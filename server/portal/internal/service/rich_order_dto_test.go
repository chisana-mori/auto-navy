package service

import (
	"testing"
	"time"

	"navy-ng/models/portal"
)

func TestToRichOrderDTO(t *testing.T) {
	// 创建测试数据
	now := portal.NavyTime(time.Now())

	order := &portal.Order{
		BaseModel: portal.BaseModel{
			ID:        1,
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrderNumber:   "ORD123456789",
		Name:          "测试订单",
		Description:   "这是一个测试订单",
		Type:          portal.OrderTypeElasticScaling,
		Status:        portal.OrderStatusPending,
		Executor:      "admin",
		CreatedBy:     "test_user",
		FailureReason: "",
		ElasticScalingDetail: &portal.ElasticScalingOrderDetail{
			BaseModel: portal.BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
			},
			OrderID:                1,
			ClusterID:              100,
			StrategyID:             nil,
			ActionType:             "pool_entry",
			DeviceCount:            2,
			StrategyTriggeredValue: "80%",
			StrategyThresholdValue: "70%",
			// Devices field is removed
		},
	}

	// 测试转换
	dto := ToRichOrderDTO(order)

	// 验证基础订单信息
	if dto == nil {
		t.Fatal("DTO转换失败，返回nil")
	}

	if dto.ID != 1 {
		t.Errorf("期望ID为1，实际为%d", dto.ID)
	}

	if dto.OrderNumber != "ORD123456789" {
		t.Errorf("期望订单号为ORD123456789，实际为%s", dto.OrderNumber)
	}

	if dto.Name != "测试订单" {
		t.Errorf("期望订单名称为'测试订单'，实际为%s", dto.Name)
	}

	if dto.Type != portal.OrderTypeElasticScaling {
		t.Errorf("期望订单类型为%s，实际为%s", portal.OrderTypeElasticScaling, dto.Type)
	}

	if dto.Status != portal.OrderStatusPending {
		t.Errorf("期望订单状态为%s，实际为%s", portal.OrderStatusPending, dto.Status)
	}

	// 验证弹性伸缩详情
	if dto.ElasticScalingDetail == nil {
		t.Fatal("弹性伸缩详情为nil")
	}

	if dto.ElasticScalingDetail.ClusterID != 100 {
		t.Errorf("期望集群ID为100，实际为%d", dto.ElasticScalingDetail.ClusterID)
	}

	if dto.ElasticScalingDetail.ActionType != "pool_entry" {
		t.Errorf("期望动作类型为pool_entry，实际为%s", dto.ElasticScalingDetail.ActionType)
	}

	if dto.ElasticScalingDetail.DeviceCount != 2 {
		t.Errorf("期望设备数量为2，实际为%d", dto.ElasticScalingDetail.DeviceCount)
	}

	// Device-related assertions are removed as the device list is no longer populated in this DTO.

	// 验证统计信息
	if dto.DeviceCount != 2 {
		t.Errorf("期望DTO设备总数为2，实际为%d", dto.DeviceCount)
	}
}

func TestToRichOrderDTOList(t *testing.T) {
	// 测试空列表
	emptyResult := ToRichOrderDTOList([]*portal.Order{})
	if len(emptyResult) != 0 {
		t.Errorf("期望空列表长度为0，实际为%d", len(emptyResult))
	}

	// 测试nil列表
	nilResult := ToRichOrderDTOList(nil)
	if len(nilResult) != 0 {
		t.Errorf("期望nil列表长度为0，实际为%d", len(nilResult))
	}
}

func TestToRichOrderDTO_NilInput(t *testing.T) {
	// 测试nil输入
	result := ToRichOrderDTO(nil)
	if result != nil {
		t.Error("期望nil输入返回nil，实际返回了非nil值")
	}
}
