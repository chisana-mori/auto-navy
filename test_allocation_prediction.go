package main

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

github.com/navy-ng/server/portal/models/portal
)

func main() {
	// 连接数据库
	db, err := gorm.Open(sqlite.Open("navy.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}



	fmt.Println("=== 测试资源分配率预测功能 ===")

	// 由于generateOrderDescription是私有方法，我们通过查看数据库中的订单来验证功能
	fmt.Println("\n验证最新的无设备订单（应该没有分配率预测）:")

	var latestNoDeviceOrder portal.Order
	err = db.Where("device_count = 0 AND created_by = 'system/auto'").Order("created_at DESC").First(&latestNoDeviceOrder).Error
	if err != nil {
		fmt.Printf("未找到无设备订单: %v\n", err)
	} else {
		fmt.Printf("订单ID %d 描述: %s\n", latestNoDeviceOrder.ID, latestNoDeviceOrder.Description)

		// 检查是否包含预测信息
		if strings.Contains(latestNoDeviceOrder.Description, "预计") {
			fmt.Println("❌ 错误：无设备时不应该包含分配率预测")
		} else {
			fmt.Println("✅ 正确：无设备时没有分配率预测")
		}
	}

	fmt.Println("\n验证有设备订单（应该包含CPU和内存分配率预测）:")

	var latestDeviceOrder portal.Order
	err = db.Where("device_count > 0").Order("created_at DESC").First(&latestDeviceOrder).Error
	if err != nil {
		fmt.Printf("未找到有设备订单: %v\n", err)
	} else {
		fmt.Printf("订单ID %d 描述: %s\n", latestDeviceOrder.ID, latestDeviceOrder.Description)

		// 检查是否包含CPU和内存预测
		hasCPUPrediction := strings.Contains(latestDeviceOrder.Description, "CPU分配率")
		hasMemPrediction := strings.Contains(latestDeviceOrder.Description, "内存分配率")

		if hasCPUPrediction && hasMemPrediction {
			fmt.Println("✅ 正确：有设备时包含CPU和内存分配率预测")
		} else {
			fmt.Printf("❌ 错误：有设备时应该包含CPU和内存分配率预测 (CPU: %v, 内存: %v)\n", hasCPUPrediction, hasMemPrediction)
		}
	}

	fmt.Println("\n=== 测试完成 ===")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
