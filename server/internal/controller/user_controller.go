package controller

import (
	"encoding/json"
	"myflowhub/pkg/protocol"
	"myflowhub/server/internal/hub"
	"myflowhub/server/internal/service"
)

type UserController struct {
	users *service.UserService
	perm  *service.PermissionService
}

func NewUserController(users *service.UserService, perm *service.PermissionService) *UserController {
	return &UserController{users: users, perm: perm}
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
