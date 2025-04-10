package database

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"navy-ng/models/portal"
)

// InitDB 初始化数据库连接
func InitDB() (*gorm.DB, error) {
	// 配置 GORM 日志
	gormLogger := logger.New(
		logger.Default.LogMode(logger.Info).(logger.Writer),
		logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			IgnoreRecordNotFoundError: true,        // 忽略记录未找到错误
			Colorful:                  true,        // 彩色输出
			LogLevel:                  logger.Info, // 设置日志级别为 Info
		},
	)

	// 创建SQLite数据库文件
	db, err := gorm.Open(sqlite.Open("navy.db"), &gorm.Config{
		Logger: gormLogger,
		// 启用详细日志
		PrepareStmt: true,
		// 打印 SQL 语句
		DryRun: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %v", err)
	}

	// 获取底层 SQL DB 并设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移数据库表
	if err := db.AutoMigrate(
		&portal.K8sNode{},
		&portal.K8sNodeLabel{},
		&portal.K8sNodeTaint{},
		&portal.Device{},
		&portal.F5Info{},
		&portal.OpsJob{},
		&portal.QueryTemplate{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	return db, nil
}

// GenerateTestData 生成所有测试数据
func GenerateTestData(db *gorm.DB) error {
	// 生成device数据
	devices := []portal.Device{
		{
			DeviceID:     "worker-北京-UAT-1",
			IP:           "192.168.1.1",
			MachineType:  "physical",
			Cluster:      "UAT 集群-北京",
			Role:         "worker",
			Arch:         "amd64",
			IDC:          "BJ",
			Room:         "Room-北京-worker",
			Datacenter:   "北京",
			Cabinet:      "Cabinet-01",
			Network:      "Production",
			AppID:        "app-001",
			ResourcePool: "kubernetes",
		},
		{
			DeviceID:     "worker-北京-UAT-2",
			IP:           "192.168.1.2",
			MachineType:  "physical",
			Cluster:      "UAT 集群-北京",
			Role:         "worker",
			Arch:         "amd64",
			IDC:          "BJ",
			Room:         "Room-北京-worker",
			Datacenter:   "北京",
			Cabinet:      "Cabinet-02",
			Network:      "Production",
			AppID:        "app-002",
			ResourcePool: "kubernetes",
		},
		{
			DeviceID:     "worker-北京-UAT-3",
			IP:           "192.168.1.3",
			MachineType:  "physical",
			Cluster:      "UAT 集群-北京",
			Role:         "worker",
			Arch:         "amd64",
			IDC:          "BJ",
			Room:         "Room-北京-worker",
			Datacenter:   "北京",
			Cabinet:      "Cabinet-03",
			Network:      "Production",
			AppID:        "app-003",
			ResourcePool: "kubernetes",
		},
		{
			DeviceID:     "worker-深圳-UAT-1",
			IP:           "192.168.2.1",
			MachineType:  "physical",
			Cluster:      "UAT 集群-深圳",
			Role:         "worker",
			Arch:         "amd64",
			IDC:          "SZ",
			Room:         "Room-深圳-worker",
			Datacenter:   "深圳",
			Cabinet:      "Cabinet-01",
			Network:      "Production",
			AppID:        "app-004",
			ResourcePool: "kubernetes",
		},
		{
			DeviceID:     "worker-深圳-UAT-2",
			IP:           "192.168.2.2",
			MachineType:  "physical",
			Cluster:      "UAT 集群-深圳",
			Role:         "worker",
			Arch:         "amd64",
			IDC:          "SZ",
			Room:         "Room-深圳-worker",
			Datacenter:   "深圳",
			Cabinet:      "Cabinet-02",
			Network:      "Production",
			AppID:        "app-005",
			ResourcePool: "kubernetes",
		},
	}

	for _, device := range devices {
		if err := db.Create(&device).Error; err != nil {
			return fmt.Errorf("failed to create device: %v", err)
		}
	}

	// 为每个device生成对应的k8s_node数据
	for _, device := range devices {
		// 生成k8s_node
		node := portal.K8sNode{
			NodeName: device.DeviceID,
			Role:     device.Role,
			Status:   "Ready",
		}
		if err := db.Create(&node).Error; err != nil {
			return fmt.Errorf("failed to create k8s node: %v", err)
		}

		// 生成label数据
		labels := []struct {
			key   string
			value string
		}{
			{"env", "prod"},
			{"app", device.AppID},
			{"cluster", device.Cluster},
			{"idc", device.IDC},
			{"room", device.Room},
			{"datacenter", device.Datacenter},
			{"network", device.Network},
			{"resource_pool", device.ResourcePool},
		}

		// 为每个节点生成3-5个label
		numLabels := rand.Intn(3) + 3
		usedKeys := make(map[string]bool)

		for i := 0; i < numLabels; i++ {
			// 随机选择一个未使用的label
			var selectedLabel struct {
				key   string
				value string
			}
			for {
				selectedLabel = labels[rand.Intn(len(labels))]
				if !usedKeys[selectedLabel.key] {
					usedKeys[selectedLabel.key] = true
					break
				}
			}

			label := portal.K8sNodeLabel{
				NodeID: node.ID,
				Key:    selectedLabel.key,
				Value:  selectedLabel.value,
				BaseModel: portal.BaseModel{
					CreatedAt: portal.NavyTime(time.Now().Truncate(24 * time.Hour)),
					UpdatedAt: portal.NavyTime(time.Now().Truncate(24 * time.Hour)),
				},
			}
			if err := db.Create(&label).Error; err != nil {
				return fmt.Errorf("failed to create label: %v", err)
			}
		}

		// 生成taint数据
		taints := []struct {
			key    string
			value  string
			effect string
		}{
			{"dedicated", "gpu", "NoSchedule"},
			{"dedicated", "cpu", "NoSchedule"},
			{"node.kubernetes.io/unschedulable", "", "NoSchedule"},
			{"node.kubernetes.io/memory-pressure", "", "NoSchedule"},
			{"node.kubernetes.io/disk-pressure", "", "NoSchedule"},
		}

		// 为每个节点生成1-2个taint
		numTaints := rand.Intn(2) + 1
		usedKeys = make(map[string]bool)

		for i := 0; i < numTaints; i++ {
			// 随机选择一个未使用的taint
			var selectedTaint struct {
				key    string
				value  string
				effect string
			}
			for {
				selectedTaint = taints[rand.Intn(len(taints))]
				if !usedKeys[selectedTaint.key] {
					usedKeys[selectedTaint.key] = true
					break
				}
			}

			taint := portal.K8sNodeTaint{
				NodeID: node.ID,
				Key:    selectedTaint.key,
				Value:  selectedTaint.value,
				Effect: selectedTaint.effect,
				BaseModel: portal.BaseModel{
					CreatedAt: portal.NavyTime(time.Now().Truncate(24 * time.Hour)),
					UpdatedAt: portal.NavyTime(time.Now().Truncate(24 * time.Hour)),
				},
			}
			if err := db.Create(&taint).Error; err != nil {
				return fmt.Errorf("failed to create taint: %v", err)
			}
		}
	}

	ClearAndSeedData(db)

	return nil
}
