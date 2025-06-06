package service

import (
	"errors"
	"fmt"
	"navy-ng/models/portal"
	"strings"
	"time"
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

	return stats, nil
}

// GetResourcePoolTypes 获取所有资源池类型
func (s *ElasticScalingService) GetResourcePoolTypes() ([]string, error) {
	var resourceTypes []string

	// 获取当天的日期
	today := time.Now().Format("2006-01-02")

	// 查询当天的所有快照数据，获取不同的资源池类型
	err := s.db.Model(&portal.ResourceSnapshot{}).
		Where("DATE(created_at) = ?", today).
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
