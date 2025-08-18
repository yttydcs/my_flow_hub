package controller

import (
	"fmt"
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
// List is transport-agnostic method to list system logs with userKey authorization.
func (c *SystemLogController) List(userKey string, req SystemLogListRequest) (*service.SystemLogListOutput, error) {
	if c.authz == nil {
		return nil, fmt.Errorf("not configured")
	}
	authCtx, ok := c.authz.ResolveAuthContextFromKey(userKey)
	if !ok || !(c.authz.HasPermission(authCtx, "log.read") || authCtx.IsAdmin) {
		return nil, fmt.Errorf("permission denied")
	}
	return c.svc.List(service.SystemLogListInput{
		Level:    req.Level,
		Source:   req.Source,
		Keyword:  req.Keyword,
		StartAt:  req.StartAt,
		EndAt:    req.EndAt,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}
