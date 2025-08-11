package repository

import (
	"myflowhub/pkg/database"

	"gorm.io/gorm"
)

// DeviceRepository 提供了访问设备数据的方法
type DeviceRepository struct {
	db *gorm.DB
}

// NewDeviceRepository 创建一个新的 DeviceRepository
func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// FindAll 返回所有设备
func (r *DeviceRepository) FindAll() ([]database.Device, error) {
	var devices []database.Device
	err := r.db.Preload("Parent").Find(&devices).Error
	return devices, err
}

// FindByUID 根据 UID 查找设备
func (r *DeviceRepository) FindByUID(uid uint64) (*database.Device, error) {
	var device database.Device
	err := r.db.Where("device_uid = ?", uid).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// FindByHardwareID 根据硬件 ID 查找设备
func (r *DeviceRepository) FindByHardwareID(hid string) (*database.Device, error) {
	var device database.Device
	err := r.db.Where("hardware_id = ?", hid).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// Create 创建一个新设备
func (r *DeviceRepository) Create(device *database.Device) error {
	return r.db.Create(device).Error
}

// UpdateParentID 更新设备的父级 ID
func (r *DeviceRepository) UpdateParentID(deviceID, parentID uint64) error {
	return r.db.Model(&database.Device{}).Where("id = ?", deviceID).Update("parent_id", parentID).Error
}
