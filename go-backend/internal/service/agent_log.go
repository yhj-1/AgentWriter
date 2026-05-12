package service

import (
	"log"

	"github.com/yupi/ai-passage-creator/internal/model"
	"github.com/yupi/ai-passage-creator/internal/store"
)

// AgentLogService 智能体日志服务
type AgentLogService struct {
	store *store.AgentLogStore
}

// NewAgentLogService 创建智能体日志服务
func NewAgentLogService(store *store.AgentLogStore) *AgentLogService {
	return &AgentLogService{
		store: store,
	}
}

// SaveLogAsync 异步保存日志
func (s *AgentLogService) SaveLogAsync(agentLog *model.AgentLog) {
	// 使用 goroutine 异步保存，避免阻塞主流程
	go func() {
		if err := s.store.Create(agentLog); err != nil {
			log.Printf("保存智能体日志失败, taskId=%s, agentName=%s, error=%v",
				agentLog.TaskID, agentLog.AgentName, err)
		} else {
			log.Printf("智能体日志已保存, taskId=%s, agentName=%s, status=%s, durationMs=%v",
				agentLog.TaskID, agentLog.AgentName, agentLog.Status, agentLog.DurationMs)
		}
	}()
}

// GetLogsByTaskID 获取任务的所有日志
func (s *AgentLogService) GetLogsByTaskID(taskID string) ([]*model.AgentLog, error) {
	return s.store.GetByTaskID(taskID)
}

// GetExecutionStats 获取任务执行统计
func (s *AgentLogService) GetExecutionStats(taskID string) (*model.AgentExecutionStats, error) {
	logs, err := s.GetLogsByTaskID(taskID)
	if err != nil {
		return nil, err
	}

	if len(logs) == 0 {
		return &model.AgentExecutionStats{
			TaskID:          taskID,
			AgentCount:      0,
			TotalDurationMs: 0,
			OverallStatus:   "NOT_FOUND",
			AgentDurations:  make(map[string]int),
			Logs:            []*model.AgentLog{},
		}, nil
	}

	// 计算统计数据
	totalDuration := 0
	agentDurations := make(map[string]int)
	overallStatus := "SUCCESS"

	for _, logEntry := range logs {
		// 累加总耗时
		if logEntry.DurationMs != nil {
			totalDuration += *logEntry.DurationMs
			agentDurations[logEntry.AgentName] = *logEntry.DurationMs
		}

		// 判断总体状态
		if logEntry.Status == "FAILED" {
			overallStatus = "FAILED"
		} else if logEntry.Status == "RUNNING" && overallStatus != "FAILED" {
			overallStatus = "RUNNING"
		}
	}

	return &model.AgentExecutionStats{
		TaskID:          taskID,
		TotalDurationMs: totalDuration,
		AgentCount:      len(logs),
		AgentDurations:  agentDurations,
		OverallStatus:   overallStatus,
		Logs:            logs,
	}, nil
}
