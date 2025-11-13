package handlers

import (
	"autobot/internal/barkhistory"
	"autobot/internal/database"
	"autobot/internal/executor"
	"autobot/internal/logmanager"
	"autobot/internal/models"
	"autobot/internal/scheduler"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// 全局调度器和日志管理器实例
var globalScheduler *scheduler.Scheduler
var globalLogManager *logmanager.LogManager

// SetScheduler 设置全局调度器
func SetScheduler(s *scheduler.Scheduler) {
	globalScheduler = s
}

// SetLogManager 设置全局日志管理器
func SetLogManager(lm *logmanager.LogManager) {
	globalLogManager = lm
}

// 前端页面处理器

// IndexHandler 首页
func IndexHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "计划任务平台",
	})
}

// NewTaskHandler 新建任务页面
func NewTaskHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "task_form.html", gin.H{
		"title": "创建新任务",
		"task":  nil,
	})
}

// EditTaskHandler 编辑任务页面
func EditTaskHandler(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "无效的任务ID",
		})
		return
	}

	var task models.Task
	if err := database.GetDB().First(&task, taskID).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "任务不存在",
		})
		return
	}

	c.HTML(http.StatusOK, "task_form.html", gin.H{
		"title": "编辑任务",
		"task":  task,
	})
}

// LogsHandler 日志页面
func LogsHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "logs.html", gin.H{
		"title": "执行日志",
	})
}

// TaskDetailHandler 任务详情页面
func TaskDetailHandler(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "无效的任务ID",
		})
		return
	}

	var task models.Task
	if err := database.GetDB().First(&task, taskID).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "任务不存在",
		})
		return
	}

	c.HTML(http.StatusOK, "task_detail.html", gin.H{
		"title": "任务详情 - " + task.Name,
		"task":  task,
	})
}

// API 处理器

// CreateTask 创建任务
func CreateTask(c *gin.Context) {
	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 cron 表达式（支持6位格式：秒 分 时 日 月 周）
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(req.CronExpr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 cron 表达式: " + err.Error()})
		return
	}

	// 设置任务状态，如果未指定则默认为 inactive
	status := req.Status
	if status == "" {
		status = "inactive"
	}

	task := models.Task{
		Name:                req.Name,
		Description:         req.Description,
		Script:              req.Script,
		CronExpr:            req.CronExpr,
		Status:              status,
		BarkConfig:          req.BarkConfig,
		TimeExclusionConfig: req.TimeExclusionConfig,
	}

	// 使用重试机制创建任务
	err = database.WithRetry(func(db *gorm.DB) error {
		return db.Create(&task).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	// 如果任务是激活状态，添加到调度器
	if task.Status == "active" && globalScheduler != nil {
		if err := globalScheduler.AddTask(&task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "添加任务到调度器失败"})
			return
		}
	}

	c.JSON(http.StatusCreated, task)
}

// GetTasks 获取任务列表
func GetTasks(c *gin.Context) {
	var tasks []models.Task
	var total int64

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// 状态筛选
	status := c.Query("status")

	// 获取总数 - 使用重试机制
	err := database.WithRetry(func(db *gorm.DB) error {
		query := db.Model(&models.Task{})
		if status != "" {
			query = query.Where("status = ?", status)
		}
		return query.Count(&total).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务总数失败"})
		return
	}

	// 获取任务列表 - 使用重试机制
	err = database.WithRetry(func(db *gorm.DB) error {
		query := db
		if status != "" {
			query = query.Where("status = ?", status)
		}
		return query.Offset(offset).Limit(limit).Order("created_at desc").Find(&tasks).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetTask 获取单个任务
func GetTask(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	var task models.Task
	if err := database.GetDB().First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// UpdateTask 更新任务
func UpdateTask(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	var req models.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var task models.Task
	if err := database.GetDB().First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	// 验证 cron 表达式（如果有更新）
	if req.CronExpr != "" {
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		_, err := parser.Parse(req.CronExpr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 cron 表达式: " + err.Error()})
			return
		}
		task.CronExpr = req.CronExpr
	}

	// 更新字段
	if req.Name != "" {
		task.Name = req.Name
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.Script != "" {
		task.Script = req.Script
	}
	if req.Status != "" {
		task.Status = req.Status
	}
	if req.BarkConfig != "" {
		task.BarkConfig = req.BarkConfig
	}
	if req.TimeExclusionConfig != "" {
		task.TimeExclusionConfig = req.TimeExclusionConfig
	}

	// 使用重试机制保存任务
	err = database.WithRetry(func(db *gorm.DB) error {
		return db.Save(&task).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新任务失败"})
		return
	}

	// 更新调度器中的任务
	if globalScheduler != nil {
		if err := globalScheduler.UpdateTask(&task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新调度器失败"})
			return
		}
	}

	c.JSON(http.StatusOK, task)
}

// DeleteTask 删除任务
func DeleteTask(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	// 检查任务是否存在
	var task models.Task
	if err := database.GetDB().First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	// 从调度器中移除任务
	if globalScheduler != nil {
		globalScheduler.RemoveTask(uint(taskID))
	}

	// 统计要删除的日志数量
	var logCount int64
	database.GetDB().Model(&models.TaskLog{}).Where("task_id = ?", taskID).Count(&logCount)

	// 开启事务，确保删除操作的原子性 - 使用新session避免污染全局DB状态
	tx := database.GetDB().Session(&gorm.Session{}).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // 重新抛出panic
		}
	}()

	// 先删除相关的日志
	if err := tx.Where("task_id = ?", taskID).Delete(&models.TaskLog{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除任务日志失败"})
		return
	}

	// 再删除任务本身
	if err := tx.Delete(&models.Task{}, taskID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除任务失败"})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除操作提交失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "任务删除成功",
		"task_name":    task.Name,
		"deleted_logs": logCount,
	})
}

// GetTaskLogs 获取任务日志
func GetTaskLogs(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	var logs []models.TaskLog
	var total int64

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// 状态筛选
	status := c.Query("status")

	// 获取总数 - 使用重试机制
	err = database.WithRetry(func(db *gorm.DB) error {
		query := db.Model(&models.TaskLog{}).Where("task_id = ?", taskID)
		if status != "" {
			query = query.Where("status = ?", status)
		}
		return query.Count(&total).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日志总数失败"})
		return
	}

	// 获取日志列表 - 使用重试机制
	err = database.WithRetry(func(db *gorm.DB) error {
		query := db.Where("task_id = ?", taskID)
		if status != "" {
			query = query.Where("status = ?", status)
		}
		return query.Preload("Task").Offset(offset).Limit(limit).Order("created_at desc").Find(&logs).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日志失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// RunTaskNow 立即执行任务
func RunTaskNow(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	var task models.Task
	if err := database.GetDB().First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	// 异步执行任务
	go func() {
		executor.ExecuteTask(&task)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "任务已开始执行"})
}

// ValidateScript 验证脚本语法
func ValidateScript(c *gin.Context) {
	var req struct {
		Script string `json:"script" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用执行器的验证函数
	if err := executor.ValidatePythonScript(req.Script); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "脚本语法正确",
	})
}

// DeleteAllLogs 删除所有日志
func DeleteAllLogs(c *gin.Context) {
	// 删除所有日志记录
	result := database.GetDB().Exec("DELETE FROM task_logs")
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除日志失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "所有日志已删除",
		"deleted_count": result.RowsAffected,
	})
}

// DeleteTaskLogs 删除特定任务的所有日志
func DeleteTaskLogs(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	// 删除指定任务的所有日志记录
	result := database.GetDB().Where("task_id = ?", taskID).Delete(&models.TaskLog{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除任务日志失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "任务日志已删除",
		"task_id":       taskID,
		"deleted_count": result.RowsAffected,
	})
}

// GetBarkRecords 获取Bark发送记录
func GetBarkRecords(c *gin.Context) {
	taskIDStr := c.Query("task_id")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	var taskID uint64 = 0
	if taskIDStr != "" {
		var err error
		taskID, err = strconv.ParseUint(taskIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
			return
		}
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	historyManager := barkhistory.NewBarkHistoryManager()
	records, total, err := historyManager.GetBarkRecords(uint(taskID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取Bark记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records":   records,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetBarkStats 获取Bark发送统计信息
func GetBarkStats(c *gin.Context) {
	historyManager := barkhistory.NewBarkHistoryManager()
	stats := historyManager.GetBarkStats()
	c.JSON(http.StatusOK, stats)
}

// DeleteAllBarkRecords 删除所有Bark记录
func DeleteAllBarkRecords(c *gin.Context) {
	historyManager := barkhistory.NewBarkHistoryManager()
	deletedCount, err := historyManager.DeleteAllBarkRecords()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除Bark记录失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "所有Bark记录已删除",
		"deleted_count": deletedCount,
	})
}

// GetLogStats 获取日志统计信息
func GetLogStats(c *gin.Context) {
	if globalLogManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "日志管理器未初始化"})
		return
	}

	stats := globalLogManager.GetLogStats()
	c.JSON(http.StatusOK, stats)
}

// GetTaskResult 获取任务的最新执行结果
func GetTaskResult(c *gin.Context) {
	id := c.Param("id")
	_, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	// 获取最新的成功执行日志
	var taskLog models.TaskLog
	if err := database.GetDB().Where("task_id = ? AND status = 'success'", id).
		Order("created_at desc").First(&taskLog).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到成功执行的结果"})
		return
	}

	// 解析 JSON 结果
	var result map[string]interface{}
	if taskLog.Result != "" {
		if err := json.Unmarshal([]byte(taskLog.Result), &result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解析结果失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id":    taskLog.TaskID,
		"log_id":     taskLog.ID,
		"result":     result,
		"created_at": taskLog.CreatedAt,
	})
}

// UpdateBarkConfig 更新任务的 Bark 配置
func UpdateBarkConfig(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	var req struct {
		BarkConfig string `json:"bark_config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 Bark 配置 JSON 格式
	var barkConfig models.BarkConfig
	if err := json.Unmarshal([]byte(req.BarkConfig), &barkConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 Bark 配置格式"})
		return
	}

	// 更新任务
	var task models.Task
	if err := database.GetDB().First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	task.BarkConfig = req.BarkConfig
	// 使用重试机制保存Bark配置
	err = database.WithRetry(func(db *gorm.DB) error {
		return db.Save(&task).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新 Bark 配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bark 配置更新成功"})
}

// GetTaskBarkKeys 获取任务可用的 Bark Keys
func GetTaskBarkKeys(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	// 获取任务
	var task models.Task
	if err := database.GetDB().First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	// 从最新执行记录中获取可用的 keys（不限制状态）
	var resultKeys []string
	var hasExecutionResult bool

	var taskLog models.TaskLog
	if err := database.GetDB().Where("task_id = ?", taskID).
		Order("created_at desc").First(&taskLog).Error; err == nil {

		hasExecutionResult = true

		// 使用与notifier相同的逻辑来获取可用的keys
		result := make(map[string]interface{})

		// 尝试解析JSON结果
		if taskLog.Result != "" {
			var rawResult map[string]interface{}
			if err := json.Unmarshal([]byte(taskLog.Result), &rawResult); err == nil {
				// JSON 解析成功，使用解析后的数据
				for key, value := range rawResult {
					result[key] = value
				}
			}
		}

		// 提取所有可用的keys
		for key := range result {
			resultKeys = append(resultKeys, key)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"keys":                 resultKeys,         // 从最新执行结果中获取的 keys
		"has_execution_result": hasExecutionResult, // 是否有执行结果
		"task_id":              taskID,
	})
}

// Bark管理页面处理器

// BarkManagementHandler Bark管理页面
func BarkManagementHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "bark_management.html", gin.H{
		"title": "Bark 管理",
	})
}

// Bark服务器管理API

// CreateBarkServer 创建Bark服务器
func CreateBarkServer(c *gin.Context) {
	var req models.CreateBarkServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果设置为默认服务器，先取消其他默认服务器
	if req.IsDefault {
		database.GetDB().Model(&models.BarkServer{}).Where("is_default = true").Update("is_default", false)
	}

	server := models.BarkServer{
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		IsDefault:   req.IsDefault,
		Status:      "active",
	}

	// 使用重试机制创建服务器
	err := database.WithRetry(func(db *gorm.DB) error {
		return db.Create(&server).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建服务器失败"})
		return
	}

	c.JSON(http.StatusCreated, server)
}

// GetBarkServers 获取Bark服务器列表
func GetBarkServers(c *gin.Context) {
	var servers []models.BarkServer

	query := database.GetDB()

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	var total int64
	query.Model(&models.BarkServer{}).Count(&total)

	// 获取服务器列表
	if err := query.Offset(offset).Limit(limit).Order("created_at desc").Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取服务器列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"servers": servers,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// GetBarkServer 获取单个Bark服务器
func GetBarkServer(c *gin.Context) {
	id := c.Param("id")
	serverID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的服务器ID"})
		return
	}

	var server models.BarkServer
	if err := database.GetDB().First(&server, serverID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "服务器不存在"})
		return
	}

	c.JSON(http.StatusOK, server)
}

// UpdateBarkServer 更新Bark服务器
func UpdateBarkServer(c *gin.Context) {
	id := c.Param("id")
	serverID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的服务器ID"})
		return
	}

	var req models.UpdateBarkServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var server models.BarkServer
	if err := database.GetDB().First(&server, serverID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "服务器不存在"})
		return
	}

	// 如果设置为默认服务器，先取消其他默认服务器
	if req.IsDefault && !server.IsDefault {
		database.GetDB().Model(&models.BarkServer{}).Where("is_default = true").Update("is_default", false)
	}

	// 更新字段
	if req.Name != "" {
		server.Name = req.Name
	}
	if req.URL != "" {
		server.URL = req.URL
	}
	if req.Description != "" {
		server.Description = req.Description
	}
	if req.Status != "" {
		server.Status = req.Status
	}
	server.IsDefault = req.IsDefault

	// 使用重试机制保存服务器
	err = database.WithRetry(func(db *gorm.DB) error {
		return db.Save(&server).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新服务器失败"})
		return
	}

	c.JSON(http.StatusOK, server)
}

// DeleteBarkServer 删除Bark服务器
func DeleteBarkServer(c *gin.Context) {
	id := c.Param("id")
	serverID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的服务器ID"})
		return
	}

	// 检查服务器是否存在
	var server models.BarkServer
	if err := database.GetDB().First(&server, serverID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "服务器不存在"})
		return
	}

	// 检查是否有设备关联到此服务器
	var deviceCount int64
	database.GetDB().Model(&models.BarkDevice{}).Where("server_id = ?", serverID).Count(&deviceCount)

	if deviceCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法删除：仍有设备关联到此服务器"})
		return
	}

	// 删除服务器 - 使用重试机制
	err = database.WithRetry(func(db *gorm.DB) error {
		return db.Delete(&models.BarkServer{}, serverID).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除服务器失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "服务器删除成功",
		"server_name": server.Name,
	})
}

// Bark设备管理API

// CreateBarkDevice 创建Bark设备
func CreateBarkDevice(c *gin.Context) {
	var req models.CreateBarkDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查设备密钥是否已存在
	var existingDevice models.BarkDevice
	if err := database.GetDB().Where("device_key = ?", req.DeviceKey).First(&existingDevice).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备密钥已存在"})
		return
	}

	// 如果设置为默认设备，先取消其他默认设备
	if req.IsDefault {
		database.GetDB().Model(&models.BarkDevice{}).Where("is_default = true").Update("is_default", false)
	}

	device := models.BarkDevice{
		Name:        req.Name,
		DeviceKey:   req.DeviceKey,
		Description: req.Description,
		ServerID:    req.ServerID,
		IsDefault:   req.IsDefault,
		Status:      "active",
	}

	// 使用重试机制创建设备
	err := database.WithRetry(func(db *gorm.DB) error {
		return db.Create(&device).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建设备失败"})
		return
	}

	// 预加载服务器信息
	database.GetDB().Preload("Server").First(&device, device.ID)

	c.JSON(http.StatusCreated, device)
}

// GetBarkDevices 获取Bark设备列表
func GetBarkDevices(c *gin.Context) {
	var devices []models.BarkDevice

	query := database.GetDB().Preload("Server")

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 服务器筛选
	if serverID := c.Query("server_id"); serverID != "" {
		query = query.Where("server_id = ?", serverID)
	}

	// 获取总数
	var total int64
	query.Model(&models.BarkDevice{}).Count(&total)

	// 获取设备列表
	if err := query.Offset(offset).Limit(limit).Order("created_at desc").Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设备列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// GetBarkDevice 获取单个Bark设备
func GetBarkDevice(c *gin.Context) {
	id := c.Param("id")
	deviceID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的设备ID"})
		return
	}

	var device models.BarkDevice
	if err := database.GetDB().Preload("Server").First(&device, deviceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		return
	}

	c.JSON(http.StatusOK, device)
}

// UpdateBarkDevice 更新Bark设备
func UpdateBarkDevice(c *gin.Context) {
	id := c.Param("id")
	deviceID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的设备ID"})
		return
	}

	var req models.UpdateBarkDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var device models.BarkDevice
	if err := database.GetDB().First(&device, deviceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		return
	}

	// 检查设备密钥是否已被其他设备使用
	if req.DeviceKey != "" && req.DeviceKey != device.DeviceKey {
		var existingDevice models.BarkDevice
		if err := database.GetDB().Where("device_key = ? AND id != ?", req.DeviceKey, deviceID).First(&existingDevice).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "设备密钥已被其他设备使用"})
			return
		}
	}

	// 如果设置为默认设备，先取消其他默认设备
	if req.IsDefault && !device.IsDefault {
		database.GetDB().Model(&models.BarkDevice{}).Where("is_default = true").Update("is_default", false)
	}

	// 更新字段
	if req.Name != "" {
		device.Name = req.Name
	}
	if req.DeviceKey != "" {
		device.DeviceKey = req.DeviceKey
	}
	if req.Description != "" {
		device.Description = req.Description
	}
	if req.ServerID != 0 {
		device.ServerID = req.ServerID
	}
	if req.Status != "" {
		device.Status = req.Status
	}
	device.IsDefault = req.IsDefault

	// 使用重试机制保存设备
	err = database.WithRetry(func(db *gorm.DB) error {
		return db.Save(&device).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新设备失败"})
		return
	}

	// 预加载服务器信息
	database.GetDB().Preload("Server").First(&device, device.ID)

	c.JSON(http.StatusOK, device)
}

// DeleteBarkDevice 删除Bark设备
func DeleteBarkDevice(c *gin.Context) {
	id := c.Param("id")
	deviceID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的设备ID"})
		return
	}

	// 检查设备是否存在
	var device models.BarkDevice
	if err := database.GetDB().First(&device, deviceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		return
	}

	// 删除设备 - 使用重试机制
	err = database.WithRetry(func(db *gorm.DB) error {
		return db.Delete(&models.BarkDevice{}, deviceID).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除设备失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "设备删除成功",
		"device_name": device.Name,
	})
}

// GetBarkDevicesForSelection 获取用于选择的Bark设备列表（简化版本）
func GetBarkDevicesForSelection(c *gin.Context) {
	var devices []models.BarkDevice

	// 只获取活跃的设备，包含基本信息
	if err := database.GetDB().
		Select("id, name, device_key, description, is_default").
		Where("status = ?", "active").
		Order("is_default desc, name asc").
		Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设备列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
	})
}
