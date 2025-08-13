package service

import (
	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// DeviceService 提供了设备相关的业务逻辑
type DeviceService struct {
	deviceRepo   *repository.DeviceRepository
	variableRepo *repository.VariableRepository
	db           *gorm.DB
}

// NewDeviceService 创建一个新的 DeviceService
func NewDeviceService(deviceRepo *repository.DeviceRepository, variableRepo *repository.VariableRepository, db *gorm.DB) *DeviceService {
	return &DeviceService{
		deviceRepo:   deviceRepo,
		variableRepo: variableRepo,
		db:           db,
	}
}

// GetAllDevices 获取所有设备
func (s *DeviceService) GetAllDevices() ([]database.Device, error) {
	return s.deviceRepo.FindAll()
}

// GetDeviceByUID 根据 UID 获取设备
func (s *DeviceService) GetDeviceByUID(uid uint64) (*database.Device, error) {
	return s.deviceRepo.FindByUID(uid)
}

// GetDeviceByID 根据数据库 ID 获取设备
func (s *DeviceService) GetDeviceByID(id uint64) (*database.Device, error) {
	return s.deviceRepo.FindByID(id)
}

// GetDeviceByHardwareID 根据硬件 ID 获取设备
func (s *DeviceService) GetDeviceByHardwareID(hid string) (*database.Device, error) {
	return s.deviceRepo.FindByHardwareID(hid)
}

// CreateDevice 创建一个新设备
func (s *DeviceService) CreateDevice(device *database.Device) error {
	return s.deviceRepo.Create(device)
}

// UpdateDeviceParentID 更新设备的父级 ID
func (s *DeviceService) UpdateDeviceParentID(deviceID, parentID uint64) error {
	return s.deviceRepo.UpdateParentID(deviceID, parentID)
}

// GetDeviceByUIDOrName 根据 UID 或名称获取设备
func (s *DeviceService) GetDeviceByUIDOrName(identifier string) (*database.Device, error) {
	if strings.HasPrefix(identifier, "[") && strings.HasSuffix(identifier, "]") {
		uid, err := strconv.ParseUint(strings.Trim(identifier, "[]"), 10, 64)
		if err != nil {
			return nil, err
		}
		return s.deviceRepo.FindByUID(uid)
	}
	return s.deviceRepo.FindByHardwareID(strings.Trim(identifier, "()"))
}

// UpdateDevice 更新设备信息
func (s *DeviceService) UpdateDevice(device *database.Device) error {
	return s.deviceRepo.Update(device)
}

// DeleteDevice 删除设备及其关联的变量
func (s *DeviceService) DeleteDevice(id uint64) error {
	// 启动数据库事务
	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 在事务中删除变量
	if err := tx.Where("owner_device_id = ?", id).Delete(&database.DeviceVariable{}).Error; err != nil {
		tx.Rollback() // 出错时回滚
		return err
	}

	// 在事务中删除设备
	if err := tx.Delete(&database.Device{}, id).Error; err != nil {
		tx.Rollback() // 出错时回滚
		return err
	}

	// 提交事务
	return tx.Commit().Error
}
