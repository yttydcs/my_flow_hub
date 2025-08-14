package controller

import (
	"encoding/json"
	"myflowhub/pkg/protocol"
	"myflowhub/server/internal/hub"
	"myflowhub/server/internal/repository"
	"myflowhub/server/internal/service"
)

type LogController struct {
	audit *service.AuditService
	authz *service.AuthzService
}

func NewLogController(audit *service.AuditService) *LogController {
	return &LogController{audit: audit}
}

func (c *LogController) SetAuthzService(a *service.AuthzService) { c.authz = a }

// HandleLogList 管理端查询审计日志（需要 log.read 或 admin.manage）
func (c *LogController) HandleLogList(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	// 优先使用 userKey 做权限判断
	var userKey string
	var filter repository.AuditListFilter
	if m, ok := msg.Payload.(map[string]interface{}); ok {
		if uk, ok := m["userKey"].(string); ok {
			userKey = uk
		}
		b, _ := json.Marshal(m)
		_ = json.Unmarshal(b, &filter)
	}
	if c.authz == nil {
		s.SendErrorResponse(client, msg.ID, "not configured")
		return
	}
	uid, ok := c.authz.ResolveUserIDFromKey(userKey)
	if !ok || !(c.authz.HasUserPermission(uid, "log.read") || c.authz.HasUserPermission(uid, "admin.manage")) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
	// 查询
	page, err := c.audit.List(filter)
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": page})
}
