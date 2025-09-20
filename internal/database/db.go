package database

import (
	"autobot/internal/models"
	"database/sql"
	"log"
	"os"
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
	sqlDB, err := sql.Open("sqlite", "autobot.db")
	if err != nil {
		return err
	}

	DB, err = gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "autobot.db",
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
