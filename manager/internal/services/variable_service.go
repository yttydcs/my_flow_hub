package services

import (
	"encoding/json"
	"errors"
	"myflowhub/manager/internal/client"
	"myflowhub/manager/internal/repository"
	"myflowhub/pkg/database"
	"myflowhub/pkg/protocol"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// VariableService 变量业务逻辑层
type VariableService struct {
	variableRepo *repository.VariableRepository
	deviceRepo   *repository.DeviceRepository
	hubClient    *client.HubClient
}

// NewVariableService 创建变量服务实例
func NewVariableService(variableRepo *repository.VariableRepository, deviceRepo *repository.DeviceRepository, hubClient *client.HubClient) *VariableService {
	return &VariableService{
		variableRepo: variableRepo,
		deviceRepo:   deviceRepo,
		hubClient:    hubClient,
	}
}

// CreateVariableRequest 创建变量请求
type CreateVariableRequest struct {
	Name          string      `json:"name"`
	Value         interface{} `json:"value"`
	OwnerDeviceID uint64      `json:"deviceId"`
}

// UpdateVariableRequest 更新变量请求
type UpdateVariableRequest struct {
	ID    uint64      `json:"id"`
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// GetAllVariables 获取所有变量
func (s *VariableService) GetAllVariables() ([]database.DeviceVariable, error) {
	return s.variableRepo.GetAll()
}

// GetVariablesByDeviceID 根据设备ID获取变量
func (s *VariableService) GetVariablesByDeviceID(deviceID uint64) ([]database.DeviceVariable, error) {
	return s.variableRepo.GetByDeviceID(deviceID)
}

// GetVariableByID 根据ID获取变量
func (s *VariableService) GetVariableByID(id uint64) (*database.DeviceVariable, error) {
	return s.variableRepo.GetByID(id)
}

// CreateVariable 创建变量
func (s *VariableService) CreateVariable(req CreateVariableRequest) (*database.DeviceVariable, error) {
	// 验证必填字段
	if req.Name == "" {
		return nil, errors.New("variable name is required")
	}

	// 检查设备是否存在
	_, err := s.deviceRepo.GetByID(req.OwnerDeviceID)
	if err != nil {
		return nil, errors.New("device not found")
	}

	// 检查变量名是否在设备中已存在
	existing, err := s.variableRepo.GetByDeviceIDAndName(req.OwnerDeviceID, req.Name)
	if err == nil && existing != nil {
		return nil, errors.New("variable with this name already exists for this device")
	}

	// 转换值为JSON格式
	valueJSON, err := json.Marshal(req.Value)
	if err != nil {
		return nil, errors.New("invalid variable value")
	}

	// 创建变量对象
	variable := &database.DeviceVariable{
		VariableName:  req.Name,
		Value:         datatypes.JSON(valueJSON),
		OwnerDeviceID: req.OwnerDeviceID,
	}

	// 保存到数据库
	if err := s.variableRepo.Create(variable); err != nil {
		return nil, err
	}

	// 预加载设备信息
	variable, err = s.variableRepo.GetByID(variable.ID)
	if err != nil {
		return nil, err
	}

	// 通知Server创建变量
	if s.hubClient.IsConnected() {
		s.notifyServerVariableCreate(variable)
	}

	return variable, nil
}

// UpdateVariable 更新变量
func (s *VariableService) UpdateVariable(req UpdateVariableRequest) (*database.DeviceVariable, error) {
	// 查找变量
	variable, err := s.variableRepo.GetByID(req.ID)
	if err != nil {
		return nil, errors.New("variable not found")
	}

	// 转换值为JSON格式
	valueJSON, err := json.Marshal(req.Value)
	if err != nil {
		return nil, errors.New("invalid variable value")
	}

	// 更新字段
	variable.VariableName = req.Name
	variable.Value = datatypes.JSON(valueJSON)

	// 保存更改
	if err := s.variableRepo.Update(variable); err != nil {
		return nil, err
	}

	// 预加载设备信息
	variable, err = s.variableRepo.GetByID(variable.ID)
	if err != nil {
		return nil, err
	}

	// 通知Server更新变量
	if s.hubClient.IsConnected() {
		s.notifyServerVariableUpdate(variable)
	}

	return variable, nil
}

// DeleteVariable 删除变量
func (s *VariableService) DeleteVariable(id uint64) error {
	// 检查变量是否存在
	variable, err := s.variableRepo.GetByID(id)
	if err != nil {
		return errors.New("variable not found")
	}

	// 通知Server删除变量
	if s.hubClient.IsConnected() {
		s.notifyServerVariableDelete(variable)
	}

	// 删除变量
	return s.variableRepo.Delete(id)
}

// notifyServerVariableCreate 通知Server创建变量
func (s *VariableService) notifyServerVariableCreate(variable *database.DeviceVariable) {
	msg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "variable_create",
		Target:    variable.Device.DeviceUID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"variable": variable,
		},
	}
	s.hubClient.SendMessage(msg)
}

// notifyServerVariableUpdate 通知Server更新变量
func (s *VariableService) notifyServerVariableUpdate(variable *database.DeviceVariable) {
	msg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "variable_update",
		Target:    variable.Device.DeviceUID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"variable": variable,
		},
	}
	s.hubClient.SendMessage(msg)
}

// notifyServerVariableDelete 通知Server删除变量
func (s *VariableService) notifyServerVariableDelete(variable *database.DeviceVariable) {
	msg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "variable_delete",
		Target:    variable.Device.DeviceUID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"variableId":   variable.ID,
			"variableName": variable.VariableName,
		},
	}
	s.hubClient.SendMessage(msg)
}
