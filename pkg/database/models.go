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
	Approved      bool       `gorm:"default:false;index"` // 审批通过后方可使用网络功能
	ParentID      *uint64
	Parent        *Device  `gorm:"foreignKey:ParentID"`
	Children      []Device `gorm:"foreignKey:ParentID"`
	Name          string
	OwnerUserID   *uint64 // 设备所有者用户ID，可为空（无主）
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

// User 用户表
type User struct {
	ID           uint64 `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex;size:100;not null"`
	PasswordHash string `gorm:"not null"`
	DisplayName  string `gorm:"size:200"`
	Disabled     bool   `gorm:"default:false"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Permission 允许型权限节点（无显式拒绝）
type Permission struct {
	ID          uint64 `gorm:"primaryKey"`
	SubjectType string `gorm:"size:20;index:idx_perm_subject"` // user/device/key
	SubjectID   uint64 `gorm:"index:idx_perm_subject"`
	Node        string `gorm:"size:512;index"`
	CreatedBy   *uint64
	CreatedAt   time.Time
}

// Key 密钥/会话令牌
type Key struct {
	ID              uint64     `gorm:"primaryKey"`
	OwnerUserID     *uint64    `gorm:"index"`         // 谁签发的
	BindSubjectType *string    `gorm:"size:20;index"` // user/device(optional)
	BindSubjectID   *uint64    `gorm:"index"`
	SecretHash      string     `gorm:"not null" comment:"sha256(secret), not the secret itself"`
	ExpiresAt       *time.Time `gorm:"index"`
	MaxUses         *int
	RemainingUses   *int
	Revoked         bool `gorm:"index"`
	IssuedBy        *uint64
	IssuedAt        time.Time
	Meta            datatypes.JSON
}

// Grant 用户->用户的借用授权（可选）
type Grant struct {
	ID            uint64         `gorm:"primaryKey"`
	GrantorUserID uint64         `gorm:"index"`
	GranteeUserID uint64         `gorm:"index"`
	ScopeNodes    datatypes.JSON // 节点数组
	ExpiresAt     *time.Time     `gorm:"index"`
	Revoked       bool           `gorm:"index"`
	CreatedAt     time.Time
}

// AuditLog 审计日志
type AuditLog struct {
	ID          uint64 `gorm:"primaryKey"`
	SubjectType string `gorm:"size:20"`
	SubjectID   *uint64
	Action      string    `gorm:"size:100"`
	Resource    string    `gorm:"size:512"`
	Decision    string    `gorm:"size:20"`
	IP          string    `gorm:"size:64"`
	UA          string    `gorm:"size:256"`
	At          time.Time `gorm:"index"`
	Extra       datatypes.JSON
}

// SystemLog 系统日志：记录系统级信息/错误；详细信息统一放入 Details(JSON)
type SystemLog struct {
	ID      uint64         `gorm:"primaryKey"`
	Level   string         `gorm:"size:20;index"`  // info | warn | error
	Source  string         `gorm:"size:100;index"` // 模块/组件来源
	Message string         `gorm:"size:512"`
	Details datatypes.JSON // 任意结构：可包含 ip/ua/stack 等
	At      time.Time      `gorm:"index"`
}
