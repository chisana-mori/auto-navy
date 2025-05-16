package service

import (
	"errors"
	"fmt"

	// "log" // This might become unused if all log.Printf are replaced - Removed

	"math/rand"
	"navy-ng/models/portal"
	"strings"
	"time"

	// Removed "navy-ng/pkg/redis" as it's no longer directly used after introducing interface

	"go.uber.org/zap" // Added zap import
	"gorm.io/gorm"
)

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
}

// NewElasticScalingService 创建弹性伸缩服务实例
// 接受数据库连接、RedisHandlerInterface 实例和 logger 作为参数
func NewElasticScalingService(db *gorm.DB, redisHandler RedisHandlerInterface, logger *zap.Logger) *ElasticScalingService {
	return &ElasticScalingService{
		db:           db,
		redisHandler: redisHandler,
		logger:       logger, // Assign logger
	}
}

// CreateStrategy 创建弹性伸缩策略
func (s *ElasticScalingService) CreateStrategy(dto StrategyDTO) (int64, error) {
	// 参数验证
	if err := s.validateStrategyDTO(&dto); err != nil {
		return 0, err
	}

	// 构建策略模型
	strategy := portal.ElasticScalingStrategy{
		Name:                   dto.Name,
		Description:            dto.Description,
		ThresholdTriggerAction: dto.ThresholdTriggerAction,
		DeviceCount:            dto.DeviceCount,
		NodeSelector:           dto.NodeSelector,
		ResourceTypes:          dto.ResourceTypes,
		Status:                 dto.Status,
		CreatedBy:              dto.CreatedBy,
		DurationMinutes:        dto.DurationMinutes,
		CooldownMinutes:        dto.CooldownMinutes,
	}

	// 设置可选字段
	if dto.CPUThresholdValue != nil {
		strategy.CPUThresholdValue = *dto.CPUThresholdValue
		strategy.CPUThresholdType = *dto.CPUThresholdType

		if dto.CPUTargetValue != nil {
			strategy.CPUTargetValue = *dto.CPUTargetValue
		}
	}

	if dto.MemoryThresholdValue != nil {
		strategy.MemoryThresholdValue = *dto.MemoryThresholdValue
		strategy.MemoryThresholdType = *dto.MemoryThresholdType

		if dto.MemoryTargetValue != nil {
			strategy.MemoryTargetValue = *dto.MemoryTargetValue
		}
	}

	if dto.CPUThresholdValue != nil && dto.MemoryThresholdValue != nil {
		strategy.ConditionLogic = dto.ConditionLogic
	}

	// 使用事务确保策略与集群关联的原子性
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 创建策略
		if err := tx.Create(&strategy).Error; err != nil {
			return err
		}

		// 创建集群关联关系
		for _, clusterID := range dto.ClusterIDs {
			association := portal.StrategyClusterAssociation{
				StrategyID: strategy.ID,
				ClusterID:  clusterID,
			}
			if err := tx.Create(&association).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return strategy.ID, nil
}

// UpdateStrategy 更新弹性伸缩策略
func (s *ElasticScalingService) UpdateStrategy(id int64, dto StrategyDTO) error {
	// 参数验证
	if err := s.validateStrategyDTO(&dto); err != nil {
		return err
	}

	// 检查策略是否存在
	var strategy portal.ElasticScalingStrategy
	if err := s.db.First(&strategy, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("策略不存在: %d", id)
		}
		return err
	}

	// 更新策略字段
	strategy.Name = dto.Name
	strategy.Description = dto.Description
	strategy.ThresholdTriggerAction = dto.ThresholdTriggerAction
	strategy.DeviceCount = dto.DeviceCount
	strategy.NodeSelector = dto.NodeSelector
	strategy.ResourceTypes = dto.ResourceTypes
	strategy.Status = dto.Status
	strategy.DurationMinutes = dto.DurationMinutes
	strategy.CooldownMinutes = dto.CooldownMinutes

	// 设置可选字段
	if dto.CPUThresholdValue != nil {
		strategy.CPUThresholdValue = *dto.CPUThresholdValue
		strategy.CPUThresholdType = *dto.CPUThresholdType

		if dto.CPUTargetValue != nil {
			strategy.CPUTargetValue = *dto.CPUTargetValue
		} else {
			strategy.CPUTargetValue = 0
		}
	} else {
		strategy.CPUThresholdValue = 0
		strategy.CPUThresholdType = ""
		strategy.CPUTargetValue = 0
	}

	if dto.MemoryThresholdValue != nil {
		strategy.MemoryThresholdValue = *dto.MemoryThresholdValue
		strategy.MemoryThresholdType = *dto.MemoryThresholdType

		if dto.MemoryTargetValue != nil {
			strategy.MemoryTargetValue = *dto.MemoryTargetValue
		} else {
			strategy.MemoryTargetValue = 0
		}
	} else {
		strategy.MemoryThresholdValue = 0
		strategy.MemoryThresholdType = ""
		strategy.MemoryTargetValue = 0
	}

	if dto.CPUThresholdValue != nil && dto.MemoryThresholdValue != nil {
		strategy.ConditionLogic = dto.ConditionLogic
	} else {
		strategy.ConditionLogic = "OR" // 默认值
	}

	// 使用事务确保更新操作的原子性
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 更新策略
		if err := tx.Save(&strategy).Error; err != nil {
			return err
		}

		// 删除现有关联
		if err := tx.Where("strategy_id = ?", id).Delete(&portal.StrategyClusterAssociation{}).Error; err != nil {
			return err
		}

		// 创建新的关联
		for _, clusterID := range dto.ClusterIDs {
			association := portal.StrategyClusterAssociation{
				StrategyID: strategy.ID,
				ClusterID:  clusterID,
			}
			if err := tx.Create(&association).Error; err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

// GetStrategy 获取策略详情
func (s *ElasticScalingService) GetStrategy(id int64) (*StrategyDetailDTO, error) {
	// 获取策略基本信息
	var strategy portal.ElasticScalingStrategy
	if err := s.db.First(&strategy, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("策略不存在: %d", id)
		}
		return nil, err
	}

	// 获取关联集群
	var associations []portal.StrategyClusterAssociation
	if err := s.db.Where("strategy_id = ?", id).Find(&associations).Error; err != nil {
		return nil, err
	}

	clusterIDs := make([]int64, len(associations))
	for i, assoc := range associations {
		clusterIDs[i] = assoc.ClusterID
	}

	// 获取执行历史
	var histories []portal.StrategyExecutionHistory
	if err := s.db.Where("strategy_id = ?", id).Order("execution_time DESC").Limit(20).Find(&histories).Error; err != nil {
		return nil, err
	}

	// 获取相关订单
	var orders []portal.ElasticScalingOrder
	if err := s.db.Where("strategy_id = ?", id).Order("created_at DESC").Limit(10).Find(&orders).Error; err != nil {
		return nil, err
	}

	// 转换为DTO
	dto := &StrategyDetailDTO{
		StrategyDTO: StrategyDTO{
			ID:                     strategy.ID,
			Name:                   strategy.Name,
			Description:            strategy.Description,
			ThresholdTriggerAction: strategy.ThresholdTriggerAction,
			DeviceCount:            strategy.DeviceCount,
			NodeSelector:           strategy.NodeSelector,
			ResourceTypes:          strategy.ResourceTypes,
			Status:                 strategy.Status,
			CreatedBy:              strategy.CreatedBy,
			CreatedAt:              time.Time(strategy.CreatedAt),
			UpdatedAt:              time.Time(strategy.UpdatedAt),
			DurationMinutes:        strategy.DurationMinutes,
			CooldownMinutes:        strategy.CooldownMinutes,
			ClusterIDs:             clusterIDs,
		},
		ExecutionHistory: make([]StrategyExecutionHistoryDTO, len(histories)),
		RelatedOrders:    make([]OrderListItemDTO, len(orders)),
	}

	// 设置可选阈值字段
	if strategy.CPUThresholdValue > 0 {
		cpuValue := strategy.CPUThresholdValue
		cpuType := strategy.CPUThresholdType
		dto.CPUThresholdValue = &cpuValue
		dto.CPUThresholdType = &cpuType

		// 添加CPU目标值
		if strategy.CPUTargetValue > 0 {
			cpuTargetVal := strategy.CPUTargetValue
			dto.CPUTargetValue = &cpuTargetVal
		}
	}

	if strategy.MemoryThresholdValue > 0 {
		memValue := strategy.MemoryThresholdValue
		memType := strategy.MemoryThresholdType
		dto.MemoryThresholdValue = &memValue
		dto.MemoryThresholdType = &memType

		// 添加内存目标值
		if strategy.MemoryTargetValue > 0 {
			memTargetVal := strategy.MemoryTargetValue
			dto.MemoryTargetValue = &memTargetVal
		}
	}

	dto.ConditionLogic = strategy.ConditionLogic

	// 添加持续时间和冷却时间
	dto.DurationMinutes = strategy.DurationMinutes
	dto.CooldownMinutes = strategy.CooldownMinutes

	// 添加资源类型
	dto.ResourceTypes = strategy.ResourceTypes

	// 转换执行历史
	for i, h := range histories {
		dto.ExecutionHistory[i] = StrategyExecutionHistoryDTO{
			ID:             h.ID,
			ExecutionTime:  time.Time(h.ExecutionTime),
			TriggeredValue: h.TriggeredValue,
			ThresholdValue: h.ThresholdValue,
			Result:         h.Result,
			OrderID:        h.OrderID,
			Reason:         h.Reason,
		}
	}

	// 转换相关订单
	for i, o := range orders {
		// 获取集群名称
		var cluster portal.K8sCluster
		clusterName := "未知集群"
		if err := s.db.Select("clustername").First(&cluster, o.ClusterID).Error; err == nil {
			clusterName = cluster.ClusterName
		}

		dto.RelatedOrders[i] = OrderListItemDTO{
			ID:           o.ID,
			OrderNumber:  o.OrderNumber,
			ClusterID:    o.ClusterID,
			ClusterName:  clusterName,
			StrategyID:   o.StrategyID,
			StrategyName: strategy.Name,
			ActionType:   o.ActionType,
			Status:       o.Status,
			DeviceCount:  o.DeviceCount,
			CreatedBy:    o.CreatedBy,
			CreatedAt:    time.Time(o.CreatedAt),
		}
	}

	return dto, nil
}

// ListStrategies 获取策略列表
func (s *ElasticScalingService) ListStrategies(name string, status string, action string, page, pageSize int) ([]StrategyListItemDTO, int64, error) {
	var total int64
	query := s.db.Model(&portal.ElasticScalingStrategy{})

	// 应用过滤条件
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if action != "" {
		query = query.Where("threshold_trigger_action = ?", action)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var strategies []portal.ElasticScalingStrategy
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&strategies).Error; err != nil {
		return nil, 0, err
	}

	result := make([]StrategyListItemDTO, len(strategies))
	for i, strategy := range strategies {
		// 获取关联集群
		var associations []portal.StrategyClusterAssociation
		if err := s.db.Where("strategy_id = ?", strategy.ID).Find(&associations).Error; err != nil {
			return nil, 0, err
		}

		// 获取集群名称
		clusterNames := make([]string, 0, len(associations))
		for _, assoc := range associations {
			var cluster portal.K8sCluster
			if err := s.db.Select("clustername").First(&cluster, assoc.ClusterID).Error; err == nil {
				clusterNames = append(clusterNames, cluster.ClusterName)
			}
		}

		// 构建DTO
		dto := StrategyListItemDTO{
			ID:                     strategy.ID,
			Name:                   strategy.Name,
			Description:            strategy.Description,
			ThresholdTriggerAction: strategy.ThresholdTriggerAction,
			DeviceCount:            strategy.DeviceCount,
			Status:                 strategy.Status,
			CreatedAt:              time.Time(strategy.CreatedAt),
			UpdatedAt:              time.Time(strategy.UpdatedAt),
			Clusters:               clusterNames,
			CPUTargetValue:         &strategy.CPUTargetValue,
			MemoryTargetValue:      &strategy.MemoryTargetValue,
			DurationMinutes:        strategy.DurationMinutes,
			CooldownMinutes:        strategy.CooldownMinutes,
			ResourceTypes:          strategy.ResourceTypes,
		}

		// 设置可选阈值字段
		if strategy.CPUThresholdValue > 0 {
			cpuValue := strategy.CPUThresholdValue
			cpuType := strategy.CPUThresholdType
			dto.CPUThresholdValue = &cpuValue
			dto.CPUThresholdType = &cpuType
		}

		if strategy.MemoryThresholdValue > 0 {
			memValue := strategy.MemoryThresholdValue
			memType := strategy.MemoryThresholdType
			dto.MemoryThresholdValue = &memValue
			dto.MemoryThresholdType = &memType
		}

		dto.ConditionLogic = strategy.ConditionLogic

		result[i] = dto
	}

	return result, total, nil
}

// UpdateStrategyStatus 更新策略状态
func (s *ElasticScalingService) UpdateStrategyStatus(id int64, status string) error {
	if status != "enabled" && status != "disabled" {
		return fmt.Errorf("无效的策略状态: %s", status)
	}

	var strategy portal.ElasticScalingStrategy
	if err := s.db.First(&strategy, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("策略不存在: %d", id)
		}
		return err
	}

	strategy.Status = status
	return s.db.Save(&strategy).Error
}

// DeleteStrategy 删除策略
func (s *ElasticScalingService) DeleteStrategy(id int64) error {
	// 检查是否有关联的执行中订单
	var count int64
	if err := s.db.Model(&portal.ElasticScalingOrder{}).
		Where("strategy_id = ? AND status IN ('pending', 'processing')", id).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("策略有正在执行的订单，无法删除")
	}

	// 使用事务保证删除操作的完整性
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除策略集群关联
		if err := tx.Where("strategy_id = ?", id).Delete(&portal.StrategyClusterAssociation{}).Error; err != nil {
			return err
		}

		// 删除策略
		if err := tx.Delete(&portal.ElasticScalingStrategy{}, id).Error; err != nil {
			return err
		}

		return nil
	})
}

// validateStrategyDTO 验证策略DTO
func (s *ElasticScalingService) validateStrategyDTO(dto *StrategyDTO) error {
	if dto.Name == "" {
		return errors.New("策略名称不能为空")
	}

	if dto.ThresholdTriggerAction != "pool_entry" && dto.ThresholdTriggerAction != "pool_exit" {
		return errors.New("无效的触发动作类型")
	}

	if dto.CPUThresholdValue == nil && dto.MemoryThresholdValue == nil {
		return errors.New("至少需要设置CPU或内存阈值")
	}

	if dto.CPUThresholdValue != nil {
		if *dto.CPUThresholdValue <= 0 || *dto.CPUThresholdValue > 100 {
			return errors.New("CPU阈值必须在0-100之间")
		}
		if *dto.CPUThresholdType != "usage" && *dto.CPUThresholdType != "allocated" {
			return errors.New("无效的CPU阈值类型")
		}

		// 验证CPU目标值
		if dto.CPUTargetValue != nil {
			if *dto.CPUTargetValue <= 0 || *dto.CPUTargetValue > 100 {
				return errors.New("CPU目标值必须在0-100之间")
			}

			// 根据动作类型验证目标值与阈值的关系
			if dto.ThresholdTriggerAction == "pool_entry" && *dto.CPUTargetValue >= *dto.CPUThresholdValue {
				return errors.New("入池动作的CPU目标值必须小于阈值")
			} else if dto.ThresholdTriggerAction == "pool_exit" && *dto.CPUTargetValue <= *dto.CPUThresholdValue {
				return errors.New("退池动作的CPU目标值必须大于阈值")
			}
		}
	}

	if dto.MemoryThresholdValue != nil {
		if *dto.MemoryThresholdValue <= 0 || *dto.MemoryThresholdValue > 100 {
			return errors.New("内存阈值必须在0-100之间")
		}
		if *dto.MemoryThresholdType != "usage" && *dto.MemoryThresholdType != "allocated" {
			return errors.New("无效的内存阈值类型")
		}

		// 验证内存目标值
		if dto.MemoryTargetValue != nil {
			if *dto.MemoryTargetValue <= 0 || *dto.MemoryTargetValue > 100 {
				return errors.New("内存目标值必须在0-100之间")
			}

			// 根据动作类型验证目标值与阈值的关系
			if dto.ThresholdTriggerAction == "pool_entry" && *dto.MemoryTargetValue >= *dto.MemoryThresholdValue {
				return errors.New("入池动作的内存目标值必须小于阈值")
			} else if dto.ThresholdTriggerAction == "pool_exit" && *dto.MemoryTargetValue <= *dto.MemoryThresholdValue {
				return errors.New("退池动作的内存目标值必须大于阈值")
			}
		}
	}

	if dto.CPUThresholdValue != nil && dto.MemoryThresholdValue != nil {
		if dto.ConditionLogic != "AND" && dto.ConditionLogic != "OR" {
			return errors.New("无效的条件逻辑，必须为AND或OR")
		}
	}

	if dto.DeviceCount <= 0 {
		return errors.New("设备数量必须大于0")
	}

	if dto.Status != "enabled" && dto.Status != "disabled" {
		return errors.New("无效的策略状态")
	}

	if len(dto.ClusterIDs) == 0 {
		return errors.New("至少需要关联一个集群")
	}

	if dto.DurationMinutes <= 0 {
		return errors.New("持续时间必须大于0")
	}

	if dto.CooldownMinutes <= 0 {
		return errors.New("冷却时间必须大于0")
	}

	// 验证资源类型（可选）
	if dto.ResourceTypes != "" {
		validTypes := map[string]bool{
			"total":    true,
			"compute":  true,
			"memory":   true,
			"storage":  true,
			"network":  true,
			"database": true,
			"gpu":      true,
		}

		types := strings.Split(dto.ResourceTypes, ",")
		for _, t := range types {
			trimmed := strings.TrimSpace(t)
			if !validTypes[trimmed] {
				return fmt.Errorf("无效的资源类型: %s", trimmed)
			}
		}
	}

	return nil
}

// CreateOrder 创建弹性伸缩订单
func (s *ElasticScalingService) CreateOrder(dto OrderDTO) (int64, error) {
	// 生成唯一订单号
	orderNumber := s.generateOrderNumber()

	// 创建订单模型
	order := portal.ElasticScalingOrder{
		OrderNumber: orderNumber,
		ClusterID:   dto.ClusterID,
		StrategyID:  dto.StrategyID,
		ActionType:  dto.ActionType,
		Status:      "pending", // 初始状态为待处理
		DeviceCount: dto.DeviceCount,
		DeviceID:    dto.DeviceID,
		CreatedBy:   dto.CreatedBy,
		StrategyTriggeredValue: dto.StrategyTriggeredValue, // 保存策略触发值
		StrategyThresholdValue: dto.StrategyThresholdValue, // 保存策略阈值
	}

	// 设置维护相关字段（如果是维护订单）
	if dto.ActionType == "maintenance_request" || dto.ActionType == "maintenance_uncordon" {
		if dto.MaintenanceStartTime != nil {
			navyStartTime := portal.NavyTime(*dto.MaintenanceStartTime)
			order.MaintenanceStartTime = &navyStartTime
		}
		if dto.MaintenanceEndTime != nil {
			navyEndTime := portal.NavyTime(*dto.MaintenanceEndTime)
			order.MaintenanceEndTime = &navyEndTime
		}
		order.ExternalTicketID = dto.ExternalTicketID
	}

	// 保存订单
	if err := s.db.Create(&order).Error; err != nil {
		return 0, err
	}

	// 如果提供了设备列表，创建关联
	if len(dto.Devices) > 0 {
		for _, deviceID := range dto.Devices {
			orderDevice := portal.OrderDevice{
				OrderID:  order.ID,
				DeviceID: deviceID,
				Status:   "pending",
			}
			if err := s.db.Create(&orderDevice).Error; err != nil {
				return order.ID, fmt.Errorf("订单创建成功，但关联设备时出错: %v", err)
			}
		}
	}

	return order.ID, nil
}

// GetOrder 获取订单详情
func (s *ElasticScalingService) GetOrder(id int64) (*OrderDetailDTO, error) {
	// 获取订单基本信息
	var order portal.ElasticScalingOrder
	if err := s.db.First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("订单不存在: %d", id)
		}
		return nil, err
	}

	// 获取集群名称
	var cluster portal.K8sCluster
	clusterName := "未知集群"
	if err := s.db.Select("clustername").First(&cluster, order.ClusterID).Error; err == nil {
		clusterName = cluster.ClusterName
	}

	// 获取策略名称（如果有关联策略）
	strategyName := ""
	if order.StrategyID != nil {
		var strategy portal.ElasticScalingStrategy
		if err := s.db.Select("name").First(&strategy, *order.StrategyID).Error; err == nil {
			strategyName = strategy.Name
		}
	}

	// 获取关联设备
	var orderDevices []portal.OrderDevice
	if err := s.db.Where("order_id = ?", id).Find(&orderDevices).Error; err != nil {
		return nil, err
	}

	// 准备设备ID列表
	deviceIDs := make([]int64, len(orderDevices))
	for i, od := range orderDevices {
		deviceIDs[i] = od.DeviceID
	}

	// 获取设备详情
	var devices []portal.Device
	if len(deviceIDs) > 0 {
		if err := s.db.Where("id IN ?", deviceIDs).Find(&devices).Error; err != nil {
			return nil, err
		}
	}

	// 获取特定设备详情（如果是维护订单）
	var deviceInfo *DeviceDTO
	if order.DeviceID != nil {
		var device portal.Device
		if err := s.db.First(&device, *order.DeviceID).Error; err == nil {
			deviceInfo = &DeviceDTO{
				ID:           device.ID,
				CICode:       device.CICode,
				IP:           device.IP,
				ArchType:     device.ArchType,
				CPU:          device.CPU,
				Memory:       device.Memory,
				Status:       device.Status,
				Role:         device.Role,
				Cluster:      device.Cluster,
				ClusterID:    device.ClusterID,
				IsSpecial:    device.IsSpecial,
				FeatureCount: device.FeatureCount,
			}
		}
	}

	// 构建DTO
	dto := &OrderDetailDTO{
		OrderDTO: OrderDTO{
			ID:                   order.ID,
			OrderNumber:          order.OrderNumber,
			ClusterID:            order.ClusterID,
			ClusterName:          clusterName,
			StrategyID:           order.StrategyID,
			StrategyName:         strategyName,
			ActionType:           order.ActionType,
			Status:               order.Status,
			DeviceCount:          order.DeviceCount,
			DeviceID:             order.DeviceID,
			DeviceInfo:           deviceInfo,
			Approver:             order.Approver,
			Executor:             order.Executor,
			CreatedBy:            order.CreatedBy,
			CreatedAt:            time.Time(order.CreatedAt),
			FailureReason:        order.FailureReason,
			MaintenanceStartTime: nil,
			MaintenanceEndTime:   nil,
			ExternalTicketID:     order.ExternalTicketID,
		},
		Devices: make([]DeviceDTO, len(devices)),
	}

	// Proper handling of maintenance time fields
	if order.MaintenanceStartTime != nil {
		startTime := time.Time(*order.MaintenanceStartTime)
		dto.MaintenanceStartTime = &startTime
	}

	if order.MaintenanceEndTime != nil {
		endTime := time.Time(*order.MaintenanceEndTime)
		dto.MaintenanceEndTime = &endTime
	}

	// Fix execution and completion time in GetOrder
	if order.ExecutionTime != nil {
		execTime := time.Time(*order.ExecutionTime)
		dto.ExecutionTime = &execTime
	}

	if order.CompletionTime != nil {
		complTime := time.Time(*order.CompletionTime)
		dto.CompletionTime = &complTime
	}

	// 转换设备列表
	deviceStatusMap := make(map[int64]string)
	for _, od := range orderDevices {
		deviceStatusMap[od.DeviceID] = od.Status
	}

	for i, device := range devices {
		deviceDTO := DeviceDTO{
			ID:           device.ID,
			CICode:       device.CICode,
			IP:           device.IP,
			ArchType:     device.ArchType,
			CPU:          device.CPU,
			Memory:       device.Memory,
			Status:       device.Status,
			Role:         device.Role,
			Cluster:      device.Cluster,
			ClusterID:    device.ClusterID,
			IsSpecial:    device.IsSpecial,
			FeatureCount: device.FeatureCount,
		}

		// 添加设备在订单中的状态
		if status, ok := deviceStatusMap[device.ID]; ok {
			deviceDTO.OrderStatus = status
		}

		dto.Devices[i] = deviceDTO
	}

	return dto, nil
}

// ListOrders 获取订单列表
func (s *ElasticScalingService) ListOrders(clusterID int64, strategyID int64, actionType string, status string, page, pageSize int) ([]OrderListItemDTO, int64, error) {
	var total int64
	query := s.db.Model(&portal.ElasticScalingOrder{})

	// 应用过滤条件
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	if strategyID > 0 {
		query = query.Where("strategy_id = ?", strategyID)
	}
	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var orders []portal.ElasticScalingOrder
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	// 准备结果
	result := make([]OrderListItemDTO, len(orders))
	for i, order := range orders {
		// 获取集群名称
		var cluster portal.K8sCluster
		clusterName := "未知集群"
		if err := s.db.Select("clustername").First(&cluster, order.ClusterID).Error; err == nil {
			clusterName = cluster.ClusterName
		}

		// 获取策略名称（如果有关联策略）
		var strategyName string
		if order.StrategyID != nil {
			var strategy portal.ElasticScalingStrategy
			if err := s.db.Select("name").First(&strategy, *order.StrategyID).Error; err == nil {
				strategyName = strategy.Name
			}
		}

		result[i] = OrderListItemDTO{
			ID:           order.ID,
			OrderNumber:  order.OrderNumber,
			ClusterID:    order.ClusterID,
			ClusterName:  clusterName,
			StrategyID:   order.StrategyID,
			StrategyName: strategyName,
			ActionType:   order.ActionType,
			Status:       order.Status,
			DeviceCount:  order.DeviceCount,
			CreatedBy:    order.CreatedBy,
			CreatedAt:    time.Time(order.CreatedAt),
		}
	}

	return result, total, nil
}

// UpdateOrderStatus 更新订单状态
func (s *ElasticScalingService) UpdateOrderStatus(id int64, status string, executor string, reason string) error {
	// 验证状态
	validStatuses := map[string]bool{
		"pending":                   true,
		"processing":                true,
		"completed":                 true,
		"failed":                    true,
		"cancelled":                 true,
		"pending_confirmation":      true,
		"scheduled_for_maintenance": true,
		"maintenance_in_progress":   true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("无效的订单状态: %s", status)
	}

	var order portal.ElasticScalingOrder
	if err := s.db.First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("订单不存在: %d", id)
		}
		return err
	}

	// 更新订单状态
	updates := map[string]interface{}{
		"status": status,
	}

	// 根据状态设置相关字段
	if status == "processing" {
		now := time.Now()
		updates["execution_time"] = now
		updates["executor"] = executor
	} else if status == "completed" {
		now := time.Now()
		updates["completion_time"] = now
	} else if status == "failed" {
		updates["failure_reason"] = reason
	}

	err := s.db.Model(&order).Updates(updates).Error
	if err != nil {
		return err
	}

	// 如果订单状态更新为 processing 或 completed，并且是由策略生成的，则记录策略执行历史
	if (status == "processing" || status == "completed") && order.StrategyID != nil {
		s.logger.Info("Order status updated, recording strategy execution history",
			zap.Int64("orderID", order.ID),
			zap.String("newStatus", status),
			zap.Int64p("strategyID", order.StrategyID))

		var historyResult string
		var executionTimeForHistory portal.NavyTime

		if status == "processing" && order.ExecutionTime != nil {
			historyResult = "order_processing_started"
			executionTimeForHistory = *order.ExecutionTime
		} else if status == "completed" && order.CompletionTime != nil {
			historyResult = "order_completed"
			executionTimeForHistory = *order.CompletionTime
		} else {
			// 如果时间戳缺失，则使用当前时间，但这不理想
			s.logger.Warn("Execution/Completion time is nil for order, using current time for history",
				zap.Int64("orderID", order.ID),
				zap.String("status", status))
			executionTimeForHistory = portal.NavyTime(time.Now())
			if status == "processing" {
				historyResult = "order_processing_started_no_exec_time"
			} else {
				historyResult = "order_completed_no_compl_time"
			}
		}
		
		// 从订单中获取保存的触发值和阈值
		reasonForHistory := fmt.Sprintf("Order %s by strategy %d.", status, *order.StrategyID)
		if order.FailureReason != "" && status == "failed" { // 虽然这里是 processing/completed, 但以防万一
			reasonForHistory = order.FailureReason
		}


		// 调用 recordStrategyExecution
		// 注意：recordStrategyExecution 内部的 ExecutionTime 将被我们这里提供的 executionTimeForHistory 覆盖
		// triggeredValue 和 thresholdValue 将从 order 对象中获取
		errRecord := s.recordStrategyExecution(
			*order.StrategyID,
			historyResult,
			&order.ID,
			reasonForHistory,
			order.StrategyTriggeredValue, // 新增参数
			order.StrategyThresholdValue, // 新增参数
			&executionTimeForHistory,     // 新增参数，传递实际的执行或完成时间
		)
		if errRecord != nil {
			s.logger.Error("Failed to record strategy execution history after order update",
				zap.Int64("orderID", order.ID),
				zap.Int64p("strategyID", order.StrategyID),
				zap.Error(errRecord))
			// 不返回错误，因为主操作（更新订单状态）已成功
		}
	}

	return nil
}

// GetOrderDevices 获取订单关联的设备
func (s *ElasticScalingService) GetOrderDevices(orderID int64) ([]DeviceDTO, error) {
	var orderDevices []portal.OrderDevice
	if err := s.db.Where("order_id = ?", orderID).Find(&orderDevices).Error; err != nil {
		return nil, err
	}

	if len(orderDevices) == 0 {
		return []DeviceDTO{}, nil
	}

	// 提取设备ID
	deviceIDs := make([]int64, len(orderDevices))
	deviceStatusMap := make(map[int64]string)
	for i, od := range orderDevices {
		deviceIDs[i] = od.DeviceID
		deviceStatusMap[od.DeviceID] = od.Status
	}

	// 获取设备详情
	var devices []portal.Device
	if err := s.db.Where("id IN ?", deviceIDs).Find(&devices).Error; err != nil {
		return nil, err
	}

	// 构建结果
	result := make([]DeviceDTO, len(devices))
	for i, device := range devices {
		result[i] = DeviceDTO{
			ID:           device.ID,
			CICode:       device.CICode,
			IP:           device.IP,
			ArchType:     device.ArchType,
			CPU:          device.CPU,
			Memory:       device.Memory,
			Status:       device.Status,
			Role:         device.Role,
			Cluster:      device.Cluster,
			ClusterID:    device.ClusterID,
			IsSpecial:    device.IsSpecial,
			FeatureCount: device.FeatureCount,
			OrderStatus:  deviceStatusMap[device.ID],
		}
	}

	return result, nil
}

// UpdateOrderDeviceStatus 更新订单中设备的状态
func (s *ElasticScalingService) UpdateOrderDeviceStatus(orderID int64, deviceID int64, status string) error {
	// 验证状态
	validStatuses := map[string]bool{
		"pending":    true,
		"processing": true,
		"completed":  true,
		"failed":     true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("无效的设备状态: %s", status)
	}

	var orderDevice portal.OrderDevice
	result := s.db.Where("order_id = ? AND device_id = ?", orderID, deviceID).First(&orderDevice)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("订单中不存在该设备")
		}
		return result.Error
	}

	orderDevice.Status = status
	return s.db.Save(&orderDevice).Error
}

// generateOrderNumber 生成唯一订单号
func (s *ElasticScalingService) generateOrderNumber() string {
	// 生成格式为 "ESO" + 年月日 + 6位随机数的订单号
	now := time.Now()
	dateStr := now.Format("20060102")
	randomStr := fmt.Sprintf("%06d", rand.Intn(1000000))
	return "ESO" + dateStr + randomStr
}

// GetDashboardStats 获取工作台统计数据
func (s *ElasticScalingService) GetDashboardStats() (*DashboardStatsDTO, error) {
	stats := &DashboardStatsDTO{}

	// 获取策略总数
	var strategyCount int64
	if err := s.db.Model(&portal.ElasticScalingStrategy{}).Count(&strategyCount).Error; err != nil {
		return nil, err
	}
	stats.StrategyCount = int(strategyCount)

	// 获取已启用策略数
	var enabledStrategyCount int64
	if err := s.db.Model(&portal.ElasticScalingStrategy{}).Where("status = ?", "enabled").Count(&enabledStrategyCount).Error; err != nil {
		return nil, err
	}
	stats.EnabledStrategyCount = int(enabledStrategyCount)

	// 获取今日已触发策略数（今日执行历史中有order_created结果的不同策略数）
	today := time.Now().Format("2006-01-02")
	var triggeredStrategyIDs []int64
	if err := s.db.Model(&portal.StrategyExecutionHistory{}).
		Select("DISTINCT strategy_id").
		Where("DATE(execution_time) = ? AND result = ?", today, "order_created").
		Pluck("strategy_id", &triggeredStrategyIDs).Error; err != nil {
		return nil, err
	}
	stats.TriggeredTodayCount = len(triggeredStrategyIDs)

	// 获取集群总数
	var clusterCount int64
	if err := s.db.Model(&portal.K8sCluster{}).Count(&clusterCount).Error; err != nil {
		return nil, err
	}
	stats.ClusterCount = int(clusterCount)

	// 获取异常集群数（根据资源快照中的异常状态判断）
	// 这里假设MaxCpuUsageRatio > 80 或 MaxMemoryUsageRatio > 80 视为异常
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	var abnormalClusterCount int64
	if err := s.db.Model(&portal.ResourceSnapshot{}).
		Where("DATE(created_at) >= ? AND (max_cpu > 80 OR max_memory > 80)", yesterday).
		Distinct("cluster_id").
		Count(&abnormalClusterCount).Error; err != nil {
		return nil, err
	}
	stats.AbnormalClusterCount = int(abnormalClusterCount)

	// 获取待处理订单数
	var pendingOrderCount int64
	if err := s.db.Model(&portal.ElasticScalingOrder{}).Where("status = ?", "pending").Count(&pendingOrderCount).Error; err != nil {
		return nil, err
	}
	stats.PendingOrderCount = int(pendingOrderCount)

	// 获取设备总数
	var deviceCount int64
	if err := s.db.Model(&portal.Device{}).Count(&deviceCount).Error; err != nil {
		return nil, err
	}
	stats.DeviceCount = int(deviceCount)

	// 获取可用设备数（没有分配到集群的设备）
	var availableDeviceCount int64
	if err := s.db.Model(&portal.Device{}).Where("cluster_id = 0 OR cluster_id IS NULL").Count(&availableDeviceCount).Error; err != nil {
		return nil, err
	}
	stats.AvailableDeviceCount = int(availableDeviceCount)

	// 获取池内设备数（已分配到集群的设备）
	var inPoolDeviceCount int64
	if err := s.db.Model(&portal.Device{}).Where("cluster_id > 0").Count(&inPoolDeviceCount).Error; err != nil {
		return nil, err
	}
	stats.InPoolDeviceCount = int(inPoolDeviceCount)

	return stats, nil
}

// GetResourceAllocationTrend 获取资源分配趋势
func (s *ElasticScalingService) GetResourceAllocationTrend(clusterID int64, timeRange string, resourceTypes string) (*ResourceAllocationTrendDTO, error) {
	if clusterID <= 0 {
		return nil, errors.New("无效的集群ID")
	}

	// 确定查询时间范围
	var startTime time.Time
	now := time.Now()

	switch timeRange {
	case "24h":
		startTime = now.Add(-24 * time.Hour)
	case "7d":
		startTime = now.Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = now.Add(-30 * 24 * time.Hour)
	default:
		startTime = now.Add(-24 * time.Hour) // 默认24小时
	}

	// 解析资源类型
	resTypes := []string{"total"} // 默认使用total
	if resourceTypes != "" {
		resTypes = strings.Split(resourceTypes, ",")
		// 清理每个资源类型字符串
		for i, rt := range resTypes {
			resTypes[i] = strings.TrimSpace(rt)
		}
	}

	// 准备结果数据结构
	result := &ResourceAllocationTrendDTO{
		Timestamps:         []time.Time{},
		CPUUsageRatio:      []float64{},
		CPUAllocationRatio: []float64{},
		MemUsageRatio:      []float64{},
		MemAllocationRatio: []float64{},
		ResourceTypes:      resTypes,
		ResourceTypeData:   make(map[string]*ResourceTypeDataDTO),
	}

	// 对每个资源类型获取数据
	for _, resourceType := range resTypes {
		// 获取指定集群的资源快照数据
		var snapshots []portal.ResourceSnapshot
		if err := s.db.Where("cluster_id = ? AND resource_type = ? AND created_at >= ?", clusterID, resourceType, startTime).
			Order("created_at ASC").
			Find(&snapshots).Error; err != nil {
			return nil, err
		}

		// 如果没有数据，继续下一个资源类型
		if len(snapshots) == 0 {
			continue
		}

		// 如果是第一个有数据的资源类型，初始化主要时间轴
		if len(result.Timestamps) == 0 {
			result.Timestamps = make([]time.Time, len(snapshots))
			result.CPUUsageRatio = make([]float64, len(snapshots))
			result.CPUAllocationRatio = make([]float64, len(snapshots))
			result.MemUsageRatio = make([]float64, len(snapshots))
			result.MemAllocationRatio = make([]float64, len(snapshots))

			for i, snapshot := range snapshots {
				result.Timestamps[i] = time.Time(snapshot.CreatedAt)

				// CPU使用率 = MaxCpuUsageRatio（资源快照中已有）
				result.CPUUsageRatio[i] = snapshot.MaxCpuUsageRatio

				// CPU分配率 = CpuRequest / CpuCapacity * 100（如果容量为0，设为0避免除零错误）
				if snapshot.CpuCapacity > 0 {
					result.CPUAllocationRatio[i] = snapshot.CpuRequest / snapshot.CpuCapacity * 100
				} else {
					result.CPUAllocationRatio[i] = 0
				}

				// 内存使用率 = MaxMemoryUsageRatio（资源快照中已有）
				result.MemUsageRatio[i] = snapshot.MaxMemoryUsageRatio

				// 内存分配率 = MemRequest / MemoryCapacity * 100（如果容量为0，设为0避免除零错误）
				if snapshot.MemoryCapacity > 0 {
					result.MemAllocationRatio[i] = snapshot.MemRequest / snapshot.MemoryCapacity * 100
				} else {
					result.MemAllocationRatio[i] = 0
				}
			}
		}

		// 为每个资源类型创建单独的数据集
		typeData := &ResourceTypeDataDTO{
			Timestamps:         make([]time.Time, len(snapshots)),
			CPUUsageRatio:      make([]float64, len(snapshots)),
			CPUAllocationRatio: make([]float64, len(snapshots)),
			MemUsageRatio:      make([]float64, len(snapshots)),
			MemAllocationRatio: make([]float64, len(snapshots)),
		}

		for i, snapshot := range snapshots {
			typeData.Timestamps[i] = time.Time(snapshot.CreatedAt)

			// CPU使用率
			typeData.CPUUsageRatio[i] = snapshot.MaxCpuUsageRatio

			// CPU分配率
			if snapshot.CpuCapacity > 0 {
				typeData.CPUAllocationRatio[i] = snapshot.CpuRequest / snapshot.CpuCapacity * 100
			}

			// 内存使用率
			typeData.MemUsageRatio[i] = snapshot.MaxMemoryUsageRatio

			// 内存分配率
			if snapshot.MemoryCapacity > 0 {
				typeData.MemAllocationRatio[i] = snapshot.MemRequest / snapshot.MemoryCapacity * 100
			}
		}

		// 将该资源类型的数据添加到结果中
		result.ResourceTypeData[resourceType] = typeData
	}

	return result, nil
}

// GetOrderStats 获取订单统计
func (s *ElasticScalingService) GetOrderStats(timeRange string) (*OrderStatsDTO, error) {
	// 确定查询时间范围
	var startTime time.Time
	now := time.Now()

	switch timeRange {
	case "7d":
		startTime = now.Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = now.Add(-30 * 24 * time.Hour)
	case "90d":
		startTime = now.Add(-90 * 24 * time.Hour)
	default:
		startTime = now.Add(-30 * 24 * time.Hour) // 默认30天
	}

	stats := &OrderStatsDTO{}

	// 获取总订单数
	var totalCount int64
	if err := s.db.Model(&portal.ElasticScalingOrder{}).
		Where("created_at >= ?", startTime).
		Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats.TotalCount = int(totalCount)

	// 获取各状态订单数
	var pendingCount int64
	if err := s.db.Model(&portal.ElasticScalingOrder{}).
		Where("created_at >= ? AND status = ?", startTime, "pending").
		Count(&pendingCount).Error; err != nil {
		return nil, err
	}
	stats.PendingCount = int(pendingCount)

	var processingCount int64
	if err := s.db.Model(&portal.ElasticScalingOrder{}).
		Where("created_at >= ? AND status = ?", startTime, "processing").
		Count(&processingCount).Error; err != nil {
		return nil, err
	}
	stats.ProcessingCount = int(processingCount)

	var completedCount int64
	if err := s.db.Model(&portal.ElasticScalingOrder{}).
		Where("created_at >= ? AND status = ?", startTime, "completed").
		Count(&completedCount).Error; err != nil {
		return nil, err
	}
	stats.CompletedCount = int(completedCount)

	var failedCount int64
	if err := s.db.Model(&portal.ElasticScalingOrder{}).
		Where("created_at >= ? AND status = ?", startTime, "failed").
		Count(&failedCount).Error; err != nil {
		return nil, err
	}
	stats.FailedCount = int(failedCount)

	var cancelledCount int64
	if err := s.db.Model(&portal.ElasticScalingOrder{}).
		Where("created_at >= ? AND status = ?", startTime, "cancelled").
		Count(&cancelledCount).Error; err != nil {
		return nil, err
	}
	stats.CancelledCount = int(cancelledCount)

	return stats, nil
}

// EvaluateStrategies 评估策略并可能创建订单
// 该函数通常由定时任务调用，评估所有启用的策略
func (s *ElasticScalingService) EvaluateStrategies() error {
	s.logger.Info("Starting to evaluate all enabled strategies")
	// 获取所有启用的策略
	var strategies []portal.ElasticScalingStrategy
	if err := s.db.Where("status = ?", "enabled").Find(&strategies).Error; err != nil {
		s.logger.Error("Failed to fetch enabled strategies", zap.Error(err))
		return err
	}
	s.logger.Info("Fetched enabled strategies", zap.Int("count", len(strategies)))

	for _, strategy := range strategies {
		s.logger.Info("Evaluating strategy in loop", zap.Int64("strategyID", strategy.ID), zap.String("strategyName", strategy.Name))
		// 使用独立锁键避免策略评估相互阻塞
		if err := s.evaluateStrategy(&strategy); err != nil {
			// 记录错误但继续处理其他策略
			s.logger.Error("Error evaluating strategy",
				zap.Int64("strategyID", strategy.ID),
				zap.String("strategyName", strategy.Name),
				zap.Error(err))
		}
	}

	return nil
}

// evaluateStrategy 评估单个策略
func (s *ElasticScalingService) evaluateStrategy(strategy *portal.ElasticScalingStrategy) error {
	s.logger.Info("Starting single strategy evaluation",
		zap.Int64("strategyID", strategy.ID),
		zap.String("strategyName", strategy.Name))

	// 生成策略特定的锁键
	lockKey := fmt.Sprintf("elastic_scaling:strategy:%d:lock", strategy.ID)

	// 使用Redis锁确保只有一个实例评估该策略
	// 使用注入的 redisHandler
	// Note: The original redis.Handler.Expire sets a default, not for a specific key.
	// We will rely on AcquireLock's expiry for the lock itself.
	// s.redisHandler.Expire(lockKey, 30 * time.Second) // Removed key parameter

	// Generate unique lock value
	lockValue := fmt.Sprintf("eval:%d:%d", strategy.ID, time.Now().UnixNano())

	// 尝试获取锁
	success, err := s.redisHandler.AcquireLock(lockKey, lockValue, 30*time.Second)
	s.logger.Debug("Redis lock acquisition attempt for strategy evaluation",
		zap.Int64("strategyID", strategy.ID),
		zap.String("lockKey", lockKey),
		zap.Bool("success", success),
		zap.Error(err))

	if err != nil {
		s.logger.Error("Failed to acquire Redis lock for strategy evaluation",
			zap.Int64("strategyID", strategy.ID),
			zap.String("lockKey", lockKey),
			zap.Error(err))
		return fmt.Errorf("获取策略锁失败: %v", err)
	}

	if !success {
		// 其他实例正在评估此策略，记录后退出
		s.logger.Info("Strategy lock not acquired, possibly already being evaluated by another instance",
			zap.Int64("strategyID", strategy.ID),
			zap.String("lockKey", lockKey))
		return nil
	}

	// 确保在函数返回时释放锁
	defer s.redisHandler.Delete(lockKey)

	// 检查冷却期
	// 获取策略最近一次成功执行的历史记录
	var latestHistory portal.StrategyExecutionHistory
	result := s.db.Where("strategy_id = ? AND result = ?", strategy.ID, "order_created").
		Order("execution_time DESC").
		First(&latestHistory)

	if result.Error == nil {
		// 如果存在最近执行记录，检查是否在冷却期内
		var cooldownEndTime time.Time
		latestHistory.ExecutionTime.Scan(&cooldownEndTime)
		cooldownEndTime = cooldownEndTime.Add(time.Duration(strategy.CooldownMinutes) * time.Minute)
		if time.Now().Before(cooldownEndTime) {
			s.logger.Info("Strategy skipped due to cooldown period",
				zap.Int64("strategyID", strategy.ID),
				zap.String("strategyName", strategy.Name),
				zap.Time("cooldownEndTime", cooldownEndTime))
			return nil
		}
	}

	// 获取关联集群
	var associations []portal.StrategyClusterAssociation
	if err := s.db.Where("strategy_id = ?", strategy.ID).Find(&associations).Error; err != nil {
		s.logger.Error("Failed to fetch associations for strategy", zap.Int64("strategyID", strategy.ID), zap.Error(err))
		return err
	}
	s.logger.Info("Fetched associations for strategy", zap.Int64("strategyID", strategy.ID), zap.Int("count", len(associations)))
	if len(associations) == 0 {
		s.logger.Info("No associations found for strategy, skipping evaluation for this strategy.", zap.Int64("strategyID", strategy.ID))
		// Optionally record a specific history type for no associations if desired
		// s.recordStrategyExecution(strategy.ID, "skipped", nil, "策略没有关联任何集群")
		return nil // Nothing to evaluate if no clusters are associated
	}

	// 解析策略的资源类型
	var resourceTypes []string
	if strategy.ResourceTypes != "" {
		resourceTypes = strings.Split(strategy.ResourceTypes, ",")
		// 清理每个资源类型字符串
		for i, rt := range resourceTypes {
			resourceTypes[i] = strings.TrimSpace(rt)
		}
	} else {
		// 如果未指定资源类型，默认使用total
		resourceTypes = []string{"total"}
	}

	// 对每个关联集群评估策略条件
	for _, assoc := range associations {
		clusterId := assoc.ClusterID
		s.logger.Info("Evaluating strategy for cluster", zap.Int64("strategyID", strategy.ID), zap.Int64("clusterID", clusterId))

		// 对每个资源类型评估条件
		for _, resourceType := range resourceTypes {
			s.logger.Info("Evaluating for resource type",
				zap.Int64("strategyID", strategy.ID),
				zap.Int64("clusterID", clusterId),
				zap.String("resourceType", resourceType))

			// 获取最新的资源快照
			query := s.db.Where("cluster_id = ? AND resource_type = ?", clusterId, resourceType)
			// 只有当策略的 ResourceTypes 不完全是 "total" 时才考虑资源类型过滤
			applyResourceTypeFilter := true
			if resourceType == "total" {
				applyResourceTypeFilter = false
				s.logger.Debug("ResourceType is 'total', not applying additional filtering for snapshot query.",
					zap.Int64("strategyID", strategy.ID), zap.String("resourceType", resourceType))
			}

			if applyResourceTypeFilter {
				// 将resourceType作为筛选条件，例如如果resourceType是"compute"，则筛选资源池也是"compute"的快照
				query = query.Where("resource_pool = ?", resourceType)
				s.logger.Debug("Applying resource_type filter for snapshot query",
					zap.Int64("strategyID", strategy.ID), zap.String("resourceType", resourceType))
			}

			var snapshot portal.ResourceSnapshot
			queryString := query.Session(&gorm.Session{DryRun: true}).Order("created_at DESC").First(&snapshot).Statement.SQL.String()
			s.logger.Debug("Snapshot query string", zap.String("sql", queryString))

			if err := query.Order("created_at DESC").First(&snapshot).Error; err != nil {
				// Determine the effective resourcePool string for logging, considering the default.
				loggedResourcePool := resourceType
				if resourceType == "total" {
					loggedResourcePool = "[NONE_APPLIED_FOR_TOTAL]"
				}

				logMsg := fmt.Sprintf("集群 %d 没有资源类型 %s 资源池 %s 的快照数据",
					clusterId, resourceType, loggedResourcePool)

				if errors.Is(err, gorm.ErrRecordNotFound) {
					s.logger.Info("No resource snapshot found, skipping strategy evaluation for this cluster/resourceType",
						zap.Int64("strategyID", strategy.ID),
						zap.String("strategyName", strategy.Name),
						zap.Int64("clusterID", clusterId),
						zap.String("resourceType", resourceType),
						zap.String("resourcePoolForQuery", loggedResourcePool),
						// zap.String("actualQuery", queryString), // queryString might be too verbose for regular info log
						zap.Error(err),
						zap.String("reason", logMsg))
				} else {
					s.logger.Error("Failed to fetch resource snapshot, skipping strategy evaluation for this cluster/resourceType",
						zap.Int64("strategyID", strategy.ID),
						zap.String("strategyName", strategy.Name),
						zap.Int64("clusterID", clusterId),
						zap.String("resourceType", resourceType),
						zap.String("resourcePoolForQuery", loggedResourcePool),
						// zap.String("actualQuery", queryString),
						zap.Error(err),
						zap.String("reason", logMsg))
					logMsg = fmt.Sprintf("查询集群 %d 资源类型 %s 资源池 %s 的快照数据失败: %v", // logMsg is still prepared for potential other uses or if needed for a more detailed error.
						clusterId, resourceType, loggedResourcePool, err)
				}
				// s.recordStrategyExecution(strategy.ID, "skipped", nil, logMsg) // Removed: Skipped executions are now only logged.
				continue
			}

			// ... existing monitoring and condition checking code ...
		}
	}

	return nil
}

// recordStrategyExecution 记录策略执行历史
// 添加了 triggeredValue, thresholdValue 和 specificExecutionTime 参数
func (s *ElasticScalingService) recordStrategyExecution(
	strategyID int64,
	result string,
	orderID *int64,
	reason string,
	triggeredValue string, // 新增：从订单传递过来的触发值
	thresholdValue string, // 新增：从订单传递过来的阈值
	specificExecutionTime *portal.NavyTime, // 新增：特定的执行时间，如订单的执行/完成时间
) error {
	s.logger.Info("Recording strategy execution history",
		zap.Int64("strategyID", strategyID),
		zap.String("result", result),
		zap.Any("orderID", orderID),
		zap.String("reason", reason),
		zap.String("triggeredValue", triggeredValue),
		zap.String("thresholdValue", thresholdValue),
		zap.Time("specificExecutionTime", time.Time(*specificExecutionTime)),
	)

	execTime := portal.NavyTime(time.Now()) // 默认使用当前时间
	if specificExecutionTime != nil {
		execTime = *specificExecutionTime // 如果提供了特定时间，则使用它
	}

	history := portal.StrategyExecutionHistory{
		StrategyID:     strategyID,
		ExecutionTime:  execTime,
		TriggeredValue: triggeredValue, // 使用传递过来的值
		ThresholdValue: thresholdValue, // 使用传递过来的值
		Result:         result,
		OrderID:        orderID,
		Reason:         reason,
	}

	err := s.db.Create(&history).Error
	if err != nil {
		s.logger.Error("Failed to create strategy execution history entry in DB",
			zap.Int64("strategyID", strategyID),
			zap.String("result", result),
			zap.Error(err))
	}
	return err
}

// safePercentage calculates percentage safely, returning 0 if denominator is zero.
func safePercentage(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator * 100
}
