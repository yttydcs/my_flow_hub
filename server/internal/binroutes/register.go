package binroutes

import (
	binproto "myflowhub/pkg/protocol/binproto"
	"myflowhub/server/internal/controller"
	"myflowhub/server/internal/hub"
	"myflowhub/server/internal/repository"
	"myflowhub/server/internal/service"
)

// RegisterAuthRoutes registers manager auth and user login/me/logout binary routes using controller adapters.
func RegisterAuthRoutes(s *hub.Server, authService *service.AuthService, deviceService *service.DeviceService, userRepo *repository.UserRepository, keyService *service.KeyService, permRepo *repository.PermissionRepository) {
	ac := controller.NewAuthController(authService, deviceService)
	ac.SetKeyService(keyService)
	ac.SetPermissionRepository(permRepo)
	ac.SetUserRepository(userRepo)
	ab := &controller.AuthBin{C: ac}
	s.RegisterBinRoute(binproto.TypeManagerAuthReq, ab.ManagerAuth)
	s.RegisterBinRoute(binproto.TypeUserLoginReq, ab.UserLogin)
	s.RegisterBinRoute(binproto.TypeUserMeReq, ab.UserMe)
	s.RegisterBinRoute(binproto.TypeUserLogoutReq, ab.UserLogout)
}

// RegisterDeviceRoutes wires device routes without anonymous handlers.
func RegisterDeviceRoutes(s *hub.Server, deviceService *service.DeviceService, permService *service.PermissionService, authzService *service.AuthzService) {
	dc := controller.NewDeviceController(deviceService, permService)
	dc.SetAuthzService(authzService)
	db := &controller.DeviceBin{C: dc}
	s.RegisterBinRoute(binproto.TypeQueryNodesReq, db.QueryNodes)
	s.RegisterBinRoute(binproto.TypeCreateDeviceReq, db.Create)
	s.RegisterBinRoute(binproto.TypeUpdateDeviceReq, db.Update)
	s.RegisterBinRoute(binproto.TypeDeleteDeviceReq, db.Delete)
}

// RegisterVariableRoutes wires variable routes using controller adapters.
func RegisterVariableRoutes(s *hub.Server, variableService *service.VariableService, deviceService *service.DeviceService, permService *service.PermissionService, authzService *service.AuthzService) {
	vc := controller.NewVariableController(variableService, deviceService, permService)
	vc.SetAuthzService(authzService)
	vb := &controller.VariableBin{C: vc}
	s.RegisterBinRoute(binproto.TypeVarUpdateReq, vb.Update)
	s.RegisterBinRoute(binproto.TypeVarDeleteReq, vb.Delete)
	s.RegisterBinRoute(binproto.TypeVarListReq, vb.List)
}

// RegisterKeyRoutes wires key CRUD routes.
func RegisterKeyRoutes(s *hub.Server, keyService *service.KeyService, permService *service.PermissionService) {
	_ = permService // reserved for future authz in controller
	kc := controller.NewKeyController(keyService)
	kb := &controller.KeyBin{C: kc}
	s.RegisterBinRoute(binproto.TypeKeyListReq, kb.List)
	s.RegisterBinRoute(binproto.TypeKeyCreateReq, kb.Create)
	s.RegisterBinRoute(binproto.TypeKeyUpdateReq, kb.Update)
	s.RegisterBinRoute(binproto.TypeKeyDeleteReq, kb.Delete)
}

// RegisterKeyDevicesRoute wires key-visible devices route.
func RegisterKeyDevicesRoute(s *hub.Server, keyService *service.KeyService) {
	kc := controller.NewKeyController(keyService)
	kb := &controller.KeyBin{C: kc}
	s.RegisterBinRoute(binproto.TypeKeyDevicesReq, kb.Devices)
}

// RegisterSystemLogRoutes wires system log listing route.
func RegisterSystemLogRoutes(s *hub.Server, keyService *service.KeyService, authzService *service.AuthzService, systemLogService *service.SystemLogService) {
	_ = keyService // not needed directly; left for parity with existing signature
	sc := controller.NewSystemLogController(systemLogService)
	sc.SetAuthzService(authzService)
	slb := &controller.SystemLogBin{C: sc}
	s.RegisterBinRoute(binproto.TypeSystemLogListReq, slb.List)
}

// RegisterUserRoutes wires user management routes (admin + self operations).
func RegisterUserRoutes(s *hub.Server, userService *service.UserService, permService *service.PermissionService, permRepo *repository.PermissionRepository, authzService *service.AuthzService) {
	uc := controller.NewUserController(userService, permService, permRepo)
	uc.SetAuthzService(authzService)
	ub := &controller.UserBin{Users: uc}
	// Admin
	s.RegisterBinRoute(binproto.TypeUserListReq, ub.List)
	s.RegisterBinRoute(binproto.TypeUserCreateReq, ub.Create)
	s.RegisterBinRoute(binproto.TypeUserUpdateReq, ub.Update)
	s.RegisterBinRoute(binproto.TypeUserDeleteReq, ub.Delete)
	s.RegisterBinRoute(binproto.TypeUserPermListReq, ub.PermList)
	s.RegisterBinRoute(binproto.TypeUserPermAddReq, ub.PermAdd)
	s.RegisterBinRoute(binproto.TypeUserPermRemoveReq, ub.PermRemove)
	// Self
	s.RegisterBinRoute(binproto.TypeUserSelfUpdateReq, ub.SelfUpdate)
	s.RegisterBinRoute(binproto.TypeUserSelfPasswordReq, ub.SelfPassword)
}

// RegisterParentAuth 路由
func RegisterParentAuth(s *hub.Server) {
	pc := controller.NewParentAuthController()
	pb := &controller.ParentAuthBin{C: pc}
	s.RegisterBinRoute(binproto.TypeParentAuthReq, pb.Handle)
}
