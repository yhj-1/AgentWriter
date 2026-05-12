package store

import (
	"time"

	"github.com/yupi/ai-passage-creator/internal/model"
	"gorm.io/gorm"
)

// AgentLogStore 智能体日志数据访问
type AgentLogStore struct {
	db *gorm.DB
}

// NewAgentLogStore 创建智能体日志 Store
func NewAgentLogStore(db *gorm.DB) *AgentLogStore {
	return &AgentLogStore{db: db}
}

// Create 创建日志记录
func (s *AgentLogStore) Create(log *model.AgentLog) error {
	return s.db.Create(log).Error
}

// GetByTaskID 根据任务ID查询所有日志
func (s *AgentLogStore) GetByTaskID(taskID string) ([]*model.AgentLog, error) {
	var logs []*model.AgentLog
	err := s.db.Where("taskId = ?", taskID).
		Where("isDelete = ?", 0).
		Order("createTime ASC").
		Find(&logs).Error
	return logs, err
}

// CountByDateRange 统计时间范围内的日志数量
func (s *AgentLogStore) CountByDateRange(start, end time.Time) (int64, error) {
	var count int64
	err := s.db.Model(&model.AgentLog{}).
		Where("createTime >= ? AND createTime <= ?", start, end).
		Where("isDelete = ?", 0).
		Count(&count).Error
	return count, err
}
