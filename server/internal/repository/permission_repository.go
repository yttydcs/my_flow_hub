package repository

import (
	"myflowhub/pkg/database"

	"gorm.io/gorm"
)

type PermissionRepository struct{ db *gorm.DB }

func NewPermissionRepository(db *gorm.DB) *PermissionRepository { return &PermissionRepository{db: db} }

func (r *PermissionRepository) ListByUserID(userID uint64) ([]database.Permission, error) {
	var ps []database.Permission
	err := r.db.Where("subject_type = ? AND subject_id = ?", "user", userID).Find(&ps).Error
	return ps, err
}

func (r *PermissionRepository) AddUserNode(userID uint64, node string, createdBy *uint64) error {
	p := &database.Permission{SubjectType: "user", SubjectID: userID, Node: node, CreatedBy: createdBy}
	return r.db.Create(p).Error
}

func (r *PermissionRepository) RemoveUserNode(userID uint64, node string) error {
	return r.db.Where("subject_type = ? AND subject_id = ? AND node = ?", "user", userID, node).Delete(&database.Permission{}).Error
}
