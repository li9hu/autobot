package main

import (
	"autobot/internal/database"
	"autobot/internal/executor"
	"autobot/internal/handlers"
	"autobot/internal/logmanager"
	"autobot/internal/middleware"
	"autobot/internal/scheduler"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	if err := database.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 初始化调度器
	taskScheduler := scheduler.NewScheduler()
	taskScheduler.Start()
	defer taskScheduler.Stop()

	// 初始化日志管理器
	logMgr := logmanager.NewLogManager()

	// 设置全局调度器和日志管理器
	handlers.SetScheduler(taskScheduler)
	handlers.SetLogManager(logMgr)

	// 设置日志清理回调函数
	executor.SetLogCleanupCallback(logMgr.CleanupLogsAfterExecution)

	// 设置 Gin 路由 - 使用发布模式减少日志输出
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 静态文件服务
	r.Static("/static", "./web/static")

	// 设置HTML模板路径
	r.LoadHTMLGlob("web/templates/*")

	// 公开路由（无需鉴权）
	public := r.Group("/")
	{
		// 登录相关页面和API
		public.GET("/login", handlers.LoginHandler)
		public.POST("/login", handlers.Login)
		public.POST("/register", handlers.Register)
		public.GET("/api/auth/check-registration", handlers.CheckRegistrationAvailable)
	}

	// 需要鉴权的路由
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// 前端页面路由
		protected.GET("/", handlers.IndexHandler)
		protected.GET("/tasks/new", handlers.NewTaskHandler)
		protected.GET("/tasks/:id/edit", handlers.EditTaskHandler)
		protected.GET("/tasks/:id", handlers.TaskDetailHandler)
		protected.GET("/logs", handlers.LogsHandler)
		protected.GET("/bark", handlers.BarkManagementHandler)
	}

	// API 路由（需要鉴权）
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// 鉴权相关API
		api.POST("/logout", handlers.Logout)
		api.GET("/me", handlers.GetCurrentUser)
		// 任务相关API
		api.POST("/tasks", handlers.CreateTask)
		api.GET("/tasks", handlers.GetTasks)
		api.GET("/tasks/:id", handlers.GetTask)
		api.PUT("/tasks/:id", handlers.UpdateTask)
		api.DELETE("/tasks/:id", handlers.DeleteTask)
		api.GET("/tasks/:id/logs", handlers.GetTaskLogs)
		api.POST("/tasks/:id/run", handlers.RunTaskNow)
		api.POST("/validate-script", handlers.ValidateScript)
		api.GET("/tasks/:id/result", handlers.GetTaskResult)
		api.PUT("/tasks/:id/bark-config", handlers.UpdateBarkConfig)
		api.GET("/tasks/:id/bark-keys", handlers.GetTaskBarkKeys)

		// 日志相关API
		api.DELETE("/logs/all", handlers.DeleteAllLogs)
		api.DELETE("/tasks/:id/logs", handlers.DeleteTaskLogs)
		api.GET("/logs/stats", handlers.GetLogStats)

		// Bark服务器管理API
		api.POST("/bark/servers", handlers.CreateBarkServer)
		api.GET("/bark/servers", handlers.GetBarkServers)
		api.GET("/bark/servers/:id", handlers.GetBarkServer)
		api.PUT("/bark/servers/:id", handlers.UpdateBarkServer)
		api.DELETE("/bark/servers/:id", handlers.DeleteBarkServer)

		// Bark设备管理API
		api.POST("/bark/devices", handlers.CreateBarkDevice)
		api.GET("/bark/devices", handlers.GetBarkDevices)
		api.GET("/bark/devices/selection", handlers.GetBarkDevicesForSelection)
		api.GET("/bark/devices/:id", handlers.GetBarkDevice)
		api.PUT("/bark/devices/:id", handlers.UpdateBarkDevice)
		api.DELETE("/bark/devices/:id", handlers.DeleteBarkDevice)

		// Bark历史记录API
		api.GET("/bark/records", handlers.GetBarkRecords)
		api.GET("/bark/stats", handlers.GetBarkStats)
		api.DELETE("/bark/records/all", handlers.DeleteAllBarkRecords)
	}

	log.Println("Server starting on port 8080...")
	r.Run(":8080")
}
