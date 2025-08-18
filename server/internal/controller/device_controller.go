package controller

import (
	"fmt"
	"myflowhub/pkg/database"
	"myflowhub/server/internal/service"
)

// DeviceController 负责处理设备相关的消息
type DeviceController struct {
	service *service.DeviceService
	perm    *service.PermissionService
	authz   *service.AuthzService
	syslog  *service.SystemLogService
}

// NewDeviceController 创建一个新的 DeviceController
func NewDeviceController(service *service.DeviceService, perm *service.PermissionService) *DeviceController {
	return &DeviceController{service: service, perm: perm}
}

// SetAuthzService 可选注入统一授权服务
func (c *DeviceController) SetAuthzService(a *service.AuthzService)         { c.authz = a }
func (c *DeviceController) SetSystemLogService(s *service.SystemLogService) { c.syslog = s }

// Business methods for binary routes (transport-agnostic)
func (c *DeviceController) QueryVisibleDevices(userKey string, requesterDeviceUID uint64) ([]database.Device, error) {
	// 三来源优先
	if c.authz != nil && userKey != "" {
		if authCtx, ok := c.authz.ResolveAuthContextFromKey(userKey); ok {
			ds, err := c.authz.VisibleDevices(authCtx, requesterDeviceUID)
			if err != nil {
				return nil, err
			}
			return ds, nil
		}
		return nil, fmt.Errorf("unauthorized")
	}
	// 旧逻辑：管理员设备可见全部，否则空
	if !c.perm.CanAccessAllDevices(requesterDeviceUID) {
		return []database.Device{}, nil
	}
	return c.service.GetAllDevices()
}

func (c *DeviceController) CreateDevice(userKey string, item database.Device, requesterDeviceUID uint64) error {
	if c.authz != nil && userKey != "" {
		if authCtx, ok := c.authz.ResolveAuthContextFromKey(userKey); ok {
			if !authCtx.IsAdmin {
				if item.OwnerUserID != nil {
					if *item.OwnerUserID != authCtx.UserID {
						return fmt.Errorf("permission denied")
					}
				} else {
					item.OwnerUserID = &authCtx.UserID
				}
				if item.ParentID == nil {
					return fmt.Errorf("parent required")
				}
				parent, err := c.service.GetDeviceByID(*item.ParentID)
				if err != nil {
					return fmt.Errorf("parent not found")
				}
				if !c.authz.CanControlDevice(requesterDeviceUID, parent.DeviceUID, authCtx) {
					return fmt.Errorf("permission denied")
				}
			}
			return c.service.CreateDevice(&item)
		}
		return fmt.Errorf("unauthorized")
	}
	if !c.perm.CanManageDevice(requesterDeviceUID, item.DeviceUID) {
		return fmt.Errorf("permission denied")
	}
	return c.service.CreateDevice(&item)
}

func (c *DeviceController) UpdateDevice(userKey string, item database.Device, requesterDeviceUID uint64) error {
	if c.authz != nil && userKey != "" {
		if authCtx, ok := c.authz.ResolveAuthContextFromKey(userKey); ok {
			if !c.authz.CanControlDevice(requesterDeviceUID, item.DeviceUID, authCtx) {
				return fmt.Errorf("permission denied")
			}
			if !authCtx.IsAdmin {
				if item.OwnerUserID != nil && *item.OwnerUserID != authCtx.UserID {
					return fmt.Errorf("permission denied")
				}
				if item.ParentID != nil {
					parent, err := c.service.GetDeviceByID(*item.ParentID)
					if err != nil {
						return fmt.Errorf("parent not found")
					}
					if !c.authz.CanControlDevice(requesterDeviceUID, parent.DeviceUID, authCtx) {
						return fmt.Errorf("permission denied")
					}
				}
			}
			return c.service.UpdateDevice(&item)
		}
		return fmt.Errorf("unauthorized")
	}
	if !c.perm.CanManageDevice(requesterDeviceUID, item.DeviceUID) {
		return fmt.Errorf("permission denied")
	}
	return c.service.UpdateDevice(&item)
}

func (c *DeviceController) DeleteDevice(userKey string, id uint64, requesterDeviceUID uint64) error {
	if c.authz != nil && userKey != "" {
		if authCtx, ok := c.authz.ResolveAuthContextFromKey(userKey); ok {
			target, err := c.service.GetDeviceByID(id)
			if err != nil {
				return fmt.Errorf("not found")
			}
			if authCtx.IsAdmin {
				return c.service.DeleteDevice(id)
			}
			if target.OwnerUserID != nil && *target.OwnerUserID == authCtx.UserID {
				return c.service.DeleteDevice(id)
			}
			if c.authz.CanControlDevice(requesterDeviceUID, target.DeviceUID, authCtx) && target.OwnerUserID != nil && *target.OwnerUserID == authCtx.UserID {
				return c.service.DeleteDevice(id)
			}
			return fmt.Errorf("permission denied")
		}
		return fmt.Errorf("unauthorized")
	}
	if !c.perm.IsAdminDevice(requesterDeviceUID) {
		return fmt.Errorf("permission denied")
	}
	return c.service.DeleteDevice(id)
}

// HandleQueryNodes 处理节点查询请求
// 所有 JSON 兼容 Handler 已移除，二进制专用
