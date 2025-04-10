package database

import (
	"encoding/json"
	"log"

	"gorm.io/gorm"
)

// InsertMockQueryTemplates 插入查询模板模拟数据
func InsertMockQueryTemplates(db *gorm.DB) error {
	// 定义一些示例筛选组
	productionGroup := []map[string]interface{}{
		{
			"field":    "role",
			"operator": "=",
			"value":    "production",
		},
		{
			"field":    "datacenter",
			"operator": "=",
			"value":    "dc1",
		},
	}

	testGroup := []map[string]interface{}{
		{
			"field":    "role",
			"operator": "=",
			"value":    "test",
		},
	}

	webserverGroup := []map[string]interface{}{
		{
			"field":    "app_id",
			"operator": "=",
			"value":    "webserver",
		},
		{
			"field":    "machine_type",
			"operator": "=",
			"value":    "physical",
		},
	}

	// 将筛选组转换为JSON字符串
	productionGroupJSON, _ := json.Marshal(productionGroup)
	testGroupJSON, _ := json.Marshal(testGroup)
	webserverGroupJSON, _ := json.Marshal(webserverGroup)

	// 创建模拟数据
	templates := []map[string]interface{}{
		{
			"name":        "生产环境设备",
			"description": "查询所有生产环境的设备",
			"groups":      string(productionGroupJSON),
			"created_by":  "admin",
			"updated_by":  "admin",
		},
		{
			"name":        "测试环境设备",
			"description": "查询所有测试环境的设备",
			"groups":      string(testGroupJSON),
			"created_by":  "admin",
			"updated_by":  "admin",
		},
		{
			"name":        "物理Web服务器",
			"description": "查询所有物理机Web服务器",
			"groups":      string(webserverGroupJSON),
			"created_by":  "admin",
			"updated_by":  "admin",
		},
	}

	// 插入数据
	for _, template := range templates {
		if err := db.Table("query_template").Create(template).Error; err != nil {
			log.Printf("Failed to insert query template: %v", err)
			return err
		}
	}

	return nil
}
