package service

import (
	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"
)

// AuthzService 统一的授权判断：用户权限、设备默认（自身及子设备）、所有权，任一满足即放行
type AuthzService struct {
	keySvc     *KeyService
	deviceRepo *repository.DeviceRepository
	permRepo   *repository.PermissionRepository
}

func NewAuthzService(keySvc *KeyService, deviceRepo *repository.DeviceRepository, permRepo *repository.PermissionRepository) *AuthzService {
	return &AuthzService{keySvc: keySvc, deviceRepo: deviceRepo, permRepo: permRepo}
}

// ResolveUserIDFromKey 根据用户密钥解析用户ID（验证有效性与剩余次数）
func (a *AuthzService) ResolveUserIDFromKey(userKey string) (uint64, bool) {
	if userKey == "" {
		return 0, false
	}
	uid, _, err := a.keySvc.ValidateUserKey(userKey)
	if err != nil {
		return 0, false
	}
	return uid, true
}

// CanControlDevice 判断是否可对目标设备执行控制（写/管理）
// 规则：
// 1) 用户权限：若用户具有 admin.manage 或 **，允许
// 2) 所有权：若目标设备归该用户所有，允许
// 3) 设备默认：请求设备为目标或其祖先（可控制自身与子设备），允许
func (a *AuthzService) CanControlDevice(requesterDeviceUID, targetDeviceUID uint64, userID uint64) bool {
	// 用户权限：管理员
	if userID != 0 && a.keySvc.IsKeyManagerUser(userID) {
		return true
	}
	// 所有权
	if userID != 0 {
		if d, err := a.deviceRepo.FindByUID(targetDeviceUID); err == nil {
			if d.OwnerUserID != nil && *d.OwnerUserID == userID {
				return true
			}
		}
	}
	// 设备默认：自身及子设备
	if requesterDeviceUID != 0 {
		if requesterDeviceUID == targetDeviceUID {
			return true
		}
		ok, _ := a.deviceRepo.IsAncestorUID(requesterDeviceUID, targetDeviceUID)
		if ok {
			return true
		}
	}
	return false
}

// VisibleDevices 返回对请求方可见的设备集合
// 用户为管理员：全部；否则：用户拥有的设备 + （如为设备请求）该设备及其子设备
func (a *AuthzService) VisibleDevices(userID, requesterDeviceUID uint64) ([]database.Device, error) {
	if userID != 0 && a.keySvc.IsKeyManagerUser(userID) {
		return a.deviceRepo.FindAll()
	}
	result := make([]database.Device, 0)
	// 加入用户拥有的设备
	if userID != 0 {
		if ds, err := a.deviceRepo.ListByOwner(userID); err == nil {
			result = append(result, ds...)
		}
	}
	// 若为设备请求，加入该设备及其所有后代
	if requesterDeviceUID != 0 {
		if dev, err := a.deviceRepo.FindByUID(requesterDeviceUID); err == nil {
			result = append(result, *dev)
		}
		if ds, err := a.deviceRepo.ListDescendantsOfUID(requesterDeviceUID); err == nil {
			result = append(result, ds...)
		}
	}
	// 去重（按 ID）
	seen := map[uint64]struct{}{}
	uniq := make([]database.Device, 0, len(result))
	for _, d := range result {
		if _, ok := seen[d.ID]; ok {
			continue
		}
		seen[d.ID] = struct{}{}
		uniq = append(uniq, d)
	}
	return uniq, nil
}

// HasUserPermission: 代理到 KeyService 的用户权限判定
func (a *AuthzService) HasUserPermission(userID uint64, node string) bool {
	return a.keySvc.HasPermission(userID, node)
}
