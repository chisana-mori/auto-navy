package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"navy-ng/models/portal"
	"navy-ng/pkg/redis"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// matchDevicesForStrategy finds suitable devices based on the strategy and query template.
// 在更新后的设计中，此函数接收按天分组的快照列表，而不是分钟级别的快照列表。
func (s *ElasticScalingService) matchDevicesForStrategy(
	strategy *portal.ElasticScalingStrategy,
	clusterID int64,
	resourceType string,
	triggeredValueStr string,
	thresholdValueStr string,
) error {
	s.logger.Info("Starting device matching for strategy",
		zap.Int64("strategyID", strategy.ID),
		zap.Int64("clusterID", clusterID),
		zap.String("resourceType", resourceType),
		zap.String("action", strategy.ThresholdTriggerAction))

	currentTime := portal.NavyTime(time.Now())

	var queryTemplateID int64
	if strategy.ThresholdTriggerAction == TriggerActionPoolEntry {
		queryTemplateID = strategy.EntryQueryTemplateID
	} else if strategy.ThresholdTriggerAction == TriggerActionPoolExit {
		queryTemplateID = strategy.ExitQueryTemplateID
	}

	if queryTemplateID == 0 {
		reason := fmt.Sprintf("Query template ID is not set for action type %s on strategy ID %d.", strategy.ThresholdTriggerAction, strategy.ID)
		s.logger.Error(reason, zap.Int64("strategyID", strategy.ID))
		s.recordStrategyExecution(strategy.ID, StrategyExecutionResultFailureInvalidTemplateID, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		return errors.New(reason)
	}
	s.logger.Info("Using query template for device matching", zap.Int64("templateID", queryTemplateID), zap.Int64("strategyID", strategy.ID))

	var queryTemplateModel portal.QueryTemplate
	if err := s.db.First(&queryTemplateModel, queryTemplateID).Error; err != nil {
		reason := fmt.Sprintf("Failed to find query template ID %d: %v", queryTemplateID, err)
		s.logger.Error(reason, zap.Int64("strategyID", strategy.ID), zap.Error(err))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.recordStrategyExecution(strategy.ID, StrategyExecutionResultFailureTemplateNotFound, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		} else {
			s.recordStrategyExecution(strategy.ID, StrategyExecutionResultFailureDBError, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		}
		return err
	}

	var filterGroups []FilterGroup // Assuming service.FilterGroup from device_query.go is used
	if err := json.Unmarshal([]byte(queryTemplateModel.Groups), &filterGroups); err != nil {
		reason := fmt.Sprintf("Failed to unmarshal filter groups from query template ID %d: %v", queryTemplateID, err)
		s.logger.Error(reason, zap.Int64("strategyID", strategy.ID), zap.Error(err))
		s.recordStrategyExecution(strategy.ID, StrategyExecutionResultFailureTemplateUnmarshal, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		return err
	}

	// 创建设备查询服务，使用缓存接口
	deviceCache, ok := s.cache.(*DeviceCache)
	if !ok {
		// 如果类型断言失败，创建一个新的设备缓存
		deviceCache = NewDeviceCache(s.redisHandler.(*redis.Handler), redis.NewKeyBuilder("navy", "v1"))
	}
	deviceQuerySvc := NewDeviceQueryService(s.db, deviceCache) // Pass deviceCache
	deviceRequest := &DeviceQueryRequest{                      // Assuming service.DeviceQueryRequest
		Groups: filterGroups,
		Page:   1,
		Size:   1000, // Fetch a large number of candidates
	}

	s.logger.Info("Querying candidate devices using template",
		zap.Int64("strategyID", strategy.ID),
		zap.Int64("templateID", queryTemplateID),
		zap.Any("requestGroups", deviceRequest.Groups))

	candidateDevicesResponse, err := deviceQuerySvc.QueryDevices(context.Background(), deviceRequest)
	if err != nil {
		reason := fmt.Sprintf("Error querying devices for template ID %d: %v", queryTemplateID, err)
		s.logger.Error(reason, zap.Int64("strategyID", strategy.ID), zap.Error(err))
		s.recordStrategyExecution(strategy.ID, StrategyExecutionResultFailureDeviceQuery, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		// TODO: Placeholder for notification
		return err
	}

	if candidateDevicesResponse == nil || len(candidateDevicesResponse.List) == 0 {
		reason := fmt.Sprintf("No candidate devices found for cluster %d, resource type %s using template %d", clusterID, resourceType, queryTemplateID)
		s.logger.Info(reason, zap.Int64("strategyID", strategy.ID))
		s.recordStrategyExecution(strategy.ID, StrategyExecutionResultFailureNoDevicesFound, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		// TODO: Placeholder for notification
		return nil // No error, but no devices found
	}

	s.logger.Info("Successfully queried candidate devices",
		zap.Int64("strategyID", strategy.ID),
		zap.Int64("templateID", queryTemplateID),
		zap.Int("candidateCount", len(candidateDevicesResponse.List)))

	// 使用策略中配置的设备数量
	numDevicesToChange := strategy.DeviceCount
	if numDevicesToChange <= 0 { // Safety check, though validation should prevent this
		numDevicesToChange = 1
		s.logger.Warn("Strategy DeviceCount is not positive, defaulting to 1", zap.Int64("strategyID", strategy.ID), zap.Int("originalDeviceCount", strategy.DeviceCount))
	}

	var selectedDeviceIDs []int64
	var suitableCandidates []DeviceResponse // Using DeviceResponse from device_query.go

	if strategy.ThresholdTriggerAction == TriggerActionPoolEntry {
		// Prefer devices not in any cluster
		var unassignedDevices []DeviceResponse
		var assignedDevicesFromQuery []DeviceResponse

		for _, device := range candidateDevicesResponse.List {
			if device.ClusterID == 0 || device.Cluster == "" {
				unassignedDevices = append(unassignedDevices, device)
			} else {
				assignedDevicesFromQuery = append(assignedDevicesFromQuery, device)
			}
		}
		// Fill with unassigned first, then with others if needed
		suitableCandidates = append(suitableCandidates, unassignedDevices...)
		suitableCandidates = append(suitableCandidates, assignedDevicesFromQuery...)

	} else if strategy.ThresholdTriggerAction == TriggerActionPoolExit {
		for _, device := range candidateDevicesResponse.List {
			if int64(device.ClusterID) == clusterID { // Must be part of the current cluster
				suitableCandidates = append(suitableCandidates, device)
			}
		}
	}

	if len(suitableCandidates) == 0 {
		reason := fmt.Sprintf("No suitable devices selected after filtering for action %s on cluster %d, template %d. Candidates from query: %d.",
			strategy.ThresholdTriggerAction, clusterID, queryTemplateID, len(candidateDevicesResponse.List))
		s.logger.Info(reason, zap.Int64("strategyID", strategy.ID))
		s.recordStrategyExecution(strategy.ID, StrategyExecutionResultFailureNoSuitableDevices, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		// TODO: Placeholder for notification
		return nil
	}

	// Select up to numDevicesToChange
	for i := 0; i < len(suitableCandidates) && len(selectedDeviceIDs) < int(numDevicesToChange); i++ {
		selectedDeviceIDs = append(selectedDeviceIDs, suitableCandidates[i].ID)
	}

	s.logger.Info("Selected devices for strategy action",
		zap.Int64("strategyID", strategy.ID),
		zap.Int64s("selectedDeviceIDs", selectedDeviceIDs),
		zap.Int("numDevicesToChange", int(numDevicesToChange)),
		zap.Int("suitableCandidateCount", len(suitableCandidates)))

	if len(selectedDeviceIDs) > 0 {
		s.logger.Info("Selected devices, proceeding to generate elastic scaling order",
			zap.Int64("strategyID", strategy.ID),
			zap.Int64s("selectedDeviceIDs", selectedDeviceIDs))

		// Call generateElasticScalingOrder
		err := s.generateElasticScalingOrder(strategy, clusterID, resourceType, selectedDeviceIDs, triggeredValueStr, thresholdValueStr)
		if err != nil {
			// Error logging and history recording are handled within generateElasticScalingOrder
			s.logger.Error("Failed to generate elastic scaling order",
				zap.Int64("strategyID", strategy.ID),
				zap.Error(err))
			// No need to record history here as generateElasticScalingOrder does it.
		}
		// The history recording for "success_devices_matched_order_pending" is removed.
		// generateElasticScalingOrder will record "order_created" or "failure_order_creation_failed".

	} else {
		// This case should ideally be caught by "no suitable devices selected"
		reason := fmt.Sprintf("No devices were ultimately selected for order generation for strategy %d on cluster %d.", strategy.ID, clusterID)
		s.logger.Warn(reason, zap.Int64("strategyID", strategy.ID))
		s.recordStrategyExecution(strategy.ID, StrategyExecutionResultFailureNoDevicesForOrder, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		return nil
	}

	return nil
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
) error {
	s.logger.Info("Generating elastic scaling order",
		zap.Int64("strategyID", strategy.ID),
		zap.Int64("clusterID", clusterID),
		zap.String("actionType", strategy.ThresholdTriggerAction),
		zap.Int("deviceCount", len(selectedDeviceIDs)))

	orderDTO := OrderDTO{
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

		reason := fmt.Sprintf("Failed to create order for cluster %d, resource type %s: %v", clusterID, resourceType, err)
		s.recordStrategyExecution(strategy.ID, StrategyExecutionResultOrderFailed, nil, reason, triggeredValueStr, thresholdValueStr, &currentTime)
		return err
	}

	s.logger.Info("Successfully created elastic scaling order",
		zap.Int64("orderID", orderID),
		zap.Int64("strategyID", strategy.ID),
		zap.Int64("clusterID", clusterID))

	reason := fmt.Sprintf("Successfully created order %d for cluster %d, resource type %s", orderID, clusterID, resourceType)
	s.recordStrategyExecution(strategy.ID, StrategyExecutionResultOrderCreated, &orderID, reason, triggeredValueStr, thresholdValueStr, &currentTime)

	// TODO: 根据设计文档，需要查询当周值班人员并向其发送运维通知
	s.logger.Info("Placeholder: Trigger notification to duty roster about the new order.", zap.Int64("orderID", orderID))

	return nil
}
