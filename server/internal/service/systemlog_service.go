package service

import (
	"encoding/json"
	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"
	"time"
)

type SystemLogService struct {
	repo *repository.SystemLogRepository
}

func NewSystemLogService(repo *repository.SystemLogRepository) *SystemLogService {
	return &SystemLogService{repo: repo}
}

func (s *SystemLogService) Write(l *database.SystemLog) error {
	return s.repo.Create(l)
}

// Helper to write a simple log
func (s *SystemLogService) Info(source, message string, details any) error {
	l := &database.SystemLog{Level: "info", Source: source, Message: message, At: time.Now()}
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			l.Details = b
		}
	}
	return s.Write(l)
}
func (s *SystemLogService) Warn(source, message string, details any) error {
	l := &database.SystemLog{Level: "warn", Source: source, Message: message, At: time.Now()}
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			l.Details = b
		}
	}
	return s.Write(l)
}
func (s *SystemLogService) Error(source, message string, details any) error {
	l := &database.SystemLog{Level: "error", Source: source, Message: message, At: time.Now()}
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			l.Details = b
		}
	}
	return s.Write(l)
}

type SystemLogListInput = repository.SystemLogFilter

type SystemLogListOutput = repository.PagedSystemLogs

func (s *SystemLogService) List(in SystemLogListInput) (*SystemLogListOutput, error) {
	return s.repo.List(in)
}
