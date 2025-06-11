package events

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// FactoryUsageExample 工厂使用示例
func FactoryUsageExample(em *EventManager) {
	ctx := context.Background()

	// 示例1：弹性伸缩订单事件发布
	// 创建订单完成事件
	err := NewESOPublisher(12345).
		WithEventManager(em).
		WithOperator("admin").
		Complete(ctx, "弹性伸缩订单处理完成")
	if err != nil {
		em.logger.Error("Failed to publish ESO complete event", zap.Error(err))
	}

	// 创建订单失败事件
	err = NewESOPublisher(12346).
		WithEventManager(em).
		WithOperator("system").
		Failed(ctx, "弹性伸缩订单处理失败：资源不足")
	if err != nil {
		em.logger.Error("Failed to publish ESO failed event", zap.Error(err))
	}

	// 示例2：设备操作事件发布
	// 设备加入资源池完成
	err = NewDevicePublisher(98765, 12345, "pool_entry").
		WithEventManager(em).
		Completed(ctx, "设备成功加入资源池")
	if err != nil {
		em.logger.Error("Failed to publish device completed event", zap.Error(err))
	}

	// 设备操作失败
	err = NewDevicePublisher(98766, 12345, "pool_exit").
		WithEventManager(em).
		Failed(ctx, "设备退出资源池失败：网络连接超时")
	if err != nil {
		em.logger.Error("Failed to publish device failed event", zap.Error(err))
	}

	// 示例3：维护操作事件发布
	// 维护开始
	err = NewMaintenancePublisher(12347, 98767, "cordon").
		WithEventManager(em).
		Started(ctx, "开始对设备进行维护操作")
	if err != nil {
		em.logger.Error("Failed to publish maintenance started event", zap.Error(err))
	}

	// 维护完成
	err = NewMaintenancePublisher(12347, 98767, "cordon").
		WithEventManager(em).
		Completed(ctx, "设备维护操作完成")
	if err != nil {
		em.logger.Error("Failed to publish maintenance completed event", zap.Error(err))
	}

	// 示例4：弹性伸缩事件发布
	// 弹性伸缩触发
	selectedDevices := []int{98765, 98766, 98767}
	err = NewScalingPublisher(1001, 2001, "compute", "pool_entry").
		WithEventManager(em).
		Triggered(ctx, 3, selectedDevices)
	if err != nil {
		em.logger.Error("Failed to publish scaling triggered event", zap.Error(err))
	}

	// 弹性伸缩完成
	err = NewScalingPublisher(1001, 2001, "compute", "pool_entry").
		WithEventManager(em).
		Completed(ctx, 3, selectedDevices, "成功完成弹性伸缩操作")
	if err != nil {
		em.logger.Error("Failed to publish scaling completed event", zap.Error(err))
	}

	// 弹性伸缩取消
	err = NewScalingPublisher(1002, 2001, "compute", "pool_exit").
		WithEventManager(em).
		Cancelled(ctx, "用户手动取消操作")
	if err != nil {
		em.logger.Error("Failed to publish scaling cancelled event", zap.Error(err))
	}
}

// ChainedEventExample 链式事件发布示例
func ChainedEventExample(em *EventManager, orderID int, deviceIDs []int) error {
	ctx := context.Background()

	// 1. 发布订单创建事件
	if err := NewESOPublisher(orderID).
		WithEventManager(em).
		WithOperator("system").
		Created(ctx, "弹性伸缩订单已创建"); err != nil {
		return fmt.Errorf("failed to publish order created event: %w", err)
	}

	// 2. 发布设备操作开始事件
	for _, deviceID := range deviceIDs {
		if err := NewDevicePublisher(deviceID, orderID, "pool_entry").
			WithEventManager(em).
			Started(ctx, fmt.Sprintf("设备 %d 开始加入资源池", deviceID)); err != nil {
			return fmt.Errorf("failed to publish device started event for device %d: %w", deviceID, err)
		}
	}

	// 3. 模拟设备操作完成
	for _, deviceID := range deviceIDs {
		if err := NewDevicePublisher(deviceID, orderID, "pool_entry").
			WithEventManager(em).
			Completed(ctx, fmt.Sprintf("设备 %d 成功加入资源池", deviceID)); err != nil {
			return fmt.Errorf("failed to publish device completed event for device %d: %w", deviceID, err)
		}
	}

	// 4. 发布订单完成事件
	if err := NewESOPublisher(orderID).
		WithEventManager(em).
		WithOperator("system").
		Complete(ctx, fmt.Sprintf("弹性伸缩订单完成，共处理 %d 台设备", len(deviceIDs))); err != nil {
		return fmt.Errorf("failed to publish order completed event: %w", err)
	}

	return nil
}

// ErrorHandlingExample 错误处理示例
func ErrorHandlingExample(em *EventManager, orderID int, deviceID int) {
	ctx := context.Background()

	// 模拟设备操作失败的场景
	devicePublisher := NewDevicePublisher(deviceID, orderID, "pool_entry").WithEventManager(em)

	// 开始操作
	if err := devicePublisher.Started(ctx, "开始设备操作"); err != nil {
		em.logger.Error("Failed to publish device started event", zap.Error(err))
		return
	}

	// 模拟操作失败
	errorMsg := "设备连接超时，操作失败"
	if err := devicePublisher.Failed(ctx, errorMsg); err != nil {
		em.logger.Error("Failed to publish device failed event", zap.Error(err))
		return
	}

	// 发布订单失败事件
	if err := NewESOPublisher(orderID).
		WithEventManager(em).
		WithOperator("system").
		Failed(ctx, fmt.Sprintf("订单处理失败：设备 %d 操作异常 - %s", deviceID, errorMsg)); err != nil {
		em.logger.Error("Failed to publish order failed event", zap.Error(err))
	}
}

// BatchOperationExample 批量操作示例
func BatchOperationExample(em *EventManager, strategyID, clusterID int, deviceIDs []int) error {
	ctx := context.Background()

	// 创建弹性伸缩发布器
	scalingPublisher := NewScalingPublisher(strategyID, clusterID, "compute", "pool_entry").WithEventManager(em)

	// 1. 触发弹性伸缩
	if err := scalingPublisher.Triggered(ctx, len(deviceIDs), deviceIDs); err != nil {
		return fmt.Errorf("failed to publish scaling triggered event: %w", err)
	}

	// 2. 批量处理设备
	successDevices := make([]int, 0)
	failedDevices := make([]int, 0)

	for i, deviceID := range deviceIDs {
		// 模拟部分设备操作成功，部分失败
		if i%3 == 0 { // 每3个设备中有1个失败
			if err := NewDevicePublisher(deviceID, 0, "pool_entry").
				WithEventManager(em).
				Failed(ctx, fmt.Sprintf("设备 %d 操作失败", deviceID)); err != nil {
				em.logger.Error("Failed to publish device failed event", zap.Int("deviceID", deviceID), zap.Error(err))
			}
			failedDevices = append(failedDevices, deviceID)
		} else {
			if err := NewDevicePublisher(deviceID, 0, "pool_entry").
				WithEventManager(em).
				Completed(ctx, fmt.Sprintf("设备 %d 操作成功", deviceID)); err != nil {
				em.logger.Error("Failed to publish device completed event", zap.Int("deviceID", deviceID), zap.Error(err))
			}
			successDevices = append(successDevices, deviceID)
		}
	}

	// 3. 根据结果发布相应的弹性伸缩事件
	if len(failedDevices) > 0 {
		// 部分失败，发布完成事件但包含失败信息
		result := fmt.Sprintf("弹性伸缩部分完成：成功 %d 台，失败 %d 台", len(successDevices), len(failedDevices))
		return scalingPublisher.Completed(ctx, len(successDevices), successDevices, result)
	} else {
		// 全部成功
		return scalingPublisher.Completed(ctx, len(successDevices), successDevices, "弹性伸缩全部完成")
	}
}