package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
	"navy-ng/models/portal"
)

// MigrateOrderData 迁移订单数据从旧表到新表结构
func MigrateOrderData(db *gorm.DB) error {
	log.Println("开始迁移订单数据...")

	// 检查是否已经迁移过
	var count int64
	if err := db.Model(&portal.Order{}).Count(&count).Error; err != nil {
		return fmt.Errorf("检查订单表失败: %v", err)
	}

	if count > 0 {
		log.Printf("订单表已有 %d 条数据，跳过迁移", count)
		return nil
	}

	// 开始事务
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开始事务失败: %v", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("迁移过程中发生错误，已回滚: %v", r)
		}
	}()

	// 获取所有旧订单数据
	var oldOrders []portal.ElasticScalingOrder
	if err := tx.Find(&oldOrders).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("获取旧订单数据失败: %v", err)
	}

	log.Printf("找到 %d 条旧订单数据", len(oldOrders))

	// 迁移数据
	for _, oldOrder := range oldOrders {
		// 创建基础订单
		newOrder := portal.Order{
			BaseModel: portal.BaseModel{
				ID:        oldOrder.ID,
				CreatedAt: oldOrder.CreatedAt,
				UpdatedAt: oldOrder.UpdatedAt,
			},
			OrderNumber:    oldOrder.OrderNumber,
			Name:           oldOrder.Name,
			Description:    oldOrder.Description,
			Type:           portal.OrderTypeElasticScaling,
			Status:         portal.OrderStatus(oldOrder.Status),
			Executor:       oldOrder.Executor,
			ExecutionTime:  oldOrder.ExecutionTime,
			CreatedBy:      oldOrder.CreatedBy,
			CompletionTime: oldOrder.CompletionTime,
			FailureReason:  oldOrder.FailureReason,
		}

		// 插入基础订单（使用指定ID）
		if err := tx.Create(&newOrder).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("创建基础订单失败 (ID: %d): %v", oldOrder.ID, err)
		}

		// 创建弹性伸缩订单详情
		detail := portal.ElasticScalingOrderDetail{
			BaseModel: portal.BaseModel{
				CreatedAt: oldOrder.CreatedAt,
				UpdatedAt: oldOrder.UpdatedAt,
			},
			OrderID:                oldOrder.ID,
			ClusterID:              oldOrder.ClusterID,
			StrategyID:             oldOrder.StrategyID,
			ActionType:             oldOrder.ActionType,
			DeviceCount:            oldOrder.DeviceCount,
			MaintenanceStartTime:   oldOrder.MaintenanceStartTime,
			MaintenanceEndTime:     oldOrder.MaintenanceEndTime,
			ExternalTicketID:       oldOrder.ExternalTicketID,
			StrategyTriggeredValue: oldOrder.StrategyTriggeredValue,
			StrategyThresholdValue: oldOrder.StrategyThresholdValue,
		}

		// 插入详情
		if err := tx.Create(&detail).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("创建订单详情失败 (OrderID: %d): %v", oldOrder.ID, err)
		}

		log.Printf("成功迁移订单 ID: %d, 订单号: %s", oldOrder.ID, oldOrder.OrderNumber)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	log.Printf("订单数据迁移完成，共迁移 %d 条记录", len(oldOrders))
	return nil
}

// RollbackOrderMigration 回滚订单数据迁移
func RollbackOrderMigration(db *gorm.DB) error {
	log.Println("开始回滚订单数据迁移...")

	// 开始事务
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开始事务失败: %v", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("回滚过程中发生错误: %v", r)
		}
	}()

	// 删除弹性伸缩订单详情
	if err := tx.Unscoped().Delete(&portal.ElasticScalingOrderDetail{}, "1=1").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除订单详情失败: %v", err)
	}

	// 删除基础订单
	if err := tx.Unscoped().Delete(&portal.Order{}, "1=1").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除基础订单失败: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	log.Println("订单数据迁移回滚完成")
	return nil
}

// ValidateOrderMigration 验证订单数据迁移的正确性
func ValidateOrderMigration(db *gorm.DB) error {
	log.Println("开始验证订单数据迁移...")

	// 检查基础订单数量
	var orderCount int64
	if err := db.Model(&portal.Order{}).Count(&orderCount).Error; err != nil {
		return fmt.Errorf("检查基础订单数量失败: %v", err)
	}

	// 检查详情数量
	var detailCount int64
	if err := db.Model(&portal.ElasticScalingOrderDetail{}).Count(&detailCount).Error; err != nil {
		return fmt.Errorf("检查订单详情数量失败: %v", err)
	}

	// 检查旧订单数量
	var oldOrderCount int64
	if err := db.Model(&portal.ElasticScalingOrder{}).Count(&oldOrderCount).Error; err != nil {
		return fmt.Errorf("检查旧订单数量失败: %v", err)
	}

	log.Printf("验证结果: 旧订单 %d 条, 新基础订单 %d 条, 订单详情 %d 条", 
		oldOrderCount, orderCount, detailCount)

	if orderCount != oldOrderCount {
		return fmt.Errorf("基础订单数量不匹配: 期望 %d, 实际 %d", oldOrderCount, orderCount)
	}

	if detailCount != oldOrderCount {
		return fmt.Errorf("订单详情数量不匹配: 期望 %d, 实际 %d", oldOrderCount, detailCount)
	}

	// 检查数据完整性
	var mismatchCount int64
	if err := db.Table("orders o").
		Joins("LEFT JOIN elastic_scaling_order_details d ON o.id = d.order_id").
		Where("d.order_id IS NULL").
		Count(&mismatchCount).Error; err != nil {
		return fmt.Errorf("检查数据完整性失败: %v", err)
	}

	if mismatchCount > 0 {
		return fmt.Errorf("发现 %d 条基础订单没有对应的详情记录", mismatchCount)
	}

	log.Println("订单数据迁移验证通过")
	return nil
}
