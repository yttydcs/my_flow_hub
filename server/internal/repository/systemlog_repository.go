package repository

import (
	"myflowhub/pkg/database"

	"gorm.io/gorm"
)

type SystemLogRepository struct{ db *gorm.DB }

func NewSystemLogRepository(db *gorm.DB) *SystemLogRepository { return &SystemLogRepository{db: db} }

type SystemLogFilter struct {
	Level    string
	Source   string
	Keyword  string // search in Message
	StartAt  *int64
	EndAt    *int64
	Page     int
	PageSize int
}

type PagedSystemLogs struct {
	Items []database.SystemLog
	Total int64
	Page  int
	Size  int
}

func (r *SystemLogRepository) Create(l *database.SystemLog) error {
	return r.db.Create(l).Error
}

func (r *SystemLogRepository) List(filter SystemLogFilter) (*PagedSystemLogs, error) {
	q := r.db.Model(&database.SystemLog{})
	if filter.Level != "" {
		q = q.Where("level = ?", filter.Level)
	}
	if filter.Source != "" {
		q = q.Where("source = ?", filter.Source)
	}
	if filter.Keyword != "" {
		q = q.Where("message ILIKE ?", "%"+filter.Keyword+"%")
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
	var items []database.SystemLog
	if err := q.Order("at DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, err
	}
	return &PagedSystemLogs{Items: items, Total: total, Page: page, Size: size}, nil
}
