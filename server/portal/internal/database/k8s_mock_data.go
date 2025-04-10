package database

import (
	"log"
	"time"

	"gorm.io/gorm"

	"navy-ng/models/portal"
)

// InsertMockK8sData 插入K8s相关模拟数据
func InsertMockK8sData(db *gorm.DB) error {
	// 检查k8s_cluster表是否已有数据
	var count int64
	if err := db.Table("k8s_cluster").Count(&count).Error; err != nil {
		return err
	}

	// 如果已有数据，则不再重复插入集群和节点数据
	if count > 0 {
		log.Println("K8s集群和节点数据已存在，跳过插入...")
	} else {
		log.Println("开始插入K8s相关模拟数据...")

		// 插入K8s集群数据
		log.Println("插入K8s集群模拟数据...")
		clusters := insertMockK8sClustersNew(db)

		// 插入K8s节点数据
		log.Println("插入K8s节点模拟数据...")
		nodes := insertMockK8sNodes(db, clusters)

		// 插入K8s节点标签数据
		log.Println("插入K8s节点标签模拟数据...")
		insertMockK8sNodeLabels(db, nodes)

		// 插入K8s节点污点数据
		log.Println("插入K8s节点污点模拟数据...")
		insertMockK8sNodeTaints(db, nodes)
	}

	// 获取今天的开始和结束时间
	todayStart := time.Now().Truncate(24 * time.Hour)
	todayEnd := todayStart.Add(24 * time.Hour)

	// 检查今天是否已有标签数据
	var todayLabelCount int64
	if err := db.Table("k8s_node_label").
		Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).
		Count(&todayLabelCount).Error; err != nil {
		return err
	}

	// 如果今天没有标签数据，则插入
	if todayLabelCount == 0 {
		log.Println("插入今天的K8s节点标签数据...")
		// 获取所有节点
		var nodes []portal.K8sNode
		if err := db.Find(&nodes).Error; err != nil {
			return err
		}
		insertMockK8sNodeLabels(db, nodes)
	}

	// 检查今天是否已有污点数据
	var todayTaintCount int64
	if err := db.Table("k8s_node_taint").
		Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).
		Count(&todayTaintCount).Error; err != nil {
		return err
	}

	// 如果今天没有污点数据，则插入
	if todayTaintCount == 0 {
		log.Println("插入今天的K8s节点污点数据...")
		// 获取所有节点
		var nodes []portal.K8sNode
		if err := db.Find(&nodes).Error; err != nil {
			return err
		}
		insertMockK8sNodeTaints(db, nodes)
	}

	return nil
}

// insertMockK8sClustersNew 插入K8s集群模拟数据
func insertMockK8sClustersNew(db *gorm.DB) []portal.K8sCluster {
	// 创建模拟数据
	mockClusters := []portal.K8sCluster{
		{
			Name:     "生产集群-上海",
			Region:   "上海",
			Endpoint: "https://k8s-prod-sh.example.com:6443",
			Status:   "Running",
		},
		{
			Name:     "生产集群-北京",
			Region:   "北京",
			Endpoint: "https://k8s-prod-bj.example.com:6443",
			Status:   "Running",
		},
		{
			Name:     "测试集群-上海",
			Region:   "上海",
			Endpoint: "https://k8s-test-sh.example.com:6443",
			Status:   "Running",
		},
		{
			Name:     "开发集群-深圳",
			Region:   "深圳",
			Endpoint: "https://k8s-dev-sz.example.com:6443",
			Status:   "Stopped",
		},
		{
			Name:     "UAT集群-北京",
			Region:   "北京",
			Endpoint: "https://k8s-uat-bj.example.com:6443",
			Status:   "Running",
		},
	}

	// 插入数据
	for i := range mockClusters {
		// 设置创建时间和更新时间
		now := time.Now()
		mockClusters[i].CreatedAt = now
		mockClusters[i].UpdatedAt = now

		if err := db.Create(&mockClusters[i]).Error; err != nil {
			log.Printf("Warning: failed to create k8s cluster %s: %v", mockClusters[i].Name, err)
		}
	}

	log.Printf("成功插入 %d 条K8s集群数据", len(mockClusters))
	return mockClusters
}

// insertMockK8sNodes 插入K8s节点模拟数据
func insertMockK8sNodes(db *gorm.DB, clusters []portal.K8sCluster) []portal.K8sNode {
	// 不需要设置时间

	// 创建模拟数据
	var mockNodes []portal.K8sNode

	// 为每个集群创建节点
	for _, cluster := range clusters {
		// 创建master节点
		now := time.Now()
		masterNode := portal.K8sNode{
			BaseModel: portal.BaseModel{
				CreatedAt: portal.NavyTime(now),
				UpdatedAt: portal.NavyTime(now),
			},
			NodeName:                "master-" + cluster.Region + "-" + cluster.Name[0:4],
			HostIP:                  "192.168.1." + string([]byte{byte(cluster.ID + 10)}),
			Role:                    "master",
			OSImage:                 "Ubuntu 20.04.5 LTS",
			KernelVersion:           "5.4.0-137-generic",
			KubeletVersion:          "v1.24.10",
			KubeProxyVersion:        "v1.24.10",
			ContainerRuntimeVersion: "containerd://1.6.12",
			CPULogic:                "8",
			MemLogic:                "32Gi",
			CPUCapacity:             "8",
			MemCapacity:             "32Gi",
			CPUAllocatable:          "7",
			MemAllocatable:          "30Gi",
			FSTypeRoot:              "ext4",
			DiskRoot:                "50Gi",
			DiskDocker:              "100Gi",
			DiskKubelet:             "100Gi",
			NodeCreated:             time.Now().Format(time.RFC3339),
			Status:                  "Ready",
			K8sClusterID:            cluster.ID,
			GPU:                     "none",
			DiskCount:               2,
			DiskDetail:              "sda:250GB,sdb:250GB",
			NetworkSpeed:            1000,
		}
		if err := db.Create(&masterNode).Error; err != nil {
			log.Printf("Warning: failed to create k8s node %s: %v", masterNode.NodeName, err)
		} else {
			mockNodes = append(mockNodes, masterNode)
		}

		// 创建etcd节点
		etcdNode := portal.K8sNode{
			BaseModel: portal.BaseModel{
				CreatedAt: portal.NavyTime(now),
				UpdatedAt: portal.NavyTime(now),
			},
			NodeName:                "etcd-" + cluster.Region + "-" + cluster.Name[0:4],
			HostIP:                  "192.168.1." + string([]byte{byte(cluster.ID + 20)}),
			Role:                    "etcd",
			OSImage:                 "Ubuntu 20.04.5 LTS",
			KernelVersion:           "5.4.0-137-generic",
			KubeletVersion:          "v1.24.10",
			KubeProxyVersion:        "v1.24.10",
			ContainerRuntimeVersion: "containerd://1.6.12",
			CPULogic:                "4",
			MemLogic:                "16Gi",
			CPUCapacity:             "4",
			MemCapacity:             "16Gi",
			CPUAllocatable:          "3",
			MemAllocatable:          "14Gi",
			FSTypeRoot:              "ext4",
			DiskRoot:                "50Gi",
			DiskDocker:              "50Gi",
			DiskKubelet:             "50Gi",
			NodeCreated:             time.Now().Format(time.RFC3339),
			Status:                  "Ready",
			K8sClusterID:            cluster.ID,
			GPU:                     "none",
			DiskCount:               1,
			DiskDetail:              "sda:200GB",
			NetworkSpeed:            1000,
		}
		if err := db.Create(&etcdNode).Error; err != nil {
			log.Printf("Warning: failed to create k8s node %s: %v", etcdNode.NodeName, err)
		} else {
			mockNodes = append(mockNodes, etcdNode)
		}

		// 创建worker节点 (3个)
		for i := 1; i <= 3; i++ {
			status := "Ready"
			if i == 3 && cluster.Status == "Stopped" {
				status = "NotReady"
			}

			gpuValue := "none"
			if i == 1 {
				gpuValue = "nvidia-tesla-t4"
			}

			workerNode := portal.K8sNode{
				BaseModel: portal.BaseModel{
					CreatedAt: portal.NavyTime(now),
					UpdatedAt: portal.NavyTime(now),
				},
				NodeName:                "worker-" + cluster.Region + "-" + cluster.Name[0:4] + "-" + string([]byte{byte(i + 48)}),
				HostIP:                  "192.168.1." + string([]byte{byte(cluster.ID + 30 + int64(i))}),
				Role:                    "worker",
				OSImage:                 "Ubuntu 20.04.5 LTS",
				KernelVersion:           "5.4.0-137-generic",
				KubeletVersion:          "v1.24.10",
				KubeProxyVersion:        "v1.24.10",
				ContainerRuntimeVersion: "containerd://1.6.12",
				CPULogic:                "16",
				MemLogic:                "64Gi",
				CPUCapacity:             "16",
				MemCapacity:             "64Gi",
				CPUAllocatable:          "14",
				MemAllocatable:          "60Gi",
				FSTypeRoot:              "ext4",
				DiskRoot:                "50Gi",
				DiskDocker:              "200Gi",
				DiskKubelet:             "200Gi",
				NodeCreated:             time.Now().Format(time.RFC3339),
				Status:                  status,
				K8sClusterID:            cluster.ID,
				GPU:                     gpuValue,
				DiskCount:               2,
				DiskDetail:              "sda:500GB,sdb:500GB",
				NetworkSpeed:            1000,
			}
			if err := db.Create(&workerNode).Error; err != nil {
				log.Printf("Warning: failed to create k8s node %s: %v", workerNode.NodeName, err)
			} else {
				mockNodes = append(mockNodes, workerNode)
			}
		}
	}

	log.Printf("成功插入 %d 条K8s节点数据", len(mockNodes))
	return mockNodes
}

// insertMockK8sNodeLabels 插入K8s节点标签模拟数据
func insertMockK8sNodeLabels(db *gorm.DB, nodes []portal.K8sNode) {
	// 创建模拟数据
	var mockLabels []portal.K8sNodeLabel

	// 为每个节点创建标签
	for _, node := range nodes {
		// 设置时间
		now := time.Now()

		// 通用标签
		commonLabels := []portal.K8sNodeLabel{
			{
				BaseModel: portal.BaseModel{
					CreatedAt: portal.NavyTime(now),
					UpdatedAt: portal.NavyTime(now),
				},
				Key:    "kubernetes.io/hostname",
				Value:  node.NodeName,
				Status: "Active",
				NodeID: node.ID,
			},
			{
				BaseModel: portal.BaseModel{
					CreatedAt: portal.NavyTime(now),
					UpdatedAt: portal.NavyTime(now),
				},
				Key:    "kubernetes.io/arch",
				Value:  "amd64",
				Status: "Active",
				NodeID: node.ID,
			},
			{
				BaseModel: portal.BaseModel{
					CreatedAt: portal.NavyTime(now),
					UpdatedAt: portal.NavyTime(now),
				},
				Key:    "kubernetes.io/os",
				Value:  "linux",
				Status: "Active",
				NodeID: node.ID,
			},
		}
		mockLabels = append(mockLabels, commonLabels...)

		// 角色特定标签
		switch node.Role {
		case "master":
			masterLabels := []portal.K8sNodeLabel{
				{
					Key:    "node-role.kubernetes.io/control-plane",
					Value:  "",
					Status: "Active",
					NodeID: node.ID,
				},
				{
					Key:    "node-role.kubernetes.io/master",
					Value:  "",
					Status: "Active",
					NodeID: node.ID,
				},
			}
			mockLabels = append(mockLabels, masterLabels...)
		case "worker":
			workerLabels := []portal.K8sNodeLabel{
				{
					Key:    "node-role.kubernetes.io/worker",
					Value:  "",
					Status: "Active",
					NodeID: node.ID,
				},
			}
			mockLabels = append(mockLabels, workerLabels...)

			// GPU标签
			if node.GPU != "none" {
				gpuLabels := []portal.K8sNodeLabel{
					{
						Key:    "nvidia.com/gpu",
						Value:  "true",
						Status: "Active",
						NodeID: node.ID,
					},
					{
						Key:    "gpu.nvidia.com/model",
						Value:  node.GPU,
						Status: "Active",
						NodeID: node.ID,
					},
				}
				mockLabels = append(mockLabels, gpuLabels...)
			}
		case "etcd":
			etcdLabels := []portal.K8sNodeLabel{
				{
					Key:    "node-role.kubernetes.io/etcd",
					Value:  "",
					Status: "Active",
					NodeID: node.ID,
				},
			}
			mockLabels = append(mockLabels, etcdLabels...)
		}

		// 添加一些自定义标签
		customLabels := []portal.K8sNodeLabel{
			{
				Key:    "env",
				Value:  getEnvFromNodeName(node.NodeName),
				Status: "Active",
				NodeID: node.ID,
			},
			{
				Key:    "region",
				Value:  getRegionFromNodeName(node.NodeName),
				Status: "Active",
				NodeID: node.ID,
			},
		}
		mockLabels = append(mockLabels, customLabels...)
	}

	// 插入数据
	for i := range mockLabels {
		// 设置当前时间
		now := portal.NavyTime(time.Now())
		mockLabels[i].CreatedAt = now
		mockLabels[i].UpdatedAt = now

		if err := db.Create(&mockLabels[i]).Error; err != nil {
			log.Printf("Warning: failed to create k8s node label %s=%s: %v", mockLabels[i].Key, mockLabels[i].Value, err)
		}
	}

	log.Printf("成功插入 %d 条K8s节点标签数据", len(mockLabels))
}

// insertMockK8sNodeTaints 插入K8s节点污点模拟数据
func insertMockK8sNodeTaints(db *gorm.DB, nodes []portal.K8sNode) {
	// 创建模拟数据
	var mockTaints []portal.K8sNodeTaint

	// 为每个节点创建污点
	for _, node := range nodes {
		// 设置时间
		now := time.Now()

		// 角色特定污点
		switch node.Role {
		case "master":
			masterTaints := []portal.K8sNodeTaint{
				{
					BaseModel: portal.BaseModel{
						CreatedAt: portal.NavyTime(now),
						UpdatedAt: portal.NavyTime(now),
					},
					Key:    "node-role.kubernetes.io/master",
					Value:  "",
					Effect: "NoSchedule",
					Status: "Active",
					NodeID: node.ID,
				},
			}
			mockTaints = append(mockTaints, masterTaints...)
		case "etcd":
			etcdTaints := []portal.K8sNodeTaint{
				{
					Key:    "node-role.kubernetes.io/etcd",
					Value:  "",
					Effect: "NoSchedule",
					Status: "Active",
					NodeID: node.ID,
				},
			}
			mockTaints = append(mockTaints, etcdTaints...)
		}

		// 为不可用节点添加污点
		if node.Status == "NotReady" {
			notReadyTaints := []portal.K8sNodeTaint{
				{
					Key:    "node.kubernetes.io/not-ready",
					Value:  "",
					Effect: "NoExecute",
					Status: "Active",
					NodeID: node.ID,
				},
			}
			mockTaints = append(mockTaints, notReadyTaints...)
		}

		// 为GPU节点添加污点
		if node.GPU != "none" {
			gpuTaints := []portal.K8sNodeTaint{
				{
					Key:    "nvidia.com/gpu",
					Value:  "present",
					Effect: "NoSchedule",
					Status: "Active",
					NodeID: node.ID,
				},
			}
			mockTaints = append(mockTaints, gpuTaints...)
		}
	}

	// 插入数据
	for i := range mockTaints {
		// 设置当前时间
		now := portal.NavyTime(time.Now())
		mockTaints[i].CreatedAt = now
		mockTaints[i].UpdatedAt = now

		if err := db.Create(&mockTaints[i]).Error; err != nil {
			log.Printf("Warning: failed to create k8s node taint %s=%s:%s: %v",
				mockTaints[i].Key, mockTaints[i].Value, mockTaints[i].Effect, err)
		}
	}

	log.Printf("成功插入 %d 条K8s节点污点数据", len(mockTaints))
}

// 辅助函数：从节点名称获取环境
func getEnvFromNodeName(nodeName string) string {
	if len(nodeName) < 10 {
		return "unknown"
	}

	// 根据节点名称中的关键字判断环境
	if nodeName[7:11] == "prod" {
		return "production"
	} else if nodeName[7:11] == "test" {
		return "testing"
	} else if nodeName[7:10] == "dev" {
		return "development"
	} else if nodeName[7:10] == "uat" {
		return "uat"
	}

	return "unknown"
}

// 辅助函数：从节点名称获取区域
func getRegionFromNodeName(nodeName string) string {
	if len(nodeName) < 10 {
		return "unknown"
	}

	// 根据节点名称中的关键字判断区域
	if nodeName[12:14] == "sh" {
		return "shanghai"
	} else if nodeName[12:14] == "bj" {
		return "beijing"
	} else if nodeName[12:14] == "sz" {
		return "shenzhen"
	}

	return "unknown"
}
