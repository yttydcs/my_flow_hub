package controller

import (
	"encoding/json"
	"fmt"
	"myflowhub/pkg/database"
	"myflowhub/pkg/protocol"
	"myflowhub/server/internal/hub"
	"myflowhub/server/internal/service"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
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
func NewVariableController(service *service.VariableService, deviceService *service.DeviceService, perm *service.PermissionService) *VariableController {
	return &VariableController{
		service:       service,
		deviceService: deviceService,
		perm:          perm,
	}
}

// SetAuthzService 可选注入统一授权服务
func (c *VariableController) SetAuthzService(a *service.AuthzService) { c.authz = a }

// authzVisibleAsAdmin: 基于用户权限判断是否具备 admin.manage（或 ** 由 HasPermission 内部处理）
func (c *VariableController) authzVisibleAsAdmin(userID uint64) bool {
	if c.authz == nil || userID == 0 {
		return false
	}
	return c.authz.HasUserPermission(userID, "admin.manage")
}

// HandleVarsQuery 处理来自直接客户端的变量查询请求
func (c *VariableController) HandleVarsQuery(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var payload protocol.VarsQueryPayload
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	results := make([]interface{}, len(payload.Queries))
	for i, query := range payload.Queries {
		var deviceIdentifier, varName string
		if strings.Contains(query, ".") {
			parts := strings.SplitN(query, ".", 2)
			deviceIdentifier = parts[0]
			varName = parts[1]
		} else {
			deviceIdentifier = fmt.Sprintf("[%d]", client.DeviceID)
			varName = query
		}

		variable, err := c.service.GetVariableByDeviceUIDAndVarName(deviceIdentifier, varName)
		if err != nil {
			results[i] = nil
			continue
		}

		var val interface{}
		json.Unmarshal(variable.Value, &val)
		results[i] = val
	}

	s.SendResponse(client, msg.ID, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"results": results,
		},
	})
}

// HandleQueryVariables 处理来自 manager 的变量查询请求
func (c *VariableController) HandleQueryVariables(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		s.SendErrorResponse(client, msg.ID, "Invalid payload format")
		return
	}
	// 从 userKey 解析用户ID（可选）
	var requesterUID uint64
	if c.authz != nil {
		if uk, ok := payload["userKey"].(string); ok && uk != "" {
			if uid, ok := c.authz.ResolveUserIDFromKey(uk); ok {
				requesterUID = uid
			}
		}
	}

	if deviceIDStr, ok := payload["deviceId"].(string); ok && deviceIDStr != "" {
		// 按设备ID查询
		deviceID, err := strconv.ParseUint(deviceIDStr, 10, 64)
		if err != nil {
			s.SendErrorResponse(client, msg.ID, "Invalid deviceId")
			return
		}
		// 三来源校验
		if c.authz != nil {
			if !c.authz.CanControlDevice(client.DeviceID, deviceID, requesterUID) {
				s.SendErrorResponse(client, msg.ID, "permission denied")
				return
			}
		} else if !c.perm.CanReadVarsForDevice(client.DeviceID, deviceID) {
			s.SendErrorResponse(client, msg.ID, "permission denied")
			return
		}
		device, err := c.deviceService.GetDeviceByUID(deviceID)
		if err != nil {
			s.SendErrorResponse(client, msg.ID, "Device not found")
			return
		}
		variables, err := c.service.GetVariablesByDeviceID(device.ID)
		if err != nil {
			s.SendErrorResponse(client, msg.ID, "Failed to get variables")
			return
		}
		s.SendResponse(client, msg.ID, map[string]interface{}{
			"success": true,
			"data":    variables,
		})
	} else {
		// 查询所有变量
		if c.authz != nil {
			// 读取全部变量仅允许管理员用户（admin.manage/**）
			if requesterUID == 0 || !c.authz.CanControlDevice(client.DeviceID, client.DeviceID, requesterUID) || !c.authzVisibleAsAdmin(requesterUID) {
				s.SendErrorResponse(client, msg.ID, "permission denied")
				return
			}
		} else if !c.perm.IsAdminDevice(client.DeviceID) {
			s.SendErrorResponse(client, msg.ID, "permission denied")
			return
		}
		variables, err := c.service.GetAllVariables()
		if err != nil {
			s.SendErrorResponse(client, msg.ID, "Failed to get all variables")
			return
		}
		s.SendResponse(client, msg.ID, map[string]interface{}{
			"success": true,
			"data":    variables,
		})
	}
}

// HandleVarDelete 处理变量删除请求
func (c *VariableController) HandleVarDelete(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}
	// 从 userKey 解析用户ID（可选）
	var requesterUID uint64
	if c.authz != nil {
		if uk, ok := payload["userKey"].(string); ok && uk != "" {
			if uid, ok := c.authz.ResolveUserIDFromKey(uk); ok {
				requesterUID = uid
			}
		}
	}

	variables, ok := payload["variables"].([]interface{})
	if !ok {
		return
	}

	var deletedCount int
	for _, v := range variables {
		fqdn, _ := v.(string)
		var deviceIdentifier, varName string
		if strings.Contains(fqdn, ".") {
			parts := strings.SplitN(fqdn, ".", 2)
			deviceIdentifier = parts[0]
			varName = parts[1]
		} else {
			deviceIdentifier = fmt.Sprintf("[%d]", client.DeviceID)
			varName = fqdn
		}

		targetDevice, err := c.deviceService.GetDeviceByUIDOrName(deviceIdentifier)
		if err != nil {
			continue
		}

		if c.authz != nil {
			if !c.authz.CanControlDevice(client.DeviceID, targetDevice.DeviceUID, requesterUID) {
				continue
			}
		} else if !c.perm.CanWriteVarsForDevice(client.DeviceID, targetDevice.DeviceUID) {
			continue
		}

		if c.service.DeleteVariable(targetDevice.ID, varName) == nil {
			deletedCount++
		}
	}
	log.Info().Int("count", deletedCount).Msg("变量已删除")
}

// HandleVarUpdate 处理变量更新请求
func (c *VariableController) HandleVarUpdate(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}
	// 从 userKey 解析用户ID（可选）
	var requesterUID uint64
	if c.authz != nil {
		if uk, ok := payload["userKey"].(string); ok && uk != "" {
			if uid, ok := c.authz.ResolveUserIDFromKey(uk); ok {
				requesterUID = uid
			}
		}
	}

	variables, ok := payload["variables"].(map[string]interface{})
	if !ok {
		return
	}

	var updatedCount int
	for fqdn, value := range variables {
		var deviceIdentifier, varName string
		if strings.Contains(fqdn, ".") {
			parts := strings.SplitN(fqdn, ".", 2)
			deviceIdentifier = parts[0]
			varName = parts[1]
		} else {
			deviceIdentifier = fmt.Sprintf("[%d]", client.DeviceID)
			varName = fqdn
		}

		if !hub.IsValidVarName(varName) {
			log.Warn().Str("varName", varName).Msg("无效的变量名，已跳过")
			continue
		}

		targetDevice, err := c.deviceService.GetDeviceByUIDOrName(deviceIdentifier)
		if err != nil {
			continue
		}

		if c.authz != nil {
			if !c.authz.CanControlDevice(client.DeviceID, targetDevice.DeviceUID, requesterUID) {
				continue
			}
		} else if !c.perm.CanWriteVarsForDevice(client.DeviceID, targetDevice.DeviceUID) {
			continue
		}

		jsonValue, _ := json.Marshal(value)
		variable := &database.DeviceVariable{
			OwnerDeviceID: targetDevice.ID,
			VariableName:  varName,
			Value:         datatypes.JSON(jsonValue),
		}
		if c.service.UpsertVariable(variable) == nil {
			updatedCount++
		}
	}
	log.Info().Int("count", updatedCount).Msg("变量已更新")
}
