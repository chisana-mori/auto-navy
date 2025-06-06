package service

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Constants for Strategy Execution Results
const (

	// Strategy Execution Results (add more as needed from previous/future steps)
	StrategyExecutionResultOrderCreated               = "order_created"
	StrategyExecutionResultOrderFailed                = "failure_order_creation_failed"
	StrategyExecutionResultBreachedPendingDeviceMatch = "breached_pending_device_match" // From previous step
	StrategyExecutionResultFailureNoSnapshots         = "failure_no_snapshots_for_duration"
	StrategyExecutionResultFailureThresholdNotMet     = "failure_threshold_not_met"
	StrategyExecutionResultFailureInvalidTemplateID   = "failure_invalid_query_template_id"
	StrategyExecutionResultFailureTemplateNotFound    = "failure_query_template_not_found"
	StrategyExecutionResultFailureTemplateUnmarshal   = "failure_query_template_unmarshal_error"
	StrategyExecutionResultFailureDeviceQuery         = "failure_device_query_error"
	StrategyExecutionResultFailureNoDevicesFound      = "failure_no_devices_found"
	StrategyExecutionResultFailureNoSuitableDevices   = "failure_no_suitable_devices_selected"
	StrategyExecutionResultFailureNoDevicesForOrder   = "failure_no_devices_for_order" // If selection leads to zero, though unlikely now
	// Results for order status updates
	StrategyExecutionResultOrderProcessingStarted     = "order_processing_started"
	StrategyExecutionResultOrderCompleted             = "order_completed"
	StrategyExecutionResultOrderProcStartedNoExecTime = "order_processing_started_no_exec_time"
	StrategyExecutionResultOrderComplNoComplTime      = "order_completed_no_compl_time"
	// DB error during strategy evaluation stages
	StrategyExecutionResultFailureDBError = "failure_db_error"

	SystemAutoCreator = "system/auto"
)

// DeviceCacheInterface defines the interface for a device cache.
type DeviceCacheInterface interface {
	GetDeviceList(queryHash string) (*DeviceListResponse, error)
	SetDeviceList(queryHash string, response *DeviceListResponse) error
	InvalidateDeviceLists() error
	GetDevice(deviceID int64) (*DeviceResponse, error)
	SetDevice(deviceID int64, device *DeviceResponse) error
	GetDeviceFieldValues(field string) ([]string, error)
	SetDeviceFieldValues(field string, values []string, isLabelField bool) error
}

// RedisHandlerInterface 定义 ElasticScalingService 所需的 Redis 方法
type RedisHandlerInterface interface {
	AcquireLock(key string, value string, expiry time.Duration) (isSuccess bool, err error)
	Delete(key string) // Delete method does not return error
	// Note: The original redis.Handler.Expire sets a default, not for a specific key.
	// If key-specific expiration is needed, the redis package might need modification
	// or a different approach is required. For now, we'll use the default setter.
	Expire(expiration time.Duration) // Expire method takes only duration
}

// ElasticScalingService 弹性伸缩服务
type ElasticScalingService struct {
	db           *gorm.DB
	redisHandler RedisHandlerInterface // Use RedisHandlerInterface
	logger       *zap.Logger           // Added logger
	cache        DeviceCacheInterface  // Changed to DeviceCacheInterface
	orderService OrderService          // 通用订单服务
}

// NewElasticScalingService 创建弹性伸缩服务实例
// 接受数据库连接、RedisHandlerInterface 实例、logger 和 cache 作为参数
func NewElasticScalingService(db *gorm.DB, redisHandler RedisHandlerInterface, logger *zap.Logger, cache DeviceCacheInterface) *ElasticScalingService {
	orderService := NewOrderService(db)
	return &ElasticScalingService{
		db:           db,
		redisHandler: redisHandler,
		logger:       logger,       // Assign logger
		cache:        cache,        // Assign cache
		orderService: orderService, // 初始化通用订单服务
	}
}
