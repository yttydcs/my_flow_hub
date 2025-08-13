package service

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"

	"sync"

	"golang.org/x/crypto/bcrypt"
)

type SessionService struct {
	users *repository.UserRepository
	mu    sync.RWMutex
	// 简易内存会话：token -> userID
	sessions map[string]uint64
}

func NewSessionService(users *repository.UserRepository) *SessionService {
	return &SessionService{users: users, sessions: make(map[string]uint64)}
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
	s.mu.Lock()
	s.sessions[token] = u.ID
	s.mu.Unlock()
	return token, u, true
}

// Resolve 返回 token 对应的用户（若存在）
func (s *SessionService) Resolve(token string) (*database.User, bool) {
	s.mu.RLock()
	uid, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	u, err := s.users.FindByID(uid)
	if err != nil {
		return nil, false
	}
	if u.Disabled {
		return nil, false
	}
	return u, true
}
