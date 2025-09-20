package scheduler

import (
	"autobot/internal/database"
	"autobot/internal/executor"
	"autobot/internal/models"
	"autobot/internal/timeutils"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler 任务调度器
type Scheduler struct {
	cron    *cron.Cron
	entries map[uint]cron.EntryID // 任务ID -> cron 条目ID 的映射
}

// NewScheduler 创建新的调度器
func NewScheduler() *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()),
		entries: make(map[uint]cron.EntryID),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.cron.Start()

	// 加载所有活跃的任务
	s.loadActiveTasks()

	// 立即执行一次清理，移除可能存在的已删除任务
	s.cleanupDeletedTasks()

	// 添加定期清理任务，每小时检查一次已删除的任务
	s.cron.AddFunc("0 0 * * * *", func() {
		s.cleanupDeletedTasks()
	})

	// log.Println("Task scheduler started") // 减少启动日志
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("Task scheduler stopped")
}

// loadActiveTasks 加载所有活跃的任务
func (s *Scheduler) loadActiveTasks() {
	var tasks []models.Task
	// 只加载状态为active且未被软删除的任务
	if err := database.GetDB().Where("status = ?", "active").Find(&tasks).Error; err != nil {
		log.Printf("Failed to load active tasks: %v", err)
		return
	}

	for _, task := range tasks {
		if err := s.AddTask(&task); err != nil {
			log.Printf("Failed to add task %d to scheduler: %v", task.ID, err)
		}
	}

	// log.Printf("Loaded %d active tasks", len(tasks)) // 减少加载日志
}

// AddTask 添加任务到调度器
func (s *Scheduler) AddTask(task *models.Task) error {
	// 如果任务已经在调度器中，先移除
	if entryID, exists := s.entries[task.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, task.ID)
	}

	// 只调度活跃的任务
	if task.Status != "active" {
		return nil
	}

	// 添加新的任务到调度器
	entryID, err := s.cron.AddFunc(task.CronExpr, func() {
		// log.Printf("Executing scheduled task: %s (ID: %d)", task.Name, task.ID) // 减少执行日志

		// 更新任务的最后执行时间
		now := time.Now()

		// 获取最新的任务配置（包括时间排除配置）
		var latestTask models.Task
		if err := database.GetDB().First(&latestTask, task.ID).Error; err != nil {
			log.Printf("Failed to get latest task config for task %d: %v", task.ID, err)
			// 如果任务不存在（可能已被删除），从调度器中移除该任务
			s.RemoveTask(task.ID)
			log.Printf("Removed deleted task %d from scheduler", task.ID)
			return
		}

		// 检查时间排除
		timeExclusionConfig, err := latestTask.GetTimeExclusionConfig()
		if err != nil {
			log.Printf("Failed to parse time exclusion config for task %d: %v", task.ID, err)
		} else {
			excluded, reason := timeutils.IsTimeExcluded(now, timeExclusionConfig)
			if excluded {
				log.Printf("Skipping task execution due to time exclusion: %s (ID: %d) - %s", latestTask.Name, task.ID, reason)

				// 计算下次允许执行的时间
				parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
				if schedule, parseErr := parser.Parse(latestTask.CronExpr); parseErr == nil {
					nextAllowedTime := timeutils.GetNextAllowedTime(schedule, timeExclusionConfig, now)
					database.GetDB().Model(&models.Task{}).Where("id = ?", task.ID).Update("next_run", nextAllowedTime)
				}
				return
			}
		}

		// 计算下次执行时间（考虑时间排除）
		var nextRun *time.Time
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(latestTask.CronExpr)
		if err == nil {
			if timeExclusionConfig != nil && timeExclusionConfig.Enabled {
				next := timeutils.GetNextAllowedTime(schedule, timeExclusionConfig, now)
				nextRun = &next
			} else {
				next := schedule.Next(now)
				nextRun = &next
			}
		} else {
			log.Printf("Failed to parse cron expression '%s' during execution for task %d: %v", latestTask.CronExpr, task.ID, err)
		}

		// 更新执行时间
		database.GetDB().Model(&models.Task{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
			"last_run": now,
			"next_run": nextRun,
		})

		// 执行任务
		executor.ExecuteTask(&latestTask)
	})

	if err != nil {
		return err
	}

	// 保存 entryID
	s.entries[task.ID] = entryID

	// 计算下次执行时间（考虑时间排除）
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(task.CronExpr)
	if err == nil {
		var nextRun time.Time

		// 获取时间排除配置
		timeExclusionConfig, configErr := task.GetTimeExclusionConfig()
		if configErr == nil && timeExclusionConfig.Enabled {
			nextRun = timeutils.GetNextAllowedTime(schedule, timeExclusionConfig, time.Now())
		} else {
			nextRun = schedule.Next(time.Now())
		}

		// 只更新next_run字段，避免覆盖其他配置
		if updateErr := database.GetDB().Model(&models.Task{}).Where("id = ?", task.ID).Update("next_run", nextRun).Error; updateErr != nil {
			log.Printf("Failed to update next_run for task %d: %v", task.ID, updateErr)
		}
	} else {
		log.Printf("Failed to parse cron expression '%s' for task %d: %v", task.CronExpr, task.ID, err)
	}

	// log.Printf("Task added to scheduler: %s (ID: %d)", task.Name, task.ID) // 减少添加日志
	return nil
}

// RemoveTask 从调度器中移除任务
func (s *Scheduler) RemoveTask(taskID uint) {
	if entryID, exists := s.entries[taskID]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, taskID)
		// log.Printf("Task removed from scheduler: ID %d", taskID) // 减少移除日志
	}
}

// UpdateTask 更新调度器中的任务
func (s *Scheduler) UpdateTask(task *models.Task) error {
	// 先移除旧的任务
	s.RemoveTask(task.ID)

	// 重新添加任务
	return s.AddTask(task)
}

// GetScheduledTasks 获取当前调度的任务数量
func (s *Scheduler) GetScheduledTasks() int {
	return len(s.entries)
}

// ListEntries 列出所有调度条目
func (s *Scheduler) ListEntries() []cron.Entry {
	return s.cron.Entries()
}

// cleanupDeletedTasks 清理已删除的任务
func (s *Scheduler) cleanupDeletedTasks() {
	// 获取当前调度器中的所有任务ID
	var scheduledTaskIDs []uint
	for taskID := range s.entries {
		scheduledTaskIDs = append(scheduledTaskIDs, taskID)
	}

	if len(scheduledTaskIDs) == 0 {
		return
	}

	// 查询数据库中仍然存在且活跃的任务
	var activeTasks []models.Task
	if err := database.GetDB().Where("id IN ? AND status = ?", scheduledTaskIDs, "active").Find(&activeTasks).Error; err != nil {
		log.Printf("Failed to query active tasks during cleanup: %v", err)
		return
	}

	// 创建活跃任务ID的映射
	activeTaskMap := make(map[uint]bool)
	for _, task := range activeTasks {
		activeTaskMap[task.ID] = true
	}

	// 移除不再活跃或已删除的任务
	var removedCount int
	for _, taskID := range scheduledTaskIDs {
		if !activeTaskMap[taskID] {
			s.RemoveTask(taskID)
			removedCount++
			log.Printf("Cleaned up deleted/inactive task %d from scheduler", taskID)
		}
	}

	if removedCount > 0 {
		log.Printf("Scheduler cleanup completed: removed %d deleted/inactive tasks", removedCount)
	}
}
