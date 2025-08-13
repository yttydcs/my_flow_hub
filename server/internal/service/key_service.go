package service

import (
	"errors"
	"time"

	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"
)

type KeyService struct {
	keys  *repository.KeyRepository
	perms *repository.PermissionRepository
}

func NewKeyService(keys *repository.KeyRepository, perms *repository.PermissionRepository) *KeyService {
	return &KeyService{keys: keys, perms: perms}
}

// IsAdminKeyUser 判断用户是否具有 admin.key 权限
func (s *KeyService) IsAdminKeyUser(userID uint64) bool {
	list, err := s.perms.ListByUserID(userID)
	if err != nil {
		return false
	}
	for _, p := range list {
		if p.Node == "admin.key" || p.Node == "**" {
			return true
		}
	}
	return false
}

// ListKeys 返回用户可见的密钥集合：普通用户仅看自己，admin.key 可看全部
func (s *KeyService) ListKeys(requestUserID uint64) ([]database.Key, error) {
	if s.IsAdminKeyUser(requestUserID) {
		return s.keys.ListAll()
	}
	return s.keys.ListByOwner(requestUserID)
}

func (s *KeyService) CreateKey(ownerUserID uint64, bindType *string, bindID *uint64, secretHash string, expiresAt *time.Time, maxUses *int, metaJSON []byte) (*database.Key, error) {
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

func (s *KeyService) UpdateKey(requestUserID uint64, k *database.Key) error {
	// 普通用户只能更新自己的密钥；admin.key 可更新任意
	if !s.IsAdminKeyUser(requestUserID) {
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
	if !s.IsAdminKeyUser(requestUserID) {
		if existing.OwnerUserID == nil || *existing.OwnerUserID != requestUserID {
			return errors.New("forbidden")
		}
	}
	return s.keys.Delete(id)
}
