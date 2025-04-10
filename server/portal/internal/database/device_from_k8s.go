package database

import (
	"log"
	"time"

	"gorm.io/gorm"

	"navy-ng/models/portal"
)

// GenerateDevicesFromK8sNodes 根据K8s节点数据生成设备数据
func GenerateDevicesFromK8sNodes(db *gorm.DB) error {
	// 清空现有设备数据
	if err := db.Exec("DELETE FROM device").Error; err != nil {
		log.Printf("Warning: failed to delete device data: %v", err)
		return err
	}

	// 获取所有K8s节点
	var nodes []portal.K8sNode
	if err := db.Find(&nodes).Error; err != nil {
		log.Printf("Error: failed to get k8s nodes: %v", err)
		return err
	}

	// 根据K8s节点数据生成设备数据
	var devices []portal.Device
	for _, node := range nodes {
		// 获取节点的集群信息
		var cluster portal.K8sCluster
		if err := db.First(&cluster, node.K8sClusterID).Error; err != nil {
			log.Printf("Warning: failed to get cluster for node %s: %v", node.NodeName, err)
			continue
		}

		// 创建设备数据
		device := portal.Device{
			DeviceID:     node.NodeName,
			IP:           node.HostIP,
			MachineType:  getNodeMachineType(node),
			Cluster:      cluster.Name,
			Role:         node.Role,
			Arch:         "x86_64",
			IDC:          getNodeIDC(node, cluster),
			Room:         getNodeRoom(node, cluster),
			Datacenter:   cluster.Region,
			Cabinet:      getNodeCabinet(node),
			Network:      getNodeNetwork(node, cluster),
			AppID:        "k8s-" + cluster.Name,
			ResourcePool: "kubernetes",
			Deleted:      "",
		}

		devices = append(devices, device)
	}

	// 插入设备数据
	for _, device := range devices {
		// 设置创建时间和更新时间
		now := time.Now()
		device.CreatedAt = portal.NavyTime(now)
		device.UpdatedAt = portal.NavyTime(now)

		if err := db.Create(&device).Error; err != nil {
			log.Printf("Warning: failed to create device %s: %v", device.DeviceID, err)
		}
	}

	log.Printf("成功从K8s节点生成并插入 %d 条设备数据", len(devices))
	return nil
}

// 辅助函数：获取节点的机器类型
func getNodeMachineType(node portal.K8sNode) string {
	switch node.Role {
	case "master":
		return "k8s-master"
	case "etcd":
		return "k8s-etcd"
	case "worker":
		if node.GPU != "none" {
			return "k8s-worker-gpu"
		}
		return "k8s-worker"
	default:
		return "k8s-node"
	}
}

// 辅助函数：获取节点的IDC
func getNodeIDC(node portal.K8sNode, cluster portal.K8sCluster) string {
	switch cluster.Region {
	case "上海":
		return "SH"
	case "北京":
		return "BJ"
	case "深圳":
		return "SZ"
	default:
		return "DC-" + cluster.Region
	}
}

// 辅助函数：获取节点的机房
func getNodeRoom(node portal.K8sNode, cluster portal.K8sCluster) string {
	return "Room-" + cluster.Region + "-" + node.Role
}

// 辅助函数：获取节点的机柜
func getNodeCabinet(node portal.K8sNode) string {
	switch node.Role {
	case "master":
		return "Rack-Master"
	case "etcd":
		return "Rack-ETCD"
	case "worker":
		if node.GPU != "none" {
			return "Rack-GPU"
		}
		return "Rack-Worker"
	default:
		return "Rack-Default"
	}
}

// 辅助函数：获取节点的网络
func getNodeNetwork(node portal.K8sNode, cluster portal.K8sCluster) string {
	switch cluster.Status {
	case "Running":
		return "Production"
	case "Stopped":
		return "Maintenance"
	default:
		return "Default"
	}
}
