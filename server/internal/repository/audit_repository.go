package repository

import (
	"myflowhub/pkg/database"

	"gorm.io/gorm"
)

type AuditLogRepository struct{ db *gorm.DB }

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository { return &AuditLogRepository{db: db} }

type AuditListFilter struct {
	Keyword     string
	SubjectType string
	Decision    string
	Action      string
	StartAt     *int64 // unix seconds
	EndAt       *int64 // unix seconds
	Page        int
	PageSize    int
}

type PagedAuditLogs struct {
	Items []database.AuditLog
	Total int64
	Page  int
	Size  int
}

func (r *AuditLogRepository) Create(l *database.AuditLog) error {
	return r.db.Create(l).Error
}

func (r *AuditLogRepository) List(filter AuditListFilter) (*PagedAuditLogs, error) {
	q := r.db.Model(&database.AuditLog{})
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		q = q.Where("action ILIKE ? OR resource ILIKE ? OR ip ILIKE ? OR ua ILIKE ? OR decision ILIKE ?", like, like, like, like, like)
	}
	if filter.SubjectType != "" {
		q = q.Where("subject_type = ?", filter.SubjectType)
	}
	if filter.Decision != "" {
		q = q.Where("decision = ?", filter.Decision)
	}
	if filter.Action != "" {
		q = q.Where("action = ?", filter.Action)
	}
	if filter.StartAt != nil {
		q = q.Where("at >= to_timestamp(?)", *filter.StartAt)
	}
	if filter.EndAt != nil {
		q = q.Where("at <= to_timestamp(?)", *filter.EndAt)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	page := filter.Page
	size := filter.PageSize
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 200 {
		size = 20
	}
	items := []database.AuditLog{}
	if err := q.Order("at DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, err
	}
	return &PagedAuditLogs{Items: items, Total: total, Page: page, Size: size}, nil
}
