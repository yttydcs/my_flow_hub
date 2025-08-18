package controller

import (
	"fmt"
	"myflowhub/pkg/database"
	"myflowhub/server/internal/service"

	"gorm.io/datatypes"
)

// VariableController 负责处理变量相关的消息
type VariableController struct {
	service       *service.VariableService
	deviceService *service.DeviceService
	perm          *service.PermissionService
	authz         *service.AuthzService
}

// NewVariableController 创建一个新的 VariableController
func NewVariableController(service *service.VariableService, deviceService *service.DeviceService, perm *service.PermissionService, authz *service.AuthzService) *VariableController {
	return &VariableController{
		service:       service,
		deviceService: deviceService,
		perm:          perm,
		authz:         authz,
	}
}

// authzVisibleAsAdmin: 基于用户权限判断是否具备 admin.manage（或 ** 由 HasPermission 内部处理）
func (c *VariableController) authzVisibleAsAdmin(userID uint64) bool {
	if c.authz == nil || userID == 0 {
		return false
	}
	return c.authz.HasUserPermission(userID, "admin.manage")
}

// HandleVarsQuery 处理来自直接客户端的变量查询请求
// 所有 JSON 兼容 Handler 已移除，二进制专用

// Business methods (transport-agnostic) for binroutes
type VarKV struct {
	DeviceUID uint64
	Name      string
	Value     []byte
}
type VarKey struct {
	DeviceUID uint64
	Name      string
}

func (c *VariableController) List(userKey string, deviceUID *uint64, requesterDeviceUID uint64) ([]database.DeviceVariable, error) {
	var requesterUID uint64
	if c.authz != nil && userKey != "" {
		if uid, ok := c.authz.ResolveUserIDFromKey(userKey); ok {
			requesterUID = uid
		}
	}
	if deviceUID != nil {
		if requesterUID != 0 {
			if !c.authz.CanControlDevice(requesterDeviceUID, *deviceUID, requesterUID) {
				return nil, fmt.Errorf("permission denied")
			}
		} else if !c.perm.CanReadVarsForDevice(requesterDeviceUID, *deviceUID) {
			return nil, fmt.Errorf("permission denied")
		}
		dev, e := c.deviceService.GetDeviceByUID(*deviceUID)
		if e != nil {
			return nil, fmt.Errorf("device not found")
		}
		return c.service.GetVariablesByDeviceID(dev.ID)
	}
	if requesterUID == 0 || !c.authz.HasUserPermission(requesterUID, "admin.manage") {
		return nil, fmt.Errorf("permission denied")
	}
	return c.service.GetAllVariables()
}

func (c *VariableController) Update(userKey string, items []VarKV, requesterDeviceUID uint64) (int, error) {
	var uid uint64
	if c.authz != nil && userKey != "" {
		if u, ok := c.authz.ResolveUserIDFromKey(userKey); ok {
			uid = u
		}
	}
	updated := 0
	for _, it := range items {
		if uid != 0 {
			if !c.authz.CanControlDevice(requesterDeviceUID, it.DeviceUID, uid) {
				continue
			}
		} else if !c.perm.CanWriteVarsForDevice(requesterDeviceUID, it.DeviceUID) {
			continue
		}
		dev, e := c.deviceService.GetDeviceByUID(it.DeviceUID)
		if e != nil {
			continue
		}
		v := &database.DeviceVariable{OwnerDeviceID: dev.ID, VariableName: it.Name, Value: datatypes.JSON(it.Value)}
		if c.service.UpsertVariable(v) == nil {
			updated++
		}
	}
	return updated, nil
}

func (c *VariableController) Delete(userKey string, items []VarKey, requesterDeviceUID uint64) (int, error) {
	var uid uint64
	if c.authz != nil && userKey != "" {
		if u, ok := c.authz.ResolveUserIDFromKey(userKey); ok {
			uid = u
		}
	}
	deleted := 0
	for _, it := range items {
		if uid != 0 {
			if !c.authz.CanControlDevice(requesterDeviceUID, it.DeviceUID, uid) {
				continue
			}
		} else if !c.perm.CanWriteVarsForDevice(requesterDeviceUID, it.DeviceUID) {
			continue
		}
		dev, e := c.deviceService.GetDeviceByUID(it.DeviceUID)
		if e != nil {
			continue
		}
		if c.service.DeleteVariable(dev.ID, it.Name) == nil {
			deleted++
		}
	}
	return deleted, nil
}
