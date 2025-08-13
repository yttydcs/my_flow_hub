package controller

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"myflowhub/pkg/database"
	"myflowhub/pkg/protocol"
	"myflowhub/server/internal/hub"
	"myflowhub/server/internal/service"
)

type KeyController struct {
	keys    *service.KeyService
	session *service.SessionService
}

func NewKeyController(keys *service.KeyService, session *service.SessionService) *KeyController {
	return &KeyController{keys: keys, session: session}
}

// HandleKeyList 按规则返回可见密钥列表
func (c *KeyController) HandleKeyList(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	// 读取 manager 转发的用户 key 或 token
	var requesterID uint64
	if m, ok := msg.Payload.(map[string]interface{}); ok {
		if uk, ok := m["userKey"].(string); ok && uk != "" {
			if uid, _, err := c.keys.ValidateUserKey(uk); err == nil {
				requesterID = uid
			}
		}
		// 回退：若未能通过 userKey 解析，则尝试 token 会话
		if requesterID == 0 {
			if t, ok := m["token"].(string); ok && c.session != nil {
				if u, ok := c.session.Resolve(t); ok {
					requesterID = u.ID
				}
			}
		}
	}
	if requesterID == 0 {
		s.SendErrorResponse(client, msg.ID, "unauthorized")
		return
	}
	list, err := c.keys.ListKeys(requesterID)
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": list})
}

// HandleKeyCreate 由服务器生成密钥，并返回给客户端一次性展示
func (c *KeyController) HandleKeyCreate(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var body struct {
		Token   string          `json:"token"`
		UserKey string          `json:"userKey"`
		BindTyp *string         `json:"bindType"`
		BindID  *uint64         `json:"bindId"`
		Expires *time.Time      `json:"expiresAt"`
		MaxUses *int            `json:"maxUses"`
		Meta    json.RawMessage `json:"meta"`
		Nodes   []string        `json:"nodes"`
	}
	b, _ := json.Marshal(msg.Payload)
	_ = json.Unmarshal(b, &body)
	var requesterID uint64
	if body.UserKey != "" {
		if uid, _, err := c.keys.ValidateUserKey(body.UserKey); err == nil {
			requesterID = uid
		}
	}
	// 回退：若未能通过 userKey 解析，则尝试 token 会话
	if requesterID == 0 && c.session != nil && body.Token != "" {
		if u, ok := c.session.Resolve(body.Token); ok {
			requesterID = u.ID
		}
	}
	if requesterID == 0 {
		s.SendErrorResponse(client, msg.ID, "unauthorized")
		return
	}

	// 服务器生成随机密钥（十六进制字符串）；生产建议仅返回一次并保存哈希
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	secret := hex.EncodeToString(random)

	k, err := c.keys.CreateKey(requesterID, body.BindTyp, body.BindID, secret, body.Expires, body.MaxUses, body.Meta)
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	if len(body.Nodes) > 0 {
		if err := c.keys.AttachKeyPermissions(requesterID, k.ID, body.Nodes); err != nil {
			s.SendErrorResponse(client, msg.ID, "invalid key permission nodes")
			return
		}
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": k, "secret": secret, "nodes": body.Nodes})
}

func (c *KeyController) HandleKeyUpdate(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var body database.Key
	b, _ := json.Marshal(msg.Payload)
	_ = json.Unmarshal(b, &body)
	var token string
	var userKey string
	if m, ok := msg.Payload.(map[string]interface{}); ok {
		if t, ok := m["token"].(string); ok {
			token = t
		}
		if k, ok := m["userKey"].(string); ok {
			userKey = k
		}
	}
	var requesterID uint64
	if userKey != "" {
		if uid, _, err := c.keys.ValidateUserKey(userKey); err == nil {
			requesterID = uid
		}
	}
	if requesterID == 0 && c.session != nil && token != "" {
		if u, ok := c.session.Resolve(token); ok {
			requesterID = u.ID
		}
	}
	if requesterID == 0 {
		s.SendErrorResponse(client, msg.ID, "unauthorized")
		return
	}
	if err := c.keys.UpdateKey(requesterID, &body); err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}

func (c *KeyController) HandleKeyDelete(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var body struct {
		Token   string `json:"token"`
		UserKey string `json:"userKey"`
		ID      uint64 `json:"id"`
	}
	b, _ := json.Marshal(msg.Payload)
	_ = json.Unmarshal(b, &body)
	var requesterID uint64
	if body.UserKey != "" {
		if uid, _, err := c.keys.ValidateUserKey(body.UserKey); err == nil {
			requesterID = uid
		}
	}
	if requesterID == 0 && c.session != nil && body.Token != "" {
		if u, ok := c.session.Resolve(body.Token); ok {
			requesterID = u.ID
		}
	}
	if requesterID == 0 {
		s.SendErrorResponse(client, msg.ID, "unauthorized")
		return
	}
	if err := c.keys.DeleteKey(requesterID, body.ID); err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}

// HandleKeyDevices 返回当前用户在创建密钥时可选择的设备集合
func (c *KeyController) HandleKeyDevices(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var body struct {
		Token   string `json:"token"`
		UserKey string `json:"userKey"`
	}
	b, _ := json.Marshal(msg.Payload)
	_ = json.Unmarshal(b, &body)
	var requesterID uint64
	if body.UserKey != "" {
		if uid, _, err := c.keys.ValidateUserKey(body.UserKey); err == nil {
			requesterID = uid
		}
	}
	if requesterID == 0 && c.session != nil && body.Token != "" {
		if u, ok := c.session.Resolve(body.Token); ok {
			requesterID = u.ID
		}
	}
	if requesterID == 0 {
		s.SendErrorResponse(client, msg.ID, "unauthorized")
		return
	}
	list, err := c.keys.ListVisibleDevicesForKey(requesterID)
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": list})
}
