package controller

import (
	"encoding/json"
	"myflowhub/pkg/protocol"
	"myflowhub/server/internal/hub"
	"myflowhub/server/internal/service"

	"github.com/rs/zerolog/log"
)

// AuthController 负责处理认证和注册相关的消息
type AuthController struct {
	authService   *service.AuthService
	deviceService *service.DeviceService
}

// NewAuthController 创建一个新的 AuthController
func NewAuthController(authService *service.AuthService, deviceService *service.DeviceService) *AuthController {
	return &AuthController{
		authService:   authService,
		deviceService: deviceService,
	}
}

// HandleAuthRequest 处理常规设备认证请求
func (c *AuthController) HandleAuthRequest(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var payload protocol.AuthRequestPayload
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	device, ok := c.authService.AuthenticateDevice(payload.DeviceID, payload.SecretKey)
	if !ok {
		log.Warn().Uint64("deviceID", payload.DeviceID).Msg("认证失败")
		return
	}

	client.DeviceID = device.DeviceUID
	s.Clients[client.DeviceID] = client
	log.Info().Uint64("clientID", client.DeviceID).Msg("客户端在 Hub 中认证成功并注册")

	// 更新父级关系并同步变量
	parentDevice, _ := c.deviceService.GetDeviceByUID(s.DeviceID)
	if parentDevice != nil {
		c.deviceService.UpdateDeviceParentID(device.ID, parentDevice.ID)
	}
	go c.syncVarsOnLogin(s, client)
}

// HandleManagerAuthRequest 处理管理员认证请求
func (c *AuthController) HandleManagerAuthRequest(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	payload, _ := msg.Payload.(map[string]interface{})
	token, _ := payload["token"].(string)

	device, ok := c.authService.AuthenticateManager(token)
	if !ok {
		log.Warn().Str("token", token).Msg("管理员认证失败")
		return
	}

	client.DeviceID = device.DeviceUID
	s.Clients[client.DeviceID] = client
	log.Info().Uint64("clientID", client.DeviceID).Msg("管理员在 Hub 中认证成功并注册")

	s.SendResponse(client, msg.ID, map[string]interface{}{
		"success":  true,
		"deviceId": device.DeviceUID,
		"role":     "manager",
	})
}

// HandleRegisterRequest 处理设备注册请求
func (c *AuthController) HandleRegisterRequest(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var payload protocol.RegisterRequestPayload
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	device, secretKey, ok := c.authService.RegisterDevice(payload.HardwareID)
	if !ok {
		log.Warn().Str("hardwareID", payload.HardwareID).Msg("设备注册失败")
		return
	}

	client.DeviceID = device.DeviceUID
	s.Clients[client.DeviceID] = client
	log.Info().Uint64("clientID", client.DeviceID).Msg("客户端在 Hub 中注册成功并注册")

	// 更新父级关系
	parentDevice, _ := c.deviceService.GetDeviceByUID(s.DeviceID)
	if parentDevice != nil {
		c.deviceService.UpdateDeviceParentID(device.ID, parentDevice.ID)
	}

	s.SendResponse(client, msg.ID, map[string]interface{}{
		"success":   true,
		"deviceId":  device.DeviceUID,
		"secretKey": secretKey,
	})
}

// syncVarsOnLogin 客户端上线时同步变量
func (c *AuthController) syncVarsOnLogin(s *hub.Server, client *hub.Client) {
	vars, err := c.authService.GetInitialVariablesForDevice(client.DeviceID)
	if err != nil || len(vars) == 0 {
		return
	}

	s.NotifyVarChange(client.DeviceID, s.DeviceID, vars)
	log.Info().Uint64("clientID", client.DeviceID).Msg("已完成上线变量同步")
}
