package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"myflowhub/server/internal/repository"
	"myflowhub/server/internal/service"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthController 负责处理认证和注册相关的消息
type AuthController struct {
	authService   *service.AuthService
	deviceService *service.DeviceService
	permRepo      *repository.PermissionRepository
	keyService    *service.KeyService
	userRepo      *repository.UserRepository
	audit         *service.AuditService
	syslog        *service.SystemLogService
}

// NewAuthController 创建一个新的 AuthController
func NewAuthController(
	authService *service.AuthService,
	deviceService *service.DeviceService,
	keyService *service.KeyService,
	permRepo *repository.PermissionRepository,
	userRepo *repository.UserRepository,
	audit *service.AuditService,
	syslog *service.SystemLogService,
) *AuthController {
	return &AuthController{
		authService:   authService,
		deviceService: deviceService,
		keyService:    keyService,
		permRepo:      permRepo,
		userRepo:      userRepo,
		audit:         audit,
		syslog:        syslog,
	}
}

// AuthenticateManagerToken: 供二进制路由调用的纯业务方法
func (c *AuthController) AuthenticateManagerToken(token string) (deviceUID uint64, role string, err error) {
	device, ok := c.authService.AuthenticateManager(token)
	if !ok {
		return 0, "", fmt.Errorf("unauthorized")
	}
	// 某些初始化路径下 DeviceUID 可能仍为 0，回退使用数据库 ID 以保证非 0
	if device.DeviceUID != 0 {
		return device.DeviceUID, "manager", nil
	}
	return device.ID, "manager", nil
}

// Login: 用户登录，返回一次性 userKey 与权限
func (c *AuthController) Login(username, password string) (keyID, userID uint64, secret, uname, displayName string, perms []string, err error) {
	user, e := c.userRepo.FindByUsername(username)
	if e != nil {
		return 0, 0, "", "", "", nil, fmt.Errorf("invalid credentials")
	}
	if user.Disabled {
		return 0, 0, "", "", "", nil, fmt.Errorf("invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return 0, 0, "", "", "", nil, fmt.Errorf("invalid credentials")
	}
	buf := make([]byte, 32)
	if _, e := rand.Read(buf); e != nil {
		return 0, 0, "", "", "", nil, fmt.Errorf("issue failed")
	}
	secret = hex.EncodeToString(buf)
	maxExp := time.Now().Add(30 * 24 * time.Hour)
	bind := "user"
	uid := user.ID
	keyObj, e2 := c.keyService.CreateKey(user.ID, &bind, &uid, secret, &maxExp, nil, nil)
	if e2 != nil {
		return 0, 0, "", "", "", nil, fmt.Errorf("issue failed")
	}
	var permNames []string
	if c.permRepo != nil {
		if list, _ := c.permRepo.ListByUserID(user.ID); len(list) > 0 {
			permNames = make([]string, 0, len(list))
			for _, p := range list {
				permNames = append(permNames, p.Node)
			}
		}
	}
	if c.syslog != nil {
		_ = c.syslog.Info("auth", "user login", map[string]any{"userId": user.ID})
	}
	if c.audit != nil {
		uid2 := user.ID
		_ = c.audit.Write("user", &uid2, "user.login", user.Username, "allow", "", "", nil)
	}
	return keyObj.ID, user.ID, secret, user.Username, user.DisplayName, permNames, nil
}

// Me: 根据 userKey 返回用户与权限
func (c *AuthController) Me(userKey string) (userID uint64, username, displayName string, perms []string, err error) {
	uid, _, e := c.keyService.PeekUserKey(userKey)
	if e != nil || uid == 0 {
		return 0, "", "", nil, fmt.Errorf("invalid key")
	}
	u, e := c.userRepo.FindByID(uid)
	if e != nil {
		return 0, "", "", nil, fmt.Errorf("not found")
	}
	var permNames []string
	if u != nil {
		if list, _ := c.permRepo.ListByUserID(u.ID); len(list) > 0 {
			permNames = make([]string, 0, len(list))
			for _, p := range list {
				permNames = append(permNames, p.Node)
			}
		}
	}
	return u.ID, u.Username, u.DisplayName, permNames, nil
}

// Logout: 撤销 userKey
func (c *AuthController) Logout(userKey string) error {
	if userKey == "" {
		return fmt.Errorf("invalid key")
	}
	return c.keyService.DeleteBySecret(userKey)
}

// 所有 JSON 兼容 Handler 已移除，二进制专用
