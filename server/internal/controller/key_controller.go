package controller

import (
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
	// 读取 manager 转发的用户 token
	var requesterID uint64
	if m, ok := msg.Payload.(map[string]interface{}); ok {
		if t, ok := m["token"].(string); ok && c.session != nil {
			if u, ok := c.session.Resolve(t); ok {
				requesterID = u.ID
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

func (c *KeyController) HandleKeyCreate(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var body struct {
		Token   string          `json:"token"`
		BindTyp *string         `json:"bindType"`
		BindID  *uint64         `json:"bindId"`
		Secret  string          `json:"secret"`
		Expires *time.Time      `json:"expiresAt"`
		MaxUses *int            `json:"maxUses"`
		Meta    json.RawMessage `json:"meta"`
	}
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &body)
	if c.session == nil {
		s.SendErrorResponse(client, msg.ID, "unauthorized")
		return
	}
	u, ok := c.session.Resolve(body.Token)
	if !ok {
		s.SendErrorResponse(client, msg.ID, "unauthorized")
		return
	}
	if body.Secret == "" {
		s.SendErrorResponse(client, msg.ID, "invalid secret")
		return
	}
	// 这里简化：直接保存明文 hash 字段；生产应存储哈希
	k, err := c.keys.CreateKey(u.ID, body.BindTyp, body.BindID, body.Secret, body.Expires, body.MaxUses, body.Meta)
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": k})
}

func (c *KeyController) HandleKeyUpdate(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var body database.Key
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &body)
	var token string
	if m, ok := msg.Payload.(map[string]interface{}); ok {
		if t, ok := m["token"].(string); ok {
			token = t
		}
	}
	if c.session == nil {
		s.SendErrorResponse(client, msg.ID, "unauthorized")
		return
	}
	u, ok := c.session.Resolve(token)
	if !ok {
		s.SendErrorResponse(client, msg.ID, "unauthorized")
		return
	}
	if err := c.keys.UpdateKey(u.ID, &body); err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}

func (c *KeyController) HandleKeyDelete(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var body struct {
		Token string `json:"token"`
		ID    uint64 `json:"id"`
	}
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &body)
	if c.session == nil {
		s.SendErrorResponse(client, msg.ID, "unauthorized")
		return
	}
	u, ok := c.session.Resolve(body.Token)
	if !ok {
		s.SendErrorResponse(client, msg.ID, "unauthorized")
		return
	}
	if err := c.keys.DeleteKey(u.ID, body.ID); err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}
