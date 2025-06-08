package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"navy-ng/models/portal"
	"sort"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// matchDevicesForStrategy finds suitable devices based on the strategy and query template.
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

	queryTemplateID, err := s.getQueryTemplateIDFromStrategy(strategy)
	if err != nil {
		reason := err.Error()
		s.logger.Error(reason, zap.Int64("strategyID", strategy.ID))
		s.recordStrategyExecution(strategy.ID, clusterID, resourceType, StrategyExecutionResultFailureInvalidTemplateID, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		return err
	}

	filterGroups, err := s.fetchAndUnmarshalQueryTemplate(queryTemplateID, strategy.ID, clusterID, resourceType, triggeredValueStr, thresholdValueStr, &currentTime)
	if err != nil {
		// Error logging and recording are handled within the function
		return err
	}

	candidateDevices, err := s.findCandidateDevices(queryTemplateID, filterGroups, strategy.ID, clusterID, resourceType, triggeredValueStr, thresholdValueStr, &currentTime)
	if err != nil {
		// Error logging and recording are handled within the function
		return err
	}

	if len(candidateDevices) == 0 {
		// 获取集群名称用于中文描述
		var cluster portal.K8sCluster
		clusterName := "未知集群"
		if err := s.db.Select("clustername").First(&cluster, clusterID).Error; err == nil {
			clusterName = cluster.ClusterName
		}

		reason := fmt.Sprintf("集群 %s（%s类型）使用模板 %d 未找到候选设备", clusterName, resourceType, queryTemplateID)
		s.logger.Info(reason, zap.Int64("strategyID", strategy.ID))
		// 无设备时仍然生成订单，作为提醒，不记录为失败
		return s.generateElasticScalingOrder(strategy, clusterID, resourceType, []int64{}, triggeredValueStr, thresholdValueStr, cpuDelta, memDelta, latestSnapshot)
	}

	selectedDeviceIDs := s.filterAndSelectDevices(candidateDevices, strategy, clusterID, cpuDelta, memDelta)

	if len(selectedDeviceIDs) == 0 {
		// 获取集群名称用于中文描述
		var cluster portal.K8sCluster
		clusterName := "未知集群"
		if err := s.db.Select("clustername").First(&cluster, clusterID).Error; err == nil {
			clusterName = cluster.ClusterName
		}

		actionName := s.getActionName(strategy.ThresholdTriggerAction)
		reason := fmt.Sprintf("集群 %s 执行%s操作时，经过筛选后无合适设备，模板 %d 查询到候选设备 %d 台",
			clusterName, actionName, queryTemplateID, len(candidateDevices))
		s.logger.Info(reason, zap.Int64("strategyID", strategy.ID))
		// 无合适设备时仍然生成订单，作为提醒，不记录为失败
		return s.generateElasticScalingOrder(strategy, clusterID, resourceType, []int64{}, triggeredValueStr, thresholdValueStr, cpuDelta, memDelta, latestSnapshot)
	}

	s.logger.Info("Selected devices for strategy action",
		zap.Int64("strategyID", strategy.ID),
		zap.Int64s("selectedDeviceIDs", selectedDeviceIDs),
		zap.Int("numDevicesToChange", strategy.DeviceCount),
		zap.Int("suitableCandidateCount", len(selectedDeviceIDs)))

	return s.generateElasticScalingOrder(strategy, clusterID, resourceType, selectedDeviceIDs, triggeredValueStr, thresholdValueStr, cpuDelta, memDelta, latestSnapshot)
}

// GetQueryTemplateIDFromStrategyPublic is a public wrapper for testing.
func (s *ElasticScalingService) GetQueryTemplateIDFromStrategyPublic(strategy *portal.ElasticScalingStrategy) (int64, error) {
	return s.getQueryTemplateIDFromStrategy(strategy)
}

func (s *ElasticScalingService) getQueryTemplateIDFromStrategy(strategy *portal.ElasticScalingStrategy) (int64, error) {
	var queryTemplateID int64
	switch strategy.ThresholdTriggerAction {
	case TriggerActionPoolEntry:
		queryTemplateID = strategy.EntryQueryTemplateID
	case TriggerActionPoolExit:
		queryTemplateID = strategy.ExitQueryTemplateID
	}

	if queryTemplateID == 0 {
		return 0, fmt.Errorf("query template ID is not set for action type %s on strategy ID %d", strategy.ThresholdTriggerAction, strategy.ID)
	}
	return queryTemplateID, nil
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
	deviceQuerySvc := NewDeviceQueryService(s.db, s.cache)
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

	// 否则，回退到基于固定数量的旧逻辑
	numDevicesToChange := strategy.DeviceCount
	if numDevicesToChange <= 0 {
		numDevicesToChange = 1
		s.logger.Warn("Strategy DeviceCount is not positive, defaulting to 1", zap.Int64("strategyID", strategy.ID), zap.Int("originalDeviceCount", strategy.DeviceCount))
	}

	var selectedDeviceIDs []int64
	for i := 0; i < len(suitableCandidates) && len(selectedDeviceIDs) < numDevicesToChange; i++ {
		selectedDeviceIDs = append(selectedDeviceIDs, suitableCandidates[i].ID)
	}

	return selectedDeviceIDs
}

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
	orderDescription := s.generateOrderDescription(strategy, clusterID, resourceType, selectedDeviceIDs, cpuDelta, memDelta, latestSnapshot)

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
	} else if len(selectedDeviceIDs) < strategy.DeviceCount {
		executionResult = StrategyExecutionResultOrderCreatedPartial
		reason = fmt.Sprintf("已为集群 %s（%s类型）创建部分%s订单 %d，匹配设备 %d 台（需要 %d 台）", clusterName, resourceType, actionName, orderID, len(selectedDeviceIDs), strategy.DeviceCount)
	} else {
		executionResult = StrategyExecutionResultOrderCreated
		reason = fmt.Sprintf("已为集群 %s（%s类型）成功创建%s订单 %d，涉及设备 %d 台", clusterName, resourceType, actionName, orderID, len(selectedDeviceIDs))
	}

	s.recordStrategyExecution(strategy.ID, clusterID, resourceType, executionResult, &orderID, reason, triggeredValueStr, thresholdValueStr, &currentTime)

	// TODO: 根据设计文档，需要查询当周值班人员并向其发送运维通知
	s.logger.Info("Placeholder: Trigger notification to duty roster about the new order.", zap.Int64("orderID", orderID))

	return nil
}
