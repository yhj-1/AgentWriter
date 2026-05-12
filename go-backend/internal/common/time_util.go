package common

import "time"

// GetTodayStart 获取今天开始时间（00:00:00）
func GetTodayStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// GetWeekStart 获取本周开始时间（周一 00:00:00）
func GetWeekStart() time.Time {
	now := time.Now()
	weekday := int(now.Weekday())

	// 周日调整为7
	if weekday == 0 {
		weekday = 7
	}

	// 计算周一的日期
	daysToMonday := weekday - 1
	monday := now.AddDate(0, 0, -daysToMonday)

	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
}

// GetMonthStart 获取本月开始时间（1号 00:00:00）
func GetMonthStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}
