package controller

import (
	"encoding/json"
	"myflowhub/pkg/database"
	"myflowhub/pkg/protocol"
	"myflowhub/server/internal/hub"
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

// HandleQueryNodes 处理节点查询请求
func (c *DeviceController) HandleQueryNodes(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	// 如提供 userKey 且已注入 authz，则返回按三来源计算的可见设备
	if c.authz != nil {
		if m, ok := msg.Payload.(map[string]interface{}); ok {
			if uk, ok := m["userKey"].(string); ok && uk != "" {
				uid, ok := c.authz.ResolveUserIDFromKey(uk)
				if ok {
					if ds, err := c.authz.VisibleDevices(uid, client.DeviceID); err == nil {
						s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": ds})
						return
					}
				}
			}
		}
	}
	// 兼容旧逻辑：管理员可见全部，否则空
	if !c.perm.CanAccessAllDevices(client.DeviceID) {
		s.SendResponse(client, msg.ID, map[string]interface{}{"success": true, "data": []database.Device{}})
		return
	}

	devices, err := c.service.GetAllDevices()
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "Failed to query nodes")
		return
	}

	// 在这里，我们可能需要一个辅助函数来构建设备树，
	// 或者我们可以直接返回设备列表，让客户端处理。
	// 为了简单起见，我们暂时直接返回列表。
	s.SendResponse(client, msg.ID, map[string]interface{}{
		"success": true,
		"data":    devices,
	})
}

// HandleCreateDevice 处理创建设备请求
func (c *DeviceController) HandleCreateDevice(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var payload database.Device
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	// 支持 userKey：
	// - 管理员（admin.manage/**）：允许自由创建
	// - 非管理员：
	//   * 不得将 OwnerUserID 设为他人；若未设置，则默认为自己
	//   * 必须指定 ParentID，且对 parent 具备控制权（设备默认/所有权其一满足）
	if c.authz != nil {
		if m, ok := msg.Payload.(map[string]interface{}); ok {
			if uk, ok := m["userKey"].(string); ok && uk != "" {
				if uid, ok := c.authz.ResolveUserIDFromKey(uk); ok {
					isAdmin := c.authz.HasUserPermission(uid, "admin.manage")
					if !isAdmin {
						// Owner 限制
						if payload.OwnerUserID != nil {
							if *payload.OwnerUserID != uid {
								s.SendErrorResponse(client, msg.ID, "permission denied")
								return
							}
						} else {
							// 默认归属创建者
							payload.OwnerUserID = &uid
						}
						// 必须指定父设备且具备控制权
						if payload.ParentID == nil {
							s.SendErrorResponse(client, msg.ID, "parent required")
							return
						}
						parent, err := c.service.GetDeviceByID(*payload.ParentID)
						if err != nil {
							s.SendErrorResponse(client, msg.ID, "parent not found")
							return
						}
						if !c.authz.CanControlDevice(client.DeviceID, parent.DeviceUID, uid) {
							s.SendErrorResponse(client, msg.ID, "permission denied")
							return
						}
					}
					goto CREATE_OK
				}
			}
		}
	}
	// 旧逻辑：仅管理员设备可创建
	if !c.perm.CanManageDevice(client.DeviceID, payload.DeviceUID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
CREATE_OK:
	if err := c.service.CreateDevice(&payload); err != nil {
		s.SendErrorResponse(client, msg.ID, "Failed to create device")
		return
	}
	if c.syslog != nil {
		_ = c.syslog.Info("device", "device created", map[string]any{"deviceUID": payload.DeviceUID, "name": payload.Name, "by": client.DeviceID})
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}

// HandleUpdateDevice 处理更新设备请求
func (c *DeviceController) HandleUpdateDevice(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var payload database.Device
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)
	// 若提供 userKey 则按三来源判定；否则退回旧的管理员判定
	if c.authz != nil {
		if m, ok := msg.Payload.(map[string]interface{}); ok {
			if uk, ok := m["userKey"].(string); ok && uk != "" {
				if uid, ok := c.authz.ResolveUserIDFromKey(uk); ok {
					isAdmin := c.authz.HasUserPermission(uid, "admin.manage")
					if !c.authz.CanControlDevice(client.DeviceID, payload.DeviceUID, uid) {
						s.SendErrorResponse(client, msg.ID, "permission denied")
						return
					}
					if !isAdmin {
						// 非管理员不得将 OwnerUserID 设为他人
						if payload.OwnerUserID != nil && *payload.OwnerUserID != uid {
							s.SendErrorResponse(client, msg.ID, "permission denied")
							return
						}
						// 非管理员如变更 ParentID，必须能控制新父节点
						if payload.ParentID != nil {
							parent, err := c.service.GetDeviceByID(*payload.ParentID)
							if err != nil {
								s.SendErrorResponse(client, msg.ID, "parent not found")
								return
							}
							if !c.authz.CanControlDevice(client.DeviceID, parent.DeviceUID, uid) {
								s.SendErrorResponse(client, msg.ID, "permission denied")
								return
							}
						}
					}
					goto UPDATE_OK
				}
			}
		}
	}
	if !c.perm.CanManageDevice(client.DeviceID, payload.DeviceUID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
UPDATE_OK:

	if err := c.service.UpdateDevice(&payload); err != nil {
		s.SendErrorResponse(client, msg.ID, "Failed to update device")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}

// HandleDeleteDevice 处理删除设备请求
func (c *DeviceController) HandleDeleteDevice(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}
	idf, ok := payload["id"].(float64)
	if !ok {
		return
	}
	id := uint64(idf)

	// 提升：支持 userKey。删除需具备 admin.manage 或者对目标设备有控制权（且自身拥有者时更安全）。
	if c.authz != nil {
		if uk, ok := payload["userKey"].(string); ok && uk != "" {
			if uid, ok := c.authz.ResolveUserIDFromKey(uk); ok {
				// 获取目标以判定其 UID
				target, err := c.service.GetDeviceByID(id)
				if err != nil {
					s.SendErrorResponse(client, msg.ID, "Device not found")
					return
				}
				// 管理员允许删除；或所有者允许删除自己设备
				if c.authz.HasUserPermission(uid, "admin.manage") {
					goto DELETE_OK
				}
				// 所有者删除自己
				if target.OwnerUserID != nil && *target.OwnerUserID == uid {
					goto DELETE_OK
				}
				// 设备默认：请求者是目标或其祖先（谨慎允许，通常删除仍建议管理员/所有者）
				if c.authz.CanControlDevice(client.DeviceID, target.DeviceUID, uid) && target.OwnerUserID != nil && *target.OwnerUserID == uid {
					goto DELETE_OK
				}
				s.SendErrorResponse(client, msg.ID, "permission denied")
				return
			}
		}
	}
	// 旧逻辑：仅管理员设备可删除
	if !c.perm.IsAdminDevice(client.DeviceID) {
		s.SendErrorResponse(client, msg.ID, "permission denied")
		return
	}
DELETE_OK:
	if err := c.service.DeleteDevice(id); err != nil {
		s.SendErrorResponse(client, msg.ID, "Failed to delete device")
		return
	}
	if c.syslog != nil {
		_ = c.syslog.Info("device", "device deleted", map[string]any{"id": id, "by": client.DeviceID})
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}
