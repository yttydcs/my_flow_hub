package controller

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"myflowhub/pkg/protocol"
	"myflowhub/server/internal/hub"
	"myflowhub/server/internal/repository"
	"myflowhub/server/internal/service"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// AuthController 负责处理认证和注册相关的消息
type AuthController struct {
	authService   *service.AuthService
	deviceService *service.DeviceService
	perm          *service.PermissionService
	permRepo      *repository.PermissionRepository
	keyService    *service.KeyService
	userRepo      *repository.UserRepository
}

// NewAuthController 创建一个新的 AuthController
func NewAuthController(authService *service.AuthService, deviceService *service.DeviceService) *AuthController {
	return &AuthController{
		authService:   authService,
		deviceService: deviceService,
	}
}

// SetKeyService 注入 KeyService，用于登录签发会话密钥
func (c *AuthController) SetKeyService(k *service.KeyService) { c.keyService = k }

// SetPermissionRepository 注入权限仓库，用于查询用户权限节点
func (c *AuthController) SetPermissionRepository(p *repository.PermissionRepository) { c.permRepo = p }

// SetUserRepository 注入用户仓库，用于登录校验
func (c *AuthController) SetUserRepository(u *repository.UserRepository) { c.userRepo = u }

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

// HandleUserLogin 处理用户登录，返回用户会话密钥（key-only 模式）
func (c *AuthController) HandleUserLogin(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var payload struct {
		Username  string     `json:"username"`
		Password  string     `json:"password"`
		ExpiresAt *time.Time `json:"expiresAt"`
		MaxUses   *int       `json:"maxUses"`
	}
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	if c.userRepo == nil || c.keyService == nil {
		s.SendErrorResponse(client, msg.ID, "login not configured")
		return
	}
	// 校验用户名/密码
	user, err := c.userRepo.FindByUsername(payload.Username)
	if err != nil || user.Disabled {
		s.SendErrorResponse(client, msg.ID, "invalid credentials")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password)) != nil {
		s.SendErrorResponse(client, msg.ID, "invalid credentials")
		return
	}

	// 生成随机 secret，并创建绑定到 user 的密钥（有效期上限 30 天）
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		s.SendErrorResponse(client, msg.ID, "failed to issue key")
		return
	}
	secret := hex.EncodeToString(buf)
	// 限制最大 30 天
	maxExp := time.Now().Add(30 * 24 * time.Hour)
	var finalExp *time.Time
	if payload.ExpiresAt == nil || payload.ExpiresAt.After(maxExp) {
		finalExp = &maxExp
	} else {
		finalExp = payload.ExpiresAt
	}
	bind := "user"
	uid := user.ID
	keyObj, err2 := c.keyService.CreateKey(user.ID, &bind, &uid, secret, finalExp, payload.MaxUses, nil)
	if err2 != nil {
		s.SendErrorResponse(client, msg.ID, "failed to issue key")
		return
	}

	// 查询权限节点快照
	perms := []string{}
	if c.permRepo != nil {
		if list, err := c.permRepo.ListByUserID(user.ID); err == nil {
			for _, p := range list {
				perms = append(perms, p.Node)
			}
		}
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{
		"success":     true,
		"token":       secret,
		"keyId":       keyObj.ID,
		"user":        map[string]interface{}{"id": user.ID, "username": user.Username, "displayName": user.DisplayName},
		"permissions": perms,
	})
}

// HandleUserMe 返回当前 userKey 对应的用户信息与权限
func (c *AuthController) HandleUserMe(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	if c.permRepo == nil || c.userRepo == nil || c.keyService == nil {
		s.SendErrorResponse(client, msg.ID, "not configured")
		return
	}
	payload, _ := msg.Payload.(map[string]interface{})
	userKey, _ := payload["userKey"].(string)
	uid, _, err := c.keyService.PeekUserKey(userKey)
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "invalid key")
		return
	}
	u, err := c.userRepo.FindByID(uid)
	if err != nil {
		s.SendErrorResponse(client, msg.ID, "not found")
		return
	}
	perms := []string{}
	if list, err := c.permRepo.ListByUserID(uid); err == nil {
		for _, p := range list {
			perms = append(perms, p.Node)
		}
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{
		"success":     true,
		"user":        map[string]interface{}{"id": u.ID, "username": u.Username, "displayName": u.DisplayName},
		"permissions": perms,
	})
}

// HandleUserLogout 撤销当前 userKey
func (c *AuthController) HandleUserLogout(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	payload, _ := msg.Payload.(map[string]interface{})
	userKey, _ := payload["userKey"].(string)
	if userKey == "" {
		s.SendErrorResponse(client, msg.ID, "invalid key")
		return
	}
	if err := c.keyService.DeleteBySecret(userKey); err != nil {
		s.SendErrorResponse(client, msg.ID, "failed")
		return
	}
	s.SendResponse(client, msg.ID, map[string]interface{}{"success": true})
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
