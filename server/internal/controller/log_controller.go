package controller

import (
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
// JSON 兼容 Handler 已移除；保留审计服务供二进制路由使用
