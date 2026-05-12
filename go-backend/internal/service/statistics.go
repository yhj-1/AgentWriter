package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/model"
	"github.com/yupi/ai-passage-creator/internal/store"
	"gorm.io/gorm"
)

const (
	StatisticsCacheKey = "statistics:overview"
	StatisticsCacheTTL = time.Hour
)

// StatisticsService 统计服务
type StatisticsService struct {
	db           *gorm.DB
	userStore    *store.UserStore
	articleStore *store.ArticleStore
	redisClient  *redis.Client
}

// NewStatisticsService 创建统计服务
func NewStatisticsService(db *gorm.DB, userStore *store.UserStore, articleStore *store.ArticleStore, redisClient *redis.Client) *StatisticsService {
	return &StatisticsService{
		db:           db,
		userStore:    userStore,
		articleStore: articleStore,
		redisClient:  redisClient,
	}
}

// GetStatistics 获取系统统计数据
func (s *StatisticsService) GetStatistics() (*model.StatisticsVO, error) {
	ctx := context.Background()

	// 1. 尝试从 Redis 获取缓存
	if s.redisClient != nil {
		cachedData, err := s.redisClient.Get(ctx, StatisticsCacheKey).Result()
		if err == nil && cachedData != "" {
			var stats model.StatisticsVO
			if err := json.Unmarshal([]byte(cachedData), &stats); err == nil {
				log.Println("从缓存获取统计数据")
				return &stats, nil
			}
		}
	}

	// 2. 缓存不存在，重新计算
	now := time.Now()
	todayStart := common.GetTodayStart()
	weekStart := common.GetWeekStart()
	monthStart := common.GetMonthStart()

	// 今日创作数量
	todayCount := s.countArticlesByDateRange(todayStart, now)

	// 本周创作数量
	weekCount := s.countArticlesByDateRange(weekStart, now)

	// 本月创作数量
	monthCount := s.countArticlesByDateRange(monthStart, now)

	// 总创作数量
	totalCount := s.countTotalArticles()

	// 成功率统计
	successRate := s.calculateSuccessRate()

	// 平均耗时统计
	avgDurationMs := s.calculateAvgDuration()

	// 活跃用户统计（本周有创作的用户）
	activeUserCount := s.countActiveUsers(weekStart)

	// 总用户数
	totalUserCount := s.countTotalUsers()

	// VIP 用户数
	vipUserCount := s.countVipUsers()

	// 配额使用情况
	quotaUsed := s.calculateQuotaUsed()

	statistics := &model.StatisticsVO{
		TodayCount:      todayCount,
		WeekCount:       weekCount,
		MonthCount:      monthCount,
		TotalCount:      totalCount,
		SuccessRate:     successRate,
		AvgDurationMs:   avgDurationMs,
		ActiveUserCount: activeUserCount,
		TotalUserCount:  totalUserCount,
		VipUserCount:    vipUserCount,
		QuotaUsed:       quotaUsed,
	}

	// 3. 存入 Redis 缓存（1小时过期）
	if s.redisClient != nil {
		data, err := json.Marshal(statistics)
		if err == nil {
			if err := s.redisClient.Set(ctx, StatisticsCacheKey, data, StatisticsCacheTTL).Err(); err == nil {
				log.Printf("统计数据已缓存，过期时间: %v", StatisticsCacheTTL)
			}
		}
	}

	return statistics, nil
}

// countArticlesByDateRange 统计指定时间范围内的文章数量
func (s *StatisticsService) countArticlesByDateRange(start, end time.Time) int64 {
	var count int64
	err := s.db.Model(&model.Article{}).
		Where("createTime >= ? AND createTime <= ?", start, end).
		Where("isDelete = ?", 0).
		Count(&count).Error

	if err != nil {
		log.Printf("统计时间范围内文章数量失败: %v", err)
		return 0
	}
	return count
}

// countTotalArticles 统计总文章数量
func (s *StatisticsService) countTotalArticles() int64 {
	var count int64
	err := s.db.Model(&model.Article{}).
		Where("isDelete = ?", 0).
		Count(&count).Error

	if err != nil {
		log.Printf("统计总文章数量失败: %v", err)
		return 0
	}
	return count
}

// calculateSuccessRate 计算成功率
func (s *StatisticsService) calculateSuccessRate() float64 {
	totalCount := s.countTotalArticles()
	if totalCount == 0 {
		return 0.0
	}

	var successCount int64
	err := s.db.Model(&model.Article{}).
		Where("status = ?", "COMPLETED").
		Where("isDelete = ?", 0).
		Count(&successCount).Error

	if err != nil {
		log.Printf("统计成功文章数量失败: %v", err)
		return 0.0
	}

	return (float64(successCount) / float64(totalCount)) * 100
}

// calculateAvgDuration 计算平均耗时（从创建到完成的平均时间）
func (s *StatisticsService) calculateAvgDuration() int {
	var articles []model.Article
	err := s.db.Model(&model.Article{}).
		Select("createTime", "completedTime").
		Where("status = ?", "COMPLETED").
		Where("completedTime IS NOT NULL").
		Where("isDelete = ?", 0).
		Find(&articles).Error

	if err != nil || len(articles) == 0 {
		if err != nil {
			log.Printf("查询已完成文章失败: %v", err)
		}
		return 0
	}

	// 计算每篇文章的耗时并求平均值
	var totalDuration int64
	validCount := 0

	for _, article := range articles {
		if article.CompletedTime != nil {
			duration := article.CompletedTime.Sub(article.CreateTime).Milliseconds()
			totalDuration += duration
			validCount++
		}
	}

	if validCount == 0 {
		return 0
	}

	return int(totalDuration / int64(validCount))
}

// countActiveUsers 统计活跃用户数（本周有创作的用户）
func (s *StatisticsService) countActiveUsers(weekStart time.Time) int64 {
	var userIDs []int64
	err := s.db.Model(&model.Article{}).
		Select("DISTINCT userId").
		Where("createTime >= ?", weekStart).
		Where("isDelete = ?", 0).
		Pluck("userId", &userIDs).Error

	if err != nil {
		log.Printf("统计活跃用户失败: %v", err)
		return 0
	}

	return int64(len(userIDs))
}

// countTotalUsers 统计总用户数
func (s *StatisticsService) countTotalUsers() int64 {
	var count int64
	err := s.db.Model(&model.User{}).
		Where("isDelete = ?", 0).
		Count(&count).Error

	if err != nil {
		log.Printf("统计总用户数失败: %v", err)
		return 0
	}
	return count
}

// countVipUsers 统计 VIP 用户数
func (s *StatisticsService) countVipUsers() int64 {
	var count int64
	err := s.db.Model(&model.User{}).
		Where("userRole = ?", common.VIPRole).
		Where("isDelete = ?", 0).
		Count(&count).Error

	if err != nil {
		log.Printf("统计 VIP 用户数失败: %v", err)
		return 0
	}
	return count
}

// calculateQuotaUsed 计算配额使用量
func (s *StatisticsService) calculateQuotaUsed() int64 {
	// 配额使用量 = (普通用户数 * 初始配额) - 当前剩余配额总和
	var users []model.User
	err := s.db.Model(&model.User{}).
		Select("quota").
		Where("userRole = ?", common.UserRole).
		Where("isDelete = ?", 0).
		Find(&users).Error

	if err != nil {
		log.Printf("计算配额使用量失败: %v", err)
		return 0
	}

	normalUserCount := int64(len(users))

	// 统计剩余配额总和
	var remainingQuota int64
	for _, user := range users {
		remainingQuota += int64(user.Quota)
	}

	totalQuota := normalUserCount * common.DefaultQuota
	used := totalQuota - remainingQuota

	if used < 0 {
		used = 0
	}

	return used
}
