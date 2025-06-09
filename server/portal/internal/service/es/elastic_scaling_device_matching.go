package es

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"navy-ng/models/portal"
	"sort"
	"time"

	. "navy-ng/server/portal/internal/service"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// matchDevicesForStrategy 根据策略匹配设备并生成订单
func (s *ElasticScalingService) matchDevicesForStrategy(
	strategy *portal.ElasticScalingStrategy,
	clusterID int64,
	resourceType string,
	triggeredValueStr string,
	thresholdValueStr string,
	cpuDelta float64,
	memDelta float64,
	latestSnapshot *portal.ResourceSnapshot,
) error {
	s.logger.Info("Starting device matching for strategy",
		zap.Int64("strategyID", strategy.ID),
		zap.Int64("clusterID", clusterID),
		zap.String("resourceType", resourceType),
		zap.String("action", strategy.ThresholdTriggerAction))

	currentTime := portal.NavyTime(time.Now())

	// 步骤1: 根据资源类型和动作类型获取设备匹配策略
	policies, err := s.getDeviceMatchingPolicies(resourceType, strategy.ThresholdTriggerAction)
	if err != nil {
		reason := fmt.Sprintf("获取设备匹配策略失败: %s", err.Error())
		s.logger.Error(reason, zap.Int64("strategyID", strategy.ID))
		s.recordStrategyExecution(strategy.ID, clusterID, resourceType, StrategyExecutionResultFailureInvalidTemplateID, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		return err
	}

	var allSelectedDeviceIDs []int64
	var totalCandidateCount int

	// 步骤2: 遍历所有匹配策略，执行设备查询和选择
	for _, policy := range policies {
		// 组装查询参数
		filterGroups, err := s.assembleQueryParameters(policy, strategy.ID, clusterID, resourceType, triggeredValueStr, thresholdValueStr, &currentTime)
		if err != nil {
			continue // 继续尝试下一个策略
		}

		// 查询候选设备
		candidateDevices, err := s.findCandidateDevices(policy.QueryTemplateID, filterGroups, strategy.ID, clusterID, resourceType, triggeredValueStr, thresholdValueStr, &currentTime)
		if err != nil {
			continue // 继续尝试下一个策略
		}

		totalCandidateCount += len(candidateDevices)

		// 筛选和选择设备
		selectedDeviceIDs := s.filterAndSelectDevices(candidateDevices, strategy, clusterID, cpuDelta, memDelta)
		allSelectedDeviceIDs = append(allSelectedDeviceIDs, selectedDeviceIDs...)

		s.logger.Info("Processed device matching policy",
			zap.Int64("policyID", policy.ID),
			zap.String("policyName", policy.Name),
			zap.Int("candidateCount", len(candidateDevices)),
			zap.Int("selectedCount", len(selectedDeviceIDs)))
	}

	// 步骤3: 处理结果
	if totalCandidateCount == 0 {
		// 获取集群名称用于中文描述
		var cluster portal.K8sCluster
		clusterName := "未知集群"
		if err := s.db.Select("clustername").First(&cluster, clusterID).Error; err == nil {
			clusterName = cluster.ClusterName
		}

		reason := fmt.Sprintf("集群 %s（%s类型）未找到候选设备", clusterName, resourceType)
		s.logger.Info(reason, zap.Int64("strategyID", strategy.ID))
		// 无设备时仍然生成订单，作为提醒，不记录为失败
		return s.generateElasticScalingOrder(strategy, clusterID, resourceType, []int64{}, triggeredValueStr, thresholdValueStr, cpuDelta, memDelta, latestSnapshot)
	}

	if len(allSelectedDeviceIDs) == 0 {
		// 获取集群名称用于中文描述
		var cluster portal.K8sCluster
		clusterName := "未知集群"
		if err := s.db.Select("clustername").First(&cluster, clusterID).Error; err == nil {
			clusterName = cluster.ClusterName
		}

		actionName := s.getActionName(strategy.ThresholdTriggerAction)
		reason := fmt.Sprintf("集群 %s 执行%s操作时，经过筛选后无合适设备，查询到候选设备 %d 台",
			clusterName, actionName, totalCandidateCount)
		s.logger.Info(reason, zap.Int64("strategyID", strategy.ID))
		// 无合适设备时仍然生成订单，作为提醒，不记录为失败
		return s.generateElasticScalingOrder(strategy, clusterID, resourceType, []int64{}, triggeredValueStr, thresholdValueStr, cpuDelta, memDelta, latestSnapshot)
	}

	// 去重选中的设备ID
	uniqueDeviceIDs := s.deduplicateDeviceIDs(allSelectedDeviceIDs)

	s.logger.Info("Selected devices for strategy action",
		zap.Int64("strategyID", strategy.ID),
		zap.Int64s("selectedDeviceIDs", uniqueDeviceIDs),
		zap.Int("totalCandidateCount", totalCandidateCount),
		zap.Int("finalSelectedCount", len(uniqueDeviceIDs)))

	return s.generateElasticScalingOrder(strategy, clusterID, resourceType, uniqueDeviceIDs, triggeredValueStr, thresholdValueStr, cpuDelta, memDelta, latestSnapshot)
}

// GetDeviceMatchingPoliciesPublic is a public wrapper for testing.
func (s *ElasticScalingService) GetDeviceMatchingPoliciesPublic(resourceType, actionType string) ([]ResourcePoolDeviceMatchingPolicy, error) {
	return s.getDeviceMatchingPolicies(resourceType, actionType)
}

// getDeviceMatchingPolicies 根据资源类型和动作类型获取设备匹配策略
func (s *ElasticScalingService) getDeviceMatchingPolicies(resourceType, actionType string) ([]ResourcePoolDeviceMatchingPolicy, error) {
	// 创建资源池设备匹配策略服务
	// 注意：这里需要类型断言，因为接口类型不能直接传递
	var deviceCache *DeviceCache
	if s.cache != nil {
		if dc, ok := s.cache.(*DeviceCache); ok {
			deviceCache = dc
		}
	}
	policyService := NewResourcePoolDeviceMatchingPolicyService(s.db, deviceCache)

	// 根据资源类型和动作类型获取匹配策略
	policies, err := policyService.GetResourcePoolDeviceMatchingPoliciesByType(
		context.Background(),
		resourceType,
		actionType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get device matching policies for resourceType=%s, actionType=%s: %w", resourceType, actionType, err)
	}

	if len(policies) == 0 {
		return nil, fmt.Errorf("no enabled device matching policy found for resourceType=%s, actionType=%s", resourceType, actionType)
	}

	return policies, nil
}

// assembleQueryParameters 组装查询参数，包含查询模板和额外动态条件
func (s *ElasticScalingService) assembleQueryParameters(
	policy ResourcePoolDeviceMatchingPolicy,
	strategyID, clusterID int64,
	resourceType, triggeredValueStr, thresholdValueStr string,
	currentTime *portal.NavyTime,
) ([]FilterGroup, error) {
	s.logger.Info("Assembling query parameters",
		zap.Int64("policyID", policy.ID),
		zap.Int64("templateID", policy.QueryTemplateID),
		zap.Strings("additionConds", policy.AdditionConds))

	// 获取基础查询模板
	var queryTemplateModel portal.QueryTemplate
	if err := s.db.First(&queryTemplateModel, policy.QueryTemplateID).Error; err != nil {
		reason := fmt.Sprintf("查询模板 ID %d 查找失败：%v", policy.QueryTemplateID, err)
		s.logger.Error(reason, zap.Int64("strategyID", strategyID), zap.Error(err))
		result := StrategyExecutionResultFailureDBError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result = StrategyExecutionResultFailureTemplateNotFound
		}
		s.recordStrategyExecution(strategyID, clusterID, resourceType, result, nil, reason, triggeredValueStr, thresholdValueStr, currentTime)
		return nil, err
	}

	// 解析基础查询条件组
	var filterGroups []FilterGroup
	if err := json.Unmarshal([]byte(queryTemplateModel.Groups), &filterGroups); err != nil {
		reason := fmt.Sprintf("查询模板 ID %d 的过滤组解析失败：%v", policy.QueryTemplateID, err)
		s.logger.Error(reason, zap.Int64("strategyID", strategyID), zap.Error(err))
		s.recordStrategyExecution(strategyID, clusterID, resourceType, StrategyExecutionResultFailureTemplateUnmarshal, nil, reason, triggeredValueStr, thresholdValueStr, currentTime)
		return nil, err
	}

	// 组装额外动态条件（如果有）
	if len(policy.AdditionConds) > 0 {
		// TODO: 实现额外动态条件的组装逻辑
		// 这里可以根据policy.AdditionConds中的条件，动态添加到filterGroups中
		s.logger.Info("Additional conditions found, but not implemented yet",
			zap.Strings("conditions", policy.AdditionConds))
	}

	return filterGroups, nil
}

// FetchAndUnmarshalQueryTemplatePublic is a public wrapper for testing.
func (s *ElasticScalingService) FetchAndUnmarshalQueryTemplatePublic(queryTemplateID, strategyID int64, triggeredValueStr, thresholdValueStr string, currentTime *portal.NavyTime) ([]FilterGroup, error) {
	return s.fetchAndUnmarshalQueryTemplate(queryTemplateID, strategyID, 0, "", triggeredValueStr, thresholdValueStr, currentTime)
}

func (s *ElasticScalingService) fetchAndUnmarshalQueryTemplate(queryTemplateID, strategyID, clusterID int64, resourceType, triggeredValueStr, thresholdValueStr string, currentTime *portal.NavyTime) ([]FilterGroup, error) {
	s.logger.Info("Using query template for device matching", zap.Int64("templateID", queryTemplateID), zap.Int64("strategyID", strategyID))

	var queryTemplateModel portal.QueryTemplate
	if err := s.db.First(&queryTemplateModel, queryTemplateID).Error; err != nil {
		reason := fmt.Sprintf("查询模板 ID %d 查找失败：%v", queryTemplateID, err)
		s.logger.Error(reason, zap.Int64("strategyID", strategyID), zap.Error(err))
		result := StrategyExecutionResultFailureDBError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result = StrategyExecutionResultFailureTemplateNotFound
		}
		s.recordStrategyExecution(strategyID, clusterID, resourceType, result, nil, reason, triggeredValueStr, thresholdValueStr, currentTime)
		return nil, err
	}

	var filterGroups []FilterGroup
	if err := json.Unmarshal([]byte(queryTemplateModel.Groups), &filterGroups); err != nil {
		reason := fmt.Sprintf("查询模板 ID %d 的过滤组解析失败：%v", queryTemplateID, err)
		s.logger.Error(reason, zap.Int64("strategyID", strategyID), zap.Error(err))
		s.recordStrategyExecution(strategyID, clusterID, resourceType, StrategyExecutionResultFailureTemplateUnmarshal, nil, reason, triggeredValueStr, thresholdValueStr, currentTime)
		return nil, err
	}
	return filterGroups, nil
}

func (s *ElasticScalingService) findCandidateDevices(queryTemplateID int64, filterGroups []FilterGroup, strategyID, clusterID int64, resourceType, triggeredValueStr, thresholdValueStr string, currentTime *portal.NavyTime) ([]DeviceResponse, error) {
	// 创建设备查询服务
	var deviceCache DeviceCacheInterface
	if s.cache != nil {
		deviceCache = s.cache
	}
	deviceQuerySvc := NewDeviceQueryService(s.db, deviceCache)
	deviceRequest := &DeviceQueryRequest{
		Groups: filterGroups,
		Page:   1,
		Size:   1000, // Fetch a large number of candidates
	}

	s.logger.Info("Querying candidate devices using template",
		zap.Int64("strategyID", strategyID),
		zap.Int64("templateID", queryTemplateID),
		zap.Any("requestGroups", deviceRequest.Groups))

	candidateDevicesResponse, err := deviceQuerySvc.QueryDevices(context.Background(), deviceRequest)
	if err != nil {
		reason := fmt.Sprintf("使用模板 ID %d 查询设备失败：%v", queryTemplateID, err)
		s.logger.Error(reason, zap.Int64("strategyID", strategyID), zap.Error(err))
		s.recordStrategyExecution(strategyID, clusterID, resourceType, StrategyExecutionResultFailureDeviceQuery, nil, reason, triggeredValueStr, thresholdValueStr, currentTime)
		return nil, err
	}

	if candidateDevicesResponse == nil {
		return []DeviceResponse{}, nil
	}

	s.logger.Info("Successfully queried candidate devices",
		zap.Int64("strategyID", strategyID),
		zap.Int64("templateID", queryTemplateID),
		zap.Int("candidateCount", len(candidateDevicesResponse.List)))

	return candidateDevicesResponse.List, nil
}

// FilterAndSelectDevicesPublic is a public wrapper for testing.
func (s *ElasticScalingService) FilterAndSelectDevicesPublic(candidates []DeviceResponse, strategy *portal.ElasticScalingStrategy, clusterID int64, cpuDelta, memDelta float64) []int64 {
	return s.filterAndSelectDevices(candidates, strategy, clusterID, cpuDelta, memDelta)
}

func (s *ElasticScalingService) filterAndSelectDevices(candidates []DeviceResponse, strategy *portal.ElasticScalingStrategy, clusterID int64, cpuDelta, memDelta float64) []int64 {
	var suitableCandidates []DeviceResponse
	if strategy.ThresholdTriggerAction == TriggerActionPoolEntry {
		var unassignedDevices, assignedDevices []DeviceResponse
		for _, device := range candidates {
			if device.ClusterID == 0 || device.Cluster == "" {
				unassignedDevices = append(unassignedDevices, device)
			} else {
				assignedDevices = append(assignedDevices, device)
			}
		}
		suitableCandidates = append(unassignedDevices, assignedDevices...)
	} else { // TriggerActionPoolExit
		for _, device := range candidates {
			if int64(device.ClusterID) == clusterID {
				suitableCandidates = append(suitableCandidates, device)
			}
		}
	}

	// 如果是基于资源增量，则使用贪婪算法
	if cpuDelta > 0 || memDelta > 0 || cpuDelta < 0 || memDelta < 0 {
		return s.greedySelectDevices(suitableCandidates, cpuDelta, memDelta, strategy.ThresholdTriggerAction)
	}

	// 否则，使用动态计算的设备数量
	numDevicesToChange := s.calculateRequiredDeviceCount(cpuDelta, memDelta, suitableCandidates)
	if numDevicesToChange <= 0 {
		numDevicesToChange = 1
		s.logger.Warn("Calculated device count is not positive, defaulting to 1",
			zap.Int64("strategyID", strategy.ID),
			zap.Int("calculatedDeviceCount", numDevicesToChange))
	}

	var selectedDeviceIDs []int64
	for i := 0; i < len(suitableCandidates) && len(selectedDeviceIDs) < numDevicesToChange; i++ {
		selectedDeviceIDs = append(selectedDeviceIDs, suitableCandidates[i].ID)
	}

	return selectedDeviceIDs
}

// calculateRequiredDeviceCount 根据资源需求动态计算所需设备数量
func (s *ElasticScalingService) calculateRequiredDeviceCount(cpuDelta, memDelta float64, candidateDevices []DeviceResponse) int {
	if len(candidateDevices) == 0 {
		return 0
	}

	// 如果有明确的资源增量，使用贪婪算法计算
	if cpuDelta != 0 || memDelta != 0 {
		// 计算平均设备资源
		var avgCPU, avgMemory float64
		for _, device := range candidateDevices {
			avgCPU += device.CPU
			avgMemory += device.Memory
		}
		avgCPU /= float64(len(candidateDevices))
		avgMemory /= float64(len(candidateDevices))

		// 根据资源缺口计算所需设备数量
		cpuDeviceCount := int(cpuDelta / avgCPU)
		memDeviceCount := int(memDelta / avgMemory)

		// 取较大值，确保能满足资源需求
		deviceCount := cpuDeviceCount
		if memDeviceCount > deviceCount {
			deviceCount = memDeviceCount
		}

		// 至少需要1台设备
		if deviceCount <= 0 {
			deviceCount = 1
		}

		s.logger.Info("Calculated device count based on resource demand",
			zap.Float64("cpuDelta", cpuDelta),
			zap.Float64("memDelta", memDelta),
			zap.Float64("avgCPU", avgCPU),
			zap.Float64("avgMemory", avgMemory),
			zap.Int("calculatedCount", deviceCount))

		return deviceCount
	}

	// 如果没有明确资源增量，默认返回1台设备
	return 1
}

// deduplicateDeviceIDs 去重设备ID列表
func (s *ElasticScalingService) deduplicateDeviceIDs(deviceIDs []int64) []int64 {
	seen := make(map[int64]bool)
	var unique []int64

	for _, id := range deviceIDs {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	return unique
}

// greedySelectDevices 贪婪算法选择设备
func (s *ElasticScalingService) greedySelectDevices(devices []DeviceResponse, cpuDemand, memDemand float64, action string) []int64 {
	var selectedDeviceIDs []int64
	var cpuFulfilled, memFulfilled float64

	// 对于入池，我们希望用最少的设备满足最大的需求，所以按CPU或内存（取决于哪个需求更大）降序排序
	// 对于出池，我们希望移除最空闲的设备，所以按CPU或内存升序排序
	sort.Slice(devices, func(i, j int) bool {
		// 简单的排序逻辑：优先考虑CPU，可以根据策略进行扩展
		if action == TriggerActionPoolEntry {
			return devices[i].CPU > devices[j].CPU
		}
		return devices[i].CPU < devices[j].CPU
	})

	for _, device := range devices {
		if action == TriggerActionPoolEntry {
			if cpuFulfilled >= cpuDemand && memFulfilled >= memDemand {
				break // 需求已满足
			}
			selectedDeviceIDs = append(selectedDeviceIDs, device.ID)
			cpuFulfilled += device.CPU
			memFulfilled += device.Memory
		} else { // TriggerActionPoolExit
			// 对于出池，cpuDemand和memDemand是负数
			if cpuFulfilled <= cpuDemand && memFulfilled <= memDemand {
				break // 需求已满足
			}
			selectedDeviceIDs = append(selectedDeviceIDs, device.ID)
			cpuFulfilled -= device.CPU
			memFulfilled -= device.Memory
		}
	}

	s.logger.Info("Greedy device selection completed",
		zap.String("action", action),
		zap.Float64("cpuDemand", cpuDemand),
		zap.Float64("memDemand", memDemand),
		zap.Float64("cpuFulfilled", cpuFulfilled),
		zap.Float64("memFulfilled", memFulfilled),
		zap.Int64s("selectedDeviceIDs", selectedDeviceIDs))

	return selectedDeviceIDs
}

// GreedySelectDevicesPublic is a public wrapper for testing.
func (s *ElasticScalingService) GreedySelectDevicesPublic(devices []DeviceResponse, cpuDemand, memDemand float64, action string) []int64 {
	return s.greedySelectDevices(devices, cpuDemand, memDemand, action)
}

// generateElasticScalingOrder creates an order based on a successful strategy evaluation and device selection.
// 在更新后的设计中，此函数在连续多天阈值被突破后被调用，而不是在分钟级别的阈值突破后。
func (s *ElasticScalingService) generateElasticScalingOrder(
	strategy *portal.ElasticScalingStrategy,
	clusterID int64,
	resourceType string, // Keep for logging/reason context
	selectedDeviceIDs []int64,
	triggeredValueStr string,
	thresholdValueStr string,
	cpuDelta float64,
	memDelta float64,
	latestSnapshot *portal.ResourceSnapshot,
) error {
	s.logger.Info("Generating elastic scaling order",
		zap.Int64("strategyID", strategy.ID),
		zap.Int64("clusterID", clusterID),
		zap.String("actionType", strategy.ThresholdTriggerAction),
		zap.Int("deviceCount", len(selectedDeviceIDs)))

	// 生成订单名称
	orderName := s.generateOrderName(strategy, len(selectedDeviceIDs))

	// 生成订单描述
	orderDescription := s.generateOrderDescription(strategy, clusterID, resourceType, selectedDeviceIDs, latestSnapshot)

	orderDTO := OrderDTO{
		Name:                   orderName,
		Description:            orderDescription,
		ClusterID:              clusterID,
		StrategyID:             &strategy.ID,
		ActionType:             strategy.ThresholdTriggerAction,
		DeviceCount:            len(selectedDeviceIDs),
		Devices:                selectedDeviceIDs,
		StrategyTriggeredValue: triggeredValueStr,
		StrategyThresholdValue: thresholdValueStr,
		CreatedBy:              SystemAutoCreator,
		// Status will be set by CreateOrder, typically to "pending"
	}

	orderID, err := s.CreateOrder(orderDTO)
	currentTime := portal.NavyTime(time.Now())

	if err != nil {
		s.logger.Error("Failed to create elastic scaling order",
			zap.Int64("strategyID", strategy.ID),
			zap.Int64("clusterID", clusterID),
			zap.Error(err))

		// 获取集群名称用于中文描述
		var cluster portal.K8sCluster
		clusterName := "未知集群"
		if err := s.db.Select("clustername").First(&cluster, clusterID).Error; err == nil {
			clusterName = cluster.ClusterName
		}

		reason := fmt.Sprintf("为集群 %s（%s类型）创建订单失败：%v", clusterName, resourceType, err)
		s.recordStrategyExecution(strategy.ID, clusterID, resourceType, StrategyExecutionResultOrderFailed, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		return err
	}

	s.logger.Info("Successfully created elastic scaling order",
		zap.Int64("orderID", orderID),
		zap.Int64("strategyID", strategy.ID),
		zap.Int64("clusterID", clusterID),
		zap.Int("deviceCount", len(selectedDeviceIDs)))

	// 根据设备数量记录不同的执行结果
	var executionResult string
	var reason string

	// 获取集群名称用于中文描述
	var cluster portal.K8sCluster
	clusterName := "未知集群"
	if err := s.db.Select("clustername").First(&cluster, clusterID).Error; err == nil {
		clusterName = cluster.ClusterName
	}

	actionName := s.getActionName(strategy.ThresholdTriggerAction)

	if len(selectedDeviceIDs) == 0 {
		executionResult = StrategyExecutionResultOrderCreatedNoDevices
		reason = fmt.Sprintf("已为集群 %s（%s类型）创建%s提醒订单 %d，但无可用设备匹配", clusterName, resourceType, actionName, orderID)
	} else {
		// 移除对固定设备数量的检查，都记录为成功创建
		executionResult = StrategyExecutionResultOrderCreated
		reason = fmt.Sprintf("已为集群 %s（%s类型）成功创建%s订单 %d，涉及设备 %d 台", clusterName, resourceType, actionName, orderID, len(selectedDeviceIDs))
	}

	s.recordStrategyExecution(strategy.ID, clusterID, resourceType, executionResult, &orderID, reason, triggeredValueStr, thresholdValueStr, &currentTime)

	// TODO: 根据设计文档，需要查询当周值班人员并向其发送运维通知
	s.logger.Info("Placeholder: Trigger notification to duty roster about the new order.", zap.Int64("orderID", orderID))

	return nil
}
