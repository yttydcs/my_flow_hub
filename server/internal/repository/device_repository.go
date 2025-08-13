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

// ListByOwner 返回指定用户拥有的设备
func (r *DeviceRepository) ListByOwner(ownerUserID uint64) ([]database.Device, error) {
	var devices []database.Device
	err := r.db.Where("owner_user_id = ?", ownerUserID).Find(&devices).Error
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

// FindByID 根据 ID 查找设备
func (r *DeviceRepository) FindByID(id uint64) (*database.Device, error) {
	var device database.Device
	if err := r.db.First(&device, id).Error; err != nil {
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

// Update 更新设备信息
func (r *DeviceRepository) Update(device *database.Device) error {
	return r.db.Save(device).Error
}

// Delete 删除设备
func (r *DeviceRepository) Delete(id uint64) error {
	return r.db.Delete(&database.Device{}, id).Error
}

// IsAncestorUID 判断 ancestorUID 是否为 descendantUID 的祖先（或相等时视为可控在上层处理）
func (r *DeviceRepository) IsAncestorUID(ancestorUID, descendantUID uint64) (bool, error) {
	desc, err := r.FindByUID(descendantUID)
	if err != nil {
		return false, err
	}
	const maxSteps = 1024
	steps := 0
	cur := desc
	for cur != nil && cur.ParentID != nil && steps < maxSteps {
		steps++
		p, err := r.FindByID(*cur.ParentID)
		if err != nil {
			break
		}
		if p.DeviceUID == ancestorUID {
			return true, nil
		}
		cur = p
	}
	return false, nil
}

// ListDescendantsOfUID 返回某设备 UID 的所有后代（不含自身）
func (r *DeviceRepository) ListDescendantsOfUID(uid uint64) ([]database.Device, error) {
	root, err := r.FindByUID(uid)
	if err != nil {
		return nil, err
	}
	result := make([]database.Device, 0)
	// 逐层查找 children
	frontier := []uint64{root.ID}
	const maxLevels = 1024
	levels := 0
	for len(frontier) > 0 && levels < maxLevels {
		levels++
		var children []database.Device
		if err := r.db.Where("parent_id IN ?", frontier).Find(&children).Error; err != nil {
			return nil, err
		}
		if len(children) == 0 {
			break
		}
		result = append(result, children...)
		next := make([]uint64, 0, len(children))
		for _, c := range children {
			next = append(next, c.ID)
		}
		frontier = next
	}
	return result, nil
}
