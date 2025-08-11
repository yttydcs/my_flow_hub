package services

import (
	"errors"
	"myflowhub/manager/internal/client"
	"myflowhub/manager/internal/repository"
	"myflowhub/pkg/database"
	"myflowhub/pkg/protocol"
	"time"

	"github.com/google/uuid"
)

// DeviceService 设备业务逻辑层
type DeviceService struct {
	deviceRepo   *repository.DeviceRepository
	variableRepo *repository.VariableRepository
	hubClient    *client.HubClient
}

// NewDeviceService 创建设备服务实例
func NewDeviceService(deviceRepo *repository.DeviceRepository, variableRepo *repository.VariableRepository, hubClient *client.HubClient) *DeviceService {
	return &DeviceService{
		deviceRepo:   deviceRepo,
		variableRepo: variableRepo,
		hubClient:    hubClient,
	}
}

// CreateDeviceRequest 创建设备请求
type CreateDeviceRequest struct {
	HardwareID string  `json:"hardwareId"`
	Name       string  `json:"name"`
	Role       string  `json:"role"`
	ParentID   *uint64 `json:"parentId"`
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	ID       uint64  `json:"id"`
	Name     string  `json:"name"`
	Role     string  `json:"role"`
	ParentID *uint64 `json:"parentId"`
}

// GetAllDevices 获取所有设备
func (s *DeviceService) GetAllDevices() ([]database.Device, error) {
	return s.deviceRepo.GetAll()
}

// GetDeviceByID 根据ID获取设备
func (s *DeviceService) GetDeviceByID(id uint64) (*database.Device, error) {
	return s.deviceRepo.GetByID(id)
}

// CreateDevice 创建设备
func (s *DeviceService) CreateDevice(req CreateDeviceRequest) (*database.Device, error) {
	// 验证必填字段
	if req.HardwareID == "" || req.Role == "" {
		return nil, errors.New("HardwareID and Role are required")
	}

	// 检查硬件ID是否已存在
	existing, err := s.deviceRepo.GetByHardwareID(req.HardwareID)
	if err == nil && existing != nil {
		return nil, errors.New("device with this hardware ID already exists")
	}

	// 转换角色类型
	role, err := s.parseRole(req.Role)
	if err != nil {
		return nil, err
	}

	// 如果指定了父设备，验证父设备是否存在
	if req.ParentID != nil {
		_, err := s.deviceRepo.GetByID(*req.ParentID)
		if err != nil {
			return nil, errors.New("parent device not found")
		}
	}

	// 创建设备对象
	device := &database.Device{
		HardwareID: req.HardwareID,
		Name:       req.Name,
		Role:       role,
		ParentID:   req.ParentID,
	}

	// 保存到数据库
	if err := s.deviceRepo.Create(device); err != nil {
		return nil, err
	}

	// 通知Server创建设备
	if s.hubClient.IsConnected() {
		s.notifyServerDeviceCreate(device)
	}

	return device, nil
}

// UpdateDevice 更新设备
func (s *DeviceService) UpdateDevice(req UpdateDeviceRequest) (*database.Device, error) {
	// 查找设备
	device, err := s.deviceRepo.GetByID(req.ID)
	if err != nil {
		return nil, errors.New("device not found")
	}

	// 转换角色类型
	role, err := s.parseRole(req.Role)
	if err != nil {
		return nil, err
	}

	// 如果指定了父设备，验证父设备是否存在
	if req.ParentID != nil {
		_, err := s.deviceRepo.GetByID(*req.ParentID)
		if err != nil {
			return nil, errors.New("parent device not found")
		}
	}

	// 更新字段
	device.Name = req.Name
	device.Role = role
	device.ParentID = req.ParentID

	// 保存更改
	if err := s.deviceRepo.Update(device); err != nil {
		return nil, err
	}

	// 通知Server更新设备
	if s.hubClient.IsConnected() {
		s.notifyServerDeviceUpdate(device)
	}

	return device, nil
}

// DeleteDevice 删除设备
func (s *DeviceService) DeleteDevice(id uint64) error {
	// 检查设备是否存在
	device, err := s.deviceRepo.GetByID(id)
	if err != nil {
		return errors.New("device not found")
	}

	// 检查是否有子设备
	children, err := s.deviceRepo.GetChildren(id)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		return errors.New("cannot delete device with child devices")
	}

	// 通知Server踢出设备
	if s.hubClient.IsConnected() {
		s.notifyServerDeviceKick(device)
	}

	// 先删除该设备的所有变量（避免外键约束违反）
	if err := s.variableRepo.DeleteByDeviceID(id); err != nil {
		return errors.New("failed to delete device variables: " + err.Error())
	}

	// 删除设备
	return s.deviceRepo.Delete(id)
}

// parseRole 解析角色字符串
func (s *DeviceService) parseRole(roleStr string) (database.DeviceRole, error) {
	switch roleStr {
	case "node":
		return database.RoleNode, nil
	case "relay":
		return database.RoleRelay, nil
	case "hub":
		return database.RoleHub, nil
	case "manager":
		return database.RoleManager, nil
	default:
		return "", errors.New("invalid role. Must be one of: node, relay, hub, manager")
	}
}

// notifyServerDeviceCreate 通知Server创建设备
func (s *DeviceService) notifyServerDeviceCreate(device *database.Device) {
	msg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "device_create",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"device": device,
		},
	}
	s.hubClient.SendMessage(msg)
}

// notifyServerDeviceUpdate 通知Server更新设备
func (s *DeviceService) notifyServerDeviceUpdate(device *database.Device) {
	msg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "device_update",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"device": device,
		},
	}
	s.hubClient.SendMessage(msg)
}

// notifyServerDeviceKick 通知Server踢出设备
func (s *DeviceService) notifyServerDeviceKick(device *database.Device) {
	msg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "device_kick",
		Target:    device.DeviceUID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"reason": "deleted",
		},
	}
	s.hubClient.SendMessage(msg)
}
