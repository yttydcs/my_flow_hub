package repository

import (
	"myflowhub/pkg/database"

	"gorm.io/gorm"
)

// VariableRepository 变量数据访问层
type VariableRepository struct {
	db *gorm.DB
}

// NewVariableRepository 创建变量仓储实例
func NewVariableRepository(db *gorm.DB) *VariableRepository {
	return &VariableRepository{db: db}
}

// GetAll 获取所有变量
func (r *VariableRepository) GetAll() ([]database.DeviceVariable, error) {
	var variables []database.DeviceVariable
	err := r.db.Preload("Device").Find(&variables).Error
	return variables, err
}

// GetByID 根据ID获取变量
func (r *VariableRepository) GetByID(id uint64) (*database.DeviceVariable, error) {
	var variable database.DeviceVariable
	err := r.db.Preload("Device").First(&variable, id).Error
	if err != nil {
		return nil, err
	}
	return &variable, nil
}

// GetByDeviceID 根据设备ID获取变量
func (r *VariableRepository) GetByDeviceID(deviceID uint64) ([]database.DeviceVariable, error) {
	var variables []database.DeviceVariable
	err := r.db.Preload("Device").Where("owner_device_id = ?", deviceID).Find(&variables).Error
	return variables, err
}

// GetByDeviceIDAndName 根据设备ID和变量名获取变量
func (r *VariableRepository) GetByDeviceIDAndName(deviceID uint64, name string) (*database.DeviceVariable, error) {
	var variable database.DeviceVariable
	err := r.db.Where("owner_device_id = ? AND variable_name = ?", deviceID, name).First(&variable).Error
	if err != nil {
		return nil, err
	}
	return &variable, nil
}

// Create 创建变量
func (r *VariableRepository) Create(variable *database.DeviceVariable) error {
	return r.db.Create(variable).Error
}

// Update 更新变量
func (r *VariableRepository) Update(variable *database.DeviceVariable) error {
	return r.db.Save(variable).Error
}

// Delete 删除变量
func (r *VariableRepository) Delete(id uint64) error {
	return r.db.Delete(&database.DeviceVariable{}, id).Error
}

// DeleteByDeviceID 删除设备的所有变量
func (r *VariableRepository) DeleteByDeviceID(deviceID uint64) error {
	return r.db.Where("owner_device_id = ?", deviceID).Delete(&database.DeviceVariable{}).Error
}

// Count 获取变量总数
func (r *VariableRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&database.DeviceVariable{}).Count(&count).Error
	return count, err
}

// CountByDeviceID 获取指定设备的变量数量
func (r *VariableRepository) CountByDeviceID(deviceID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&database.DeviceVariable{}).Where("owner_device_id = ?", deviceID).Count(&count).Error
	return count, err
}
