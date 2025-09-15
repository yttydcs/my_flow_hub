package binproto

//go:generate go run ../../../tools/binproto-gen/main.go -input types.go -output generated.go

// TypeID constants will be generated from these structs.
// The generator will automatically assign TypeIDs based on the order of definition below,
// starting from a base value or mapping from a config. For now, we'll assume a simple ordering.
// Request structs should be followed by their Response structs where applicable.

// -------------------------
// General & System Messages
// -------------------------

// OKResp corresponds to TypeOKResp (0)
type OKResp struct {
	RequestID uint64 `bin:"u64"`
	Code      int32  `bin:"i32"`
	Message   string `bin:"string"`
}

// ErrResp corresponds to TypeErrResp (1)
type ErrResp struct {
	RequestID uint64 `bin:"u64"`
	Code      int32  `bin:"i32"`
	Message   string `bin:"string"`
}

// -------------------------
// Auth & User Management
// -------------------------

// ManagerAuthReq corresponds to TypeManagerAuthReq (100)
type ManagerAuthReq struct {
	Token string `bin:"string"`
}

// ManagerAuthResp corresponds to TypeManagerAuthResp (101)
type ManagerAuthResp struct {
	RequestID uint64 `bin:"u64"`
	DeviceUID uint64 `bin:"u64"`
	Role      string `bin:"string,optional"`
}

// UserLoginReq corresponds to TypeUserLoginReq (110)
type UserLoginReq struct {
	Username string `bin:"string"`
	Password string `bin:"string"`
}

// UserLoginResp corresponds to TypeUserLoginResp (111)
type UserLoginResp struct {
	RequestID   uint64   `bin:"u64"`
	KeyID       uint64   `bin:"u64"`
	UserID      uint64   `bin:"u64"`
	Secret      string   `bin:"string"`
	Username    string   `bin:"string"`
	DisplayName string   `bin:"string"`
	Permissions []string `bin:"slice,varint"`
}

// UserMeReq corresponds to TypeUserMeReq (112)
type UserMeReq struct {
	UserKey string `bin:"string"`
}

// UserMeResp corresponds to TypeUserMeResp (113)
type UserMeResp struct {
	RequestID   uint64   `bin:"u64"`
	UserID      uint64   `bin:"u64"`
	Username    string   `bin:"string"`
	DisplayName string   `bin:"string"`
	Permissions []string `bin:"slice,varint"`
}

// UserLogoutReq corresponds to TypeUserLogoutReq (114)
type UserLogoutReq struct {
	UserKey string `bin:"string"`
}

// UserItem is a reusable struct for user information.
type UserItem struct {
	ID           uint64 `bin:"u64"`
	Username     string `bin:"string"`
	DisplayName  string `bin:"string"`
	Disabled     bool   `bin:"bool"`
	CreatedAtSec int64  `bin:"i64"`
	UpdatedAtSec int64  `bin:"i64"`
}

// UserListReq corresponds to TypeUserListReq (180)
type UserListReq struct {
	UserKey string `bin:"string"`
}

// UserListResp corresponds to TypeUserListResp (181)
type UserListResp struct {
	RequestID uint64     `bin:"u64"`
	Users     []UserItem `bin:"slice,u32"`
}

// UserCreateReq corresponds to TypeUserCreateReq (182)
type UserCreateReq struct {
	UserKey     string `bin:"string"`
	Username    string `bin:"string"`
	DisplayName string `bin:"string"`
	Password    string `bin:"string"`
}

// UserCreateResp corresponds to TypeUserCreateResp (183)
type UserCreateResp struct {
	RequestID uint64 `bin:"u64"`
	UserID    uint64 `bin:"u64"`
}

// UserUpdateReq corresponds to TypeUserUpdateReq (184)
type UserUpdateReq struct {
	UserKey     string  `bin:"string"`
	ID          uint64  `bin:"u64"`
	DisplayName *string `bin:"string,optional"`
	Password    *string `bin:"string,optional"`
	Disabled    *bool   `bin:"bool,optional"`
}

// UserDeleteReq corresponds to TypeUserDeleteReq (185)
type UserDeleteReq struct {
	UserKey string `bin:"string"`
	ID      uint64 `bin:"u64"`
}

// -------------------------
// Device Management
// -------------------------

// DeviceItem is a reusable struct for device information.
type DeviceItem struct {
	ID           uint64  `bin:"u64"`
	DeviceUID    uint64  `bin:"u64"`
	HardwareID   string  `bin:"string"`
	Role         string  `bin:"string"`
	Name         string  `bin:"string"`
	ParentID     *uint64 `bin:"u64,optional"`
	OwnerUserID  *uint64 `bin:"u64,optional"`
	LastSeenSec  *int64  `bin:"i64,optional"`
	CreatedAtSec int64   `bin:"i64"`
	UpdatedAtSec int64   `bin:"i64"`
}

// QueryNodesReq corresponds to TypeQueryNodesReq (20)
type QueryNodesReq struct {
	UserKey string `bin:"string,optional"`
}

// QueryNodesResp corresponds to TypeQueryNodesResp (120)
type QueryNodesResp struct {
	RequestID uint64       `bin:"u64"`
	Devices   []DeviceItem `bin:"slice,varint"`
}

// CreateDeviceReq corresponds to TypeCreateDeviceReq (21)
type CreateDeviceReq struct {
	UserKey string     `bin:"string,optional"`
	Device  DeviceItem `bin:"struct"`
}

// UpdateDeviceReq corresponds to TypeUpdateDeviceReq (22)
type UpdateDeviceReq struct {
	UserKey string     `bin:"string,optional"`
	Device  DeviceItem `bin:"struct"`
}

// DeleteDeviceReq corresponds to TypeDeleteDeviceReq (23)
type DeleteDeviceReq struct {
	UserKey string `bin:"string,optional"`
	ID      uint64 `bin:"u64"`
}