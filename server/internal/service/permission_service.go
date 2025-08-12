package service

import (
	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"
)

// PermissionService 提供基础权限判断（最小可行版）
// 后续可扩展为基于权限节点与密钥的完整判定。
type PermissionService struct {
	deviceRepo *repository.DeviceRepository
}

func NewPermissionService(deviceRepo *repository.DeviceRepository) *PermissionService {
	return &PermissionService{deviceRepo: deviceRepo}
}

// IsAdminDevice: 设备是否为管理器（视为具备 admin.manage）
func (p *PermissionService) IsAdminDevice(deviceUID uint64) bool {
	dev, err := p.deviceRepo.FindByUID(deviceUID)
	if err != nil {
		return false
	}
	return dev.Role == database.RoleManager
}

// CanAccessAllDevices: 只有管理员可获取所有设备
func (p *PermissionService) CanAccessAllDevices(requesterUID uint64) bool {
	return p.IsAdminDevice(requesterUID)
}

// CanReadVarsForDevice: 管理员或目标即自己
func (p *PermissionService) CanReadVarsForDevice(requesterUID uint64, targetUID uint64) bool {
	if p.IsAdminDevice(requesterUID) {
		return true
	}
	return requesterUID == targetUID
}

// CanWriteVarsForDevice: 管理员或目标即自己
func (p *PermissionService) CanWriteVarsForDevice(requesterUID uint64, targetUID uint64) bool {
	if p.IsAdminDevice(requesterUID) {
		return true
	}
	return requesterUID == targetUID
}

// CanManageDevice: 只有管理员（最小实现；未来加入所有者判断）
func (p *PermissionService) CanManageDevice(requesterUID uint64, targetUID uint64) bool {
	return p.IsAdminDevice(requesterUID)
}
