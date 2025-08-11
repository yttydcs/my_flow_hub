package repository

import (
	"myflowhub/pkg/database"
	"strings"

	"gorm.io/gorm"
)

// VariableRepository 提供了访问变量数据的方法
type VariableRepository struct {
	db *gorm.DB
}

// NewVariableRepository 创建一个新的 VariableRepository
func NewVariableRepository(db *gorm.DB) *VariableRepository {
	return &VariableRepository{db: db}
}

// FindByDeviceID 根据设备 ID 查找变量
func (r *VariableRepository) FindByDeviceID(deviceID uint64) ([]database.DeviceVariable, error) {
	var variables []database.DeviceVariable
	err := r.db.Where("owner_device_id = ?", deviceID).Find(&variables).Error
	return variables, err
}

// FindByOwnerAndName 根据所有者和变量名查找变量
func (r *VariableRepository) FindByOwnerAndName(ownerID uint64, name string) (*database.DeviceVariable, error) {
	var variable database.DeviceVariable
	err := r.db.Where("owner_device_id = ? AND variable_name = ?", ownerID, name).First(&variable).Error
	if err != nil {
		return nil, err
	}
	return &variable, nil
}

// Upsert 更新或创建变量
func (r *VariableRepository) Upsert(variable *database.DeviceVariable) error {
	return r.db.Where("owner_device_id = ? AND variable_name = ?", variable.OwnerDeviceID, variable.VariableName).
		Assign(database.DeviceVariable{Value: variable.Value}).
		FirstOrCreate(variable).Error
}

// Delete 删除变量
func (r *VariableRepository) Delete(ownerID uint64, name string) error {
	return r.db.Where("owner_device_id = ? AND variable_name = ?", ownerID, name).Delete(&database.DeviceVariable{}).Error
}

// FindAll 返回所有变量
func (r *VariableRepository) FindAll() ([]database.DeviceVariable, error) {
	var variables []database.DeviceVariable
	err := r.db.Preload("Device").Find(&variables).Error
	return variables, err
}

// FindByDeviceUIDAndVarName 根据设备 UID 和变量名查找
func (r *VariableRepository) FindByDeviceUIDAndVarName(deviceUID, varName string) (*database.DeviceVariable, error) {
	var targetDevice database.Device
	var err error

	if strings.HasPrefix(deviceUID, "[") && strings.HasSuffix(deviceUID, "]") {
		err = r.db.Where("device_uid = ?", strings.Trim(deviceUID, "[]")).First(&targetDevice).Error
	} else {
		err = r.db.Where("name = ?", strings.Trim(deviceUID, "()")).First(&targetDevice).Error
	}
	if err != nil {
		return nil, err
	}

	return r.FindByOwnerAndName(targetDevice.ID, varName)
}
