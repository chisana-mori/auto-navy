package service

import (
	"fmt"
	"navy-ng/pkg/redis"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap" // Added zap import
	"gorm.io/gorm"
)

// ElasticScalingMonitor 弹性伸缩监控服务
type ElasticScalingMonitor struct {
	db             *gorm.DB
	redisHandler   *redis.Handler
	scalingService *ElasticScalingService
	config         MonitorConfig
	logger         *zap.Logger // Added logger
	stopChan       chan struct{}
	isRunning      bool
	cron           *cron.Cron
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	MonitorCron        string        // 监控任务的 Cron 表达式
	EvaluationInterval time.Duration // 策略评估间隔
	LockTimeout        time.Duration // Redis锁超时时间
	LockRetryInterval  time.Duration // Redis锁重试间隔
	LockMaxRetries     int           // Redis锁最大重试次数
}

// DefaultMonitorConfig 默认监控配置
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		MonitorCron:        "* * * * *",        // 每分钟运行一次
		EvaluationInterval: 10 * time.Minute, // 每10分钟评估一次策略
		LockTimeout:        30 * time.Second, // 锁超时时间30秒
		LockRetryInterval:  1 * time.Second,  // 锁重试间隔1秒
		LockMaxRetries:     3,                // 最大重试3次
	}
}

// NewElasticScalingMonitor 创建弹性伸缩监控服务实例
func NewElasticScalingMonitor(db *gorm.DB, config MonitorConfig, logger *zap.Logger) *ElasticScalingMonitor {
	// 使用默认Redis连接
	redisHandler := redis.NewRedisHandler("default")
	// 设置锁的过期时间
	redisHandler.Expire(config.LockTimeout)

	// 创建设备缓存
	deviceCache := NewDeviceCache(redisHandler, redis.NewKeyBuilder("navy", "v1"))

	monitor := &ElasticScalingMonitor{
		db:             db,
		redisHandler:   redisHandler,
		scalingService: NewElasticScalingService(db, redisHandler, logger, deviceCache), // Pass redisHandler, logger and cache
		config:         config,
		logger:         logger, // Assign logger
		stopChan:       make(chan struct{}),
		isRunning:      false,
	}
	monitor.cron = cron.New(cron.WithLogger(cron.DefaultLogger))
	return monitor
}

// Start 启动监控服务
func (m *ElasticScalingMonitor) Start() {
	if m.isRunning {
		m.logger.Info("Monitoring service is already running")
		return
	}

	m.isRunning = true
	m.logger.Info("Starting elastic scaling monitoring service")

	// 启动策略评估协程
	go m.startStrategyEvaluator()
	m.cron.Start()
}

// Stop 停止监控服务
func (m *ElasticScalingMonitor) Stop() {
	if !m.isRunning {
		return
	}

	ctx := m.cron.Stop()
	<-ctx.Done()
	close(m.stopChan)
	m.isRunning = false
	m.logger.Info("Stopping elastic scaling monitoring service")
}

// startStrategyEvaluator 启动策略评估
func (m *ElasticScalingMonitor) startStrategyEvaluator() {
	_, err := m.cron.AddFunc(m.config.MonitorCron, func() {
		if err := m.evaluateStrategiesWithLock(); err != nil {
			m.logger.Error("Strategy evaluation failed", zap.Error(err))
		}
	})
	if err != nil {
		m.logger.Error("Failed to add cron job", zap.Error(err))
	}

	<-m.stopChan
}

// evaluateStrategiesWithLock 使用Redis分布式锁评估策略
func (m *ElasticScalingMonitor) evaluateStrategiesWithLock() error {
	lockKey := "elastic_scaling:strategy_evaluation_lock"
	lockValue := fmt.Sprintf("monitor:%d", time.Now().UnixNano())

	// 使用项目现有的Redis锁机制
	// 设置锁超时时间
	m.redisHandler.Expire(m.config.LockTimeout)

	// 尝试获取分布式锁
	success, err := m.redisHandler.AcquireLock(lockKey, lockValue, m.config.LockTimeout)
	if err != nil {
		return fmt.Errorf("获取分布式锁失败: %w", err)
	}

	if !success {
		m.logger.Info("Could not acquire strategy evaluation lock, another instance might be executing", zap.String("lockKey", lockKey))
		return nil
	}

	// 确保锁释放
	defer m.redisHandler.Delete(lockKey)

	m.logger.Info("Starting evaluation of elastic scaling strategies...")
	// 调用现有的策略评估方法
	if err := m.scalingService.EvaluateStrategies(); err != nil {
		return fmt.Errorf("策略评估失败: %w", err)
	}

	m.logger.Info("Strategy evaluation completed")
	return nil
}
