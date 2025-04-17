package database

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"navy-ng/models/portal"
)

// InitDB 初始化数据库连接
func InitDB() (*gorm.DB, error) {
	// 配置 GORM 日志
	gormLogger := logger.New(
		logger.Default.LogMode(logger.Info).(logger.Writer),
		logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			IgnoreRecordNotFoundError: true,        // 忽略记录未找到错误
			Colorful:                  true,        // 彩色输出
			LogLevel:                  logger.Info, // 设置日志级别为 Info
		},
	)

	// 创建SQLite数据库文件
	db, err := gorm.Open(sqlite.Open("navy.db"), &gorm.Config{
		Logger: gormLogger,
		// 启用详细日志
		PrepareStmt: true,
		// 打印 SQL 语句
		DryRun: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %v", err)
	}

	// 获取底层 SQL DB 并设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移数据库表
	if err := db.AutoMigrate(
		&portal.K8sNode{},
		&portal.K8sNodeLabel{},
		&portal.K8sNodeTaint{},
		&portal.Device{},
		&portal.F5Info{},
		&portal.OpsJob{},
		&portal.QueryTemplate{},
		&portal.LabelValue{},
		&portal.LabelManagement{},
		&portal.TaintManagement{},
		&portal.K8sETCD{},
		&portal.DeviceApp{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	ClearAndSeedDatabase(db)
	return db, nil
}
