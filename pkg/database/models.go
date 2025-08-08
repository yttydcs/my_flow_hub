package database

import (
	"time"

	"gorm.io/datatypes"
)

type DeviceRole string
type PermissionAction string

const (
	RoleHub     DeviceRole = "hub"
	RoleRelay   DeviceRole = "relay"
	RoleNode    DeviceRole = "node"
	RoleManager DeviceRole = "manager"

	ActionRead        PermissionAction = "read"
	ActionWrite       PermissionAction = "write"
	ActionSendMessage PermissionAction = "send_message"
)

// Device 对应于 'devices' 表
type Device struct {
	ID            uint64     `gorm:"primaryKey"`
	DeviceUID     uint64     `gorm:"unique;not null;autoIncrement;start:10000"`
	SecretKeyHash string     `gorm:"not null"`
	HardwareID    string     `gorm:"unique"`
	Role          DeviceRole `gorm:"type:varchar(20)"`
	ParentID      *uint64
	Parent        *Device `gorm:"foreignKey:ParentID"`
	Name          string
	LastSeen      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// DeviceVariable 对应于 'device_variables' 表
type DeviceVariable struct {
	ID            uint64 `gorm:"primaryKey"`
	OwnerDeviceID uint64 `gorm:"not null"`
	Device        Device `gorm:"foreignKey:OwnerDeviceID"`
	VariableName  string `gorm:"not null"`
	Value         datatypes.JSON
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// AccessPermission 对应于 'access_permissions' 表
type AccessPermission struct {
	ID                 uint64 `gorm:"primaryKey"`
	RequesterDeviceID  uint64 `gorm:"not null"`
	RequesterDevice    Device `gorm:"foreignKey:RequesterDeviceID"`
	TargetDeviceID     uint64 `gorm:"not null"`
	TargetDevice       Device `gorm:"foreignKey:TargetDeviceID"`
	TargetVariableName string
	Action             PermissionAction `gorm:"type:varchar(20)"`
	CreatedAt          time.Time
}
