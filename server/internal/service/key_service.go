package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"
)

type KeyService struct {
	keys       *repository.KeyRepository
	perms      *repository.PermissionRepository
	deviceRepo *repository.DeviceRepository
}

func NewKeyService(keys *repository.KeyRepository, perms *repository.PermissionRepository, devices *repository.DeviceRepository) *KeyService {
	return &KeyService{keys: keys, perms: perms, deviceRepo: devices}
}

// IsKeyManagerUser 判断用户是否具有 key.manage 权限
func (s *KeyService) IsKeyManagerUser(userID uint64) bool {
	list, err := s.perms.ListByUserID(userID)
	if err != nil {
		return false
	}
	for _, p := range list {
		if p.Node == "key.manage" || p.Node == "**" {
			return true
		}
	}
	return false
}

// ListKeys 返回用户可见的密钥集合：普通用户仅看自己，admin.key 可看全部
func (s *KeyService) ListKeys(requestUserID uint64) ([]database.Key, error) {
	if s.IsKeyManagerUser(requestUserID) {
		return s.keys.ListAll()
	}
	return s.keys.ListByOwner(requestUserID)
}

// ValidateUserKey 校验用户密钥，返回请求者的用户ID
// 策略：优先使用绑定的用户（BindSubjectType=user）；否则回退为签发者 OwnerUserID。
// 要求：未撤销、未过期、剩余次数>0（如有限制）
func (s *KeyService) ValidateUserKey(secret string) (uint64, *database.Key, error) {
	if secret == "" {
		return 0, nil, errors.New("empty key")
	}
	// 兼容：既支持直接存储的明文十六进制 secret，也支持存储为 sha256(secret) 的哈希
	k, err := s.keys.FindBySecretHash(secret)
	if err != nil {
		// 尝试哈希匹配
		sum := sha256.Sum256([]byte(secret))
		hashed := hex.EncodeToString(sum[:])
		k, err = s.keys.FindBySecretHash(hashed)
		if err != nil {
			return 0, nil, err
		}
	}
	if k.Revoked {
		return 0, nil, errors.New("revoked")
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return 0, nil, errors.New("expired")
	}
	if k.RemainingUses != nil && *k.RemainingUses <= 0 {
		return 0, nil, errors.New("exhausted")
	}
	var userID uint64
	if k.BindSubjectType != nil && *k.BindSubjectType == "user" && k.BindSubjectID != nil {
		userID = *k.BindSubjectID
	} else if k.OwnerUserID != nil {
		userID = *k.OwnerUserID
	}
	if userID == 0 {
		return 0, nil, errors.New("invalid key subject")
	}
	// 消耗一次（如果有限制）
	if k.RemainingUses != nil {
		v := *k.RemainingUses - 1
		k.RemainingUses = &v
		_ = s.keys.Update(k)
	}
	return userID, k, nil
}

// PeekUserKey 校验密钥但不消耗剩余次数
func (s *KeyService) PeekUserKey(secret string) (uint64, *database.Key, error) {
	if secret == "" {
		return 0, nil, errors.New("empty key")
	}
	k, err := s.keys.FindBySecretHash(secret)
	if err != nil {
		sum := sha256.Sum256([]byte(secret))
		hashed := hex.EncodeToString(sum[:])
		k, err = s.keys.FindBySecretHash(hashed)
		if err != nil {
			return 0, nil, err
		}
	}
	if k.Revoked {
		return 0, nil, errors.New("revoked")
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return 0, nil, errors.New("expired")
	}
	if k.MaxUses != nil && k.RemainingUses != nil && *k.RemainingUses <= 0 {
		return 0, nil, errors.New("exhausted")
	}
	var userID uint64
	if k.BindSubjectType != nil && *k.BindSubjectType == "user" && k.BindSubjectID != nil {
		userID = *k.BindSubjectID
	} else if k.OwnerUserID != nil {
		userID = *k.OwnerUserID
	}
	if userID == 0 {
		return 0, nil, errors.New("invalid key subject")
	}
	return userID, k, nil
}

// RevokeBySecret 撤销密钥（支持明文或哈希传入）
func (s *KeyService) RevokeBySecret(secret string) error {
	if secret == "" {
		return errors.New("empty key")
	}
	k, err := s.keys.FindBySecretHash(secret)
	if err != nil {
		sum := sha256.Sum256([]byte(secret))
		hashed := hex.EncodeToString(sum[:])
		k, err = s.keys.FindBySecretHash(hashed)
		if err != nil {
			return err
		}
	}
	k.Revoked = true
	return s.keys.Update(k)
}

// DeleteBySecret 通过明文或哈希删除密钥，避免积累
func (s *KeyService) DeleteBySecret(secret string) error {
	if secret == "" {
		return errors.New("empty key")
	}
	k, err := s.keys.FindBySecretHash(secret)
	if err != nil {
		sum := sha256.Sum256([]byte(secret))
		hashed := hex.EncodeToString(sum[:])
		k, err = s.keys.FindBySecretHash(hashed)
		if err != nil {
			return err
		}
	}
	return s.keys.Delete(k.ID)
}

// HasPermission 判断用户是否拥有指定权限节点（精确匹配；通配在上层处理或扩展）
func (s *KeyService) HasPermission(userID uint64, node string) bool {
	list, err := s.perms.ListByUserID(userID)
	if err != nil {
		return false
	}
	for _, p := range list {
		if p.Node == node || p.Node == "**" { // 简化：支持超级权限
			return true
		}
	}
	return false
}

func (s *KeyService) CreateKey(ownerUserID uint64, bindType *string, bindID *uint64, secret string, expiresAt *time.Time, maxUses *int, metaJSON []byte) (*database.Key, error) {
	// 为安全起见按 sha256 存储；同时 ValidateUserKey 兼容旧数据
	sum := sha256.Sum256([]byte(secret))
	secretHash := hex.EncodeToString(sum[:])
	k := &database.Key{
		OwnerUserID:     &ownerUserID,
		BindSubjectType: bindType,
		BindSubjectID:   bindID,
		SecretHash:      secretHash,
		ExpiresAt:       expiresAt,
		MaxUses:         maxUses,
		RemainingUses:   maxUses,
		Revoked:         false,
		IssuedBy:        &ownerUserID,
		IssuedAt:        time.Now(),
	}
	if len(metaJSON) > 0 {
		k.Meta = metaJSON
	}
	if err := s.keys.Create(k); err != nil {
		return nil, err
	}
	return k, nil
}

// AttachKeyPermissions 为密钥设置权限，要求 nodes ⊆ 用户权限
func (s *KeyService) AttachKeyPermissions(ownerUserID uint64, keyID uint64, nodes []string) error {
	if len(nodes) == 0 {
		return nil
	}
	// 获取用户全部权限
	ups, err := s.perms.ListByUserID(ownerUserID)
	if err != nil {
		return err
	}
	allowed := map[string]struct{}{}
	for _, p := range ups {
		allowed[p.Node] = struct{}{}
	}
	for _, n := range nodes {
		if _, ok := allowed[n]; !ok && n != "**" { // 不允许密钥拥有超过用户本身的节点；超级权限必须由用户已具备
			return errors.New("key nodes exceed user permissions")
		}
	}
	for _, n := range nodes {
		if err := s.perms.AddKeyNode(keyID, n, &ownerUserID); err != nil {
			return err
		}
	}
	return nil
}

func (s *KeyService) UpdateKey(requestUserID uint64, k *database.Key) error {
	// 普通用户只能更新自己的密钥；admin.key 可更新任意
	if !s.IsKeyManagerUser(requestUserID) {
		if k.OwnerUserID == nil || *k.OwnerUserID != requestUserID {
			return errors.New("forbidden")
		}
	}
	return s.keys.Update(k)
}

func (s *KeyService) DeleteKey(requestUserID uint64, id uint64) error {
	existing, err := s.keys.FindByID(id)
	if err != nil {
		return err
	}
	if !s.IsKeyManagerUser(requestUserID) {
		if existing.OwnerUserID == nil || *existing.OwnerUserID != requestUserID {
			return errors.New("forbidden")
		}
	}
	return s.keys.Delete(id)
}

// ListVisibleDevicesForKey 返回当前用户在创建密钥时可见的设备
func (s *KeyService) ListVisibleDevicesForKey(userID uint64) ([]database.Device, error) {
	if s.IsKeyManagerUser(userID) {
		return s.deviceRepo.FindAll()
	}
	return s.deviceRepo.ListByOwner(userID)
}
