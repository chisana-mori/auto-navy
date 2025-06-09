package service

import (
	"errors"
	"fmt"
	"navy-ng/models/portal"
	"time"

	"gorm.io/gorm"
)

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

		ResourceTypes:   dto.ResourceTypes,
		Status:          dto.Status,
		CreatedBy:       dto.CreatedBy,
		DurationMinutes: dto.DurationMinutes,
		CooldownMinutes: dto.CooldownMinutes,
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

	// 获取相关订单（使用新的订单模型）
	var orders []portal.Order
	if err := s.db.Preload("ElasticScalingDetail").
		Joins("JOIN ng_elastic_scaling_order_details ON orders.id = ng_elastic_scaling_order_details.order_id").
		Where("ng_elastic_scaling_order_details.strategy_id = ?", id).
		Order("orders.created_at DESC").Limit(10).Find(&orders).Error; err != nil {
		return nil, err
	}

	// 转换为DTO
	dto := &StrategyDetailDTO{
		StrategyDTO: StrategyDTO{
			ID:                     strategy.ID,
			Name:                   strategy.Name,
			Description:            strategy.Description,
			ThresholdTriggerAction: strategy.ThresholdTriggerAction,

			ResourceTypes:   strategy.ResourceTypes,
			Status:          strategy.Status,
			CreatedBy:       strategy.CreatedBy,
			CreatedAt:       time.Time(strategy.CreatedAt),
			UpdatedAt:       time.Time(strategy.UpdatedAt),
			DurationMinutes: strategy.DurationMinutes,
			CooldownMinutes: strategy.CooldownMinutes,
			ClusterIDs:      clusterIDs,
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
			ClusterID:      h.ClusterID,
			ResourceType:   h.ResourceType,
			ExecutionTime:  time.Time(h.ExecutionTime),
			TriggeredValue: h.TriggeredValue,
			ThresholdValue: h.ThresholdValue,
			Result:         h.Result,
			OrderID:        h.OrderID,
			Reason:         h.Reason,
		}
	}

	// 转换相关订单
	for i, order := range orders {
		if order.ElasticScalingDetail == nil {
			continue
		}

		detail := order.ElasticScalingDetail

		// 获取集群名称
		var cluster portal.K8sCluster
		clusterName := "未知集群"
		if err := s.db.Select("clustername").First(&cluster, detail.ClusterID).Error; err == nil {
			clusterName = cluster.ClusterName
		}

		dto.RelatedOrders[i] = OrderListItemDTO{
			ID:           order.ID,
			OrderNumber:  order.OrderNumber,
			Name:         order.Name,        // 订单名称
			Description:  order.Description, // 订单描述
			ClusterID:    detail.ClusterID,
			ClusterName:  clusterName,
			StrategyID:   detail.StrategyID,
			StrategyName: strategy.Name,
			ActionType:   detail.ActionType,
			Status:       string(order.Status),
			DeviceCount:  detail.DeviceCount,
			CreatedBy:    order.CreatedBy,
			CreatedAt:    time.Time(order.CreatedAt),
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

			Status:            strategy.Status,
			CreatedAt:         time.Time(strategy.CreatedAt),
			UpdatedAt:         time.Time(strategy.UpdatedAt),
			Clusters:          clusterNames,
			CPUTargetValue:    &strategy.CPUTargetValue,
			MemoryTargetValue: &strategy.MemoryTargetValue,
			DurationMinutes:   strategy.DurationMinutes,
			CooldownMinutes:   strategy.CooldownMinutes,
			ResourceTypes:     strategy.ResourceTypes,
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
	// 检查是否有关联的执行中订单（使用新的订单模型）
	var count int64
	if err := s.db.Table("orders").
		Joins("JOIN ng_elastic_scaling_order_details ON orders.id = ng_elastic_scaling_order_details.order_id").
		Where("ng_elastic_scaling_order_details.strategy_id = ? AND orders.status IN ('pending', 'processing')", id).
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

	if dto.ThresholdTriggerAction != TriggerActionPoolEntry && dto.ThresholdTriggerAction != TriggerActionPoolExit {
		return errors.New("无效的触发动作类型")
	}

	if dto.CPUThresholdValue == nil && dto.MemoryThresholdValue == nil {
		return errors.New("至少需要设置CPU或内存阈值")
	}

	if dto.CPUThresholdValue != nil {
		if *dto.CPUThresholdValue <= 0 || *dto.CPUThresholdValue > 100 {
			return errors.New("CPU阈值必须在0-100之间")
		}
		if *dto.CPUThresholdType != ThresholdTypeUsage && *dto.CPUThresholdType != ThresholdTypeAllocated {
			return errors.New("无效的CPU阈值类型")
		}

		// 验证CPU目标值
		if dto.CPUTargetValue != nil {
			if *dto.CPUTargetValue <= 0 || *dto.CPUTargetValue > 100 {
				return errors.New("CPU目标值必须在0-100之间")
			}

			// 根据动作类型验证目标值与阈值的关系
			if dto.ThresholdTriggerAction == TriggerActionPoolEntry && *dto.CPUTargetValue >= *dto.CPUThresholdValue {
				return errors.New("入池动作的CPU目标值必须小于阈值")
			} else if dto.ThresholdTriggerAction == TriggerActionPoolExit && *dto.CPUTargetValue <= *dto.CPUThresholdValue {
				return errors.New("退池动作的CPU目标值必须大于阈值")
			}
		}
	}

	if dto.MemoryThresholdValue != nil {
		if *dto.MemoryThresholdValue <= 0 || *dto.MemoryThresholdValue > 100 {
			return errors.New("内存阈值必须在0-100之间")
		}
		if *dto.MemoryThresholdType != ThresholdTypeUsage && *dto.MemoryThresholdType != ThresholdTypeAllocated {
			return errors.New("无效的内存阈值类型")
		}

		// 验证内存目标值
		if dto.MemoryTargetValue != nil {
			if *dto.MemoryTargetValue <= 0 || *dto.MemoryTargetValue > 100 {
				return errors.New("内存目标值必须在0-100之间")
			}

			// 根据动作类型验证目标值与阈值的关系
			if dto.ThresholdTriggerAction == TriggerActionPoolEntry && *dto.MemoryTargetValue >= *dto.MemoryThresholdValue {
				return errors.New("入池动作的内存目标值必须小于阈值")
			} else if dto.ThresholdTriggerAction == TriggerActionPoolExit && *dto.MemoryTargetValue <= *dto.MemoryThresholdValue {
				return errors.New("退池动作的内存目标值必须大于阈值")
			}
		}
	}

	if dto.CPUThresholdValue != nil && dto.MemoryThresholdValue != nil {
		if dto.ConditionLogic != ConditionLogicAnd && dto.ConditionLogic != ConditionLogicOr {
			return errors.New("无效的条件逻辑，必须为AND或OR")
		}
	}

	// 移除设备数量验证，设备数量将根据实际资源需求动态计算

	if dto.Status != StrategyStatusEnabled && dto.Status != StrategyStatusDisabled {
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

	return nil
}
