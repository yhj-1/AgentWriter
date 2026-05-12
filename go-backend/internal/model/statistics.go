package model

// StatisticsVO 统计数据 VO
type StatisticsVO struct {
	TodayCount      int64   `json:"todayCount"`      // 今日创作数量
	WeekCount       int64   `json:"weekCount"`       // 本周创作数量
	MonthCount      int64   `json:"monthCount"`      // 本月创作数量
	TotalCount      int64   `json:"totalCount"`      // 总创作数量
	SuccessRate     float64 `json:"successRate"`     // 成功率（百分比）
	AvgDurationMs   int     `json:"avgDurationMs"`   // 平均耗时（毫秒）
	ActiveUserCount int64   `json:"activeUserCount"` // 活跃用户数（本周）
	TotalUserCount  int64   `json:"totalUserCount"`  // 总用户数
	VipUserCount    int64   `json:"vipUserCount"`    // VIP 用户数
	QuotaUsed       int64   `json:"quotaUsed"`       // 配额总使用量
}

// AgentExecutionStats 智能体执行统计
type AgentExecutionStats struct {
	TaskID          string         `json:"taskId"`          // 任务ID
	TotalDurationMs int            `json:"totalDurationMs"` // 总耗时（毫秒）
	AgentCount      int            `json:"agentCount"`      // 智能体数量
	AgentDurations  map[string]int `json:"agentDurations"`  // 各智能体耗时
	OverallStatus   string         `json:"overallStatus"`   // 总体状态：SUCCESS/FAILED/RUNNING
	Logs            []*AgentLog    `json:"logs"`            // 详细日志列表
}
