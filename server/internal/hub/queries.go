package hub

import (
	"myflowhub/pkg/database"
	"myflowhub/pkg/protocol"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// DeviceTreeNode 用于构建带有完整子节点信息的设备树
type DeviceTreeNode struct {
	database.Device
	Children []DeviceTreeNode `json:"children"`
}

// buildDeviceTree 构建设备树形结构
func buildDeviceTree(devices []database.Device) []DeviceTreeNode {
	// 创建设备映射，便于查找
	deviceMap := make(map[uint64]*DeviceTreeNode)
	var rootDevices []DeviceTreeNode

	// 首先创建所有设备节点
	for _, device := range devices {
		node := DeviceTreeNode{
			Device:   device,
			Children: make([]DeviceTreeNode, 0),
		}
		deviceMap[device.ID] = &node
	}

	// 然后构建父子关系
	for _, device := range devices {
		if device.ParentID != nil {
			// 有父节点，添加到父节点的children中
			if parent, exists := deviceMap[*device.ParentID]; exists {
				parent.Children = append(parent.Children, *deviceMap[device.ID])
			}
		} else {
			// 没有父节点，是根节点
			rootDevices = append(rootDevices, *deviceMap[device.ID])
		}
	}

	return rootDevices
}

// handleQueryNodes 处理节点查询请求
func handleQueryNodes(s *Server, client *Client, msg protocol.BaseMessage) {
	var devices []database.Device

	// 获取所有设备，包括Parent信息
	if err := database.DB.Preload("Parent").Find(&devices).Error; err != nil {
		log.Error().Err(err).Msg("查询设备列表失败")
		response := protocol.BaseMessage{
			ID:        uuid.New().String(),
			Source:    s.DeviceID,
			Target:    client.DeviceID,
			Type:      "response",
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"success":     false,
				"error":       "Failed to query devices",
				"original_id": msg.ID,
			},
		}
		client.Send <- mustMarshal(response)
		return
	}

	// 构建带有完整子节点信息的设备树
	deviceTree := buildDeviceTree(devices)

	response := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Source:    s.DeviceID,
		Target:    client.DeviceID,
		Type:      "response",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"success":     true,
			"data":        deviceTree,
			"original_id": msg.ID,
		},
	}
	client.Send <- mustMarshal(response)
	log.Info().Uint64("clientID", client.DeviceID).Int("deviceCount", len(devices)).Msg("节点查询请求已处理（含完整树形结构）")
}

// handleQueryVariables 处理变量查询请求
func handleQueryVariables(s *Server, client *Client, msg protocol.BaseMessage) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		sendErrorResponse(s, client, msg.ID, "Invalid payload format")
		return
	}

	deviceIDStr, _ := payload["deviceId"].(string)

	query := database.DB.Preload("Device")
	if deviceIDStr != "" {
		if deviceID, err := strconv.ParseUint(deviceIDStr, 10, 64); err == nil {
			var device database.Device
			if err := database.DB.Where("device_uid = ?", deviceID).First(&device).Error; err == nil {
				query = query.Where("owner_device_id = ?", device.ID)
			}
		}
	}

	var variables []database.DeviceVariable
	if err := query.Find(&variables).Error; err != nil {
		log.Error().Err(err).Msg("查询变量列表失败")
		sendErrorResponse(s, client, msg.ID, "Failed to query variables")
		return
	}

	response := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Source:    s.DeviceID,
		Target:    client.DeviceID,
		Type:      "response",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"success":     true,
			"data":        variables,
			"original_id": msg.ID,
		},
	}
	client.Send <- mustMarshal(response)
	log.Info().Uint64("clientID", client.DeviceID).Int("variableCount", len(variables)).Msg("变量查询请求已处理")
}

// sendErrorResponse 发送错误响应
func sendErrorResponse(s *Server, client *Client, originalID, errorMsg string) {
	response := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Source:    s.DeviceID,
		Target:    client.DeviceID,
		Type:      "response",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"success":     false,
			"error":       errorMsg,
			"original_id": originalID,
		},
	}
	client.Send <- mustMarshal(response)
}
