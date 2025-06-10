package es

import (
	"errors"
	"fmt"
	"navy-ng/models/portal"
	. "navy-ng/server/portal/internal/service"
	"sort"
	"strings"
	"time"

	"github.com/jinzhu/now"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// 日志消息
	logEvaluatingStrategies    = "Starting to evaluate all enabled strategies"
	logFailedToFetchStrategies = "Failed to fetch enabled strategies"
	logLockNotAcquired         = "Strategy lock not acquired, another instance is likely evaluating"
	logStrategyInCooldown      = "Strategy skipped due to cooldown period"
	logNoAssociatedClusters    = "No clusters associated with strategy, skipping"

	// 错误信息
	errFailedToCheckLock       = "failed to check redis lock for strategy %d: %w"
	errFailedToCheckCooldown   = "failed to check cooldown for strategy %d: %w"
	errFailedToGetAssociations = "failed to get associations for strategy %d: %w"

	// Redis 锁
	lockKeyFormat   = "elastic_scaling:strategy:%d:lock"
	lockValueFormat = "eval:%d:%d"

	// Zap 日志字段键
	zapKeyStrategyID = "strategyID"

	// GORM 查询
	queryStatusEnabled       = "status = ?"
	queryStrategyIDAndResult = "strategy_id = ? AND result = ?"
	queryResourceTypeAndPool = "resource_type = ? AND resource_pool = ?"
	resultOrderCreated       = "order_created"
	orderByExecutionTimeDesc = "execution_time DESC"

	// 资源类型
	resourceTypeTotal = "total"

	// 日期格式
	dateFormat = time.DateOnly
)

// EvaluateStrategies 评估所有启用的策略，并可能创建订单。
// 该函数是策略评估的入口点，通常由定时任务调用。
func (s *ElasticScalingService) EvaluateStrategies() error {
	s.logger.Info(logEvaluatingStrategies)
	var strategies []portal.ElasticScalingStrategy
	if err := s.db.Where(queryStatusEnabled, portal.StrategyStatusEnabled).Find(&strategies).Error; err != nil {
		s.logger.Error(logFailedToFetchStrategies, zap.Error(err))
		return err
	}
	s.logger.Info("Fetched enabled strategies", zap.Int("count", len(strategies)))

	for _, strategy := range strategies {
		// 为每个策略单独评估，记录错误但继续处理其他策略
		if err := s.evaluateStrategy(&strategy); err != nil {
			s.logger.Error("Error evaluating strategy",
				zap.Int64("strategyID", strategy.ID),
				zap.String("strategyName", strategy.Name),
				zap.Error(err))
		}
	}
	return nil
}

// evaluateStrategy 评估单个策略的完整流程。
// 它负责锁、数据获取和触发评估。冷却期检查已移至资源池级别。
func (s *ElasticScalingService) evaluateStrategy(strategy *portal.ElasticScalingStrategy) error {
	s.logger.Info("Starting single strategy evaluation", zap.Int64("strategyID", strategy.ID), zap.String("strategyName", strategy.Name))

	// 1. 尝试获取分布式锁
	lockKey := fmt.Sprintf(lockKeyFormat, strategy.ID)
	lockValue := fmt.Sprintf(lockValueFormat, strategy.ID, time.Now().UnixNano())
	locked, err := s.redisHandler.AcquireLock(lockKey, lockValue, 30*time.Second)
	if err != nil {
		return fmt.Errorf(errFailedToCheckLock, strategy.ID, err)
	}
	if !locked {
		s.logger.Info(logLockNotAcquired, zap.Int64(zapKeyStrategyID, strategy.ID))
		return nil
	}
	defer s.redisHandler.Delete(lockKey)

	// 2. 获取关联集群
	associations, err := s.getStrategyClusterAssociations(strategy.ID)
	if err != nil {
		return fmt.Errorf(errFailedToGetAssociations, strategy.ID, err)
	}
	if len(associations) == 0 {
		s.logger.Warn(logNoAssociatedClusters, zap.Int64(zapKeyStrategyID, strategy.ID))
		return nil
	}

	// 3. 循环评估每个关联
	for _, assoc := range associations {
		s.evaluateAssociation(strategy, assoc.ClusterID)
	}

	return nil
}

// evaluateAssociation 评估策略与单个集群的关联。
func (s *ElasticScalingService) evaluateAssociation(strategy *portal.ElasticScalingStrategy, clusterID int64) {
	resourceTypes := parseResourceTypes(strategy.ResourceTypes)

	for _, resourceType := range resourceTypes {
		s.logger.Info("Evaluating for resource type",
			zap.Int64("strategyID", strategy.ID),
			zap.Int64("clusterID", clusterID),
			zap.String("resourceType", resourceType))

		// 检查该集群+资源池是否在冷却期内
		inCooldown, err := s.isClusterResourcePoolInCooldown(strategy, clusterID, resourceType)
		if err != nil {
			s.logger.Error("Failed to check cooldown for cluster resource pool",
				zap.Error(err),
				zap.Int64("strategyID", strategy.ID),
				zap.Int64("clusterID", clusterID),
				zap.String("resourceType", resourceType))
			continue
		}
		if inCooldown {
			// 获取集群名称用于中文描述
			var cluster portal.K8sCluster
			clusterName := "未知集群"
			if err := s.db.Select("clustername").First(&cluster, clusterID).Error; err == nil {
				clusterName = cluster.ClusterName
			}

			reason := fmt.Sprintf("集群 %s（%s类型）处于冷却期内，跳过本次评估", clusterName, resourceType)
			s.logger.Info(reason,
				zap.Int64("strategyID", strategy.ID),
				zap.Int64("clusterID", clusterID),
				zap.String("resourceType", resourceType))

			// 记录冷却期执行历史
			currentTime := portal.NavyTime(time.Now())
			s.recordStrategyExecution(strategy.ID, clusterID, resourceType, StrategyExecutionResultSkippedCooldown, nil, reason, "", "", &currentTime)
			continue
		}

		// 获取每日快照
		requiredDays := getRequiredConsecutiveDays(strategy)
		daysToCheck := requiredDays + 3 // 多查询几天以确保有足够的历史数据进行判断
		snapshots, err := s.getOrderedDailySnapshots(clusterID, resourceType, daysToCheck)
		if err != nil {
			s.logger.Error("Failed to get daily snapshots", zap.Error(err), zap.Int64("clusterID", clusterID))
			continue
		}

		if len(snapshots) == 0 {
			// 获取集群名称用于中文描述
			var cluster portal.K8sCluster
			clusterName := "未知集群"
			if err := s.db.Select("clustername").First(&cluster, clusterID).Error; err == nil {
				clusterName = cluster.ClusterName
			}

			logMsg := fmt.Sprintf("集群 %s（%s类型）在过去 %d 天内未找到资源快照数据", clusterName, resourceType, daysToCheck)
			s.logger.Info(logMsg, zap.Int64("strategyID", strategy.ID))
			currentTime := portal.NavyTime(time.Now())
			s.recordStrategyExecution(strategy.ID, clusterID, resourceType, StrategyExecutionResultFailureNoSnapshots, nil, logMsg, "", "", &currentTime)
			continue
		}

		// 核心评估逻辑
		breached, consecutiveDays, triggeredValue, thresholdValue := s.EvaluateSnapshots(snapshots, strategy)

		// 根据评估结果执行操作
		currentTime := portal.NavyTime(time.Now())
		if breached {
			s.logger.Info("Threshold consistently breached for strategy",
				zap.Int64("strategyID", strategy.ID),
				zap.Int64("clusterID", clusterID),
				zap.Int("consecutiveDays", consecutiveDays),
				zap.Int("requiredDays", requiredDays))

			// 计算资源增量
			cpuDelta, memDelta := s.calculateResourceDelta(snapshots[len(snapshots)-1], strategy)

			s.logger.Info("Calculated resource delta",
				zap.Int64("strategyID", strategy.ID),
				zap.Int64("clusterID", clusterID),
				zap.Float64("cpuDelta", cpuDelta),
				zap.Float64("memDelta", memDelta))

			// 触发设备匹配和订单创建
			if err := s.matchDevicesForStrategyFunc(strategy, clusterID, resourceType, triggeredValue, thresholdValue, cpuDelta, memDelta, &snapshots[len(snapshots)-1]); err != nil {
				s.logger.Error("Error during device matching for strategy", zap.Error(err), zap.Int64("strategyID", strategy.ID))
			}
		} else {
			s.logger.Info("Threshold not consistently breached for strategy",
				zap.Int64("strategyID", strategy.ID),
				zap.Int64("clusterID", clusterID),
				zap.Int("consecutiveDays", consecutiveDays),
				zap.Int("requiredDays", requiredDays))

			// 获取集群名称用于中文描述
			var cluster portal.K8sCluster
			clusterName := "未知集群"
			if err := s.db.Select("clustername").First(&cluster, clusterID).Error; err == nil {
				clusterName = cluster.ClusterName
			}

			reason := fmt.Sprintf("集群 %s（%s类型）阈值未连续满足条件，当前连续天数 %d 天（需要 %d 天）",
				clusterName, resourceType, consecutiveDays, requiredDays)
			s.recordStrategyExecution(strategy.ID, clusterID, resourceType, StrategyExecutionResultFailureThresholdNotMet, nil, reason, triggeredValue, thresholdValue, &currentTime)
		}
	}
}

// isClusterResourcePoolInCooldown 检查指定集群+资源池是否在冷却期内
// 冷却期基于订单：如果该集群+资源池生成了非取消状态的订单，
// 则该资源池在冷却期内不会重复生成订单
func (s *ElasticScalingService) isClusterResourcePoolInCooldown(strategy *portal.ElasticScalingStrategy, clusterID int64, resourceType string) (bool, error) {
	if strategy.CooldownMinutes <= 0 {
		return false, nil
	}

	// 查询该集群+资源池类型的最近一次非取消状态订单
	var latestOrder portal.ElasticScalingOrderDetail
	err := s.db.Table("ng_orders o").
		Select("o.created_at").
		Joins("JOIN ng_elastic_scaling_order_details esd ON o.id = esd.order_id").
		Where("esd.strategy_id = ? AND esd.cluster_id = ? AND esd.resource_pool_type = ? AND o.status != ?",
			strategy.ID, clusterID, resourceType, portal.OrderStatusCancelled).
		Order("o.created_at DESC").
		First(&latestOrder).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 没有找到订单，不在冷却期
			return false, nil
		}
		return false, fmt.Errorf("failed to query latest order for cluster %d resource %s: %w", clusterID, resourceType, err)
	}

	// 计算时间差距和冷却期信息
	now := time.Now()
	lastOrderTime := time.Time(latestOrder.CreatedAt)
	timeDiff := now.Sub(lastOrderTime)
	cooldownEndTime := lastOrderTime.Add(time.Duration(strategy.CooldownMinutes) * time.Minute)
	remainingCooldown := cooldownEndTime.Sub(now)

	days := int(timeDiff.Hours() / 24)
	hours := int(timeDiff.Hours()) % 24
	minutes := int(timeDiff.Minutes()) % 60

	var statusMsg string
	if remainingCooldown > 0 {
		remainingDays := int(remainingCooldown.Hours() / 24)
		remainingHours := int(remainingCooldown.Hours()) % 24
		remainingMinutes := int(remainingCooldown.Minutes()) % 60
		statusMsg = fmt.Sprintf("剩余冷却时间: %d天%d时%d分钟", remainingDays, remainingHours, remainingMinutes)
	} else {
		statusMsg = "冷却期已结束"
	}

	s.logger.Info(fmt.Sprintf("[ElasticScaling] 集群 %d 资源池 %s | 最近订单时间差距: %d天%d时%d分钟 | 预期冷却时间: %d分钟 | %s",
		clusterID, resourceType, days, hours, minutes, strategy.CooldownMinutes, statusMsg))
	return now.Before(cooldownEndTime), nil
}

// getStrategyClusterAssociations 获取策略关联的集群。
func (s *ElasticScalingService) getStrategyClusterAssociations(strategyID int64) ([]portal.StrategyClusterAssociation, error) {
	var associations []portal.StrategyClusterAssociation
	if err := s.db.Where("strategy_id = ?", strategyID).Find(&associations).Error; err != nil {
		return nil, err
	}
	return associations, nil
}

// getOrderedDailySnapshots 获取并处理每日资源快照。
func (s *ElasticScalingService) getOrderedDailySnapshots(clusterID int64, resourceType string, days int) ([]portal.ResourceSnapshot, error) {
	endDate := now.EndOfDay()
	startDate := endDate.AddDate(0, 0, -days+1) // 包含今天在内的过去N天

	query := s.db.Where("cluster_id = ? AND created_at between ? and ?", clusterID, startDate, endDate)

	if resourceType != resourceTypeTotal {
		query = query.Where(queryResourceTypeAndPool, resourceType, resourceType)
	}

	var snapshots []portal.ResourceSnapshot
	if err := query.Order(OrderByCreatedAtDesc).Find(&snapshots).Error; err != nil {
		return nil, err
	}

	// 按天分组，每天只取最新的一个快照
	dailySnapshotMap := make(map[string]portal.ResourceSnapshot)
	for _, snapshot := range snapshots {
		day := time.Time(snapshot.CreatedAt).Format(dateFormat)
		if _, exists := dailySnapshotMap[day]; !exists {
			dailySnapshotMap[day] = snapshot
		}
	}

	var orderedDailySnapshots []portal.ResourceSnapshot
	for _, snapshot := range dailySnapshotMap {
		orderedDailySnapshots = append(orderedDailySnapshots, snapshot)
	}

	// 按创建时间升序排序
	sort.Slice(orderedDailySnapshots, func(i, j int) bool {
		return time.Time(orderedDailySnapshots[i].CreatedAt).Before(time.Time(orderedDailySnapshots[j].CreatedAt))
	})

	return orderedDailySnapshots, nil
}

// EvaluateSnapshots 是核心评估逻辑，无副作用，易于测试。
// 它接收快照和策略，返回是否触发、连续天数以及相关的监控值。
func (s *ElasticScalingService) EvaluateSnapshots(
	snapshots []portal.ResourceSnapshot,
	strategy *portal.ElasticScalingStrategy,
) (breached bool, consecutiveDays int, triggeredValueStr string, thresholdValueStr string) {
	var maxConsecutiveDays int
	var lastBreachedTriggeredValue string
	var lastBreachedThresholdValue string

	for _, snapshot := range snapshots {
		singleBreached, singleTriggeredValue, singleThresholdValue := s.checkSingleSnapshotBreach(snapshot, strategy)

		if singleBreached {
			consecutiveDays++
			lastBreachedTriggeredValue = singleTriggeredValue
			lastBreachedThresholdValue = singleThresholdValue
		} else {
			// 更新最大连续天数并重置计数器
			if consecutiveDays > maxConsecutiveDays {
				maxConsecutiveDays = consecutiveDays
			}
			consecutiveDays = 0
		}
	}
	// 循环结束后再次更新最大连续天数
	if consecutiveDays > maxConsecutiveDays {
		maxConsecutiveDays = consecutiveDays
	}

	requiredDays := getRequiredConsecutiveDays(strategy)
	// 只有当评估周期末尾的连续天数满足条件时才算触发
	if consecutiveDays >= requiredDays {
		return true, consecutiveDays, lastBreachedTriggeredValue, lastBreachedThresholdValue
	}

	// 如果未触发，返回观察到的最大连续天数和最后一次触发时的值
	return false, maxConsecutiveDays, lastBreachedTriggeredValue, lastBreachedThresholdValue
}

// checkSingleSnapshotBreach 检查单个快照是否满足策略阈值。
func (s *ElasticScalingService) checkSingleSnapshotBreach(snapshot portal.ResourceSnapshot, strategy *portal.ElasticScalingStrategy) (
	breached bool, triggeredValueStr string, thresholdValueStr string) {

	var cpuMet, memMet bool
	var cpuVal, memVal float64 = -1.0, -1.0

	// CPU检查 - 统一使用分配率计算（cpuRequest/cpuCapacity）
	if strategy.CPUThresholdValue > 0 {
		cpuVal = safePercentage(snapshot.CpuRequest, snapshot.CpuCapacity)
		cpuMet = compare(cpuVal, float64(strategy.CPUThresholdValue), strategy.ThresholdTriggerAction)
	} else {
		cpuMet = true // 没有定义CPU阈值，则默认满足
	}

	// 内存检查 - 统一使用分配率计算（memRequest/memoryCapacity）
	if strategy.MemoryThresholdValue > 0 {
		memVal = safePercentage(snapshot.MemRequest, snapshot.MemoryCapacity)
		memMet = compare(memVal, float64(strategy.MemoryThresholdValue), strategy.ThresholdTriggerAction)
	} else {
		memMet = true // 没有定义内存阈值，则默认满足
	}

	// 逻辑组合
	if strategy.CPUThresholdValue > 0 && strategy.MemoryThresholdValue > 0 {
		if strategy.ConditionLogic == ConditionLogicAnd {
			breached = cpuMet && memMet
		} else {
			breached = cpuMet || memMet
		}
	} else if strategy.CPUThresholdValue > 0 {
		breached = cpuMet
	} else if strategy.MemoryThresholdValue > 0 {
		breached = memMet
	} else {
		breached = false // 策略无效
	}

	triggeredValueStr = s.buildTriggeredValueString(cpuVal, memVal, strategy)
	thresholdValueStr = s.buildThresholdString(strategy)

	return breached, triggeredValueStr, thresholdValueStr
}

// calculateResourceDelta 计算需要调整的资源量
func (s *ElasticScalingService) calculateResourceDelta(latestSnapshot portal.ResourceSnapshot, strategy *portal.ElasticScalingStrategy) (cpuDelta float64, memDelta float64) {
	// 目标是调整资源容量使分配率达到阈值水平
	// 入池：当分配率超过阈值时，需要增加容量以降低分配率
	// 出池：当分配率低于阈值时，需要减少容量以提高分配率
	// 计算基于分配率：Request/Capacity

	if strategy.ThresholdTriggerAction == TriggerActionPoolEntry {
		if strategy.CPUThresholdValue > 0 {
			currentCPUAllocation := safePercentage(latestSnapshot.CpuRequest, latestSnapshot.CpuCapacity)
			if currentCPUAllocation > strategy.CPUThresholdValue {
				// 我们希望将分配率降至目标值，例如阈值本身
				targetCPUAllocation := strategy.CPUThresholdValue
				// (currentRequest/currentCapacity - targetRequest/newCapacity)
				// 假设 targetRequest = currentRequest, 求解 newCapacity
				// currentRequest / targetCPUAllocation = newCapacity
				newCapacity := latestSnapshot.CpuRequest / (targetCPUAllocation / 100)
				cpuDelta = newCapacity - latestSnapshot.CpuCapacity
			}
		}
		if strategy.MemoryThresholdValue > 0 {
			currentMemAllocation := safePercentage(latestSnapshot.MemRequest, latestSnapshot.MemoryCapacity)
			if currentMemAllocation > strategy.MemoryThresholdValue {
				targetMemAllocation := strategy.MemoryThresholdValue
				newCapacity := latestSnapshot.MemRequest / (targetMemAllocation / 100)
				memDelta = newCapacity - latestSnapshot.MemoryCapacity
			}
		}
	} else if strategy.ThresholdTriggerAction == TriggerActionPoolExit {
		if strategy.CPUThresholdValue > 0 {
			currentCPUAllocation := safePercentage(latestSnapshot.CpuRequest, latestSnapshot.CpuCapacity)
			if currentCPUAllocation < strategy.CPUThresholdValue {
				// 我们希望将分配率提升至目标值
				targetCPUAllocation := strategy.CPUThresholdValue
				// 假设 targetRequest = currentRequest, 求解 newCapacity
				// currentRequest / targetCPUAllocation = newCapacity
				newCapacity := latestSnapshot.CpuRequest / (targetCPUAllocation / 100)
				cpuDelta = newCapacity - latestSnapshot.CpuCapacity // Delta will be negative
			}
		}
		if strategy.MemoryThresholdValue > 0 {
			currentMemAllocation := safePercentage(latestSnapshot.MemRequest, latestSnapshot.MemoryCapacity)
			if currentMemAllocation < strategy.MemoryThresholdValue {
				targetMemAllocation := strategy.MemoryThresholdValue
				newCapacity := latestSnapshot.MemRequest / (targetMemAllocation / 100)
				memDelta = newCapacity - latestSnapshot.MemoryCapacity // Delta will be negative
			}
		}
	}

	return cpuDelta, memDelta
}

// compare 辅助函数，根据扩容或缩容操作比较值。
func compare(current, threshold float64, action string) bool {
	if action == TriggerActionPoolEntry { // 扩容：当前值 > 阈值
		return current > threshold
	}
	return current < threshold // 缩容：当前值 < 阈值
}

// parseResourceTypes 解析资源类型字符串。
func parseResourceTypes(resourceTypesStr string) []string {
	if resourceTypesStr == "" {
		return []string{"total"}
	}
	types := strings.Split(resourceTypesStr, ",")
	for i, rt := range types {
		types[i] = strings.TrimSpace(rt)
	}
	return types
}

// getRequiredConsecutiveDays 从策略中计算需要的天数。
func getRequiredConsecutiveDays(strategy *portal.ElasticScalingStrategy) int {
	// DurationMinutes 字段可能被误用为天数，这里做兼容处理
	// 假设如果值小于100，它代表天数；否则代表分钟
	if strategy.DurationMinutes > 0 && strategy.DurationMinutes < 100 {
		return strategy.DurationMinutes
	}
	days := strategy.DurationMinutes / (24 * 60)
	// 允许0天持续时间，表示立即触发
	if days < 0 {
		return 0
	}
	return days
}

// buildThresholdString 构建策略阈值的字符串表示。
func (s *ElasticScalingService) buildThresholdString(strategy *portal.ElasticScalingStrategy) string {
	var parts []string
	actionStr := ">"
	if strategy.ThresholdTriggerAction == TriggerActionPoolExit {
		actionStr = "<"
	}

	if strategy.CPUThresholdValue > 0 {
		parts = append(parts, fmt.Sprintf("CPU 分配率 %s %.2f%%", actionStr, strategy.CPUThresholdValue))
	}
	if strategy.MemoryThresholdValue > 0 {
		parts = append(parts, fmt.Sprintf("Memory 分配率 %s %.2f%%", actionStr, strategy.MemoryThresholdValue))
	}

	logic := " "
	if len(parts) > 1 {
		logic = fmt.Sprintf(" %s ", strategy.ConditionLogic)
	}

	return fmt.Sprintf("%s for %d days", strings.Join(parts, logic), getRequiredConsecutiveDays(strategy))
}

// buildTriggeredValueString 构建实际触发值的字符串表示。
func (s *ElasticScalingService) buildTriggeredValueString(cpuValue, memValue float64, strategy *portal.ElasticScalingStrategy) string {
	var parts []string
	if strategy.CPUThresholdValue > 0 {
		if cpuValue >= 0 {
			parts = append(parts, fmt.Sprintf("CPU 分配率: %.2f%%", cpuValue))
		} else {
			parts = append(parts, fmt.Sprintf("CPU 分配率: N/A"))
		}
	}

	if strategy.MemoryThresholdValue > 0 {
		if memValue >= 0 {
			parts = append(parts, fmt.Sprintf("Memory 分配率: %.2f%%", memValue))
		} else {
			parts = append(parts, fmt.Sprintf("Memory 分配率: N/A"))
		}
	}

	if len(parts) == 0 {
		return "No relevant metrics recorded"
	}
	return strings.Join(parts, ", ")
}

// recordStrategyExecution 记录策略执行历史。
func (s *ElasticScalingService) recordStrategyExecution(
	strategyID int64,
	clusterID int64,
	resourceType string,
	result string,
	orderID *int64,
	reason string,
	triggeredValue string,
	thresholdValue string,
	specificExecutionTime *portal.NavyTime,
) error {
	execTime := portal.NavyTime(time.Now())
	if specificExecutionTime != nil {
		execTime = *specificExecutionTime
	}

	history := portal.StrategyExecutionHistory{
		StrategyID:     strategyID,
		ClusterID:      clusterID,
		ResourceType:   resourceType,
		ExecutionTime:  execTime,
		TriggeredValue: triggeredValue,
		ThresholdValue: thresholdValue,
		Result:         result,
		OrderID:        orderID,
		Reason:         reason,
	}

	if err := s.db.Create(&history).Error; err != nil {
		s.logger.Error("Failed to create strategy execution history entry in DB", zap.Error(err), zap.Int64("strategyID", strategyID))
		return err
	}
	return nil
}

// isGormRecordNotFoundError 检查错误是否为gorm.ErrRecordNotFound
func isGormRecordNotFoundError(err error) bool {
	return err != nil && err == gorm.ErrRecordNotFound
}
