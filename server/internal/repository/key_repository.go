package repository

import (
	"myflowhub/pkg/database"

	"gorm.io/gorm"
)

type KeyRepository struct{ db *gorm.DB }

func NewKeyRepository(db *gorm.DB) *KeyRepository { return &KeyRepository{db: db} }

func (r *KeyRepository) FindByID(id uint64) (*database.Key, error) {
	var k database.Key
	if err := r.db.First(&k, id).Error; err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *KeyRepository) ListAll() ([]database.Key, error) {
	var ks []database.Key
	err := r.db.Find(&ks).Error
	return ks, err
}

func (r *KeyRepository) ListByOwner(ownerID uint64) ([]database.Key, error) {
	var ks []database.Key
	err := r.db.Where("owner_user_id = ?", ownerID).Find(&ks).Error
	return ks, err
}

func (r *KeyRepository) Create(k *database.Key) error {
	return r.db.Create(k).Error
}

func (r *KeyRepository) Update(k *database.Key) error {
	return r.db.Save(k).Error
}

func (r *KeyRepository) Delete(id uint64) error {
	return r.db.Delete(&database.Key{}, id).Error
}

func (r *KeyRepository) FindBySecretHash(hash string) (*database.Key, error) {
	var k database.Key
	if err := r.db.Where("secret_hash = ? AND revoked = FALSE", hash).First(&k).Error; err != nil {
		return nil, err
	}
	return &k, nil
}
