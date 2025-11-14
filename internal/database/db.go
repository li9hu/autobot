package database

import (
	"autobot/internal/models"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "modernc.org/sqlite"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() error {
	var err error

	// 创建一个完全静默的logger
	silentLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // 慢SQL阈值
			LogLevel:                  logger.Silent, // 日志级别
			IgnoreRecordNotFoundError: true,          // 忽略ErrRecordNotFound错误
			Colorful:                  false,         // 禁用彩色打印
		},
	)

	// 使用 SQLite 数据库，通过 modernc.org/sqlite 驱动
	// 添加 SQLite 配置参数以改善并发性能
	// _busy_timeout=60000: 60秒超时，给长时间操作足够时间
	// _journal_mode=WAL: WAL模式，支持多读单写
	// _synchronous=NORMAL: 平衡性能和安全性
	// _cache_size=2000: 缓存大小
	// _foreign_keys=true: 启用外键约束
	// _temp_store=memory: 临时表存储在内存中
	// _txlock=immediate: 立即获取写锁，避免死锁
	dsn := "autobot.db?_busy_timeout=60000&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=2000&_foreign_keys=true&_temp_store=memory&_txlock=immediate"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}

	// 配置连接池 - 针对并发任务执行优化
	// SQLite with WAL mode 可以支持多个读连接和1个写连接
	// 设置足够的连接数以支持并发任务执行
	sqlDB.SetMaxOpenConns(20)              // 最多20个连接，足够支持10+个并发任务
	sqlDB.SetMaxIdleConns(5)               // 保持5个空闲连接，减少连接创建开销
	sqlDB.SetConnMaxLifetime(10 * time.Minute) // 连接生命周期10分钟

	DB, err = gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dsn,
		Conn:       sqlDB,
	}, &gorm.Config{
		Logger: silentLogger,
	})

	if err != nil {
		return err
	}

	// 自动迁移数据表
	err = DB.AutoMigrate(
		&models.Task{},
		&models.TaskLog{},
		&models.BarkServer{},
		&models.BarkDevice{},
		&models.BarkRecord{},
		&models.User{},
	)

	if err != nil {
		return err
	}

	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

// WithRetry 执行数据库操作，如果遇到 SQLITE_BUSY 错误则重试
// 针对SQLite的写锁竞争问题，使用更激进的重试策略
func WithRetry(operation func(*gorm.DB) error) error {
	const maxRetries = 15  // 增加到15次重试，应对高并发写操作
	const initialDelay = 10 * time.Millisecond
	const maxDelay = 2 * time.Second

	var err error
	for i := 0; i < maxRetries; i++ {
		// 为每次操作创建新的会话，避免事务状态污染
		// 使用PrepareStmt=false来减少锁持有时间，确保操作立即提交
		session := DB.Session(&gorm.Session{
			PrepareStmt: false, // 禁用预编译语句，减少锁持有时间
		})
		err = operation(session)
		if err == nil {
			return nil
		}

		// 检查是否是 SQLITE_BUSY 或事务错误
		errMsg := err.Error()
		if strings.Contains(errMsg, "database is locked") || 
			strings.Contains(errMsg, "SQLITE_BUSY") ||
			strings.Contains(errMsg, "cannot start a transaction within a transaction") {
			if i < maxRetries-1 { // 不是最后一次重试
				// 使用指数退避策略，但限制最大延迟
				// 延迟序列：10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.28s, 2s, 2s...
				delay := initialDelay * time.Duration(1<<uint(i))
				if delay > maxDelay {
					delay = maxDelay
				}
				log.Printf("Database error, retrying in %v (attempt %d/%d): %v", delay, i+1, maxRetries, err)
				time.Sleep(delay)
				continue
			}
		}

		// 非可重试错误或已达到最大重试次数
		break
	}

	return err
}
