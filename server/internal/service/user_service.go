package service

import (
	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) List() ([]database.User, error) {
	return s.repo.FindAll()
}

func (s *UserService) Get(id uint64) (*database.User, error) {
	return s.repo.FindByID(id)
}

func (s *UserService) GetByUsername(username string) (*database.User, error) {
	return s.repo.FindByUsername(username)
}

func (s *UserService) Create(username, displayName, password string) (*database.User, error) {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	u := &database.User{Username: username, DisplayName: displayName, PasswordHash: string(hash)}
	if err := s.repo.Create(u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) Update(id uint64, displayName *string, password *string, disabled *bool) error {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if displayName != nil {
		u.DisplayName = *displayName
	}
	if password != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		u.PasswordHash = string(hash)
	}
	if disabled != nil {
		u.Disabled = *disabled
	}
	return s.repo.Update(u)
}

func (s *UserService) Delete(id uint64) error {
	return s.repo.Delete(id)
}
