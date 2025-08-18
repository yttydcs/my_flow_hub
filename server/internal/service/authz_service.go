package service

import (
	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"
)

// AuthContext 封装了单次请求的授权上下文
type AuthContext struct {
	UserID      uint64
	Permissions map[string]struct{}
	IsAdmin     bool
}

// AuthzService 统一的授权判断服务
type AuthzService struct {
	keySvc     *KeyService
	deviceRepo *repository.DeviceRepository
	permRepo   *repository.PermissionRepository
}

func NewAuthzService(keySvc *KeyService, deviceRepo *repository.DeviceRepository, permRepo *repository.PermissionRepository) *AuthzService {
	return &AuthzService{keySvc: keySvc, deviceRepo: deviceRepo, permRepo: permRepo}
}

// ResolveAuthContextFromKey 根据用户密钥解析完整的授权上下文
func (a *AuthzService) ResolveAuthContextFromKey(userKey string) (*AuthContext, bool) {
	if userKey == "" {
		return nil, false
	}

	uid, key, err := a.keySvc.PeekUserKey(userKey)
	if err != nil {
		return nil, false
	}

	// 1. 获取密钥自身的权限
	keyPerms, err := a.permRepo.ListByKeyID(key.ID)
	if err != nil {
		return nil, false
	}

	ctx := &AuthContext{
		UserID:      uid,
		Permissions: make(map[string]struct{}),
	}

	// 2. 如果密钥有独立的权限设置，则使用密钥的权限
	if len(keyPerms) > 0 {
		for _, p := range keyPerms {
			ctx.Permissions[p.Node] = struct{}{}
		}
	} else {
		// 3. 否则，继承其所有者的全部权限
		userPerms, err := a.permRepo.ListByUserID(uid)
		if err != nil {
			return nil, false
		}
		for _, p := range userPerms {
			ctx.Permissions[p.Node] = struct{}{}
		}
	}

	// 4. 检查管理员权限
	if _, ok := ctx.Permissions["**"]; ok {
		ctx.IsAdmin = true
	}
	if _, ok := ctx.Permissions["admin.manage"]; ok {
		ctx.IsAdmin = true
	}

	return ctx, true
}

// CanControlDevice 判断是否可对目标设备执行控制（写/管理）
func (a *AuthzService) CanControlDevice(requesterDeviceUID, targetDeviceUID uint64, authCtx *AuthContext) bool {
	if authCtx == nil {
		return false
	}
	// 1. 管理员权限
	if authCtx.IsAdmin {
		return true
	}

	// 2. 所有权
	if authCtx.UserID != 0 {
		if d, err := a.deviceRepo.FindByUID(targetDeviceUID); err == nil {
			if d.OwnerUserID != nil && *d.OwnerUserID == authCtx.UserID {
				return true
			}
		}
	}

	// 3. 设备默认权限：自身及子设备
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
func (a *AuthzService) VisibleDevices(authCtx *AuthContext, requesterDeviceUID uint64) ([]database.Device, error) {
	if authCtx.IsAdmin {
		return a.deviceRepo.FindAll()
	}

	result := make([]database.Device, 0)
	// 加入用户拥有的设备
	if authCtx.UserID != 0 {
		if ds, err := a.deviceRepo.ListByOwner(authCtx.UserID); err == nil {
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

	// 去重
	seen := map[uint64]struct{}{}
	uniq := make([]database.Device, 0, len(result))
	for _, d := range result {
		if _, ok := seen[d.ID]; !ok {
			seen[d.ID] = struct{}{}
			uniq = append(uniq, d)
		}
	}
	return uniq, nil
}

// HasPermission 检查授权上下文中是否包含指定权限节点
func (a *AuthzService) HasPermission(authCtx *AuthContext, node string) bool {
	if authCtx == nil {
		return false
	}
	if authCtx.IsAdmin {
		return true
	}
	return CanGrant(authCtx.Permissions, node)
}
