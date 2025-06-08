package service

import (
	"fmt"
	"navy-ng/models/portal"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Constants for Strategy Execution Results
const (

	// Strategy Execution Results (add more as needed from previous/future steps)
	StrategyExecutionResultOrderCreated               = "order_created"
	StrategyExecutionResultOrderCreatedNoDevices      = "order_created_no_devices" // 无设备时创建提醒订单
	StrategyExecutionResultOrderCreatedPartial        = "order_created_partial"    // 部分设备匹配时创建订单
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
	Delete(key string)
	Expire(expiration time.Duration)
	Get(key string) string
	SetWithExpireTime(key string, value string, expiration time.Duration)
	ScanKeys(pattern string) ([]string, error)
}

// ElasticScalingService 弹性伸缩服务
type ElasticScalingService struct {
	db                          *gorm.DB
	redisHandler                RedisHandlerInterface // Use RedisHandlerInterface
	logger                      *zap.Logger           // Added logger
	cache                       DeviceCacheInterface  // Changed to DeviceCacheInterface
	orderService                OrderService          // 通用订单服务
	matchDevicesForStrategyFunc func(strategy *portal.ElasticScalingStrategy, clusterID int64, resourceType, triggeredValue, thresholdValue string, cpuDelta, memDelta float64) error
}

// GetStrategyExecutionHistoryWithPagination 获取策略执行历史（分页）
func (s *ElasticScalingService) GetStrategyExecutionHistoryWithPagination(strategyID int64, pagination *PaginationRequest, clusterName string) ([]StrategyExecutionHistoryDetailDTO, int64, error) {
	// 首先检查策略是否存在
	var strategy portal.ElasticScalingStrategy
	if err := s.db.First(&strategy, strategyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, fmt.Errorf("策略不存在: %d", strategyID)
		}
		return nil, 0, err
	}

	// 构建基础查询条件
	baseQuery := s.db.Model(&portal.StrategyExecutionHistory{}).Where("strategy_id = ?", strategyID)

	// 如果有集群名字过滤条件，需要关联集群表进行模糊查询
	if clusterName != "" {
		baseQuery = baseQuery.Joins("JOIN k8s_clusters ON strategy_execution_histories.cluster_id = k8s_clusters.id").
			Where("k8s_clusters.cluster_name LIKE ? OR k8s_clusters.cluster_name_cn LIKE ?", "%"+clusterName+"%", "%"+clusterName+"%")
	}

	// 获取总数
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	var histories []portal.StrategyExecutionHistory
	query := s.db.Where("strategy_id = ?", strategyID)

	// 如果有集群名字过滤条件，添加相同的关联查询
	if clusterName != "" {
		query = query.Joins("JOIN k8s_clusters ON strategy_execution_histories.cluster_id = k8s_clusters.id").
			Where("k8s_clusters.cluster_name LIKE ? OR k8s_clusters.cluster_name_cn LIKE ?", "%"+clusterName+"%", "%"+clusterName+"%")
	}

	query = query.Order("execution_time DESC").
		Offset(pagination.GetOffset()).
		Limit(pagination.Size)

	if err := query.Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	// 获取所有相关的集群ID
	clusterIDs := make([]int64, 0)
	for _, history := range histories {
		clusterIDs = append(clusterIDs, history.ClusterID)
	}

	// 批量获取集群信息
	var clusters []portal.K8sCluster
	clusterMap := make(map[int64]portal.K8sCluster)
	if len(clusterIDs) > 0 {
		if err := s.db.Where("id IN ?", clusterIDs).Find(&clusters).Error; err != nil {
			return nil, 0, err
		}
		for _, cluster := range clusters {
			clusterMap[cluster.ID] = cluster
		}
	}

	// 转换为DTO
	result := make([]StrategyExecutionHistoryDetailDTO, len(histories))
	for i, history := range histories {
		clusterName := "未知集群"
		if cluster, exists := clusterMap[history.ClusterID]; exists {
			if cluster.ClusterNameCn != "" {
				clusterName = cluster.ClusterNameCn
			} else {
				clusterName = cluster.ClusterName
			}
		}

		result[i] = StrategyExecutionHistoryDetailDTO{
			ID:             history.ID,
			StrategyID:     history.StrategyID,
			StrategyName:   strategy.Name,
			ClusterID:      history.ClusterID,
			ClusterName:    clusterName,
			ResourceType:   history.ResourceType,
			ExecutionTime:  time.Time(history.ExecutionTime),
			TriggeredValue: history.TriggeredValue,
			ThresholdValue: history.ThresholdValue,
			Result:         history.Result,
			OrderID:        history.OrderID,
			HasOrder:       history.OrderID != nil,
			Reason:         history.Reason,
		}
	}

	return result, total, nil
}

// NewElasticScalingService 创建弹性伸缩服务实例
// 接受数据库连接、RedisHandlerInterface 实例、logger 和 cache 作为参数
func NewElasticScalingService(db *gorm.DB, redisHandler RedisHandlerInterface, logger *zap.Logger, cache DeviceCacheInterface) *ElasticScalingService {
	orderService := NewOrderService(db)
	s := &ElasticScalingService{
		db:           db,
		redisHandler: redisHandler,
		logger:       logger,       // Assign logger
		cache:        cache,        // Assign cache
		orderService: orderService, // 初始化通用订单服务
	}
	s.matchDevicesForStrategyFunc = s.matchDevicesForStrategy
	return s
}

// SetMatchDevicesForStrategyFunc is a test helper to mock the device matching function.
func (s *ElasticScalingService) SetMatchDevicesForStrategyFunc(f func(strategy *portal.ElasticScalingStrategy, clusterID int64, resourceType, triggeredValue, thresholdValue string, cpuDelta, memDelta float64) error) {
	s.matchDevicesForStrategyFunc = f
}
