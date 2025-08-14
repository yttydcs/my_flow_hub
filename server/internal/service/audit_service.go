package service

import (
	"time"

	"myflowhub/pkg/database"
	"myflowhub/server/internal/repository"
)

type AuditService struct {
	repo   *repository.AuditLogRepository
	keySvc *KeyService
}

func NewAuditService(repo *repository.AuditLogRepository, keySvc *KeyService) *AuditService {
	return &AuditService{repo: repo, keySvc: keySvc}
}

func (s *AuditService) Write(subjectType string, subjectID *uint64, action, resource, decision, ip, ua string, extraJSON []byte) error {
	l := &database.AuditLog{SubjectType: subjectType, SubjectID: subjectID, Action: action, Resource: resource, Decision: decision, IP: ip, UA: ua, At: time.Now()}
	if len(extraJSON) > 0 {
		l.Extra = extraJSON
	}
	return s.repo.Create(l)
}

// List 返回分页日志；需要在调用方做权限控制
func (s *AuditService) List(filter repository.AuditListFilter) (*repository.PagedAuditLogs, error) {
	return s.repo.List(filter)
}
