package barkhistory

import (
	"autobot/internal/database"
	"autobot/internal/models"
	"log"
	"time"

	"gorm.io/gorm"
)

const (
	// MaxBarkRecords 最大Bark记录条数
	MaxBarkRecords = 50000
)

// BarkHistoryManager Bark历史记录管理器
type BarkHistoryManager struct{}

// NewBarkHistoryManager 创建新的Bark历史记录管理器
func NewBarkHistoryManager() *BarkHistoryManager {
	return &BarkHistoryManager{}
}

// SaveBarkRecord 保存Bark发送记录
func (bhm *BarkHistoryManager) SaveBarkRecord(record *models.BarkRecord) error {
	db := database.GetDB()

	// 保存记录
	if err := db.Create(record).Error; err != nil {
		log.Printf("Failed to save bark record: %v", err)
		return err
	}

	// 清理旧记录
	bhm.cleanupOldRecords()

	return nil
}

// CheckDuplication 检查是否重复
func (bhm *BarkHistoryManager) CheckDuplication(record *models.BarkRecord, dedupConfig *models.BarkDeduplicationConfig) (bool, error) {
	if !dedupConfig.Enabled {
		return false, nil // 未启用去重
	}

	db := database.GetDB()

	switch dedupConfig.Mode {
	case "recentN":
		return bhm.checkRecentN(db, record, dedupConfig.RecentN)
	case "hash":
		return bhm.checkHash(db, record)
	case "timeWindow":
		return bhm.checkTimeWindow(db, record, dedupConfig.TimeWindow)
	default:
		log.Printf("Unknown deduplication mode: %s", dedupConfig.Mode)
		return false, nil
	}
}

// checkRecentN 检查最近N条记录中是否有重复
func (bhm *BarkHistoryManager) checkRecentN(db *gorm.DB, record *models.BarkRecord, recentN int) (bool, error) {
	var count int64

	// 只查询该任务最近N条成功发送的记录中是否有相同的content_hash
	// 注意：只统计 status = 'success' 的记录，跳过的记录不计入
	err := db.Model(&models.BarkRecord{}).
		Where("task_id = ? AND content_hash = ? AND status = ?", record.TaskID, record.ContentHash, "success").
		Order("created_at DESC").
		Limit(recentN).
		Count(&count).Error

	if err != nil {
		log.Printf("Failed to check recent N duplication: %v", err)
		return false, err
	}

	return count > 0, nil
}

// checkHash 检查全局是否有相同hash的记录
func (bhm *BarkHistoryManager) checkHash(db *gorm.DB, record *models.BarkRecord) (bool, error) {
	var count int64

	// 只查询全局成功发送的记录中是否有相同的content_hash
	// 注意：只统计 status = 'success' 的记录，跳过的记录不计入
	err := db.Model(&models.BarkRecord{}).
		Where("content_hash = ? AND status = ?", record.ContentHash, "success").
		Count(&count).Error

	if err != nil {
		log.Printf("Failed to check hash duplication: %v", err)
		return false, err
	}

	return count > 0, nil
}

// checkTimeWindow 检查时间窗口内是否有重复
func (bhm *BarkHistoryManager) checkTimeWindow(db *gorm.DB, record *models.BarkRecord, timeWindowMinutes int) (bool, error) {
	var count int64

	// 计算时间窗口的开始时间
	timeWindow := time.Duration(timeWindowMinutes) * time.Minute
	startTime := time.Now().Add(-timeWindow)

	// 只查询时间窗口内成功发送的记录中是否有相同的content_hash
	// 注意：只统计 status = 'success' 的记录，跳过的记录不计入
	err := db.Model(&models.BarkRecord{}).
		Where("task_id = ? AND content_hash = ? AND status = ? AND created_at >= ?",
			record.TaskID, record.ContentHash, "success", startTime).
		Count(&count).Error

	if err != nil {
		log.Printf("Failed to check time window duplication: %v", err)
		return false, err
	}

	return count > 0, nil
}

// cleanupOldRecords 清理旧记录，保持在最大记录数内
func (bhm *BarkHistoryManager) cleanupOldRecords() {
	db := database.GetDB()

	// 计算当前记录数
	var count int64
	if err := db.Model(&models.BarkRecord{}).Count(&count).Error; err != nil {
		log.Printf("Failed to count bark records: %v", err)
		return
	}

	// 如果超过限制，删除最旧的记录
	if count > MaxBarkRecords {
		excessCount := count - MaxBarkRecords

		// 获取需要删除的最旧记录的ID
		var recordIDs []uint
		if err := db.Model(&models.BarkRecord{}).
			Order("created_at ASC").
			Limit(int(excessCount)).
			Pluck("id", &recordIDs).Error; err != nil {
			log.Printf("Failed to get old bark record IDs: %v", err)
			return
		}

		// 删除这些记录
		if len(recordIDs) > 0 {
			result := db.Where("id IN ?", recordIDs).Delete(&models.BarkRecord{})
			if result.Error != nil {
				log.Printf("Failed to delete old bark records: %v", result.Error)
				return
			}
			log.Printf("Cleaned up %d old bark records (kept %d)", result.RowsAffected, MaxBarkRecords)
		}
	}
}

// GetBarkRecords 获取Bark记录（支持分页）
func (bhm *BarkHistoryManager) GetBarkRecords(taskID uint, page int, pageSize int) ([]models.BarkRecord, int64, error) {
	db := database.GetDB()

	var records []models.BarkRecord
	var total int64

	query := db.Model(&models.BarkRecord{})
	if taskID > 0 {
		query = query.Where("task_id = ?", taskID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// GetBarkStats 获取Bark发送统计信息
func (bhm *BarkHistoryManager) GetBarkStats() map[string]interface{} {
	db := database.GetDB()

	var totalRecords int64
	var successRecords int64
	var failedRecords int64

	// 总记录数
	db.Model(&models.BarkRecord{}).Count(&totalRecords)

	// 成功记录数
	db.Model(&models.BarkRecord{}).Where("status = ?", "success").Count(&successRecords)

	// 失败记录数
	db.Model(&models.BarkRecord{}).Where("status = ?", "failed").Count(&failedRecords)

	stats := map[string]interface{}{
		"total_records":   totalRecords,
		"success_records": successRecords,
		"failed_records":  failedRecords,
		"max_records":     MaxBarkRecords,
	}

	// 获取最新和最旧记录时间
	if totalRecords > 0 {
		var oldestRecord, newestRecord models.BarkRecord
		db.Order("created_at asc").First(&oldestRecord)
		db.Order("created_at desc").First(&newestRecord)

		stats["oldest_record"] = oldestRecord.CreatedAt
		stats["newest_record"] = newestRecord.CreatedAt
	}

	return stats
}

// DeleteAllBarkRecords 删除所有Bark记录
func (bhm *BarkHistoryManager) DeleteAllBarkRecords() (int64, error) {
	db := database.GetDB()

	// 先计算要删除的记录数
	var count int64
	db.Model(&models.BarkRecord{}).Count(&count)

	// 删除所有记录
	result := db.Where("1 = 1").Delete(&models.BarkRecord{})
	if result.Error != nil {
		log.Printf("Failed to delete all Bark records: %v", result.Error)
		return 0, result.Error
	}

	log.Printf("Deleted all Bark records: %d records", result.RowsAffected)
	return result.RowsAffected, nil
}
