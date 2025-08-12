package service

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type SessionService struct {
	users *repository.UserRepository
}

func NewSessionService(users *repository.UserRepository) *SessionService {
	return &SessionService{users: users}
}

// Login returns a pseudo session token (hash) if username/password checks out
func (s *SessionService) Login(username, password string) (string, *database.User, bool) {
	u, err := s.users.FindByUsername(username)
	if err != nil || u.Disabled {
		return "", nil, false
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", nil, false
	}
	// naive token: sha256(username + now). In production use JWT or DB stored token.
	h := sha256.Sum256([]byte(username + time.Now().String()))
	token := hex.EncodeToString(h[:])
	return token, u, true
}
