package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/service"
)

// StatisticsHandler 统计分析处理器
type StatisticsHandler struct {
	statisticsService *service.StatisticsService
}

// NewStatisticsHandler 创建统计分析处理器
func NewStatisticsHandler(statisticsService *service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{
		statisticsService: statisticsService,
	}
}

// GetStatistics 获取系统统计数据（仅管理员）
func (h *StatisticsHandler) GetStatistics(c *gin.Context) {
	statistics, err := h.statisticsService.GetStatistics()
	if err != nil {
		c.JSON(http.StatusOK, common.Error(common.ErrSystem.WithMessage("获取统计数据失败: "+err.Error())))
		return
	}

	c.JSON(http.StatusOK, common.Success(statistics))
}
