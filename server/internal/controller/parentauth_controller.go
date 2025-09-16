package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"strings"
	"time"

	"myflowhub/pkg/config"
	"myflowhub/pkg/database"
	bin "myflowhub/pkg/protocol/binproto"
	"myflowhub/server/internal/hub"
)

// ParentAuthController 处理父链路二进制认证
type ParentAuthController struct {
	Nonces    map[string]int64
	NoncesTTL time.Duration
}

func NewParentAuthController() *ParentAuthController {
	return &ParentAuthController{Nonces: make(map[string]int64), NoncesTTL: 10 * time.Minute}
}

// VerifyAndAssign 校验请求并返回分配的设备 UID
func (p *ParentAuthController) VerifyAndAssign(reqTsMs int64, nonce [16]byte, hardwareID, caps string) (uint64, error) {
	nowMs := time.Now().UnixMilli()
	if d := nowMs - reqTsMs; d > int64(5*time.Minute/time.Millisecond) || d < -int64(5*time.Minute/time.Millisecond) {
		return 0, ErrBadTimeWindow
	}
	if hardwareID == "" {
		return 0, ErrBadRequest
	}
	// nonce 去重
	for k, v := range p.Nonces {
		if nowMs-v > int64(p.NoncesTTL/time.Millisecond) {
			delete(p.Nonces, k)
		}
	}
	key := string(nonce[:])
	if _, exists := p.Nonces[key]; exists {
		return 0, ErrReplay
	}
	p.Nonces[key] = nowMs

	// HMAC 校验：实际签名比对在 Bin 适配器中完成；此处仅进行设备登记
	return p.ensureDevice(hardwareID, caps)
}

func (p *ParentAuthController) ensureDevice(hardwareID, caps string) (uint64, error) {
	var dev database.Device
	if err := database.DB.Where("hardware_id = ?", hardwareID).First(&dev).Error; err != nil {
		// 新登记的设备默认未审批，等待 Manager 审批后才能正常使用
		// 角色：若 caps 中包含 "relay" 则为中继，否则按普通节点处理
		role := database.RoleNode
		if strings.Contains(strings.ToLower(caps), "relay") {
			role = database.RoleRelay
		}
		dev = database.Device{HardwareID: hardwareID, Role: role, Name: hardwareID, Approved: false}
		if e2 := database.DB.Create(&dev).Error; e2 != nil {
			return 0, e2
		}
		// 确保分配稳定的 DeviceUID（避免 0 导致 Hub 审批查询失配）
		if dev.DeviceUID == 0 {
			dev.DeviceUID = dev.ID
			_ = database.DB.Model(&dev).Update("device_uid", dev.DeviceUID).Error
		}
		return dev.DeviceUID, nil
	}
	// 兼容旧记录：若 DeviceUID 仍为 0，则回填为主键 ID
	if dev.DeviceUID == 0 {
		dev.DeviceUID = dev.ID
		_ = database.DB.Model(&dev).Update("device_uid", dev.DeviceUID).Error
	}
	return dev.DeviceUID, nil
}

// 错误
var (
	ErrBadTimeWindow = &binErr{"time window exceeded"}
	ErrBadRequest    = &binErr{"bad request"}
	ErrReplay        = &binErr{"replay detected"}
)

type binErr struct{ s string }

func (e *binErr) Error() string { return e.s }

// Adapter 层：解析/封包
type ParentAuthBin struct{ C *ParentAuthController }

func (p *ParentAuthBin) Handle(s *hub.Server, c *hub.Client, h bin.HeaderV1, payload []byte) {
	version, tsMs, nonce, hardwareID, caps, sig, err := bin.DecodeParentAuthReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	_ = version
	// 计算 HMAC 并比对
	var tsBuf [8]byte
	binary.LittleEndian.PutUint64(tsBuf[:], uint64(tsMs))
	// 优先使用专用 RelayToken，其次回退 ManagerToken 以兼容
	key := config.AppConfig.Server.RelayToken
	if key == "" {
		key = config.AppConfig.Server.ManagerToken
	}
	mac := computeHMACSHA256([]byte(key), tsBuf[:], nonce[:], []byte(hardwareID), []byte(caps))
	if !hmac.Equal(sig[:], mac[:]) {
		sendErr(s, c, h, 401, "invalid signature")
		return
	}
	uid, e := p.C.VerifyAndAssign(tsMs, nonce, hardwareID, caps)
	if e != nil {
		sendErr(s, c, h, 400, e.Error())
		return
	}
	// 未审批设备：禁止加入网络与消息发送
	var dev database.Device
	if err := database.DB.Where("device_uid = ? OR id = ?", uid, uid).First(&dev).Error; err == nil {
		if !dev.Approved {
			sendErr(s, c, h, 403, "device not approved")
			return
		}
	}
	var sid [16]byte
	// 成功后将连接标记为该设备，加入 Hub 客户端表
	c.DeviceID = uid
	s.Clients[c.DeviceID] = c
	pl := bin.EncodeParentAuthResp(h.MsgID, uid, sid, 30, nil, 0, [32]byte{})
	sendFrame(s, c, h, bin.TypeParentAuthResp, pl)
}

// 使用本地计算逻辑的 HMAC-SHA256
func computeHMACSHA256(key []byte, data ...[]byte) [32]byte {
	h := hmac.New(sha256.New, key)
	for _, d := range data {
		h.Write(d)
	}
	var mac [32]byte
	copy(mac[:], h.Sum(nil))
	return mac
}
