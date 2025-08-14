package controller

import (
	"encoding/json"
	"myflowhub/pkg/protocol"
	"myflowhub/server/internal/hub"
	"myflowhub/server/internal/service"
)

type SystemLogController struct {
	svc   *service.SystemLogService
	authz *service.AuthzService
}

func NewSystemLogController(svc *service.SystemLogService) *SystemLogController {
	return &SystemLogController{svc: svc}
}
func (c *SystemLogController) SetAuthzService(a *service.AuthzService) { c.authz = a }

type SystemLogListRequest struct {
	Level    string `json:"level"`
	Source   string `json:"source"`
	Keyword  string `json:"keyword"`
	StartAt  *int64 `json:"startAt"`
	EndAt    *int64 `json:"endAt"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

// HandleSystemLogList 通过 hub 路由处理系统日志列表查询（需要 log.read 或 admin.manage）
func (c *SystemLogController) HandleSystemLogList(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var userKey string
	var req SystemLogListRequest
	if m, ok := msg.Payload.(map[string]interface{}); ok {
		if uk, ok2 := m["userKey"].(string); ok2 {
			userKey = uk
		}
		b, _ := json.Marshal(m)
		_ = json.Unmarshal(b, &req)
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
	out, err := c.svc.List(service.SystemLogListInput{
		Level:    req.Level,
		Source:   req.Source,
		Keyword:  req.Keyword,
		StartAt:  req.StartAt,
		EndAt:    req.EndAt,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": out})
}
