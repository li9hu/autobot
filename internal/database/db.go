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
	dsn := "autobot.db?_busy_timeout=60000&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=2000&_foreign_keys=true&_temp_store=memory"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}

	// 配置连接池 - 进一步减少并发连接数以避免锁定
	// SQLite 在高并发写入时容易锁定，使用更保守的设置
	sqlDB.SetMaxOpenConns(3) // 最多3个连接
	sqlDB.SetMaxIdleConns(1) // 保持1个空闲连接
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

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
func WithRetry(operation func(*gorm.DB) error) error {
	const maxRetries = 5
	const baseDelay = 50 * time.Millisecond

	var err error
	for i := 0; i < maxRetries; i++ {
		err = operation(DB)
		if err == nil {
			return nil
		}

		// 检查是否是 SQLITE_BUSY 错误
		if strings.Contains(err.Error(), "database is locked") || strings.Contains(err.Error(), "SQLITE_BUSY") {
			if i < maxRetries-1 { // 不是最后一次重试
				// 使用指数退避策略：50ms, 100ms, 200ms, 400ms
				delay := baseDelay * time.Duration(1<<i)
				log.Printf("Database busy, retrying in %v (attempt %d/%d): %v", delay, i+1, maxRetries, err)
				time.Sleep(delay)
				continue
			}
		}

		// 非 SQLITE_BUSY 错误或已达到最大重试次数
		break
	}

	return err
}
