package repository

import (
	"myflowhub/pkg/database"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindAll() ([]database.User, error) {
	var users []database.User
	err := r.db.Find(&users).Error
	return users, err
}

func (r *UserRepository) FindByID(id uint64) (*database.User, error) {
	var u database.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByUsername(username string) (*database.User, error) {
	var u database.User
	if err := r.db.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(u *database.User) error {
	return r.db.Create(u).Error
}

func (r *UserRepository) Update(u *database.User) error {
	return r.db.Save(u).Error
}

func (r *UserRepository) Delete(id uint64) error {
	return r.db.Delete(&database.User{}, id).Error
}
