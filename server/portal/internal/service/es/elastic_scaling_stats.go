package es

import (
	"errors"
	"fmt"
	"navy-ng/models/portal"
	"strings"
	"time"

	"github.com/jinzhu/now"
	_ "gorm.io/gorm" // GORM is used via s.db
)

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

	var triggeredStrategyIDs []int64
	if err := s.db.Model(&portal.StrategyExecutionHistory{}).
		Select("DISTINCT strategy_id").
		Where("execution_time between ? and ?", now.BeginningOfDay(), now.EndOfDay()).
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

	// 获取生成待处理状态的扩缩容订单的集群个数，按集群名去重
	var abnormalClusterCount int64
	if err := s.db.Table("ng_orders o").
		Joins("JOIN ng_elastic_scaling_order_details esd ON o.id = esd.order_id").
		Joins("JOIN k8s_cluster c ON esd.cluster_id = c.id").
		Where("o.type = ? AND o.status = ?", portal.OrderTypeElasticScaling, portal.OrderStatusPending).
		Distinct("c.clustername").
		Count(&abnormalClusterCount).Error; err != nil {
		return nil, err
	}
	stats.AbnormalClusterCount = int(abnormalClusterCount)

	// 获取待处理订单数（使用新的订单表）
	var pendingOrderCount int64
	if err := s.db.Model(&portal.Order{}).
		Where("type = ? AND status = ?", portal.OrderTypeElasticScaling, portal.OrderStatusPending).
		Count(&pendingOrderCount).Error; err != nil {
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

	// 计算目标巡检资源池数和已巡检资源池数
	var enabledStrategies []portal.ElasticScalingStrategy
	if err := s.db.Where("status = ?", portal.StrategyStatusEnabled).Find(&enabledStrategies).Error; err != nil {
		return nil, err
	}

	targetResourcePools := make(map[string]struct{})
	inspectedResourcePools := make(map[string]struct{})

	for _, strategy := range enabledStrategies {
		// 获取策略关联的集群
		var strategyClusters []portal.StrategyClusterAssociation
		if err := s.db.Where("strategy_id = ?", strategy.ID).Find(&strategyClusters).Error; err != nil {
			return nil, err
		}

		strategyResourceTypes := strings.Split(strategy.ResourceTypes, ",")

		for _, sc := range strategyClusters {
			for _, rt := range strategyResourceTypes {
				// 检查资源池是否在当天的快照中存在于该集群
				var count int64
				if err := s.db.Model(&portal.ResourceSnapshot{}).
					Where("cluster_id = ? AND resource_type = ? AND created_at BETWEEN ? AND ?", sc.ClusterID, rt, now.BeginningOfDay(), now.EndOfDay()).
					Count(&count).Error; err != nil {
					return nil, err
				}
				if count > 0 {
					poolKey := fmt.Sprintf("%d-%s", sc.ClusterID, rt)
					targetResourcePools[poolKey] = struct{}{}

					// 检查该策略、集群、资源池今天是否已巡检 (有执行历史记录)
					var historyCount int64
					if err := s.db.Model(&portal.StrategyExecutionHistory{}).
						Where("strategy_id = ? AND cluster_id = ? AND resource_type = ? AND execution_time BETWEEN ? AND ?",
							strategy.ID, sc.ClusterID, rt, now.BeginningOfDay(), now.EndOfDay()).
						Count(&historyCount).Error; err != nil {
						return nil, err
					}
					if historyCount > 0 {
						inspectedResourcePools[poolKey] = struct{}{}
					}
				}
			}
		}
	}

	stats.TargetResourcePoolCount = len(targetResourcePools)
	stats.InspectedResourcePoolCount = len(inspectedResourcePools)

	return stats, nil
}

// GetResourcePoolTypes 获取所有资源池类型
func (s *ElasticScalingService) GetResourcePoolTypes() ([]string, error) {
	var resourceTypes []string

	// 查询当天的所有快照数据，获取不同的资源池类型
	err := s.db.Model(&portal.ResourceSnapshot{}).
		Where("created_at between ? and ?", now.BeginningOfDay(), now.EndOfDay()).
		Distinct("resource_type").
		Pluck("resource_type", &resourceTypes).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get resource pool types: %w", err)
	}

	// 如果当天没有数据，尝试获取最近的数据
	if len(resourceTypes) == 0 {
		err = s.db.Model(&portal.ResourceSnapshot{}).
			Order("created_at DESC").
			Distinct("resource_type").
			Limit(20).
			Pluck("resource_type", &resourceTypes).Error

		if err != nil {
			return nil, fmt.Errorf("failed to get recent resource pool types: %w", err)
		}
	}

	// 如果仍然没有数据，返回预定义的资源池类型
	if len(resourceTypes) == 0 {
		resourceTypes = []string{
			string(portal.Total),
			string(portal.Intel),
			string(portal.ARM),
			string(portal.HG),
			string(portal.GPU),
			string(portal.WithTaint),
			string(portal.Common),
		}
	}

	return resourceTypes, nil
}

// GetResourceAllocationTrend 获取资源分配趋势
func (s *ElasticScalingService) GetResourceAllocationTrend(clusterID int, timeRange string, resourceTypes string) (*ResourceAllocationTrendDTO, error) {
	if clusterID <= 0 {
		return nil, errors.New("无效的集群ID")
	}

	// 确定查询时间范围
	var startTime time.Time
	currentTime := time.Now()

	switch timeRange {
	case "24h":
		startTime = now.New(currentTime).Add(-24 * time.Hour)
	case "7d":
		startTime = now.New(currentTime).Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = now.New(currentTime).Add(-30 * 24 * time.Hour)
	default:
		startTime = now.New(currentTime).Add(-24 * time.Hour) // 默认24小时
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
	currentTime := time.Now()

	switch timeRange {
	case "7d":
		startTime = now.New(currentTime).Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = now.New(currentTime).Add(-30 * 24 * time.Hour)
	case "90d":
		startTime = now.New(currentTime).Add(-90 * 24 * time.Hour)
	default:
		startTime = now.New(currentTime).Add(-30 * 24 * time.Hour) // 默认30天
	}

	stats := &OrderStatsDTO{}

	// 获取总订单数（使用新的订单表）
	var totalCount int64
	if err := s.db.Model(&portal.Order{}).
		Where("type = ? AND created_at >= ?", portal.OrderTypeElasticScaling, startTime).
		Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats.TotalCount = int(totalCount)

	// 获取各状态订单数
	var pendingCount int64
	if err := s.db.Model(&portal.Order{}).
		Where("type = ? AND created_at >= ? AND status = ?", portal.OrderTypeElasticScaling, startTime, portal.OrderStatusPending).
		Count(&pendingCount).Error; err != nil {
		return nil, err
	}
	stats.PendingCount = int(pendingCount)

	var processingCount int64
	if err := s.db.Model(&portal.Order{}).
		Where("type = ? AND created_at >= ? AND status = ?", portal.OrderTypeElasticScaling, startTime, portal.OrderStatusProcessing).
		Count(&processingCount).Error; err != nil {
		return nil, err
	}
	stats.ProcessingCount = int(processingCount)

	var completedCount int64
	if err := s.db.Model(&portal.Order{}).
		Where("type = ? AND created_at >= ? AND status = ?", portal.OrderTypeElasticScaling, startTime, portal.OrderStatusCompleted).
		Count(&completedCount).Error; err != nil {
		return nil, err
	}
	stats.CompletedCount = int(completedCount)

	var failedCount int64
	if err := s.db.Model(&portal.Order{}).
		Where("type = ? AND created_at >= ? AND status = ?", portal.OrderTypeElasticScaling, startTime, portal.OrderStatusFailed).
		Count(&failedCount).Error; err != nil {
		return nil, err
	}
	stats.FailedCount = int(failedCount)

	var cancelledCount int64
	if err := s.db.Model(&portal.Order{}).
		Where("type = ? AND created_at >= ? AND status = ?", portal.OrderTypeElasticScaling, startTime, portal.OrderStatusCancelled).
		Count(&cancelledCount).Error; err != nil {
		return nil, err
	}
	stats.CancelledCount = int(cancelledCount)

	var ignoredCount int64
	if err := s.db.Model(&portal.Order{}).
		Where("type = ? AND created_at >= ? AND status = ?", portal.OrderTypeElasticScaling, startTime, portal.OrderStatusIgnored).
		Count(&ignoredCount).Error; err != nil {
		return nil, err
	}
	// 注意：OrderStatsDTO 中可能没有 IgnoredCount 字段，根据实际DTO结构调整

	// 计算成功率（如果DTO中有该字段）
	// if stats.TotalCount > 0 {
	//     stats.SuccessRate = float64(stats.CompletedCount) / float64(stats.TotalCount) * 100
	// }

	return stats, nil
}
