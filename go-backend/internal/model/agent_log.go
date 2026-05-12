package model

import "time"

// AgentLog 智能体执行日志
type AgentLog struct {
	ID           int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TaskID       string     `gorm:"column:taskId;type:varchar(64);not null;index" json:"taskId"`
	AgentName    string     `gorm:"column:agentName;type:varchar(64);not null" json:"agentName"`
	StartTime    time.Time  `gorm:"column:startTime;not null" json:"startTime"`
	EndTime      *time.Time `gorm:"column:endTime" json:"endTime"`
	DurationMs   *int       `gorm:"column:durationMs" json:"durationMs"`
	Status       string     `gorm:"column:status;type:varchar(20);not null" json:"status"` // RUNNING/SUCCESS/FAILED
	ErrorMessage *string    `gorm:"column:errorMessage;type:text" json:"errorMessage"`
	Prompt       *string    `gorm:"column:prompt;type:text" json:"prompt"`
	InputData    *string    `gorm:"column:inputData;type:text" json:"inputData"`
	OutputData   *string    `gorm:"column:outputData;type:text" json:"outputData"`
	CreateTime   time.Time  `gorm:"column:createTime;autoCreateTime" json:"createTime"`
	UpdateTime   time.Time  `gorm:"column:updateTime;autoUpdateTime" json:"updateTime"`
	IsDelete     int        `gorm:"column:isDelete;default:0" json:"isDelete"`
}

// TableName 指定表名
func (AgentLog) TableName() string {
	return "agent_log"
}
