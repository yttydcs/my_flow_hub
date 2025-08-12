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
}

// NewDeviceController 创建一个新的 DeviceController
func NewDeviceController(service *service.DeviceService) *DeviceController {
	return &DeviceController{service: service}
}

// HandleQueryNodes 处理节点查询请求
func (c *DeviceController) HandleQueryNodes(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
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

	if err := c.service.CreateDevice(&payload); err != nil {
		s.SendErrorResponse(client, msg.ID, "Failed to create device")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}

// HandleUpdateDevice 处理更新设备请求
func (c *DeviceController) HandleUpdateDevice(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var payload database.Device
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

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
	id, ok := payload["id"].(float64)
	if !ok {
		return
	}

	if err := c.service.DeleteDevice(uint64(id)); err != nil {
		s.SendErrorResponse(client, msg.ID, "Failed to delete device")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
}
