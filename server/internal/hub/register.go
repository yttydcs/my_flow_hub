package hub

import (
	bin "myflowhub/pkg/protocol/binproto"
)

// BinHandler 是二进制消息处理器签名。
type BinHandler func(s *Server, c *Client, h bin.HeaderV1, payload []byte)

// RegisterAuthRoutes 注册认证与用户自服务相关路由（传入具体处理器函数）。
func RegisterAuthRoutes(s *Server, managerAuth, userLogin, userMe, userLogout BinHandler) {
	if managerAuth != nil {
		s.RegisterBinRoute(bin.TypeManagerAuthReq, managerAuth)
	}
	if userLogin != nil {
		s.RegisterBinRoute(bin.TypeUserLoginReq, userLogin)
	}
	if userMe != nil {
		s.RegisterBinRoute(bin.TypeUserMeReq, userMe)
	}
	if userLogout != nil {
		s.RegisterBinRoute(bin.TypeUserLogoutReq, userLogout)
	}
}

// RegisterDeviceRoutes 注册设备相关路由。
func RegisterDeviceRoutes(s *Server, queryNodes, create, update, deleteH BinHandler) {
	if queryNodes != nil {
		s.RegisterBinRoute(bin.TypeQueryNodesReq, queryNodes)
	}
	if create != nil {
		s.RegisterBinRoute(bin.TypeCreateDeviceReq, create)
	}
	if update != nil {
		s.RegisterBinRoute(bin.TypeUpdateDeviceReq, update)
	}
	if deleteH != nil {
		s.RegisterBinRoute(bin.TypeDeleteDeviceReq, deleteH)
	}
}

// RegisterVariableRoutes 注册变量相关路由。
func RegisterVariableRoutes(s *Server, update, deleteH, list BinHandler) {
	if update != nil {
		s.RegisterBinRoute(bin.TypeVarUpdateReq, update)
	}
	if deleteH != nil {
		s.RegisterBinRoute(bin.TypeVarDeleteReq, deleteH)
	}
	if list != nil {
		s.RegisterBinRoute(bin.TypeVarListReq, list)
	}
}

// RegisterKeyRoutes 注册密钥 CRUD 路由。
func RegisterKeyRoutes(s *Server, list, create, update, deleteH BinHandler) {
	if list != nil {
		s.RegisterBinRoute(bin.TypeKeyListReq, list)
	}
	if create != nil {
		s.RegisterBinRoute(bin.TypeKeyCreateReq, create)
	}
	if update != nil {
		s.RegisterBinRoute(bin.TypeKeyUpdateReq, update)
	}
	if deleteH != nil {
		s.RegisterBinRoute(bin.TypeKeyDeleteReq, deleteH)
	}
}

// RegisterKeyDevicesRoute 注册密钥可见设备列表路由。
func RegisterKeyDevicesRoute(s *Server, devices BinHandler) {
	if devices != nil {
		s.RegisterBinRoute(bin.TypeKeyDevicesReq, devices)
	}
}

// RegisterSystemLogRoutes 注册系统日志路由。
func RegisterSystemLogRoutes(s *Server, list BinHandler) {
	if list != nil {
		s.RegisterBinRoute(bin.TypeSystemLogListReq, list)
	}
}

// RegisterUserRoutes 注册用户管理（管理员 + 自助）路由。
func RegisterUserRoutes(
	s *Server,
	list, create, update, deleteH BinHandler,
	permList, permAdd, permRemove BinHandler,
	selfUpdate, selfPassword BinHandler,
) {
	if list != nil {
		s.RegisterBinRoute(bin.TypeUserListReq, list)
	}
	if create != nil {
		s.RegisterBinRoute(bin.TypeUserCreateReq, create)
	}
	if update != nil {
		s.RegisterBinRoute(bin.TypeUserUpdateReq, update)
	}
	if deleteH != nil {
		s.RegisterBinRoute(bin.TypeUserDeleteReq, deleteH)
	}
	if permList != nil {
		s.RegisterBinRoute(bin.TypeUserPermListReq, permList)
	}
	if permAdd != nil {
		s.RegisterBinRoute(bin.TypeUserPermAddReq, permAdd)
	}
	if permRemove != nil {
		s.RegisterBinRoute(bin.TypeUserPermRemoveReq, permRemove)
	}
	if selfUpdate != nil {
		s.RegisterBinRoute(bin.TypeUserSelfUpdateReq, selfUpdate)
	}
	if selfPassword != nil {
		s.RegisterBinRoute(bin.TypeUserSelfPasswordReq, selfPassword)
	}
}

// RegisterParentAuth 注册父链路认证路由。
func RegisterParentAuth(s *Server, handle BinHandler) {
	if handle != nil {
		s.RegisterBinRoute(bin.TypeParentAuthReq, handle)
	}
}
