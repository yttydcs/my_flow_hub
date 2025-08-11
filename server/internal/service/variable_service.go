package service

import (
	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"
)

// VariableService 提供了变量相关的业务逻辑
type VariableService struct {
	repo *repository.VariableRepository
}

// NewVariableService 创建一个新的 VariableService
func NewVariableService(repo *repository.VariableRepository) *VariableService {
	return &VariableService{repo: repo}
}

// GetVariablesByDeviceID 根据设备 ID 获取变量
func (s *VariableService) GetVariablesByDeviceID(deviceID uint64) ([]database.DeviceVariable, error) {
	return s.repo.FindByDeviceID(deviceID)
}

// GetVariableByOwnerAndName 根据所有者和变量名获取变量
func (s *VariableService) GetVariableByOwnerAndName(ownerID uint64, name string) (*database.DeviceVariable, error) {
	return s.repo.FindByOwnerAndName(ownerID, name)
}

// UpsertVariable 更新或创建变量
func (s *VariableService) UpsertVariable(variable *database.DeviceVariable) error {
	return s.repo.Upsert(variable)
}

// GetVariableByDeviceUIDAndVarName 根据设备 UID 和变量名获取变量
func (s *VariableService) GetVariableByDeviceUIDAndVarName(deviceUID, varName string) (*database.DeviceVariable, error) {
	return s.repo.FindByDeviceUIDAndVarName(deviceUID, varName)
}

// GetAllVariables 获取所有变量
func (s *VariableService) GetAllVariables() ([]database.DeviceVariable, error) {
	return s.repo.FindAll()
}

// DeleteVariable 删除变量
func (s *VariableService) DeleteVariable(ownerID uint64, name string) error {
	return s.repo.Delete(ownerID, name)
}
