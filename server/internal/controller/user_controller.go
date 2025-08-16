package controller

import (
	"myflowhub/server/internal/repository"
	"myflowhub/server/internal/service"
)

type UserController struct {
	users     *service.UserService
	perm      *service.PermissionService
	permsRepo *repository.PermissionRepository
	authz     *service.AuthzService
	audit     *service.AuditService
}

func NewUserController(users *service.UserService, perm *service.PermissionService, permsRepo *repository.PermissionRepository) *UserController {
	return &UserController{users: users, perm: perm, permsRepo: permsRepo}
}

// SetAuthzService 可选注入统一授权服务
func (c *UserController) SetAuthzService(a *service.AuthzService) { c.authz = a }

// 可选注入审计服务
func (c *UserController) SetAuditService(a *service.AuditService) { c.audit = a }

// 所有 JSON 兼容 Handler 已移除；该控制器将通过二进制路由调用其业务服务
