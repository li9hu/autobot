package logmanager

import (
	"autobot/internal/database"
	"autobot/internal/models"
	"log"
)

const (
	// MaxLogsPerTask 每个任务最大日志条数
	MaxLogsPerTask = 5000
	// MaxTotalLogs 全局最大日志条数
	MaxTotalLogs = 50000
)

// LogManager 日志管理器
type LogManager struct{}

// NewLogManager 创建新的日志管理器
func NewLogManager() *LogManager {
	return &LogManager{}
}

// CleanupLogsAfterExecution 在任务执行后清理日志
// 1. 首先检查单个任务是否超过5000条，如果超过则删除最旧的记录
// 2. 然后检查全局是否超过50000条，如果超过则删除全局最旧的记录
func (lm *LogManager) CleanupLogsAfterExecution(taskID uint) {
	// 1. 清理单个任务的旧日志
	lm.cleanupTaskLogs(taskID)

	// 2. 清理全局旧日志
	lm.cleanupGlobalLogs()
}

// cleanupTaskLogs 清理单个任务的旧日志
func (lm *LogManager) cleanupTaskLogs(taskID uint) {
	db := database.GetDB()

	// 计算当前任务的日志数量
	var count int64
	if err := db.Model(&models.TaskLog{}).Where("task_id = ?", taskID).Count(&count).Error; err != nil {
		log.Printf("Failed to count logs for task %d: %v", taskID, err)
		return
	}

	// 如果超过限制，删除最旧的日志
	if count > MaxLogsPerTask {
		excessCount := count - MaxLogsPerTask

		// 获取需要删除的最旧日志的ID
		var logIDs []uint
		if err := db.Model(&models.TaskLog{}).
			Where("task_id = ?", taskID).
			Order("created_at ASC").
			Limit(int(excessCount)).
			Pluck("id", &logIDs).Error; err != nil {
			log.Printf("Failed to get old log IDs for task %d: %v", taskID, err)
			return
		}

		// 删除这些日志
		if len(logIDs) > 0 {
			result := db.Where("id IN ?", logIDs).Delete(&models.TaskLog{})
			if result.Error != nil {
				log.Printf("Failed to delete old logs for task %d: %v", taskID, result.Error)
				return
			}
			log.Printf("Cleaned up %d old logs for task %d (kept %d)", result.RowsAffected, taskID, MaxLogsPerTask)
		}
	}
}

// cleanupGlobalLogs 清理全局旧日志
func (lm *LogManager) cleanupGlobalLogs() {
	db := database.GetDB()

	// 计算全局日志数量
	var count int64
	if err := db.Model(&models.TaskLog{}).Count(&count).Error; err != nil {
		log.Printf("Failed to count global logs: %v", err)
		return
	}

	// 如果超过全局限制，删除最旧的日志
	if count > MaxTotalLogs {
		excessCount := count - MaxTotalLogs

		// 获取需要删除的最旧日志的ID
		var logIDs []uint
		if err := db.Model(&models.TaskLog{}).
			Order("created_at ASC").
			Limit(int(excessCount)).
			Pluck("id", &logIDs).Error; err != nil {
			log.Printf("Failed to get old log IDs for global cleanup: %v", err)
			return
		}

		// 删除这些日志
		if len(logIDs) > 0 {
			result := db.Where("id IN ?", logIDs).Delete(&models.TaskLog{})
			if result.Error != nil {
				log.Printf("Failed to delete old logs globally: %v", result.Error)
				return
			}
			log.Printf("Cleaned up %d old logs globally (kept %d)", result.RowsAffected, MaxTotalLogs)
		}
	}
}

// GetLogStats 获取日志统计信息
func (lm *LogManager) GetLogStats() map[string]interface{} {
	db := database.GetDB()

	// 获取全局日志统计
	var totalLogs int64
	db.Model(&models.TaskLog{}).Count(&totalLogs)

	stats := map[string]interface{}{
		"total_logs":        totalLogs,
		"max_total_logs":    MaxTotalLogs,
		"max_logs_per_task": MaxLogsPerTask,
	}

	// 获取最新和最旧日志时间
	if totalLogs > 0 {
		var oldestLog, newestLog models.TaskLog
		db.Order("created_at asc").First(&oldestLog)
		db.Order("created_at desc").First(&newestLog)

		stats["oldest_log"] = oldestLog.CreatedAt
		stats["newest_log"] = newestLog.CreatedAt
	}

	return stats
}
