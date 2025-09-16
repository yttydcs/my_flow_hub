package service

import (
	"encoding/json"
	"myflowhub/pkg/config"
	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService 提供了认证和注册的业务逻辑
type AuthService struct {
	deviceRepo   *repository.DeviceRepository
	variableRepo *repository.VariableRepository
}

// NewAuthService 创建一个新的 AuthService
func NewAuthService(deviceRepo *repository.DeviceRepository, variableRepo *repository.VariableRepository) *AuthService {
	return &AuthService{
		deviceRepo:   deviceRepo,
		variableRepo: variableRepo,
	}
}

// AuthenticateDevice 认证一个常规设备
func (s *AuthService) AuthenticateDevice(deviceID uint64, secretKey string) (*database.Device, bool) {
	device, err := s.deviceRepo.FindByUID(deviceID)
	if err != nil {
		return nil, false
	}

	if err := bcrypt.CompareHashAndPassword([]byte(device.SecretKeyHash), []byte(secretKey)); err != nil {
		return nil, false
	}

	return device, true
}

// AuthenticateManager 认证一个管理员节点
func (s *AuthService) AuthenticateManager(token string) (*database.Device, bool) {
	if token != config.AppConfig.Server.ManagerToken {
		return nil, false
	}

	managerDevice, err := s.deviceRepo.FindByHardwareID("manager")
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			hashedSecret, _ := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
			newManagerDevice := &database.Device{
				HardwareID:    "manager",
				SecretKeyHash: string(hashedSecret),
				Role:          database.RoleManager,
				Approved:      true,
				Name:          "System Manager",
			}
			if err := s.deviceRepo.Create(newManagerDevice); err != nil {
				return nil, false
			}
			if newManagerDevice.DeviceUID == 0 {
				// 确保有非零 UID（某些数据库/迁移环境可能未自动分配）
				newManagerDevice.DeviceUID = 10001
				_ = s.deviceRepo.Update(newManagerDevice)
			}
			return newManagerDevice, true
		}
		return nil, false
	}
	if managerDevice.DeviceUID == 0 {
		managerDevice.DeviceUID = 10001
		_ = s.deviceRepo.Update(managerDevice)
	}
	if !managerDevice.Approved {
		managerDevice.Approved = true
		_ = s.deviceRepo.Update(managerDevice)
	}
	return managerDevice, true
}

// RegisterDevice 注册一个新设备
func (s *AuthService) RegisterDevice(hardwareID string) (*database.Device, string, bool) {
	_, err := s.deviceRepo.FindByHardwareID(hardwareID)
	if err == nil {
		return nil, "", false // 设备已存在
	}

	secretKey := "default-secret" // 在实际应用中应随机生成
	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte(secretKey), bcrypt.DefaultCost)
	newDevice := &database.Device{
		HardwareID:    hardwareID,
		SecretKeyHash: string(hashedSecret),
		Role:          database.RoleNode,
		Name:          hardwareID,
	}

	if err := s.deviceRepo.Create(newDevice); err != nil {
		return nil, "", false
	}

	return newDevice, secretKey, true
}

// GetInitialVariablesForDevice 获取设备上线时的初始变量
func (s *AuthService) GetInitialVariablesForDevice(deviceUID uint64) (map[string]interface{}, error) {
	device, err := s.deviceRepo.FindByUID(deviceUID)
	if err != nil {
		return nil, err
	}

	variables, err := s.variableRepo.FindByDeviceID(device.ID)
	if err != nil {
		return nil, err
	}

	varsMap := make(map[string]interface{})
	for _, v := range variables {
		var val interface{}
		json.Unmarshal(v.Value, &val)
		varsMap[v.VariableName] = val
	}

	return varsMap, nil
}
