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
	authz *service.AuthzService
}

func NewKeyController(keys *service.KeyService, authz *service.AuthzService) *KeyController {
	return &KeyController{keys: keys, authz: authz}
}

// 可选注入审计服务
func (c *KeyController) SetAuditService(a *service.AuditService) { c.audit = a }

// --- Business methods for binroutes ---

// List returns keys visible to the user represented by userKey.
func (c *KeyController) List(userKey string) ([]database.Key, error) {
	authCtx, ok := c.authz.ResolveAuthContextFromKey(userKey)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}
	// 权限检查：需要 "key.read" 或管理员权限
	if !c.authz.HasPermission(authCtx, "key.read") {
		return nil, fmt.Errorf("forbidden")
	}
	return c.keys.ListKeys(authCtx.UserID)
}

// Create issues a new key bound optionally and returns the generated secret with created key and attached nodes.
func (c *KeyController) Create(userKey string, bindType *string, bindID *uint64, expiresAt *time.Time, maxUses *int32, meta []byte, nodes []string) (secret string, key *database.Key, outNodes []string, err error) {
	authCtx, ok := c.authz.ResolveAuthContextFromKey(userKey)
	if !ok {
		return "", nil, nil, fmt.Errorf("unauthorized")
	}
	// 权限检查：需要 "key.create" 或管理员权限
	if !c.authz.HasPermission(authCtx, "key.create") {
		return "", nil, nil, fmt.Errorf("forbidden")
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
	k, e := c.keys.CreateKey(authCtx.UserID, bindType, bindID, secret, expPtr, maxPtr, meta)
	if e != nil {
		return "", nil, nil, fmt.Errorf("create failed")
	}
	if len(nodes) > 0 {
		if err := c.keys.AttachKeyPermissions(authCtx.UserID, k.ID, nodes); err != nil {
			return "", nil, nil, fmt.Errorf("invalid key permission nodes: %w", err)
		}
	}
	return secret, k, nodes, nil
}

// Update updates a key record, with permissions checked by userKey holder.
func (c *KeyController) Update(userKey string, item *database.Key) error {
	authCtx, ok := c.authz.ResolveAuthContextFromKey(userKey)
	if !ok {
		return fmt.Errorf("unauthorized")
	}
	// 权限检查：需要 "key.update" 或管理员权限
	if !c.authz.HasPermission(authCtx, "key.update") {
		return fmt.Errorf("forbidden")
	}
	return c.keys.UpdateKey(authCtx.UserID, item)
}

// Delete deletes a key by id with permission checks.
func (c *KeyController) Delete(userKey string, id uint64) error {
	authCtx, ok := c.authz.ResolveAuthContextFromKey(userKey)
	if !ok {
		return fmt.Errorf("unauthorized")
	}
	// 权限检查：需要 "key.delete" 或管理员权限
	if !c.authz.HasPermission(authCtx, "key.delete") {
		return fmt.Errorf("forbidden")
	}
	return c.keys.DeleteKey(authCtx.UserID, id)
}

// VisibleDevices lists devices the user can select when creating a key.
func (c *KeyController) VisibleDevices(userKey string) ([]database.Device, error) {
	authCtx, ok := c.authz.ResolveAuthContextFromKey(userKey)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}
	// 使用 authCtx 来获取可见设备
	return c.authz.VisibleDevices(authCtx, 0)
}

// HandleKeyList 按规则返回可见密钥列表

// 已移除所有 JSON Handler；仅保留业务方法供二进制路由调用
