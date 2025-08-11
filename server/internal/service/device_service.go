package service

import (
	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"
	"strconv"
	"strings"
)

// DeviceService 提供了设备相关的业务逻辑
type DeviceService struct {
	repo *repository.DeviceRepository
}

// NewDeviceService 创建一个新的 DeviceService
func NewDeviceService(repo *repository.DeviceRepository) *DeviceService {
	return &DeviceService{repo: repo}
}

// GetAllDevices 获取所有设备
func (s *DeviceService) GetAllDevices() ([]database.Device, error) {
	return s.repo.FindAll()
}

// GetDeviceByUID 根据 UID 获取设备
func (s *DeviceService) GetDeviceByUID(uid uint64) (*database.Device, error) {
	return s.repo.FindByUID(uid)
}

// GetDeviceByHardwareID 根据硬件 ID 获取设备
func (s *DeviceService) GetDeviceByHardwareID(hid string) (*database.Device, error) {
	return s.repo.FindByHardwareID(hid)
}

// CreateDevice 创建一个新设备
func (s *DeviceService) CreateDevice(device *database.Device) error {
	return s.repo.Create(device)
}

// UpdateDeviceParentID 更新设备的父级 ID
func (s *DeviceService) UpdateDeviceParentID(deviceID, parentID uint64) error {
	return s.repo.UpdateParentID(deviceID, parentID)
}

// GetDeviceByUIDOrName 根据 UID 或名称获取设备
func (s *DeviceService) GetDeviceByUIDOrName(identifier string) (*database.Device, error) {
	if strings.HasPrefix(identifier, "[") && strings.HasSuffix(identifier, "]") {
		uid, err := strconv.ParseUint(strings.Trim(identifier, "[]"), 10, 64)
		if err != nil {
			return nil, err
		}
		return s.repo.FindByUID(uid)
	}
	return s.repo.FindByHardwareID(strings.Trim(identifier, "()"))
}
