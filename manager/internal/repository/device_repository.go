package repository

import (
	"myflowhub/pkg/database"

	"gorm.io/gorm"
)

// DeviceRepository 设备数据访问层
type DeviceRepository struct {
	db *gorm.DB
}

// NewDeviceRepository 创建设备仓储实例
func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// GetAll 获取所有设备
func (r *DeviceRepository) GetAll() ([]database.Device, error) {
	var devices []database.Device
	err := r.db.Preload("Parent").Preload("Children").Find(&devices).Error
	return devices, err
}

// GetByID 根据ID获取设备
func (r *DeviceRepository) GetByID(id uint64) (*database.Device, error) {
	var device database.Device
	err := r.db.Preload("Parent").Preload("Children").First(&device, id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetByHardwareID 根据硬件ID获取设备
func (r *DeviceRepository) GetByHardwareID(hardwareID string) (*database.Device, error) {
	var device database.Device
	err := r.db.Where("hardware_id = ?", hardwareID).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// Create 创建设备
func (r *DeviceRepository) Create(device *database.Device) error {
	return r.db.Create(device).Error
}

// Update 更新设备
func (r *DeviceRepository) Update(device *database.Device) error {
	return r.db.Save(device).Error
}

// Delete 删除设备
func (r *DeviceRepository) Delete(id uint64) error {
	return r.db.Delete(&database.Device{}, id).Error
}

// GetChildren 获取子设备
func (r *DeviceRepository) GetChildren(parentID uint64) ([]database.Device, error) {
	var children []database.Device
	err := r.db.Where("parent_id = ?", parentID).Find(&children).Error
	return children, err
}

// Count 获取设备总数
func (r *DeviceRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&database.Device{}).Count(&count).Error
	return count, err
}
