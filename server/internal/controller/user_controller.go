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
}

func NewUserController(users *service.UserService, perm *service.PermissionService, permsRepo *repository.PermissionRepository) *UserController {
	return &UserController{users: users, perm: perm, permsRepo: permsRepo}
}

func (c *UserController) HandleUserList(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
	data, err := c.users.List()
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": data})
}

func (c *UserController) HandleUserCreate(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
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
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
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
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
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
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
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
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
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
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
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
