package controller

import (
	"encoding/json"
	"myflowhub/pkg/protocol"
	"myflowhub/server/internal/hub"
	"myflowhub/server/internal/repository"
	"myflowhub/server/internal/service"
)

type UserController struct {
	users     *service.UserService
	perm      *service.PermissionService
	permsRepo *repository.PermissionRepository
	authz     *service.AuthzService
}

func NewUserController(users *service.UserService, perm *service.PermissionService, permsRepo *repository.PermissionRepository) *UserController {
	return &UserController{users: users, perm: perm, permsRepo: permsRepo}
}

// SetAuthzService 可选注入统一授权服务
func (c *UserController) SetAuthzService(a *service.AuthzService) { c.authz = a }

func (c *UserController) HandleUserList(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	// 优先使用 userKey 判定是否具备 admin.manage；否则退回设备是否为管理员
	if c.authz != nil {
		if m, ok := msg.Payload.(map[string]interface{}); ok {
			if uk, ok := m["userKey"].(string); ok && uk != "" {
				if uid, ok := c.authz.ResolveUserIDFromKey(uk); ok && c.authz.HasUserPermission(uid, "admin.manage") {
					goto LIST_OK
				}
				s.SendErrorResponse(client, msg.ID, "permission denied")
				return
			}
		}
	}
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
LIST_OK:
	data, err := c.users.List()
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": data})
}

func (c *UserController) HandleUserCreate(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	if c.authz != nil {
		if m, ok := msg.Payload.(map[string]interface{}); ok {
			if uk, ok := m["userKey"].(string); ok && uk != "" {
				if uid, ok := c.authz.ResolveUserIDFromKey(uk); !(ok && c.authz.HasUserPermission(uid, "admin.manage")) {
					s.SendErrorResponse(client, msg.ID, "permission denied")
					return
				}
				goto CREATE_OK
			}
		}
	}
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
CREATE_OK:
	var payload struct{ Username, DisplayName, Password string }
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)
	u, err := c.users.Create(payload.Username, payload.DisplayName, payload.Password)
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": u})
}

func (c *UserController) HandleUserUpdate(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	if c.authz != nil {
		if m, ok := msg.Payload.(map[string]interface{}); ok {
			if uk, ok := m["userKey"].(string); ok && uk != "" {
				if uid, ok := c.authz.ResolveUserIDFromKey(uk); !(ok && c.authz.HasUserPermission(uid, "admin.manage")) {
					s.SendErrorResponse(client, msg.ID, "permission denied")
					return
				}
				goto UPDATE_OK
			}
		}
	}
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
UPDATE_OK:
	var payload struct {
		ID          uint64
		DisplayName *string
		Password    *string
		Disabled    *bool
	}
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)
	if err := c.users.Update(payload.ID, payload.DisplayName, payload.Password, payload.Disabled); err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}

func (c *UserController) HandleUserDelete(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	if c.authz != nil {
		if m, ok := msg.Payload.(map[string]interface{}); ok {
			if uk, ok := m["userKey"].(string); ok && uk != "" {
				if uid, ok := c.authz.ResolveUserIDFromKey(uk); !(ok && c.authz.HasUserPermission(uid, "admin.manage")) {
					s.SendErrorResponse(client, msg.ID, "permission denied")
					return
				}
				goto DELETE_OK
			}
		}
	}
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
DELETE_OK:
	var payload struct{ ID uint64 }
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)
	if err := c.users.Delete(payload.ID); err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}

// 权限管理：列出用户权限节点
func (c *UserController) HandleUserPermList(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	if c.authz != nil {
		if m, ok := msg.Payload.(map[string]interface{}); ok {
			if uk, ok := m["userKey"].(string); ok && uk != "" {
				if uid, ok := c.authz.ResolveUserIDFromKey(uk); !(ok && c.authz.HasUserPermission(uid, "admin.manage")) {
					s.SendErrorResponse(client, msg.ID, "permission denied")
					return
				}
				goto PERM_LIST_OK
			}
		}
	}
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
PERM_LIST_OK:
	var payload struct {
		UserID uint64 `json:"userId"`
	}
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)
	list, err := c.permsRepo.ListByUserID(payload.UserID)
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	// 仅返回节点数组，便于前端消费
	nodes := make([]string, 0, len(list))
	for _, p := range list {
		nodes = append(nodes, p.Node)
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": nodes})
}

// 添加用户权限节点
func (c *UserController) HandleUserPermAdd(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	if c.authz != nil {
		if m, ok := msg.Payload.(map[string]interface{}); ok {
			if uk, ok := m["userKey"].(string); ok && uk != "" {
				if uid, ok := c.authz.ResolveUserIDFromKey(uk); !(ok && c.authz.HasUserPermission(uid, "admin.manage")) {
					s.SendErrorResponse(client, msg.ID, "permission denied")
					return
				}
				goto PERM_ADD_OK
			}
		}
	}
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
PERM_ADD_OK:
	var payload struct {
		UserID uint64 `json:"userId"`
		Node   string `json:"node"`
	}
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)
	if payload.UserID == 0 || payload.Node == "" {
		s.SendErrorResponse(client, msg.ID, "invalid params")
		return
	}
	if err := c.permsRepo.AddUserNode(payload.UserID, payload.Node, nil); err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}

// 移除用户权限节点
func (c *UserController) HandleUserPermRemove(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	if c.authz != nil {
		if m, ok := msg.Payload.(map[string]interface{}); ok {
			if uk, ok := m["userKey"].(string); ok && uk != "" {
				if uid, ok := c.authz.ResolveUserIDFromKey(uk); !(ok && c.authz.HasUserPermission(uid, "admin.manage")) {
					s.SendErrorResponse(client, msg.ID, "permission denied")
					return
				}
				goto PERM_REMOVE_OK
			}
		}
	}
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
PERM_REMOVE_OK:
	var payload struct {
		UserID uint64 `json:"userId"`
		Node   string `json:"node"`
	}
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)
	if payload.UserID == 0 || payload.Node == "" {
		s.SendErrorResponse(client, msg.ID, "invalid params")
		return
	}
	if err := c.permsRepo.RemoveUserNode(payload.UserID, payload.Node); err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}
