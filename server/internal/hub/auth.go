package hub

import (
	"encoding/json"
	"fmt"
	"myflowhub/pkg/config"
	"myflowhub/pkg/database"
	"myflowhub/pkg/protocol"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// handleAuth 处理客户端认证
func (s *Server) handleAuth(client *Client, msg protocol.BaseMessage) bool {
	var payload protocol.AuthRequestPayload
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	var device database.Device
	if err := database.DB.Where("device_uid = ?", payload.DeviceID).First(&device).Error; err != nil {
		log.Warn().Uint64("deviceID", payload.DeviceID).Err(err).Msg("认证失败：设备不存在")
		return false
	}

	if err := bcrypt.CompareHashAndPassword([]byte(device.SecretKeyHash), []byte(payload.SecretKey)); err != nil {
		log.Warn().Uint64("deviceID", payload.DeviceID).Msg("认证失败：密钥不正确")
		return false
	}

	client.DeviceID = device.DeviceUID

	// 更新设备的 parent 关系
	updateDeviceParent(s, device.ID, s.DeviceID)

	return true
}

// handleManagerAuth 处理管理员认证
func (s *Server) handleManagerAuth(client *Client, msg protocol.BaseMessage) bool {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		log.Warn().Msg("管理员认证失败：无效的payload格式")
		return false
	}

	token, ok := payload["token"].(string)
	if !ok {
		log.Warn().Msg("管理员认证失败：缺少token")
		return false
	}

	// 验证管理员令牌 - 使用配置中的ManagerToken
	if token != config.AppConfig.Server.ManagerToken {
		log.Warn().Str("token", token).Msg("管理员认证失败：token不正确")
		return false
	}

	// 为管理员创建或获取特殊设备记录
	var managerDevice database.Device
	err := database.DB.Where("hardware_id = ?", "manager").First(&managerDevice).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建管理员设备记录
			hashedSecret, _ := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
			managerDevice = database.Device{
				HardwareID:    "manager",
				SecretKeyHash: string(hashedSecret),
				Role:          database.RoleManager,
				Name:          "System Manager",
			}
			if err := database.DB.Create(&managerDevice).Error; err != nil {
				log.Error().Err(err).Msg("创建管理员设备记录失败")
				return false
			}
		} else {
			log.Error().Err(err).Msg("查询管理员设备记录失败")
			return false
		}
	}

	client.DeviceID = managerDevice.DeviceUID

	// 发送认证成功响应
	response := protocol.BaseMessage{
		ID:   msg.ID,
		Type: "manager_auth_response",
		Payload: map[string]interface{}{
			"success":  true,
			"deviceId": managerDevice.DeviceUID,
			"role":     "manager",
		},
	}
	client.Send <- mustMarshal(response)
	return true
}

// handleRegister 处理设备注册
func (s *Server) handleRegister(client *Client, msg protocol.BaseMessage) bool {
	var payload protocol.RegisterRequestPayload
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	if payload.HardwareID == "" {
		return false
	}

	var device database.Device
	err := database.DB.Where("hardware_id = ?", payload.HardwareID).First(&device).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return false
	}
	if err == nil {
		return false
	}

	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte("default-secret"), bcrypt.DefaultCost)
	newDevice := database.Device{
		HardwareID:    payload.HardwareID,
		SecretKeyHash: string(hashedSecret),
		Role:          database.RoleNode,
		Name:          payload.HardwareID,
	}

	if err := database.DB.Create(&newDevice).Error; err != nil {
		return false
	}

	client.DeviceID = newDevice.DeviceUID

	// 更新设备的 parent 关系
	updateDeviceParent(s, newDevice.ID, s.DeviceID)

	response := protocol.BaseMessage{
		ID:   msg.ID,
		Type: "register_response",
		Payload: map[string]interface{}{
			"success":   true,
			"deviceId":  newDevice.DeviceUID,
			"secretKey": "default-secret",
		},
	}
	client.Send <- mustMarshal(response)
	return true
}

// updateDeviceParent 更新设备的父级关系
func updateDeviceParent(s *Server, deviceID, parentDeviceUID uint64) {
	// 获取父设备的数据库ID
	var parentDevice database.Device
	if err := database.DB.Where("device_uid = ?", parentDeviceUID).First(&parentDevice).Error; err != nil {
		log.Warn().Uint64("parentUID", parentDeviceUID).Err(err).Msg("无法找到父设备")
		return
	}

	// 更新设备的父级ID
	if err := database.DB.Model(&database.Device{}).Where("id = ?", deviceID).Update("parent_id", parentDevice.ID).Error; err != nil {
		log.Error().Uint64("deviceID", deviceID).Uint64("parentID", parentDevice.ID).Err(err).Msg("更新设备父级关系失败")
	} else {
		log.Info().Uint64("deviceID", deviceID).Uint64("parentID", parentDevice.ID).Msg("设备父级关系已更新")
	}
}

// handleVarUpdate 处理变量更新
func handleVarUpdate(s *Server, client *Client, msg protocol.BaseMessage) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
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

		if !IsValidVarName(varName) {
			log.Warn().Str("varName", varName).Msg("无效的变量名，已跳过")
			continue
		}

		var targetDevice database.Device
		var err error
		if strings.HasPrefix(deviceIdentifier, "[") && strings.HasSuffix(deviceIdentifier, "]") {
			err = database.DB.Where("device_uid = ?", strings.Trim(deviceIdentifier, "[]")).First(&targetDevice).Error
		} else {
			err = database.DB.Where("name = ?", strings.Trim(deviceIdentifier, "()")).First(&targetDevice).Error
		}
		if err != nil {
			continue
		}

		jsonValue, _ := json.Marshal(value)
		variable := database.DeviceVariable{
			OwnerDeviceID: targetDevice.ID,
			VariableName:  varName,
			Value:         datatypes.JSON(jsonValue),
		}
		if database.DB.Where("owner_device_id = ? AND variable_name = ?", targetDevice.ID, varName).Assign(database.DeviceVariable{Value: datatypes.JSON(jsonValue)}).FirstOrCreate(&variable).Error == nil {
			updatedCount++
		}
	}
	log.Info().Int("count", updatedCount).Msg("变量已更新")
}

// handleVarsQuery 处理变量查询
func handleVarsQuery(s *Server, client *Client, msg protocol.BaseMessage) {
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

		var targetDevice database.Device
		var err error
		if strings.HasPrefix(deviceIdentifier, "[") && strings.HasSuffix(deviceIdentifier, "]") {
			err = database.DB.Where("device_uid = ?", strings.Trim(deviceIdentifier, "[]")).First(&targetDevice).Error
		} else {
			err = database.DB.Where("name = ?", strings.Trim(deviceIdentifier, "()")).First(&targetDevice).Error
		}
		if err != nil {
			results[i] = nil
			continue
		}

		var variable database.DeviceVariable
		if err := database.DB.Where("owner_device_id = ? AND variable_name = ?", targetDevice.ID, varName).First(&variable).Error; err != nil {
			results[i] = nil
		} else {
			var val interface{}
			json.Unmarshal(variable.Value, &val)
			results[i] = val
		}
	}

	response := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Source:    s.DeviceID,
		Target:    client.DeviceID,
		Type:      "response",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"success":     true,
			"original_id": msg.ID,
			"data": map[string]interface{}{
				"results": results,
			},
		},
	}
	client.Send <- mustMarshal(response)
}

// syncVarsOnLogin 客户端上线时同步变量
func (s *Server) syncVarsOnLogin(client *Client) {
	// 获取客户端设备的数据库记录
	var clientDevice database.Device
	if err := database.DB.Where("device_uid = ?", client.DeviceID).First(&clientDevice).Error; err != nil {
		log.Warn().Uint64("deviceUID", client.DeviceID).Err(err).Msg("无法找到客户端设备记录")
		return
	}

	var variables []database.DeviceVariable
	database.DB.Where("owner_device_id = ?", clientDevice.ID).Find(&variables)

	if len(variables) == 0 {
		return
	}

	varsMap := make(map[string]interface{})
	for _, v := range variables {
		var val interface{}
		json.Unmarshal(v.Value, &val)
		varsMap[v.VariableName] = val
	}

	notifyVarChange(s, client.DeviceID, s.DeviceID, varsMap)
	log.Info().Uint64("clientID", client.DeviceID).Msg("已完成上线变量同步")
}

// notifyVarChange 通知变量变更
func notifyVarChange(s *Server, targetDeviceID, sourceDeviceID uint64, variables map[string]interface{}) {
	if targetClient, ok := s.Clients[targetDeviceID]; ok {
		notification := protocol.BaseMessage{
			ID:        uuid.New().String(),
			Source:    sourceDeviceID,
			Target:    targetDeviceID,
			Type:      "var_notify",
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"variables": variables,
			},
		}
		targetClient.Send <- mustMarshal(notification)
	}
}
