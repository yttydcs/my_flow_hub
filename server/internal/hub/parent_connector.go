package hub

import (
	"crypto/rand"
	"encoding/binary"
	"myflowhub/pkg/config"
	bin "myflowhub/pkg/protocol/binproto"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// connectToParent establishes and maintains the connection to the parent server.
func (s *Server) connectToParent() {
	u, err := url.Parse(s.ParentAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("无效的上级服务器地址")
	}

	for { // Main reconnect loop
		log.Info().Str("address", u.String()).Msg("正在连接到上级服务器...")
		dialer := *websocket.DefaultDialer
		// 请求协商二进制子协议
		dialer.Subprotocols = []string{"myflowhub.bin.v1"}
		conn, _, err := dialer.Dial(u.String(), nil)
		if err != nil {
			log.Error().Err(err).Msg("连接上级失败，将在5秒后重试")
			time.Sleep(5 * time.Second)
			continue
		}

		// Authenticate with the parent (binary ManagerAuth as MVP)
		if !s.authenticateWithParent(conn) {
			// 认证失败，关闭并重试
			_ = conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		// Start reader and writer pumps
		done := make(chan struct{})
		go s.writePumpToParent(conn, done)
		go s.readPumpFromParent(conn, done)

		<-done // Wait until a pump fails
		log.Warn().Msg("与上级的连接已断开，准备重连...")
		conn.Close()
		time.Sleep(5 * time.Second)
	}
}

// readPumpFromParent handles reading messages from the parent.
func (s *Server) readPumpFromParent(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			log.Error().Err(err).Msg("从上级读取消息失败")
			return
		}
		log.Info().Msg("收到来自上级的消息(二进制未实现，忽略)")
		// Here we could handle messages from parent, e.g., route them to our children
	}
}

// writePumpToParent handles writing messages to the parent.
func (s *Server) writePumpToParent(conn *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-s.ParentSend:
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				log.Error().Err(err).Msg("向上级写入消息失败")
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error().Err(err).Msg("向上级发送心跳失败")
				return
			}
		case <-done:
			return
		}
	}
}

// authenticateWithParent sends an authentication request to the parent.
func (s *Server) authenticateWithParent(conn *websocket.Conn) bool {
	// 优先使用 ParentAuth（二进制 HMAC 握手）；失败时回退到 ManagerAuth
	// 取 token 优先级：Relay.SharedToken > Server.RelayToken > Server.ManagerToken（兼容旧配置）
	token := config.AppConfig.Relay.SharedToken
	if token == "" {
		token = config.AppConfig.Server.RelayToken
	}
	if token == "" {
		token = config.AppConfig.Server.ManagerToken
	}
	if token == "" {
		log.Error().Msg("父链路认证失败：未配置 Relay/Shared/Manager Token")
		return false
	}

	// 构造 ParentAuthReq
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		log.Error().Err(err).Msg("生成 nonce 失败")
		return false
	}
	tsMs := time.Now().UnixMilli()
	var tsBuf [8]byte
	binary.LittleEndian.PutUint64(tsBuf[:], uint64(tsMs))
	// caps 可根据需要扩展，这里传递一个简单标识
	caps := "relay"
	mac := computeHMACSHA256([]byte(token), tsBuf[:], nonce[:], []byte(s.HardwareID), []byte(caps))
	msgID := uint64(time.Now().UnixNano())
	pl := bin.EncodeParentAuthReq(1, tsMs, nonce, s.HardwareID, caps, mac)
	h := bin.HeaderV1{TypeID: bin.TypeParentAuthReq, MsgID: msgID, Source: s.DeviceID, Target: 0, Timestamp: time.Now().UnixMilli()}
	frame, err := bin.EncodeFrame(h, pl)
	if err != nil {
		log.Error().Err(err).Msg("编码 ParentAuth 帧失败")
		return false
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		log.Error().Err(err).Msg("发送 ParentAuth 请求失败")
		return false
	}
	// 等待响应
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	mt, msg, err := conn.ReadMessage()
	if err != nil {
		log.Error().Err(err).Msg("读取 ParentAuth 响应失败")
		return false
	}
	if mt != websocket.BinaryMessage {
		log.Error().Msg("ParentAuth 响应不是二进制帧")
		return false
	}
	rh, rpl, err := bin.DecodeFrame(msg)
	if err != nil {
		log.Error().Err(err).Msg("解析 ParentAuth 响应帧失败")
		return false
	}
	switch rh.TypeID {
	case bin.TypeParentAuthResp:
		_, deviceUID, _, _, _, _, _, derr := bin.DecodeParentAuthResp(rpl)
		if derr != nil {
			log.Error().Err(derr).Msg("解码 ParentAuth 响应失败")
			return false
		}
		// 记录分配的 DeviceID，用于后续作为 Source 标识
		if deviceUID != 0 {
			s.DeviceID = deviceUID
		}
		log.Info().Uint64("deviceUID", deviceUID).Msg("父链路 ParentAuth 认证成功")
		return true
	case bin.TypeErrResp:
		_, code, msgb, e := bin.DecodeErrResp(rpl)
		if e != nil {
			log.Error().Err(e).Msg("读取 ParentAuth 错误响应失败")
			return false
		}
		log.Error().Int32("code", code).Msgf("ParentAuth 被拒绝：%s", string(msgb))
		return false
	case bin.TypeManagerAuthResp:
		// 兼容：如果上级仍返回旧的 ManagerAuthResp
		_, deviceUID, role, derr := bin.DecodeManagerAuthResp(rpl)
		if derr != nil {
			log.Error().Err(derr).Msg("解码兼容的 ManagerAuth 响应失败")
			return false
		}
		if deviceUID != 0 {
			s.DeviceID = deviceUID
		}
		log.Info().Uint64("deviceUID", deviceUID).Str("role", role).Msg("父链路使用兼容 ManagerAuth 认证成功")
		return true
	default:
		// 回退到旧协议尝试一次
		log.Warn().Uint16("typeID", rh.TypeID).Msg("ParentAuth 收到未知类型响应，尝试回退 ManagerAuth")
	}

	// 回退：ManagerAuth
	payload := bin.EncodeManagerAuthReq(token)
	header := bin.HeaderV1{TypeID: bin.TypeManagerAuthReq, MsgID: msgID + 1, Source: s.DeviceID, Target: 0, Timestamp: time.Now().UnixMilli()}
	frame2, err := bin.EncodeFrame(header, payload)
	if err != nil {
		log.Error().Err(err).Msg("编码回退 ManagerAuth 帧失败")
		return false
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, frame2); err != nil {
		log.Error().Err(err).Msg("发送回退 ManagerAuth 请求失败")
		return false
	}
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	mt2, msg2, err := conn.ReadMessage()
	if err != nil {
		log.Error().Err(err).Msg("读取回退 ManagerAuth 响应失败")
		return false
	}
	if mt2 != websocket.BinaryMessage {
		log.Error().Msg("回退 ManagerAuth 响应不是二进制帧")
		return false
	}
	h2, pl2, err := bin.DecodeFrame(msg2)
	if err != nil {
		log.Error().Err(err).Msg("解析回退 ManagerAuth 响应帧失败")
		return false
	}
	if h2.TypeID == bin.TypeManagerAuthResp {
		_, deviceUID, role, err := bin.DecodeManagerAuthResp(pl2)
		if err != nil {
			log.Error().Err(err).Msg("解码回退 ManagerAuth 负载失败")
			return false
		}
		if deviceUID != 0 {
			s.DeviceID = deviceUID
		}
		log.Info().Uint64("deviceUID", deviceUID).Str("role", role).Msg("父链路回退 ManagerAuth 认证成功")
		return true
	}
	if h2.TypeID == bin.TypeErrResp {
		_, code, msgb, e := bin.DecodeErrResp(pl2)
		if e != nil {
			log.Error().Err(e).Msg("读取回退 ManagerAuth 错误响应失败")
			return false
		}
		log.Error().Int32("code", code).Msgf("回退 ManagerAuth 被拒绝：%s", string(msgb))
		return false
	}
	log.Error().Uint16("typeID", h2.TypeID).Msg("回退 ManagerAuth 收到未知类型响应")
	return false
}
