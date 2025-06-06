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
			ExternalTicketID:       "EXT123",
			StrategyTriggeredValue: "80%",
			StrategyThresholdValue: "70%",
			Devices: []portal.Device{
				{
					BaseModel: portal.BaseModel{
						ID:        1,
						CreatedAt: now,
						UpdatedAt: now,
					},
					CICode:         "DEV001",
					IP:             "192.168.1.100",
					ArchType:       "x86_64",
					IDC:            "IDC1",
					Room:           "Room1",
					Cabinet:        "Cabinet1",
					CabinetNO:      "C001",
					InfraType:      "物理机",
					IsLocalization: false,
					NetZone:        "DMZ",
					Group:          "计算节点",
					AppID:          "APP001",
					AppName:        "测试应用",
					CPU:            16.0,
					Memory:         64.0,
					Model:          "Dell R740",
					OS:             "CentOS 7",
					Company:        "Dell",
					Status:         "运行中",
					Role:           "worker",
					Cluster:        "test-cluster",
					ClusterID:      100,
					IsSpecial:      false,
					FeatureCount:   0,
				},
				{
					BaseModel: portal.BaseModel{
						ID:        2,
						CreatedAt: now,
						UpdatedAt: now,
					},
					CICode:         "DEV002",
					IP:             "192.168.1.101",
					ArchType:       "x86_64",
					IDC:            "IDC1",
					Room:           "Room1",
					Cabinet:        "Cabinet2",
					CabinetNO:      "C002",
					InfraType:      "物理机",
					IsLocalization: true,
					NetZone:        "DMZ",
					Group:          "计算节点",
					AppID:          "APP001",
					AppName:        "测试应用",
					CPU:            32.0,
					Memory:         128.0,
					Model:          "华为 2288H",
					OS:             "openEuler 20.03",
					Company:        "华为",
					Status:         "运行中",
					Role:           "worker",
					Cluster:        "test-cluster",
					ClusterID:      100,
					IsSpecial:      true,
					FeatureCount:   2,
				},
			},
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

	// 验证设备信息
	if len(dto.Devices) != 2 {
		t.Fatalf("期望设备数量为2，实际为%d", len(dto.Devices))
	}

	// 验证第一个设备
	device1 := dto.Devices[0]
	if device1.CICode != "DEV001" {
		t.Errorf("期望第一个设备CI码为DEV001，实际为%s", device1.CICode)
	}

	if device1.IP != "192.168.1.100" {
		t.Errorf("期望第一个设备IP为192.168.1.100，实际为%s", device1.IP)
	}

	if device1.CPU != 16.0 {
		t.Errorf("期望第一个设备CPU为16.0，实际为%f", device1.CPU)
	}

	if device1.IsLocalization != false {
		t.Errorf("期望第一个设备非国产化，实际为%t", device1.IsLocalization)
	}

	// 验证第二个设备
	device2 := dto.Devices[1]
	if device2.CICode != "DEV002" {
		t.Errorf("期望第二个设备CI码为DEV002，实际为%s", device2.CICode)
	}

	if device2.IsLocalization != true {
		t.Errorf("期望第二个设备为国产化，实际为%t", device2.IsLocalization)
	}

	if device2.IsSpecial != true {
		t.Errorf("期望第二个设备为特殊设备，实际为%t", device2.IsSpecial)
	}

	if device2.FeatureCount != 2 {
		t.Errorf("期望第二个设备特性数量为2，实际为%d", device2.FeatureCount)
	}

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
