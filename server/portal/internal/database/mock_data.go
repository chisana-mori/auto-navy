package database

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"navy-ng/models/portal"
)

// Constants for mock data generation
const (
	statusRunning = "Running"
	statusReady   = "Ready"
	statusOffline = "Offline"

	clusterProdID int64 = 1
	clusterTestID int64 = 2

	labelSourceInternal = 0
	labelSourceExternal = 1

	taintTypeSystem   = "system"
	taintTypeCustom   = "custom"
	taintTypeHardware = "hardware"

	effectNoSchedule = "NoSchedule"
	effectNoExecute  = "NoExecute"

	colorGreen      = "#4CAF50"
	colorBlue       = "#2196F3"
	colorYellow     = "#FFC107"
	colorPurple     = "#9C27B0"
	colorPink       = "#E91E63"
	colorRed        = "#F44336"
	colorDeepPurple = "#673AB7"
	colorCyan       = "#00BCD4"
	colorOrange     = "#FF9800"
	colorBrown      = "#795548"

	// Resource pool types
	resourcePoolTypeTotal     = "Total"
	resourcePoolTypeIntel     = "Intel"
	resourcePoolTypeARM       = "ARM"
	resourcePoolTypeHG        = "HG"
	resourcePoolTypeGPU       = "GPU"
	resourcePoolTypeWithTaint = "WithTaint"
	resourcePoolTypeCommon    = "Common"

	// Action types
	actionTypePoolEntry = "pool_entry"
	actionTypePoolExit  = "pool_exit"
)

// ClearAndSeedDatabase clears relevant tables and seeds them with consistent mock data.
// It returns the seeded clusters for further use (e.g., seeding snapshots).
func ClearAndSeedDatabase(db *gorm.DB) ([]portal.K8sCluster, error) {
	log.Println("Starting database clearing and seeding...")

	// 1. Clear existing data in reverse dependency order (or tables that can be cleared)
	tablesToClear := []string{
		"ops_job",                              // Assuming no critical FKs pointing to it
		"f5_info",                              // Assuming no critical FKs pointing to it
		"resource_pool_device_matching_policy", // Depends on query_template
		"query_template",                       // Assuming no critical FKs pointing to it
		"device_app",                           // 添加DeviceApp表，依赖于device表
		"k8s_node_taint",                       // Depends on k8s_node, taint_feature
		"k8s_node_label",                       // Depends on k8s_node, label_feature
		"device",                               // Related to k8s_node, k8s_etcd
		"k8s_node",                             // Depends on k8s_cluster
		"k8s_etcd",                             // Depends on k8s_cluster
		"label_feature_value",                  // Depends on label_feature
		"label_feature",                        // Independent
		"taint_feature",                        // Independent
		"k8s_cluster_resource_snapshot",        // New table for resource snapshots
		"k8s_cluster",                          // Independent
	}

	log.Println("Clearing tables...")
	for _, table := range tablesToClear {
		if err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
			// Log warning but continue if possible, crucial tables might block seeding
			log.Printf("Warning: failed to clear table %s: %v. Seeding might be incomplete.", table, err)
		}
	}

	// Optional: Reset auto-increment sequences if using SQLite
	sequencesToReset := []string{
		"ops_job",
		"f5_info",
		"resource_pool_device_matching_policy",
		"query_template",
		"device_app", // 添加DeviceApp表序列重置
		"k8s_node_label",
		"k8s_node_taint",
		"device",
		"k8s_node",
		"k8s_etcd",
		"label_feature_value",
		"label_feature",
		"taint_feature",
		"k8s_cluster_resource_snapshot", // New table for resource snapshots
		"k8s_cluster",
	}
	log.Println("Resetting sequences (SQLite only)...")
	for _, seq := range sequencesToReset {
		// This command is specific to SQLite
		if err := db.Exec(fmt.Sprintf("DELETE FROM sqlite_sequence WHERE name='%s'", seq)).Error; err != nil {
			// Log warning, as this might fail on other DBs or if the table didn't exist
			// log.Printf("Warning: failed to reset sequence for %s: %v", seq, err)
		}
	}

	// 2. Seed data in dependency order
	log.Println("Seeding K8s Clusters...")
	clusters, err := seedK8sClusters(db)
	if err != nil {
		return nil, fmt.Errorf("failed to seed k8s clusters: %w", err)
	}

	log.Println("Seeding Label Features...")
	labelFeatures, err := seedLabelFeatures(db)
	if err != nil {
		return nil, fmt.Errorf("failed to seed label features: %w", err)
	}

	log.Println("Seeding Taint Features...")
	taintFeatures, err := seedTaintFeatures(db)
	if err != nil {
		return nil, fmt.Errorf("failed to seed taint features: %w", err)
	}

	log.Println("Seeding K8s Nodes...")
	nodes, err := seedK8sNodes(db, clusters)
	if err != nil {
		return nil, fmt.Errorf("failed to seed k8s nodes: %w", err)
	}

	log.Println("Seeding K8s ETCD...")
	etcds, err := seedK8sEtcd(db, clusters)
	if err != nil {
		return nil, fmt.Errorf("failed to seed k8s etcd: %w", err)
	}

	log.Println("Seeding Devices...")
	devices, err := seedDevices(db, nodes, etcds)
	if err != nil {
		return nil, fmt.Errorf("failed to seed devices: %w", err)
	}
	// Note: Devices are seeded, but their 'role' and 'cluster' might be updated later based on actual relations

	log.Println("Seeding K8s Node Labels...")
	if err := seedK8sNodeLabels(db, nodes, labelFeatures); err != nil {
		return nil, fmt.Errorf("failed to seed k8s node labels: %w", err)
	}

	log.Println("Seeding K8s Node Taints...")
	if err := seedK8sNodeTaints(db, nodes, taintFeatures); err != nil {
		return nil, fmt.Errorf("failed to seed k8s node taints: %w", err)
	}

	log.Println("Updating Device Roles/Clusters based on K8s relations...")
	if err := updateDeviceRelations(db, nodes, etcds, clusters); err != nil {
		return nil, fmt.Errorf("failed to update device relations: %w", err)
	}

	// 添加DeviceApp数据生成
	log.Println("Seeding Device Apps...")
	if err := seedDeviceApps(db, devices); err != nil {
		return nil, fmt.Errorf("failed to seed device apps: %w", err)
	}

	// Seed other independent data (optional)
	log.Println("Seeding F5 Info...")
	if err := seedF5Info(db, clusters); err != nil { // Pass clusters if F5 needs cluster IDs
		log.Printf("Warning: failed to seed F5 info: %v", err)
	}

	log.Println("Seeding Ops Jobs...")
	if err := seedOpsJobs(db); err != nil {
		log.Printf("Warning: failed to seed Ops Jobs: %v", err)
	}

	log.Println("Seeding K8s Resource Snapshots...")
	if err := seedK8sResourceSnapshots(db, clusters); err != nil {
		return nil, fmt.Errorf("failed to seed k8s resource snapshots: %w", err)
	}

	log.Println("Seeding Query Templates...")
	if err := seedQueryTemplates(db); err != nil {
		log.Printf("Warning: failed to seed query templates: %v", err)
	}

	log.Println("Seeding Resource Pool Device Matching Policies...")
	if err := seedResourcePoolDeviceMatchingPolicies(db); err != nil {
		log.Printf("Warning: failed to seed resource pool device matching policies: %v", err)
	}

	log.Println("Database seeding completed successfully.")
	return clusters, nil // Return the seeded clusters
}

// --- Seeding Functions --- //

// seedQueryTemplates generates mock data for query_template table.
func seedQueryTemplates(db *gorm.DB) error {
	// Create some basic query templates
	templates := []portal.QueryTemplate{
		{
			BaseModel:   portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Name:        "生产环境设备",
			Description: "查询所有生产环境的设备",
			Groups:      `[{"id":"group1","blocks":[{"id":"block1","type":"device","conditionType":"equal","key":"status","value":"Running","operator":"and"}],"operator":"and"}]`,
			CreatedBy:   "system",
			UpdatedBy:   "system",
		},
		{
			BaseModel:   portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Name:        "GPU设备",
			Description: "查询所有GPU设备",
			Groups:      `[{"id":"group1","blocks":[{"id":"block1","type":"nodeLabel","conditionType":"equal","key":"nvidia.com/gpu","value":"true","operator":"and"}],"operator":"and"}]`,
			CreatedBy:   "system",
			UpdatedBy:   "system",
		},
		{
			BaseModel:   portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Name:        "上海区域设备",
			Description: "查询所有上海区域的设备",
			Groups:      `[{"id":"group1","blocks":[{"id":"block1","type":"device","conditionType":"equal","key":"idc","value":"shanghai","operator":"and"}],"operator":"and"}]`,
			CreatedBy:   "system",
			UpdatedBy:   "system",
		},
		{
			BaseModel:   portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Name:        "物理机设备",
			Description: "查询所有物理机设备",
			Groups:      `[{"id":"group1","blocks":[{"id":"block1","type":"device","conditionType":"equal","key":"infra_type","value":"physical","operator":"and"}],"operator":"and"}]`,
			CreatedBy:   "system",
			UpdatedBy:   "system",
		},
	}

	if err := db.Create(&templates).Error; err != nil {
		return fmt.Errorf("failed to create query templates: %w", err)
	}

	log.Printf("Inserted %d Query Templates", len(templates))
	return nil
}

// seedResourcePoolDeviceMatchingPolicies generates mock data for resource_pool_device_matching_policy table.
func seedResourcePoolDeviceMatchingPolicies(db *gorm.DB) error {
	// 获取已创建的查询模板
	var templates []portal.QueryTemplate
	if err := db.Find(&templates).Error; err != nil {
		return fmt.Errorf("failed to get query templates: %w", err)
	}

	// 创建模板名称到ID的映射
	templateMap := make(map[string]uint)
	for _, template := range templates {
		templateMap[template.Name] = uint(template.ID)
	}

	// 获取默认模板ID（如果没有模板，使用1作为默认值）
	defaultTemplateID := uint(1)
	if len(templates) > 0 {
		defaultTemplateID = uint(templates[0].ID)
	}

	// 获取特定模板ID
	getTemplateID := func(name string) uint {
		if id, ok := templateMap[name]; ok {
			return id
		}
		return defaultTemplateID
	}

	// Create policies for different resource pool types and action types
	policies := []portal.ResourcePoolDeviceMatchingPolicy{
		{
			BaseModel:        portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Name:             "Total资源池入池策略",
			Description:      "Total资源池入池设备匹配策略",
			ResourcePoolType: resourcePoolTypeTotal,
			ActionType:       actionTypePoolEntry,
			QueryTemplateID:  getTemplateID("生产环境设备"),
			Status:           "enabled",
			CreatedBy:        "system",
			UpdatedBy:        "system",
		},
		{
			BaseModel:        portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Name:             "Total资源池退池策略",
			Description:      "Total资源池退池设备匹配策略",
			ResourcePoolType: resourcePoolTypeTotal,
			ActionType:       actionTypePoolExit,
			QueryTemplateID:  getTemplateID("生产环境设备"),
			Status:           "enabled",
			CreatedBy:        "system",
			UpdatedBy:        "system",
		},
		{
			BaseModel:        portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Name:             "GPU资源池入池策略",
			Description:      "GPU资源池入池设备匹配策略",
			ResourcePoolType: resourcePoolTypeGPU,
			ActionType:       actionTypePoolEntry,
			QueryTemplateID:  getTemplateID("GPU设备"),
			Status:           "enabled",
			CreatedBy:        "system",
			UpdatedBy:        "system",
		},
		{
			BaseModel:        portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Name:             "GPU资源池退池策略",
			Description:      "GPU资源池退池设备匹配策略",
			ResourcePoolType: resourcePoolTypeGPU,
			ActionType:       actionTypePoolExit,
			QueryTemplateID:  getTemplateID("GPU设备"),
			Status:           "enabled",
			CreatedBy:        "system",
			UpdatedBy:        "system",
		},
		{
			BaseModel:        portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Name:             "Intel资源池入池策略",
			Description:      "Intel资源池入池设备匹配策略",
			ResourcePoolType: resourcePoolTypeIntel,
			ActionType:       actionTypePoolEntry,
			QueryTemplateID:  getTemplateID("物理机设备"),
			Status:           "enabled",
			CreatedBy:        "system",
			UpdatedBy:        "system",
		},
		{
			BaseModel:        portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Name:             "Intel资源池退池策略",
			Description:      "Intel资源池退池设备匹配策略",
			ResourcePoolType: resourcePoolTypeIntel,
			ActionType:       actionTypePoolExit,
			QueryTemplateID:  getTemplateID("物理机设备"),
			Status:           "disabled", // 这个策略是禁用的
			CreatedBy:        "system",
			UpdatedBy:        "system",
		},
	}

	if err := db.Create(&policies).Error; err != nil {
		return fmt.Errorf("failed to create resource pool device matching policies: %w", err)
	}

	log.Printf("Inserted %d Resource Pool Device Matching Policies", len(policies))
	return nil
}

// seedK8sResourceSnapshots generates mock data for k8s_cluster_resource_snapshot table for the last few days.
func seedK8sResourceSnapshots(db *gorm.DB, clusters []portal.K8sCluster) error {
	log.Println("Seeding Daily Resource Snapshots...")

	resourceTypesToSeed := []portal.ResourceType{
		portal.Total,
		portal.Intel,
		portal.ARM,
		portal.HG,
		portal.GPU,
		portal.WithTaint,
		portal.Common,
	}

	now := time.Now()
	var snapshotsToInsert []portal.ResourceSnapshot

	// Generate data for the last 7 days
	for d := 0; d < 7; d++ {
		day := now.AddDate(0, 0, -d)
		// Set time to a specific hour to make snapshots appear daily at the same time
		snapshotTime := time.Date(day.Year(), day.Month(), day.Day(), 10, 0, 0, 0, day.Location())

		for _, cluster := range clusters {
			for _, resType := range resourceTypesToSeed {
				// Generate some varying mock data
				baseCPU := 1000.0 * (1.0 - float64(d)*0.05) // Decrease slightly each day
				baseMem := 2000.0 * (1.0 - float64(d)*0.03)
				baseNodes := 100 - int64(d*2)
				basePods := 1500 - int64(d*50)

				// Add some randomness
				cpuCapacity := baseCPU + rand.Float64()*baseCPU*0.1
				memCapacity := baseMem + rand.Float64()*baseMem*0.1
				nodeCount := baseNodes + rand.Int63n(10) - 5
				podCount := basePods + rand.Int63n(100) - 50

				// Ensure counts are not negative
				if nodeCount < 0 {
					nodeCount = 0
				}
				if podCount < 0 {
					podCount = 0
				}

				// Simulate usage and requests (vary based on resource type and day)
				cpuRequestRatio := 0.6 + rand.Float64()*0.2                      // 60-80%
				memRequestRatio := 0.5 + rand.Float64()*0.2                      // 50-70%
				maxCpuUsageRatio := cpuRequestRatio * (1.0 + rand.Float64()*0.1) // Slightly higher than request
				maxMemUsageRatio := memRequestRatio * (1.0 + rand.Float64()*0.1)

				cpuRequest := cpuCapacity * cpuRequestRatio
				memRequest := memCapacity * memRequestRatio

				// Adjust values for specific resource types if needed (simplified for now)
				if resType == portal.GPU {
					cpuCapacity *= 0.5 // Assume GPU nodes have less general CPU
					memCapacity *= 0.8
					// Add GPU specific metrics if the model supported them
				}
				if resType == portal.WithTaint {
					nodeCount = baseNodes / 10 // Fewer tainted nodes
					if nodeCount < 1 {
						nodeCount = 1
					}
				}
				if resType == portal.Common {
					cpuCapacity *= 0.9 // Common nodes are the majority
					memCapacity *= 0.9
				}

				snapshot := portal.ResourceSnapshot{
					BaseModel:           portal.BaseModel{CreatedAt: portal.NavyTime(snapshotTime), UpdatedAt: portal.NavyTime(snapshotTime)},
					ClusterID:           uint(cluster.ID),
					ResourceType:        string(resType),
					CpuCapacity:         cpuCapacity,
					MemoryCapacity:      memCapacity,
					CpuRequest:          cpuRequest,
					MemRequest:          memRequest,
					NodeCount:           nodeCount,
					BMCount:             nodeCount / 2, // Mock BM/VM split
					VMCount:             nodeCount - nodeCount/2,
					MaxCpuUsageRatio:    maxCpuUsageRatio * 100, // Store as percentage
					MaxMemoryUsageRatio: maxMemUsageRatio * 100,
					PerNodeCpuRequest:   cpuRequest / float64(nodeCount+1), // Avoid division by zero
					PerNodeMemRequest:   memRequest / float64(nodeCount+1),
					PodCount:            podCount,
				}
				snapshotsToInsert = append(snapshotsToInsert, snapshot)
			}
		}
	}

	if len(snapshotsToInsert) > 0 {
		if err := db.CreateInBatches(&snapshotsToInsert, 100).Error; err != nil {
			return fmt.Errorf("failed to create resource snapshots: %w", err)
		}
		log.Printf("Inserted %d Daily Resource Snapshots", len(snapshotsToInsert))
	} else {
		log.Println("No Daily Resource Snapshots to insert.")
	}

	return nil
}

// --- Seeding Functions --- //

func seedK8sClusters(db *gorm.DB) ([]portal.K8sCluster, error) {
	// Create clusters individually to handle the ID setting
	prodCluster := portal.K8sCluster{
		ClusterName: "cluster-prod",
		Zone:        "shanghai",
		ApiServer:   "https://k8s-prod.example.com:6443",
		Status:      statusRunning,
		ClusterID:   "cluster-prod-001", // 添加唯一的 cluster_id
	}
	if err := db.Create(&prodCluster).Error; err != nil {
		return nil, err
	}

	testCluster := portal.K8sCluster{
		ClusterName: "cluster-test",
		Zone:        "beijing",
		ApiServer:   "https://k8s-test.example.com:6443",
		Status:      statusRunning,
		ClusterID:   "cluster-test-001", // 添加唯一的 cluster_id
	}
	if err := db.Create(&testCluster).Error; err != nil {
		return nil, err
	}

	// Return the created clusters
	mockClusters := []portal.K8sCluster{prodCluster, testCluster}
	log.Printf("Inserted %d K8s Clusters", len(mockClusters))
	return mockClusters, nil
}

func seedLabelFeatures(db *gorm.DB) ([]portal.LabelManagement, error) {
	labels := []portal.LabelManagement{
		{Name: "Kubernetes Hostname", Key: "kubernetes.io/hostname", Source: labelSourceExternal, IsControl: true, Range: "node", IDC: "all", Status: 0, IsDenyList: false, Color: colorGreen},
		{Name: "GPU Support", Key: "nvidia.com/gpu", Source: labelSourceExternal, IsControl: true, Range: "node", IDC: "all", Status: 0, IsDenyList: false, Color: colorBlue},
		{Name: "Environment", Key: "env", Source: labelSourceInternal, IsControl: true, Range: "node", IDC: "all", Status: 0, IsDenyList: false, Color: colorYellow},
		{Name: "Region", Key: "topology.kubernetes.io/region", Source: labelSourceExternal, IsControl: true, Range: "node", IDC: "all", Status: 0, IsDenyList: false, Color: colorPurple},
		{Name: "Custom Workload", Key: "workload-type", Source: labelSourceInternal, IsControl: true, Range: "node", IDC: "all", Status: 0, IsDenyList: false, Color: colorPink},
	}

	// Add Timestamps before creation
	for i := range labels {
		currentTime := time.Now()
		labels[i].CreatedAt = portal.NavyTime(currentTime)
		labels[i].UpdatedAt = portal.NavyTime(currentTime)
	}

	if err := db.Create(&labels).Error; err != nil {
		return nil, err
	}

	// Seed Label Values (Optional, based on requirements)
	labelValuesData := map[string][]string{
		"kubernetes.io/hostname":        {"prod-node-1", "prod-node-2", "test-node-1"},
		"nvidia.com/gpu":                {"true", "false"},
		"env":                           {"prod", "staging", "dev"},
		"topology.kubernetes.io/region": {"shanghai", "beijing"},
		"workload-type":                 {"batch", "web", "database"},
	}

	labelMap := make(map[string]int64)
	for _, l := range labels {
		labelMap[l.Key] = l.ID
	}

	var valuesToInsert []portal.LabelValue
	for key, values := range labelValuesData {
		if labelID, ok := labelMap[key]; ok {
			for _, value := range values {
				valuesToInsert = append(valuesToInsert, portal.LabelValue{
					LabelID: labelID,
					Value:   value,
				})
			}
		}
	}
	if len(valuesToInsert) > 0 {
		if err := db.Create(&valuesToInsert).Error; err != nil {
			log.Printf("Warning: failed to seed label values: %v", err)
			// Continue even if values fail
		}
	}

	log.Printf("Inserted %d Label Features", len(labels))
	return labels, nil
}

func seedTaintFeatures(db *gorm.DB) ([]portal.TaintManagement, error) {
	taints := []portal.TaintManagement{
		{Key: "node-role.kubernetes.io/master", Value: "", Effect: effectNoSchedule, Description: "Master node taint", Type: taintTypeSystem, Status: 0, Color: colorRed},
		{Key: "node-role.kubernetes.io/control-plane", Value: "", Effect: effectNoSchedule, Description: "Control plane node taint", Type: taintTypeSystem, Status: 0, Color: colorRed},
		{Key: "nvidia.com/gpu", Value: "present", Effect: effectNoSchedule, Description: "GPU node taint", Type: taintTypeHardware, Status: 0, Color: colorCyan},
		{Key: "custom/maintenance", Value: "true", Effect: effectNoSchedule, Description: "Maintenance mode taint", Type: taintTypeCustom, Status: 0, Color: colorOrange},
		{Key: "special-workload", Value: "true", Effect: effectNoExecute, Description: "Taint for special workloads", Type: taintTypeCustom, Status: 0, Color: colorBrown},
	}

	// Add Timestamps before creation
	for i := range taints {
		currentTime := time.Now()
		taints[i].CreatedAt = portal.NavyTime(currentTime)
		taints[i].UpdatedAt = portal.NavyTime(currentTime)
	}

	if err := db.Create(&taints).Error; err != nil {
		return nil, err
	}
	log.Printf("Inserted %d Taint Features", len(taints))
	return taints, nil
}

func seedK8sNodes(db *gorm.DB, clusters []portal.K8sCluster) ([]portal.K8sNode, error) {
	nodes := []portal.K8sNode{
		{
			BaseModel:     portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			NodeName:      "prod-node-1", // Matches a device CICode
			HostIP:        "10.1.1.11",   // Matches a device IP
			Role:          "worker",
			OSImage:       "Ubuntu 20.04.4 LTS",
			KernelVersion: "5.4.0-109-generic",
			Status:        statusReady,
			K8sClusterID:  clusterProdID,
			DiskCount:     2,
			DiskDetail:    "sda:1TB,sdb:1TB",
			NetworkSpeed:  10000, // 10Gbps
			GPU:           "none",
		},
		{
			BaseModel:     portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			NodeName:      "prod-node-2", // Matches a device CICode
			HostIP:        "10.1.1.12",
			Role:          "master",
			OSImage:       "Ubuntu 20.04.4 LTS",
			KernelVersion: "5.4.0-109-generic",
			Status:        statusReady,
			K8sClusterID:  clusterProdID,
			DiskCount:     1,
			DiskDetail:    "sda:500GB",
			NetworkSpeed:  10000,
			GPU:           "nvidia-tesla-t4",
		},
		{
			BaseModel:     portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			NodeName:      "test-node-1", // Matches a device CICode
			HostIP:        "10.2.1.11",   // Matches a device IP
			Role:          "worker",
			OSImage:       "CentOS Stream 8",
			KernelVersion: "4.18.0-373.el8.x86_64",
			Status:        statusReady,
			K8sClusterID:  clusterTestID,
			DiskCount:     1,
			DiskDetail:    "vda:200GB",
			NetworkSpeed:  1000,
			GPU:           "none",
		},
		{
			BaseModel:     portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			NodeName:      "offline-node", // Does not match any device
			HostIP:        "10.1.5.5",
			Role:          "worker",
			OSImage:       "Ubuntu 18.04.6 LTS",
			KernelVersion: "4.15.0-171-generic",
			Status:        statusOffline,
			K8sClusterID:  clusterProdID,
			DiskCount:     1,
			DiskDetail:    "sda:500GB",
			NetworkSpeed:  1000,
			GPU:           "none",
		},
	}
	if err := db.Create(&nodes).Error; err != nil {
		return nil, err
	}
	log.Printf("Inserted %d K8s Nodes", len(nodes))
	return nodes, nil
}

func seedK8sEtcd(db *gorm.DB, clusters []portal.K8sCluster) ([]portal.K8sETCD, error) {
	etcds := []portal.K8sETCD{
		{
			BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Instance:  "10.1.1.21", // Matches a device IP
			Role:      "etcd-member-prod",
			ServerId:  "prod-etcd-1",
			ClusterID: int(clusterProdID),
		},
		{
			BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Instance:  "10.1.1.22", // Matches a device IP
			Role:      "etcd-member-prod",
			ServerId:  "prod-etcd-2",
			ClusterID: int(clusterProdID),
		},
		{
			BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now()), UpdatedAt: portal.NavyTime(time.Now())},
			Instance:  "10.2.1.21", // Matches a device IP
			Role:      "etcd-member-test",
			ServerId:  "test-etcd-1",
			ClusterID: int(clusterTestID),
		},
	}
	if err := db.Create(&etcds).Error; err != nil {
		return nil, err
	}
	log.Printf("Inserted %d K8s ETCD instances", len(etcds))
	return etcds, nil
}

func seedDevices(db *gorm.DB, nodes []portal.K8sNode, etcds []portal.K8sETCD) ([]portal.Device, error) {
	devices := []portal.Device{
		// Devices matching K8s Nodes
		{
			BaseModel: portal.BaseModel{ID: 1001},
			CICode:    "prod-node-1", // Match node[0].NodeName
			IP:        "10.1.1.11",   // Match node[0].HostIP
			ArchType:  "x86_64", IDC: "shanghai", Room: "A1", Cabinet: "R1", CabinetNO: "U10", InfraType: "physical", NetZone: "prod-net", Group: "compute", Status: statusRunning,
			AppID: "compute-1001-init", // 初始AppID
		},
		{
			BaseModel: portal.BaseModel{ID: 1002},
			CICode:    "prod-node-2", // Match node[1].NodeName
			IP:        "10.1.1.12",   // Match node[1].HostIP
			ArchType:  "x86_64", IDC: "shanghai", Room: "A1", Cabinet: "R1", CabinetNO: "U11", InfraType: "physical", NetZone: "prod-net", Group: "gpu-compute", Status: statusRunning,
			AppID: "gpu-compute-1002-init", // 初始AppID
		},
		{
			BaseModel: portal.BaseModel{ID: 1003},
			CICode:    "test-node-1", // Match node[2].NodeName
			IP:        "10.2.1.11",   // Match node[2].HostIP
			ArchType:  "x86_64", IDC: "beijing", Room: "B1", Cabinet: "R5", CabinetNO: "U05", InfraType: "virtual", NetZone: "test-net", Group: "compute", Status: statusRunning,
			AppID: "compute-1003-init", // 初始AppID
		},
		// Devices matching K8s ETCD
		{
			BaseModel: portal.BaseModel{ID: 1004},
			CICode:    "etcd-host-1",
			IP:        "10.1.1.21", // Match etcd[0].Instance
			ArchType:  "x86_64", IDC: "shanghai", Room: "A2", Cabinet: "R2", CabinetNO: "U01", InfraType: "physical", NetZone: "mgmt-net", Group: "etcd", Status: statusRunning,
			AppID: "etcd-1004-init", // 初始AppID
		},
		{
			BaseModel: portal.BaseModel{ID: 1005},
			CICode:    "etcd-host-2",
			IP:        "10.1.1.22", // Match etcd[1].Instance
			ArchType:  "x86_64", IDC: "shanghai", Room: "A2", Cabinet: "R2", CabinetNO: "U02", InfraType: "physical", NetZone: "mgmt-net", Group: "etcd", Status: statusRunning,
			AppID: "etcd-1005-init", // 初始AppID
		},
		{
			BaseModel: portal.BaseModel{ID: 1006},
			CICode:    "etcd-host-3",
			IP:        "10.2.1.21", // Match etcd[2].Instance
			ArchType:  "x86_64", IDC: "beijing", Room: "B2", Cabinet: "R6", CabinetNO: "U01", InfraType: "physical", NetZone: "mgmt-net", Group: "etcd", Status: statusRunning,
			AppID: "etcd-1006-init", // 初始AppID
		},
		// Unrelated Device
		{
			BaseModel: portal.BaseModel{ID: 1007},
			CICode:    "storage-svr-1",
			IP:        "10.5.1.100", // Does not match any node or etcd
			ArchType:  "x86_64", IDC: "shanghai", Room: "C1", Cabinet: "R10", CabinetNO: "U20", InfraType: "physical", NetZone: "storage-net", Group: "storage", Status: statusRunning,
			AppID: "storage-1007-init", // 初始AppID
		},
	}

	// Set Timestamps before creation
	now := portal.NavyTime(time.Now())
	for i := range devices {
		devices[i].CreatedAt = now
		devices[i].UpdatedAt = now
	}

	if err := db.Create(&devices).Error; err != nil {
		return nil, err
	}
	log.Printf("Inserted %d Devices", len(devices))
	return devices, nil
}

func seedK8sNodeLabels(db *gorm.DB, nodes []portal.K8sNode, features []portal.LabelManagement) error {
	// Create a map for easy feature lookup by key
	featureMap := make(map[string]portal.LabelManagement)
	for _, f := range features {
		featureMap[f.Key] = f
	}

	var labelsToInsert []portal.K8sNodeLabel
	now := portal.NavyTime(time.Now())

	// Add labels to specific nodes
	// Node 0 (prod-node-1)
	if _, ok := featureMap["env"]; ok {
		labelsToInsert = append(labelsToInsert, portal.K8sNodeLabel{BaseModel: portal.BaseModel{CreatedAt: now}, NodeID: nodes[0].ID, Key: "env", Value: "prod"})
	}
	if _, ok := featureMap["workload-type"]; ok {
		labelsToInsert = append(labelsToInsert, portal.K8sNodeLabel{BaseModel: portal.BaseModel{CreatedAt: now}, NodeID: nodes[0].ID, Key: "workload-type", Value: "web"})
	}
	if _, ok := featureMap["topology.kubernetes.io/region"]; ok {
		labelsToInsert = append(labelsToInsert, portal.K8sNodeLabel{BaseModel: portal.BaseModel{CreatedAt: now}, NodeID: nodes[0].ID, Key: "topology.kubernetes.io/region", Value: "shanghai"})
	}

	// Node 1 (prod-node-2)
	if _, ok := featureMap["env"]; ok {
		labelsToInsert = append(labelsToInsert, portal.K8sNodeLabel{BaseModel: portal.BaseModel{CreatedAt: now}, NodeID: nodes[1].ID, Key: "env", Value: "prod"})
	}
	if _, ok := featureMap["nvidia.com/gpu"]; ok {
		labelsToInsert = append(labelsToInsert, portal.K8sNodeLabel{BaseModel: portal.BaseModel{CreatedAt: now}, NodeID: nodes[1].ID, Key: "nvidia.com/gpu", Value: "true"})
	}
	if _, ok := featureMap["topology.kubernetes.io/region"]; ok {
		labelsToInsert = append(labelsToInsert, portal.K8sNodeLabel{BaseModel: portal.BaseModel{CreatedAt: now}, NodeID: nodes[1].ID, Key: "topology.kubernetes.io/region", Value: "shanghai"})
	}

	// Node 2 (test-node-1)
	if _, ok := featureMap["env"]; ok {
		labelsToInsert = append(labelsToInsert, portal.K8sNodeLabel{BaseModel: portal.BaseModel{CreatedAt: now}, NodeID: nodes[2].ID, Key: "env", Value: "test"})
	}
	if _, ok := featureMap["topology.kubernetes.io/region"]; ok {
		labelsToInsert = append(labelsToInsert, portal.K8sNodeLabel{BaseModel: portal.BaseModel{CreatedAt: now}, NodeID: nodes[2].ID, Key: "topology.kubernetes.io/region", Value: "beijing"})
	}

	if len(labelsToInsert) > 0 {
		if err := db.Create(&labelsToInsert).Error; err != nil {
			return err
		}
	}
	log.Printf("Inserted %d K8s Node Labels", len(labelsToInsert))
	return nil
}

func seedK8sNodeTaints(db *gorm.DB, nodes []portal.K8sNode, features []portal.TaintManagement) error {
	// Create a map for easy feature lookup by key
	featureMap := make(map[string]portal.TaintManagement)
	for _, f := range features {
		featureMap[f.Key] = f
	}

	var taintsToInsert []portal.K8sNodeTaint
	now := portal.NavyTime(time.Now())

	// Add taints to specific nodes
	// Node 1 (prod-node-2, master/gpu)
	if f, ok := featureMap["node-role.kubernetes.io/master"]; ok {
		taintsToInsert = append(taintsToInsert, portal.K8sNodeTaint{BaseModel: portal.BaseModel{CreatedAt: now}, NodeID: nodes[1].ID, Key: f.Key, Value: f.Value, Effect: f.Effect})
	}
	if f, ok := featureMap["nvidia.com/gpu"]; ok {
		taintsToInsert = append(taintsToInsert, portal.K8sNodeTaint{BaseModel: portal.BaseModel{CreatedAt: now}, NodeID: nodes[1].ID, Key: f.Key, Value: f.Value, Effect: f.Effect})
	}

	// Node 3 (offline-node)
	if f, ok := featureMap["custom/maintenance"]; ok {
		taintsToInsert = append(taintsToInsert, portal.K8sNodeTaint{BaseModel: portal.BaseModel{CreatedAt: now}, NodeID: nodes[3].ID, Key: f.Key, Value: f.Value, Effect: f.Effect})
	}

	if len(taintsToInsert) > 0 {
		if err := db.Create(&taintsToInsert).Error; err != nil {
			return err
		}
	}
	log.Printf("Inserted %d K8s Node Taints", len(taintsToInsert))
	return nil
}

// updateDeviceRelations updates device 'role' and 'cluster' based on matched k8s_node/k8s_etcd.
func updateDeviceRelations(db *gorm.DB, nodes []portal.K8sNode, etcds []portal.K8sETCD, clusters []portal.K8sCluster) error {
	// Create a map for fast lookup of cluster by ID
	clusterMap := make(map[int64]portal.K8sCluster)
	for _, c := range clusters {
		clusterMap[int64(c.ID)] = c
	}

	// Update based on k8s_node
	log.Println("Updating devices based on k8s_node matches...")
	for _, node := range nodes {
		cluster, ok := clusterMap[node.K8sClusterID]
		if !ok {
			log.Printf("Warning: Cluster ID %d not found for node %s", node.K8sClusterID, node.NodeName)
			continue
		}

		updates := map[string]interface{}{
			"role":       node.Role,
			"cluster":    cluster.ClusterName,
			"cluster_id": int(node.K8sClusterID), // Convert int64 to int for Device model
		}
		// Use case-insensitive matching for ci_code and nodename
		result := db.Model(&portal.Device{}).Where("LOWER(ci_code) = LOWER(?) AND ci_code != ''", node.NodeName).Updates(updates)
		if result.Error != nil {
			log.Printf("Warning: failed to update device for node %s: %v", node.NodeName, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("Updated %d device(s) for node %s", result.RowsAffected, node.NodeName)
		}
	}

	// Update based on k8s_etcd
	log.Println("Updating devices based on k8s_etcd matches (if not already updated by node)...")
	for _, etcd := range etcds {
		cluster, ok := clusterMap[int64(etcd.ClusterID)] // Convert int to int64 for map lookup
		if !ok {
			log.Printf("Warning: Cluster ID %d not found for etcd %s", etcd.ClusterID, etcd.Instance)
			continue
		}

		updates := map[string]interface{}{
			"role":       etcd.Role, // Use Role from K8sETCD model
			"cluster":    cluster.ClusterName,
			"cluster_id": etcd.ClusterID,
		}
		// Update only if IP matches AND the device wasn't already updated by a k8s_node match
		// We check if cluster_id is NULL or 0 assuming non-k8s devices won't have a cluster_id set initially
		result := db.Model(&portal.Device{}).
			Where("ip = ? AND ip != ''", etcd.Instance).
			Where("cluster_id IS NULL OR cluster_id = 0"). // Avoid overwriting node match
			Updates(updates)
		if result.Error != nil {
			log.Printf("Warning: failed to update device for etcd %s: %v", etcd.Instance, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("Updated %d device(s) for etcd %s", result.RowsAffected, etcd.Instance)
		}
	}

	// Update any device directly referencing cluster
	for _, cluster := range clusters {
		devicesCriteria := "cluster = ? OR cluster_id = ?"
		if err := db.Model(&portal.Device{}).
			Where(devicesCriteria, cluster.ClusterName, int(cluster.ID)).
			Update("cluster_id", int(cluster.ID)).
			Error; err != nil {
			return err
		}
	}

	return nil
}

// --- Other Seeding Functions (Keep or integrate as needed) --- //

func seedF5Info(db *gorm.DB, clusters []portal.K8sCluster) error {
	// Basic F5 mock data, link to existing cluster IDs if possible
	clusterIDMap := make(map[string]int64)
	for _, c := range clusters {
		clusterIDMap[c.Zone] = c.ID // Example mapping by region
	}

	prodClusterID := clusterIDMap["shanghai"] // Default to shanghai if needed
	if prodClusterID == 0 {
		prodClusterID = clusterProdID
	}
	testClusterID := clusterIDMap["beijing"] // Default to beijing if needed
	if testClusterID == 0 {
		testClusterID = clusterTestID
	}

	mockF5s := []portal.F5Info{
		{BaseModel: portal.BaseModel{ID: 1}, Name: "App1-Prod", VIP: "10.1.10.1", Port: "443", K8sClusterID: prodClusterID, Status: "active", PoolStatus: "healthy"},
		{BaseModel: portal.BaseModel{ID: 2}, Name: "App2-Test", VIP: "10.2.10.1", Port: "80", K8sClusterID: testClusterID, Status: "active", PoolStatus: "offline"},
		// Add more realistic F5 data as needed
	}

	// Set timestamps using time.Time
	currentTime := time.Now()
	for i := range mockF5s {
		mockF5s[i].CreatedAt = currentTime // Assign time.Time directly
		mockF5s[i].UpdatedAt = currentTime // Assign time.Time directly
	}

	if err := db.Create(&mockF5s).Error; err != nil {
		return err
	}
	log.Printf("Inserted %d F5 Info records", len(mockF5s))
	return nil
}

func seedOpsJobs(db *gorm.DB) error {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	mockJobs := []portal.OpsJob{
		{Name: "Deploy Web v1.2", Status: "completed", Progress: 100, StartTime: yesterday, EndTime: yesterday.Add(15 * time.Minute), LogContent: "Deployment successful"},
		{Name: "Backup Database", Status: "running", Progress: 75, StartTime: now.Add(-1 * time.Hour), LogContent: "Backup in progress..."},
		{Name: "Update Firewall", Status: "pending", Progress: 0},
		// Add more jobs
	}
	if err := db.Create(&mockJobs).Error; err != nil {
		return err
	}
	log.Printf("Inserted %d Ops Jobs", len(mockJobs))
	return nil
}

// ConfigureCORS - Keep this utility function if used by the main application setup
func ConfigureCORS(r *gin.Engine) {
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*", "*"}, // Allow both frontend origins
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		AllowWildcard:    true, // Allow wildcard matching for origins
		MaxAge:           12 * time.Hour,
	}))

	// Add a middleware to log all requests for debugging
	r.Use(func(c *gin.Context) {
		fmt.Printf("Request: %s %s\n", c.Request.Method, c.Request.URL.Path)
		c.Next()
	})
}

// --- Deprecated / To Be Removed --- //

/*
// InsertMockOpsJobs - Deprecated: Logic merged into seedOpsJobs
func InsertMockOpsJobs(db *gorm.DB) error { ... }
*/

/*
// insertMockF5Info - Deprecated: Logic merged into seedF5Info
func insertMockF5Info(db *gorm.DB) { ... }
*/

/*
// InsertMockDevices - Deprecated: Logic merged into seedDevices
func InsertMockDevices(db *gorm.DB) error { ... }
*/

/*
// InsertMockLabelAndTaintData - Deprecated: Logic merged into seedLabelFeatures/seedTaintFeatures
func InsertMockLabelAndTaintData(db *gorm.DB) error { ... }
*/

/*
// insertMockK8sClusters - Deprecated: Logic merged into seedK8sClusters
func insertMockK8sClusters(db *gorm.DB) { ... }
*/

// 添加seedDeviceApps函数
func seedDeviceApps(db *gorm.DB, devices []portal.Device) error {
	// 首先查询已有设备，确保有正确的AppID值可以关联
	var existingDevices []portal.Device
	if len(devices) == 0 {
		if err := db.Find(&existingDevices).Error; err != nil {
			return fmt.Errorf("failed to query existing devices: %w", err)
		}
		devices = existingDevices
	}

	// 为不同类型的设备组创建应用
	appTypes := map[string][]string{
		"compute":     {"web-service", "api-service", "compute-app"},
		"gpu-compute": {"ml-training", "ai-inference", "data-processing"},
		"etcd":        {"etcd-service", "key-value-store", "configuration-service"},
		"storage":     {"object-storage", "database-service", "backup-service"},
	}

	// 创建DeviceApp记录
	var deviceApps []portal.DeviceApp
	now := time.Now()

	// 用于存储已经分配的AppID，确保不重复
	assignedAppIds := make(map[string]bool)

	// 用于记录设备ID和对应的AppID，以便后续批量更新
	deviceAppMap := make(map[int64]string)

	// 为每个设备创建一个或多个应用
	for _, device := range devices {
		// 确保设备有分组信息
		if device.Group == "" {
			continue
		}

		// 为该设备组选择应用类型
		appNames, exists := appTypes[device.Group]
		if !exists {
			// 如果没有预定义的类型，使用通用应用
			appNames = []string{"general-service"}
		}

		// 为每个设备随机选择1-2个应用
		numApps := 1
		if len(appNames) > 1 {
			numApps = rand.Intn(2) + 1 // 1 或 2
		}

		for i := 0; i < numApps && i < len(appNames); i++ {
			// 创建唯一的AppID: 设备组前缀-设备ID-序号
			appID := fmt.Sprintf("%s-%d-%d", device.Group, device.ID, i+1)

			// 跳过已经分配的AppID
			if assignedAppIds[appID] {
				continue
			}
			assignedAppIds[appID] = true

			// 创建DeviceApp记录
			app := portal.DeviceApp{
				BaseModel: portal.BaseModel{
					CreatedAt: portal.NavyTime(now),
					UpdatedAt: portal.NavyTime(now),
				},
				AppId:       appID,
				Type:        0, // 0 表示设备
				Name:        appNames[i],
				Owner:       "system",
				Feature:     fmt.Sprintf("Application for %s", device.Group),
				Description: fmt.Sprintf("Auto-generated application for device %s in group %s", device.CICode, device.Group),
				Status:      1, // 1 表示停止收集
			}
			deviceApps = append(deviceApps, app)

			// 记录设备ID和对应的AppID（使用第一个应用）
			if _, exists := deviceAppMap[device.ID]; !exists {
				deviceAppMap[device.ID] = appID
			}
		}
	}

	// 创建一些组件类型的DeviceApp
	componentTypes := []string{"monitor", "logger", "network-agent", "security-scanner"}
	for i, compType := range componentTypes {
		appID := fmt.Sprintf("component-%d", i+1)
		app := portal.DeviceApp{
			BaseModel: portal.BaseModel{
				CreatedAt: portal.NavyTime(now),
				UpdatedAt: portal.NavyTime(now),
			},
			AppId:       appID,
			Type:        1, // 1 表示组件
			Name:        compType,
			Owner:       "admin",
			Feature:     fmt.Sprintf("System component for %s", compType),
			Description: fmt.Sprintf("Auto-generated component for system %s functionality", compType),
			Status:      1, // 1 表示停止收集
		}
		deviceApps = append(deviceApps, app)
	}

	// 插入DeviceApp记录
	if len(deviceApps) > 0 {
		if err := db.Create(&deviceApps).Error; err != nil {
			return fmt.Errorf("failed to create device apps: %w", err)
		}
	}

	// 批量更新设备的AppID字段，建立关联
	log.Printf("Updating AppID for %d devices", len(deviceAppMap))
	for deviceID, appID := range deviceAppMap {
		if err := db.Model(&portal.Device{}).Where("id = ?", deviceID).Update("appid", appID).Error; err != nil {
			log.Printf("Warning: failed to update device AppID for device %d: %v", deviceID, err)
		}
	}

	log.Printf("Inserted %d Device Apps", len(deviceApps))
	return nil
}
