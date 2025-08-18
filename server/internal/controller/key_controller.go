package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"myflowhub/pkg/database"
	"myflowhub/server/internal/service"
)

type KeyController struct {
	keys  *service.KeyService
	audit *service.AuditService
}

func NewKeyController(keys *service.KeyService) *KeyController {
	return &KeyController{keys: keys}
}

// 可选注入审计服务
func (c *KeyController) SetAuditService(a *service.AuditService) { c.audit = a }

// --- Business methods for binroutes ---

// List returns keys visible to the user represented by userKey.
func (c *KeyController) List(userKey string) ([]database.Key, error) {
	if userKey == "" {
		return nil, fmt.Errorf("unauthorized")
	}
	uid, _, err := c.keys.PeekUserKey(userKey)
	if err != nil || uid == 0 {
		return nil, fmt.Errorf("unauthorized")
	}
	return c.keys.ListKeys(uid)
}

// Create issues a new key bound optionally and returns the generated secret with created key and attached nodes.
func (c *KeyController) Create(userKey string, bindType *string, bindID *uint64, expiresAt *time.Time, maxUses *int32, meta []byte, nodes []string) (secret string, key *database.Key, outNodes []string, err error) {
	if userKey == "" {
		return "", nil, nil, fmt.Errorf("unauthorized")
	}
	uid, _, e := c.keys.PeekUserKey(userKey)
	if e != nil || uid == 0 {
		return "", nil, nil, fmt.Errorf("unauthorized")
	}
	// server generates 32-byte random secret
	buf := make([]byte, 32)
	if _, e := rand.Read(buf); e != nil {
		return "", nil, nil, fmt.Errorf("entropy failed")
	}
	secret = hex.EncodeToString(buf)
	var expPtr *time.Time
	if expiresAt != nil {
		v := *expiresAt
		expPtr = &v
	}
	var maxPtr *int
	if maxUses != nil {
		v := int(*maxUses)
		maxPtr = &v
	}
	k, e := c.keys.CreateKey(uid, bindType, bindID, secret, expPtr, maxPtr, meta)
	if e != nil {
		return "", nil, nil, fmt.Errorf("create failed")
	}
	if len(nodes) > 0 {
		if err := c.keys.AttachKeyPermissions(uid, k.ID, nodes); err != nil {
			return "", nil, nil, fmt.Errorf("invalid key permission nodes")
		}
	}
	return secret, k, nodes, nil
}

// Update updates a key record, with permissions checked by userKey holder.
func (c *KeyController) Update(userKey string, item *database.Key) error {
	if userKey == "" {
		return fmt.Errorf("unauthorized")
	}
	uid, _, err := c.keys.PeekUserKey(userKey)
	if err != nil || uid == 0 {
		return fmt.Errorf("unauthorized")
	}
	return c.keys.UpdateKey(uid, item)
}

// Delete deletes a key by id with permission checks.
func (c *KeyController) Delete(userKey string, id uint64) error {
	// Placeholder for future implementations
	if userKey == "" {
		return fmt.Errorf("unauthorized")
	}
	uid, _, err := c.keys.PeekUserKey(userKey)
	if err != nil || uid == 0 {
		return fmt.Errorf("unauthorized")
	}
	return c.keys.DeleteKey(uid, id)
}

// VisibleDevices lists devices the user can select when creating a key.
func (c *KeyController) VisibleDevices(userKey string) ([]database.Device, error) {
	if userKey == "" {
		return nil, fmt.Errorf("unauthorized")
	}
	uid, _, err := c.keys.PeekUserKey(userKey)
	if err != nil || uid == 0 {
		return nil, fmt.Errorf("unauthorized")
	}
	return c.keys.ListVisibleDevicesForKey(uid)
}

// HandleKeyList 按规则返回可见密钥列表

// 已移除所有 JSON Handler；仅保留业务方法供二进制路由调用
