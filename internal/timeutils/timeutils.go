package timeutils

import (
	"autobot/internal/models"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// IsTimeExcluded 检查给定时间是否被排除
func IsTimeExcluded(checkTime time.Time, config *models.TimeExclusionConfig) (bool, string) {
	if config == nil || !config.Enabled || len(config.ExclusionRules) == 0 {
		return false, ""
	}

	for _, rule := range config.ExclusionRules {
		excluded, reason := isRuleMatched(checkTime, rule)
		if excluded {
			return true, reason
		}
	}

	return false, ""
}

// isRuleMatched 检查时间是否匹配排除规则
func isRuleMatched(checkTime time.Time, rule models.TimeExclusionRule) (bool, string) {
	switch rule.Type {
	case "daily":
		return isDailyRuleMatched(checkTime, rule)
	case "weekly":
		return isWeeklyRuleMatched(checkTime, rule)
	case "date_range":
		return isDateRangeRuleMatched(checkTime, rule)
	default:
		log.Printf("Unknown time exclusion rule type: %s", rule.Type)
		return false, ""
	}
}

// isDailyRuleMatched 检查每日排除规则
func isDailyRuleMatched(checkTime time.Time, rule models.TimeExclusionRule) (bool, string) {
	if rule.StartTime == "" || rule.EndTime == "" {
		return false, ""
	}

	startTime, err := time.Parse("15:04", rule.StartTime)
	if err != nil {
		log.Printf("Failed to parse start time '%s': %v", rule.StartTime, err)
		return false, ""
	}

	endTime, err := time.Parse("15:04", rule.EndTime)
	if err != nil {
		log.Printf("Failed to parse end time '%s': %v", rule.EndTime, err)
		return false, ""
	}

	// 获取当天的时间
	checkTimeOfDay := time.Date(1970, 1, 1, checkTime.Hour(), checkTime.Minute(), checkTime.Second(), 0, time.UTC)
	startTimeOfDay := time.Date(1970, 1, 1, startTime.Hour(), startTime.Minute(), 0, 0, time.UTC)
	endTimeOfDay := time.Date(1970, 1, 1, endTime.Hour(), endTime.Minute(), 0, 0, time.UTC)

	// 处理跨天的情况（如22:00-06:00）
	if endTimeOfDay.Before(startTimeOfDay) {
		// 跨天情况：22:00-06:00
		// 检查是否在晚上时间段（>= 22:00）或早上时间段（< 06:00）
		if checkTimeOfDay.After(startTimeOfDay) || checkTimeOfDay.Equal(startTimeOfDay) ||
			checkTimeOfDay.Before(endTimeOfDay) {
			return true, fmt.Sprintf("每日排除时间段: %s-%s", rule.StartTime, rule.EndTime)
		}
	} else {
		// 同天情况：09:00-17:00
		if (checkTimeOfDay.After(startTimeOfDay) || checkTimeOfDay.Equal(startTimeOfDay)) &&
			checkTimeOfDay.Before(endTimeOfDay) {
			return true, fmt.Sprintf("每日排除时间段: %s-%s", rule.StartTime, rule.EndTime)
		}
	}

	return false, ""
}

// isWeeklyRuleMatched 检查每周排除规则
func isWeeklyRuleMatched(checkTime time.Time, rule models.TimeExclusionRule) (bool, string) {
	if len(rule.Weekdays) == 0 {
		return false, ""
	}

	// 检查是否在指定的周几
	weekday := int(checkTime.Weekday())
	weekdayMatched := false
	for _, day := range rule.Weekdays {
		if day == weekday {
			weekdayMatched = true
			break
		}
	}

	if !weekdayMatched {
		return false, ""
	}

	// 如果指定了时间段，还需要检查时间
	if rule.StartTime != "" && rule.EndTime != "" {
		matched, reason := isDailyRuleMatched(checkTime, rule)
		if matched {
			return true, fmt.Sprintf("每周排除时间段: %s", reason)
		}
		return false, ""
	}

	// 如果没有指定时间段，整天都排除
	weekdayNames := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	var dayNames []string
	for _, day := range rule.Weekdays {
		if day >= 0 && day < 7 {
			dayNames = append(dayNames, weekdayNames[day])
		}
	}
	return true, fmt.Sprintf("每周排除: %v", dayNames)
}

// isDateRangeRuleMatched 检查日期范围排除规则
func isDateRangeRuleMatched(checkTime time.Time, rule models.TimeExclusionRule) (bool, string) {
	if rule.StartDate == "" || rule.EndDate == "" {
		return false, ""
	}

	startDate, err := time.Parse("2006-01-02", rule.StartDate)
	if err != nil {
		log.Printf("Failed to parse start date '%s': %v", rule.StartDate, err)
		return false, ""
	}

	endDate, err := time.Parse("2006-01-02", rule.EndDate)
	if err != nil {
		log.Printf("Failed to parse end date '%s': %v", rule.EndDate, err)
		return false, ""
	}

	// 只比较日期部分
	checkDate := time.Date(checkTime.Year(), checkTime.Month(), checkTime.Day(), 0, 0, 0, 0, time.UTC)
	startDateOnly := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	endDateOnly := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, time.UTC)

	if checkDate.After(startDateOnly) && checkDate.Before(endDateOnly) {
		// 如果指定了时间段，还需要检查时间
		if rule.StartTime != "" && rule.EndTime != "" {
			matched, _ := isDailyRuleMatched(checkTime, rule)
			if matched {
				return true, fmt.Sprintf("日期范围排除: %s-%s %s-%s",
					rule.StartDate, rule.EndDate, rule.StartTime, rule.EndTime)
			}
			return false, ""
		}

		// 如果没有指定时间段，整天都排除
		return true, fmt.Sprintf("日期范围排除: %s-%s", rule.StartDate, rule.EndDate)
	}

	return false, ""
}

// GetNextAllowedTime 获取下一个允许执行的时间
func GetNextAllowedTime(schedule cron.Schedule, config *models.TimeExclusionConfig, fromTime time.Time) time.Time {
	if config == nil || !config.Enabled {
		return schedule.Next(fromTime)
	}

	// 最多向前查找100次，避免无限循环
	nextTime := schedule.Next(fromTime)
	for i := 0; i < 100; i++ {
		excluded, _ := IsTimeExcluded(nextTime, config)
		if !excluded {
			return nextTime
		}
		// 如果被排除，查找下一个时间
		nextTime = schedule.Next(nextTime)
	}

	// 如果100次都被排除，返回默认的下次执行时间
	return schedule.Next(fromTime)
}
