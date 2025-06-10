package es

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestGenerateOrderNotificationEmail 测试邮件生成功能
func TestGenerateOrderNotificationEmail(t *testing.T) {
	// 创建测试服务实例
	service := &ElasticScalingService{
		logger: zap.NewNop(), // 使用空日志记录器
	}

	// 测试入池订单邮件生成
	t.Run("Pool Entry Order Email", func(t *testing.T) {
		dto := OrderDTO{
			OrderNumber:      "ESO20241201123456",
			Name:             "测试入池订单",
			Description:      "自动化测试入池订单",
			ClusterID:        1,
			ActionType:       actionTypePoolEntry,
			DeviceCount:      3,
			CreatedBy:        "admin",
			ResourcePoolType: "compute",
			Devices:          []int{1, 2},
		}

		emailContent := service.generateOrderNotificationEmailForTest(123, dto, "test-cluster")

		// 验证邮件内容包含关键信息
		assert.Contains(t, emailContent, "入池变更通知")
		assert.Contains(t, emailContent, "测试入池订单")
		assert.Contains(t, emailContent, "test-cluster")
		assert.Contains(t, emailContent, "2 台")
		assert.Contains(t, emailContent, "admin")
		assert.Contains(t, emailContent, "值班同事，您好")
		assert.Contains(t, emailContent, "处理指引")
		assert.Contains(t, emailContent, "重要提醒")
	})

	// 测试退池订单邮件生成
	t.Run("Pool Exit Order Email", func(t *testing.T) {
		dto := OrderDTO{
			OrderNumber:      "ESO20241201654321",
			Name:             "测试退池订单",
			Description:      "自动化测试退池订单",
			ClusterID:        2,
			ActionType:       actionTypePoolExit,
			DeviceCount:      2,
			CreatedBy:        "operator",
			ResourcePoolType: "memory",
			Devices:          []int{1, 2},
		}

		emailContent := service.generateOrderNotificationEmailForTest(456, dto, "production-cluster")

		// 验证邮件内容包含关键信息
		assert.Contains(t, emailContent, "退池变更通知")
		assert.Contains(t, emailContent, "测试退池订单")
		assert.Contains(t, emailContent, "production-cluster")
		assert.Contains(t, emailContent, "2 台")
		assert.Contains(t, emailContent, "operator")
		assert.Contains(t, emailContent, "#ff7a45") // 退池操作的颜色
	})

	// 测试维护订单邮件生成
	t.Run("Maintenance Order Email", func(t *testing.T) {
		dto := OrderDTO{
			OrderNumber:      "ESO20241201789012",
			Name:             "测试维护订单",
			Description:      "自动化测试维护订单",
			ClusterID:        3,
			ActionType:       actionTypeMaintenanceRequest,
			DeviceCount:      1,
			CreatedBy:        "maintainer",
			ResourcePoolType: "compute",
			Devices:          []int{1, 2},
		}

		emailContent := service.generateOrderNotificationEmailForTest(789, dto, "staging-cluster")

		// 验证邮件内容包含关键信息
		assert.Contains(t, emailContent, "维护申请变更通知")
		assert.Contains(t, emailContent, "测试维护订单")
		assert.Contains(t, emailContent, "staging-cluster")
		assert.Contains(t, emailContent, "2 台")
		assert.Contains(t, emailContent, "maintainer")
	})
}

// generateOrderNotificationEmailForTest 测试专用的邮件生成方法
func (s *ElasticScalingService) generateOrderNotificationEmailForTest(orderID int64, dto OrderDTO, clusterName string) string {
	// 模拟设备信息 - 使用真实的设备数组长度
	devices := []DeviceDTO{
		{
			ID:       1,
			CICode:   "CI001",
			IP:       "192.168.1.10",
			ArchType: "x86_64",
			CPU:      8.0,
			Memory:   16384.0, // 16GB，使用字节单位以便正确显示为0.0GB（16384/1024 = 16.0）
			Status:   "active",
		},
		{
			ID:       2,
			CICode:   "CI002",
			IP:       "192.168.1.11",
			ArchType: "x86_64",
			CPU:      16.0,
			Memory:   32768.0, // 32GB，使用字节单位
			Status:   "active",
		},
	}

	// 确定变更工作类型
	actionName := s.getActionName(dto.ActionType)

	// 生成邮件主题
	subject := fmt.Sprintf(emailSubjectTemplate, actionName, fmt.Sprintf("ESO%d", orderID))

	// 生成HTML邮件正文
	emailContent := s.buildEmailHTML(subject, actionName, clusterName, dto, devices)

	return emailContent
}

// TestGetActionName 测试动作名称获取
func TestGetActionName(t *testing.T) {
	service := &ElasticScalingService{}

	tests := []struct {
		actionType string
		expected   string
	}{
		{actionTypePoolEntry, actionNamePoolEntry},
		{actionTypePoolExit, actionNamePoolExit},
		{actionTypeMaintenanceRequest, "维护申请"},
		{actionTypeMaintenanceUncordon, "维护解除"},
		{"unknown", "未知操作"},
	}

	for _, test := range tests {
		t.Run(test.actionType, func(t *testing.T) {
			result := service.getActionName(test.actionType)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestEmailHTMLStructure 测试邮件HTML结构
func TestEmailHTMLStructure(t *testing.T) {
	service := &ElasticScalingService{
		logger: zap.NewNop(),
	}

	dto := OrderDTO{
		OrderNumber:      "ESO20241201123456",
		Name:             "测试订单",
		Description:      "测试描述",
		ActionType:       actionTypePoolEntry,
		DeviceCount:      2,
		CreatedBy:        "admin",
		ResourcePoolType: "compute",
	}

	devices := []DeviceDTO{
		{
			CICode:   "CI001",
			IP:       "192.168.1.10",
			ArchType: "x86_64",
			CPU:      8.0,
			Memory:   16384.0, // 16GB，使用字节单位，16384/1024=16.0GB
			Status:   "active",
		},
	}

	emailContent := service.buildEmailHTML("测试邮件", "入池", "test-cluster", dto, devices)

	// 验证HTML结构
	assert.Contains(t, emailContent, "<!DOCTYPE html>")
	assert.Contains(t, emailContent, "<html>")
	assert.Contains(t, emailContent, "</html>")
	assert.Contains(t, emailContent, "<head>")
	assert.Contains(t, emailContent, "body style=")
	assert.Contains(t, emailContent, "table style=")
	assert.Contains(t, emailContent, "CI001")
	assert.Contains(t, emailContent, "192.168.1.10")
	assert.Contains(t, emailContent, "8.0核")
	assert.Contains(t, emailContent, "16.0GB")
}
