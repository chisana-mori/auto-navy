package service

import (
	"fmt"
	"navy-ng/models/portal"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

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
		// 注意：设计文档中提到冷却期以天为单位，但当前实现使用分钟作为单位
		// 这是为了提供更精细的控制，可以根据需要调整为天
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

			// 根据设计文档，我们需要获取最近N天的资源快照，而不仅仅是最新的
			// 计算N天前的时间点，这里我们使用策略的持续天数
			daysToCheck := 7 // 默认检查最近7天的数据，可以根据需要调整
			startDate := time.Now().AddDate(0, 0, -daysToCheck)

			// 获取每天的资源快照
			var dailySnapshots []portal.ResourceSnapshot
			query := s.db.Where("cluster_id = ? AND resource_type = ? AND created_at >= ?",
				clusterId, resourceType, startDate)

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

			// 获取每天的快照，按天分组，每天取一个代表性快照
			if err := query.Order("created_at DESC").Find(&dailySnapshots).Error; err != nil {
				s.logger.Error("Failed to fetch resource snapshots",
					zap.Int64("strategyID", strategy.ID),
					zap.String("strategyName", strategy.Name),
					zap.Int64("clusterID", clusterId),
					zap.String("resourceType", resourceType),
					zap.Error(err))
				continue
			}

			if len(dailySnapshots) == 0 {
				logMsg := fmt.Sprintf("No resource snapshots found for cluster %d, resource type %s within the last %d days.",
					clusterId, resourceType, daysToCheck)
				s.logger.Info(logMsg,
					zap.Int64("strategyID", strategy.ID),
					zap.String("strategyName", strategy.Name),
					zap.Int64("clusterID", clusterId),
					zap.String("resourceType", resourceType))

				currentTime := portal.NavyTime(time.Now())
				s.recordStrategyExecution(strategy.ID, StrategyExecutionResultFailureNoSnapshots, nil, logMsg, "", "", &currentTime)
				continue
			}

			// 按天分组快照
			dailySnapshotMap := make(map[string]portal.ResourceSnapshot)
			for _, snapshot := range dailySnapshots {
				day := time.Time(snapshot.CreatedAt).Format("2006-01-02")
				// 如果这一天还没有快照，或者当前快照时间更晚，则使用当前快照
				if _, exists := dailySnapshotMap[day]; !exists ||
					time.Time(snapshot.CreatedAt).After(time.Time(dailySnapshotMap[day].CreatedAt)) {
					dailySnapshotMap[day] = snapshot
				}
			}

			// 将每天的代表性快照转换为有序切片
			var orderedDailySnapshots []portal.ResourceSnapshot
			for _, snapshot := range dailySnapshotMap {
				orderedDailySnapshots = append(orderedDailySnapshots, snapshot)
			}

			// 按创建时间排序
			sort.Slice(orderedDailySnapshots, func(i, j int) bool {
				return time.Time(orderedDailySnapshots[i].CreatedAt).Before(time.Time(orderedDailySnapshots[j].CreatedAt))
			})

			s.logger.Info("Successfully fetched daily snapshots",
				zap.Int64("strategyID", strategy.ID),
				zap.Int64("clusterID", clusterId),
				zap.String("resourceType", resourceType),
				zap.Int("snapshotCount", len(orderedDailySnapshots)))

			// 计算连续超过阈值的天数
			var consecutiveDays int
			var maxConsecutiveDays int
			var breached bool
			var triggeredValueStr string
			var thresholdValueStr string

			// 检查每天的快照是否超过阈值
			for i, snapshot := range orderedDailySnapshots {
				// 使用现有的checkConsistentThresholdBreach函数检查单个快照
				singleBreached, singleTriggeredValueStr, singleThresholdValueStr := s.checkConsistentThresholdBreach([]portal.ResourceSnapshot{snapshot}, strategy)

				if singleBreached {
					consecutiveDays++
					// 如果是最后一天的快照，保存其触发值和阈值
					if i == len(orderedDailySnapshots)-1 {
						triggeredValueStr = singleTriggeredValueStr
						thresholdValueStr = singleThresholdValueStr
					}
				} else {
					// 重置连续天数
					consecutiveDays = 0
				}

				// 更新最大连续天数
				if consecutiveDays > maxConsecutiveDays {
					maxConsecutiveDays = consecutiveDays
				}
			}

			// 根据设计文档，只有当最后一段连续天数大于等于策略要求的持续天数时，才触发策略
			// 这里我们使用DurationMinutes字段作为持续天数的配置（需要确认这个字段的含义）
			requiredConsecutiveDays := strategy.DurationMinutes / (24 * 60) // 将分钟转换为天
			if requiredConsecutiveDays < 1 {
				requiredConsecutiveDays = 1 // 至少需要1天
			}

			breached = consecutiveDays >= requiredConsecutiveDays

			currentTime := portal.NavyTime(time.Now())
			if breached {
				s.logger.Info("Threshold consistently breached for strategy",
					zap.Int64("strategyID", strategy.ID),
					zap.String("strategyName", strategy.Name),
					zap.Int64("clusterID", clusterId),
					zap.String("resourceType", resourceType),
					zap.Int("consecutiveDays", consecutiveDays),
					zap.Int("requiredDays", requiredConsecutiveDays),
					zap.String("triggeredValue", triggeredValueStr),
					zap.String("thresholdValue", thresholdValueStr))

				// 调用设备匹配函数
				errMatch := s.matchDevicesForStrategy(strategy, clusterId, resourceType, triggeredValueStr, thresholdValueStr)
				if errMatch != nil {
					s.logger.Error("Error during device matching for strategy",
						zap.Int64("strategyID", strategy.ID),
						zap.Int64("clusterID", clusterId),
						zap.String("resourceType", resourceType),
						zap.Error(errMatch))
					// matchDevicesForStrategy函数内部会处理错误记录
				}
			} else {
				s.logger.Info("Threshold not consistently breached for strategy",
					zap.Int64("strategyID", strategy.ID),
					zap.String("strategyName", strategy.Name),
					zap.Int64("clusterID", clusterId),
					zap.String("resourceType", resourceType),
					zap.Int("consecutiveDays", consecutiveDays),
					zap.Int("requiredDays", requiredConsecutiveDays),
					zap.String("evaluatedTriggerValue", triggeredValueStr),
					zap.String("targetThresholdValue", thresholdValueStr))

				reason := fmt.Sprintf("Threshold not consistently met for cluster %d and resource type %s for %d consecutive days (required: %d days).",
					clusterId, resourceType, consecutiveDays, requiredConsecutiveDays)
				s.recordStrategyExecution(strategy.ID, StrategyExecutionResultFailureThresholdNotMet, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
			}
		}
	}

	return nil
}

// checkConsistentThresholdBreach checks if the strategy's threshold was breached for the given snapshots.
// 注意：在更新后的设计中，此函数用于评估单个日期的快照是否超过阈值，而不是评估连续时间段内的所有快照。
// 连续天数的计算已移至evaluateStrategy函数中。
func (s *ElasticScalingService) checkConsistentThresholdBreach(snapshots []portal.ResourceSnapshot, strategy *portal.ElasticScalingStrategy) (
	breached bool, triggeredValueStr string, thresholdValueStr string) {

	// If there are no snapshots, it cannot be a breach.
	if len(snapshots) == 0 {
		s.logger.Warn("checkConsistentThresholdBreach called with zero snapshots", zap.Int64("strategyID", strategy.ID))
		return false, "No snapshots available", s.buildThresholdString(strategy)
	}

	// 在当前实现中，我们只关注是否有任何快照满足条件
	// 对于按天评估的场景，通常只传入单个快照

	var actualCPUValues []float64
	var actualMemValues []float64

	allSnapshotsMetCriteria := true
	for _, snapshot := range snapshots {
		cpuMet := false
		memMet := false
		snapshotCpuVal := -1.0 // Use -1 to indicate not applicable or not calculated
		snapshotMemVal := -1.0

		// CPU Check
		if strategy.CPUThresholdValue > 0 {
			var currentCPUValue float64
			if strategy.CPUThresholdType == ThresholdTypeUsage {
				currentCPUValue = snapshot.MaxCpuUsageRatio
			} else if strategy.CPUThresholdType == ThresholdTypeAllocated {
				currentCPUValue = safePercentage(snapshot.CpuRequest, snapshot.CpuCapacity)
			}
			snapshotCpuVal = currentCPUValue
			actualCPUValues = append(actualCPUValues, currentCPUValue)

			if strategy.ThresholdTriggerAction == TriggerActionPoolEntry { // Scale Out (Pool Entry) - value > threshold
				cpuMet = currentCPUValue > float64(strategy.CPUThresholdValue)
			} else { // Scale In (Pool Exit) - value < threshold
				cpuMet = currentCPUValue < float64(strategy.CPUThresholdValue)
			}
		} else {
			cpuMet = true // No CPU threshold defined, so condition is met by default for CPU part
		}

		// Memory Check
		if strategy.MemoryThresholdValue > 0 {
			var currentMemValue float64
			if strategy.MemoryThresholdType == ThresholdTypeUsage {
				currentMemValue = snapshot.MaxMemoryUsageRatio
			} else if strategy.MemoryThresholdType == ThresholdTypeAllocated {
				currentMemValue = safePercentage(snapshot.MemRequest, snapshot.MemoryCapacity)
			}
			snapshotMemVal = currentMemValue
			actualMemValues = append(actualMemValues, currentMemValue)

			if strategy.ThresholdTriggerAction == TriggerActionPoolEntry { // Scale Out - value > threshold
				memMet = currentMemValue > float64(strategy.MemoryThresholdValue)
			} else { // Scale In - value < threshold
				memMet = currentMemValue < float64(strategy.MemoryThresholdValue)
			}
		} else {
			memMet = true // No Memory threshold defined, so condition is met by default for Memory part
		}

		// Condition Logic
		snapshotMeetsCondition := false
		if strategy.CPUThresholdValue > 0 && strategy.MemoryThresholdValue > 0 { // Both thresholds defined
			if strategy.ConditionLogic == ConditionLogicAnd {
				snapshotMeetsCondition = cpuMet && memMet
			} else { // OR logic
				snapshotMeetsCondition = cpuMet || memMet
			}
		} else if strategy.CPUThresholdValue > 0 { // Only CPU defined
			snapshotMeetsCondition = cpuMet
		} else if strategy.MemoryThresholdValue > 0 { // Only Memory defined
			snapshotMeetsCondition = memMet
		} else {
			snapshotMeetsCondition = false // Should not happen due to validation, but good to handle
			s.logger.Warn("Strategy has neither CPU nor Memory threshold defined during breach check", zap.Int64("strategyID", strategy.ID))
		}

		s.logger.Debug("Snapshot evaluation for threshold breach",
			zap.Int64("strategyID", strategy.ID),
			zap.Time("snapshotTime", time.Time(snapshot.CreatedAt)),
			zap.Float64("cpuValue", snapshotCpuVal),
			zap.Bool("cpuMet", cpuMet),
			zap.Float64("memValue", snapshotMemVal),
			zap.Bool("memMet", memMet),
			zap.String("conditionLogic", strategy.ConditionLogic),
			zap.Bool("snapshotMeetsCondition", snapshotMeetsCondition))

		if !snapshotMeetsCondition {
			allSnapshotsMetCriteria = false
			break
		}
	}

	// Construct triggeredValueStr and thresholdValueStr
	triggeredValueStr = s.buildTriggeredValueString(actualCPUValues, actualMemValues, strategy)
	thresholdValueStr = s.buildThresholdString(strategy)

	if allSnapshotsMetCriteria {
		s.logger.Debug("Snapshot met criteria for threshold breach",
			zap.Int64("strategyID", strategy.ID),
			zap.Time("snapshotTime", time.Time(snapshots[0].CreatedAt)))
		return true, triggeredValueStr, thresholdValueStr
	}

	s.logger.Debug("Snapshot did not meet criteria for threshold breach",
		zap.Int64("strategyID", strategy.ID),
		zap.Time("snapshotTime", time.Time(snapshots[0].CreatedAt)))
	return false, triggeredValueStr, thresholdValueStr
}

// buildThresholdString constructs a string representation of the strategy's thresholds.
func (s *ElasticScalingService) buildThresholdString(strategy *portal.ElasticScalingStrategy) string {
	var parts []string
	actionStr := ">"
	if strategy.ThresholdTriggerAction == TriggerActionPoolExit {
		actionStr = "<"
	}

	if strategy.CPUThresholdValue > 0 {
		parts = append(parts, fmt.Sprintf("CPU %s %s %f%%", strategy.CPUThresholdType, actionStr, strategy.CPUThresholdValue))
	}
	if strategy.MemoryThresholdValue > 0 {
		parts = append(parts, fmt.Sprintf("Memory %s %s %f%%", strategy.MemoryThresholdType, actionStr, strategy.MemoryThresholdValue))
	}

	logic := " "
	if len(parts) > 1 {
		logic = fmt.Sprintf(" %s ", strategy.ConditionLogic)
	}

	// 将分钟转换为天，以反映我们现在按天评估策略
	days := strategy.DurationMinutes / (24 * 60)
	if days < 1 {
		days = 1 // 至少需要1天
	}
	return fmt.Sprintf("%s for %d days", strings.Join(parts, logic), days)
}

// buildTriggeredValueString constructs a string representation of the actual values observed.
// It calculates averages if multiple snapshots were involved.
func (s *ElasticScalingService) buildTriggeredValueString(cpuValues []float64, memValues []float64, strategy *portal.ElasticScalingStrategy) string {
	var parts []string
	if strategy.CPUThresholdValue > 0 && len(cpuValues) > 0 {
		avgCPU := 0.0
		for _, v := range cpuValues {
			avgCPU += v
		}
		avgCPU /= float64(len(cpuValues))
		parts = append(parts, fmt.Sprintf("CPU %s: %.2f%% (avg)", strategy.CPUThresholdType, avgCPU))
	} else if strategy.CPUThresholdValue > 0 {
		parts = append(parts, fmt.Sprintf("CPU %s: N/A (no values)", strategy.CPUThresholdType))
	}

	if strategy.MemoryThresholdValue > 0 && len(memValues) > 0 {
		avgMem := 0.0
		for _, v := range memValues {
			avgMem += v
		}
		avgMem /= float64(len(memValues))
		parts = append(parts, fmt.Sprintf("Memory %s: %.2f%% (avg)", strategy.MemoryThresholdType, avgMem))
	} else if strategy.MemoryThresholdValue > 0 {
		parts = append(parts, fmt.Sprintf("Memory %s: N/A (no values)", strategy.MemoryThresholdType))
	}

	if len(parts) == 0 {
		return "No relevant metrics recorded"
	}
	return strings.Join(parts, ", ")
}

// recordStrategyExecution 记录策略执行历史
// 添加了 triggeredValue, thresholdValue 和 specificExecutionTime 参数
func (s *ElasticScalingService) recordStrategyExecution(
	strategyID int64,
	result string,
	orderID *int64,
	reason string,
	triggeredValue string, // 新增：从订单传递过来的触发值 或 评估过程中的实际值
	thresholdValue string, // 新增：从订单传递过来的阈值 或 策略定义的阈值
	specificExecutionTime *portal.NavyTime, // 新增：特定的执行时间，如订单的执行/完成时间或评估发生时间
) error {
	s.logger.Info("Recording strategy execution history",
		zap.Int64("strategyID", strategyID),
		zap.String("result", result),
		zap.Any("orderID", orderID), // Use Any for potentially nil pointer
		zap.String("reason", reason),
		zap.String("triggeredValue", triggeredValue),
		zap.String("thresholdValue", thresholdValue),
	)

	execTime := portal.NavyTime(time.Now()) // 默认使用当前时间
	if specificExecutionTime != nil {
		execTime = *specificExecutionTime // 如果提供了特定时间，则使用它
		s.logger.Debug("Using specific execution time for history record", zap.Time("execTime", time.Time(execTime)))
	} else {
		s.logger.Debug("Using current time for history record", zap.Time("execTime", time.Time(execTime)))
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
