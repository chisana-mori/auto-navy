package database

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"navy-ng/models/portal"
)

// K8s Cluster IDs.
const (
	clusterIDProdEast  int64 = 101
	clusterIDProdNorth int64 = 102
	clusterIDTestSH    int64 = 103
	clusterIDDevSZ     int64 = 104
	clusterIDUATBJ     int64 = 105
)

// F5 Info IDs.
const (
	f5IDFinanceProd   int64 = 1
	f5IDHRProd        int64 = 2
	f5IDMarketingTest int64 = 3
	f5IDCRMDev        int64 = 4
	f5IDPaymentProd   int64 = 5
	f5IDCMSUAT        int64 = 6
	f5IDSearchProd    int64 = 7
	f5IDStorageProd   int64 = 8
	f5IDAuthTest      int64 = 9
	f5IDReportUAT     int64 = 10
)

// Status Constants.
const (
	statusRunning  = "running"
	statusActive   = "active"
	statusHealthy  = "healthy"
	statusStopped  = "stopped"
	statusInactive = "inactive"
	statusOffline  = "offline"
	statusDegraded = "degraded"
)

// Port Constants.
const (
	port443  = "443"
	port80   = "80"
	port8080 = "8080"
	port8000 = "8000"
	port9200 = "9200"
)

// 空字符串常量
const emptyString = ""

// InsertMockOpsJobs 插入运维任务模拟数据
func InsertMockOpsJobs(db *gorm.DB) error {
	// 获取当前时间
	now := time.Now()
	// 创建一些不同状态的任务
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	// 创建模拟任务数据
	mockJobs := []portal.OpsJob{
		{
			Name:        "更新F5配置",
			Description: "更新生产环境F5负载均衡器配置",
			Status:      "completed",
			Progress:    100,
			StartTime:   yesterday,
			EndTime:     yesterday.Add(30 * time.Minute),
			LogContent:  "2023-05-10 10:00:00 [INFO] 开始更新F5配置\n2023-05-10 10:15:00 [INFO] 备份当前配置\n2023-05-10 10:20:00 [INFO] 应用新配置\n2023-05-10 10:30:00 [INFO] 配置更新完成",
		},
		{
			Name:        "重启应用服务器",
			Description: "重启测试环境应用服务器",
			Status:      "running",
			Progress:    60,
			StartTime:   now.Add(-30 * time.Minute),
			LogContent:  "2023-05-11 14:30:00 [INFO] 开始重启应用服务器\n2023-05-11 14:35:00 [INFO] 停止服务\n2023-05-11 14:40:00 [INFO] 清理缓存\n2023-05-11 14:45:00 [INFO] 启动服务中...",
		},
		{
			Name:        "数据库备份",
			Description: "执行每周数据库完整备份",
			Status:      "failed",
			Progress:    45,
			StartTime:   lastWeek,
			EndTime:     lastWeek.Add(2 * time.Hour),
			LogContent:  "2023-05-05 01:00:00 [INFO] 开始数据库备份\n2023-05-05 01:30:00 [INFO] 备份进行中...\n2023-05-05 02:00:00 [ERROR] 备份失败: 磁盘空间不足\n2023-05-05 02:01:00 [INFO] 清理临时文件",
		},
		{
			Name:        "部署新版本",
			Description: "部署应用v2.3.0版本到UAT环境",
			Status:      "pending",
			Progress:    0,
			LogContent:  "",
		},
		{
			Name:        "网络配置更新",
			Description: "更新防火墙规则",
			Status:      "completed",
			Progress:    100,
			StartTime:   now.Add(-2 * 24 * time.Hour),
			EndTime:     now.Add(-2 * 24 * time.Hour).Add(45 * time.Minute),
			LogContent:  "2023-05-09 09:00:00 [INFO] 开始更新防火墙规则\n2023-05-09 09:15:00 [INFO] 备份当前规则\n2023-05-09 09:30:00 [INFO] 应用新规则\n2023-05-09 09:45:00 [INFO] 规则更新完成",
		},
	}

	// 插入模拟数据
	for _, job := range mockJobs {
		if err := db.Create(&job).Error; err != nil {
			log.Printf("Warning: failed to insert mock ops job: %v", err)
			return err
		}
	}

	log.Printf("成功插入 %d 条运维任务模拟数据", len(mockJobs))
	return nil
}

// ClearAndSeedData 清空现有数据并插入样例数据
func ClearAndSeedData(db *gorm.DB) error {
	// 清空现有数据
	if err := db.Exec("DELETE FROM f5_info").Error; err != nil {
		log.Printf("Warning: failed to delete f5_info data: %v", err)
	}

	// 插入F5信息数据
	insertMockF5Info(db)

	return nil
}

// 插入K8s集群数据
func insertMockK8sClusters(db *gorm.DB) {
	// 创建模拟K8s集群数据
	mockClusters := []portal.K8sCluster{
		{
			BaseModel: portal.BaseModel{ID: clusterIDProdEast},
			Name:      "生产集群-华东",
			Region:    "上海",
			Endpoint:  "https://k8s-prod-east.example.com:6443",
			Status:    statusRunning,
		},
		{
			BaseModel: portal.BaseModel{ID: clusterIDProdNorth},
			Name:      "生产集群-华北",
			Region:    "北京",
			Endpoint:  "https://k8s-prod-north.example.com:6443",
			Status:    statusRunning,
		},
		{
			BaseModel: portal.BaseModel{ID: clusterIDTestSH},
			Name:      "测试集群-上海",
			Region:    "上海",
			Endpoint:  "https://k8s-test-sh.example.com:6443",
			Status:    statusRunning,
		},
		{
			BaseModel: portal.BaseModel{ID: clusterIDDevSZ},
			Name:      "开发集群-深圳",
			Region:    "深圳",
			Endpoint:  "https://k8s-dev-sz.example.com:6443",
			Status:    statusStopped,
		},
		{
			BaseModel: portal.BaseModel{ID: clusterIDUATBJ},
			Name:      "UAT集群-北京",
			Region:    "北京",
			Endpoint:  "https://k8s-uat-bj.example.com:6443",
			Status:    statusRunning,
		},
	}

	for _, cluster := range mockClusters {
		if err := db.Create(&cluster).Error; err != nil {
			log.Printf("Warning: failed to create k8s cluster %d: %v", cluster.ID, err)
		}
	}

	log.Printf("成功插入 %d 条K8s集群数据", len(mockClusters))
}

// 插入F5信息数据
func insertMockF5Info(db *gorm.DB) {
	// 获取当前时间
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)
	lastMonth := now.Add(-30 * 24 * time.Hour)

	// 创建模拟F5信息数据
	mockF5s := []portal.F5Info{
		{
			BaseModel:     portal.BaseModel{ID: f5IDFinanceProd},
			Name:          "财务系统-生产",
			VIP:           "10.100.1.1",
			Port:          port443,
			AppID:         "FIN-APP-001",
			InstanceGroup: "finance-prod",
			Status:        statusActive,
			PoolName:      "finance-pool-prod",
			PoolStatus:    statusHealthy,
			PoolMembers:   "192.168.1.10:443 online,192.168.1.11:443 online",
			K8sClusterID:  clusterIDProdEast,
			Domains:       "finance.example.com",
			GrafanaParams: "http://grafana.example.com/d/finance-prod",
			Ignored:       false,
			CreatedAt:     lastMonth,
			UpdatedAt:     yesterday,
		},
		{
			BaseModel:     portal.BaseModel{ID: f5IDHRProd},
			Name:          "人力资源系统-生产",
			VIP:           "10.100.1.2",
			Port:          port80,
			AppID:         "HR-APP-001",
			InstanceGroup: "hr-prod",
			Status:        statusActive,
			PoolName:      "hr-pool-prod",
			PoolStatus:    statusHealthy,
			PoolMembers:   "192.168.2.10:80 online,192.168.2.11:80 online",
			K8sClusterID:  clusterIDProdNorth,
			Domains:       "hr.example.com",
			GrafanaParams: "http://grafana.example.com/d/hr-prod",
			Ignored:       false,
			CreatedAt:     lastMonth,
			UpdatedAt:     yesterday,
		},
		{
			BaseModel:     portal.BaseModel{ID: f5IDMarketingTest},
			Name:          "营销系统-测试",
			VIP:           "10.200.1.1",
			Port:          port8080,
			AppID:         "MKT-APP-002",
			InstanceGroup: "marketing-test",
			Status:        statusActive,
			PoolName:      "marketing-pool-test",
			PoolStatus:    statusDegraded,
			PoolMembers:   "192.168.3.10:8080 online,192.168.3.11:8080 offline",
			K8sClusterID:  clusterIDTestSH,
			Domains:       "marketing-test.example.com",
			GrafanaParams: "http://grafana.example.com/d/marketing-test",
			Ignored:       false,
			CreatedAt:     lastWeek,
			UpdatedAt:     yesterday,
		},
		{
			BaseModel:     portal.BaseModel{ID: f5IDCRMDev},
			Name:          "CRM系统-开发",
			VIP:           "10.200.2.1",
			Port:          port8000,
			AppID:         "CRM-APP-003",
			InstanceGroup: "crm-dev",
			Status:        statusInactive,
			PoolName:      "crm-pool-dev",
			PoolStatus:    statusOffline,
			PoolMembers:   "192.168.4.10:8000 offline",
			K8sClusterID:  clusterIDDevSZ,
			Domains:       "crm-dev.example.com",
			GrafanaParams: "http://grafana.example.com/d/crm-dev",
			Ignored:       true,
			CreatedAt:     lastWeek,
			UpdatedAt:     now,
		},
		{
			BaseModel:     portal.BaseModel{ID: f5IDPaymentProd},
			Name:          "支付系统-生产",
			VIP:           "10.100.2.1",
			Port:          port443,
			AppID:         "PAY-APP-001",
			InstanceGroup: "payment-prod",
			Status:        statusActive,
			PoolName:      "payment-pool-prod",
			PoolStatus:    statusHealthy,
			PoolMembers:   "192.168.5.10:443 online,192.168.5.11:443 online,192.168.5.12:443 online",
			K8sClusterID:  clusterIDProdEast,
			Domains:       "payment.example.com,pay.example.com",
			GrafanaParams: "http://grafana.example.com/d/payment-prod",
			Ignored:       false,
			CreatedAt:     lastMonth,
			UpdatedAt:     now,
		},
		{
			BaseModel:     portal.BaseModel{ID: f5IDCMSUAT},
			Name:          "内容管理-UAT",
			VIP:           "10.200.3.1",
			Port:          port8080,
			AppID:         "CMS-APP-002",
			InstanceGroup: "cms-uat",
			Status:        statusActive,
			PoolName:      "cms-pool-uat",
			PoolStatus:    statusHealthy,
			PoolMembers:   "192.168.6.10:8080 online,192.168.6.11:8080 offline",
			K8sClusterID:  clusterIDUATBJ,
			Domains:       "cms-uat.example.com",
			GrafanaParams: "http://grafana.example.com/d/cms-uat",
			Ignored:       false,
			CreatedAt:     lastWeek,
			UpdatedAt:     yesterday,
		},
		{
			BaseModel:     portal.BaseModel{ID: f5IDSearchProd},
			Name:          "搜索服务-生产",
			VIP:           "10.100.3.1",
			Port:          port9200,
			AppID:         "SEARCH-APP-001",
			InstanceGroup: "search-prod",
			Status:        statusActive,
			PoolName:      "search-pool-prod",
			PoolStatus:    statusHealthy,
			PoolMembers:   "192.168.7.10:9200 online,192.168.7.11:9200 online,192.168.7.12:9200 online",
			K8sClusterID:  clusterIDProdEast,
			Domains:       "search.example.com,search-api.example.com",
			GrafanaParams: "http://grafana.example.com/d/search-prod",
			Ignored:       false,
			CreatedAt:     lastMonth,
			UpdatedAt:     lastWeek,
		},
		{
			BaseModel:     portal.BaseModel{ID: f5IDStorageProd},
			Name:          "文件存储-生产",
			VIP:           "10.100.4.1",
			Port:          port443,
			AppID:         "STORAGE-APP-001",
			InstanceGroup: "storage-prod",
			Status:        statusActive,
			PoolName:      "storage-pool-prod",
			PoolStatus:    statusHealthy,
			PoolMembers:   "192.168.8.10:443 online,192.168.8.11:443 online",
			K8sClusterID:  clusterIDProdNorth,
			Domains:       "storage.example.com,files.example.com",
			GrafanaParams: "http://grafana.example.com/d/storage-prod",
			Ignored:       false,
			CreatedAt:     lastMonth,
			UpdatedAt:     yesterday,
		},
		{
			BaseModel:     portal.BaseModel{ID: f5IDAuthTest},
			Name:          "用户认证-测试",
			VIP:           "10.200.4.1",
			Port:          port8080,
			AppID:         "AUTH-APP-002",
			InstanceGroup: "auth-test",
			Status:        statusActive,
			PoolName:      "auth-pool-test",
			PoolStatus:    statusDegraded,
			PoolMembers:   "192.168.9.10:8080 offline",
			K8sClusterID:  clusterIDTestSH,
			Domains:       "auth-test.example.com",
			GrafanaParams: "http://grafana.example.com/d/auth-test",
			Ignored:       false,
			CreatedAt:     lastWeek,
			UpdatedAt:     yesterday,
		},
		{
			BaseModel:     portal.BaseModel{ID: f5IDReportUAT},
			Name:          "报表系统-UAT",
			VIP:           "10.200.5.1",
			Port:          port8080,
			AppID:         "REPORT-APP-002",
			InstanceGroup: "report-uat",
			Status:        statusInactive,
			PoolName:      "report-pool-uat",
			PoolStatus:    statusOffline,
			PoolMembers:   "192.168.10.10:8080 offline",
			K8sClusterID:  clusterIDUATBJ,
			Domains:       "report-uat.example.com",
			GrafanaParams: "http://grafana.example.com/d/report-uat",
			Ignored:       true,
			CreatedAt:     lastWeek,
			UpdatedAt:     now,
		},
	}

	for _, f5 := range mockF5s {
		if err := db.Create(&f5).Error; err != nil {
			log.Printf("Warning: failed to create f5 info %d: %v", f5.ID, err)
		}
	}
}

// InsertMockDevices 插入设备模拟数据
func InsertMockDevices(db *gorm.DB) error {
	// 清空现有设备数据
	if err := db.Exec("DELETE FROM device").Error; err != nil {
		log.Printf("Warning: failed to delete device data: %v", err)
	}

	// 创建模拟设备数据
	mockDevices := []portal.Device{
		{
			DeviceID:     "SYSOPS00409045",
			IP:           "29.19.50.124",
			MachineType:  "qf-core601-flannel-2",
			Cluster:      "work",
			Role:         "x86",
			Arch:         "qf",
			IDC:          "601",
			Room:         "OF601-02P",
			Datacenter:   "private_cloud",
			Cabinet:      "central",
			Network:      "10703",
			AppID:        "",
			ResourcePool: "",
		},
		{
			DeviceID:     "SYSOPS00409044",
			IP:           "29.19.50.123",
			MachineType:  "qf-core601-flannel-2",
			Cluster:      "work",
			Role:         "x86",
			Arch:         "qf",
			IDC:          "601",
			Room:         "OF601-04P",
			Datacenter:   "private_cloud",
			Cabinet:      "central",
			Network:      "10703",
			AppID:        "",
			ResourcePool: "",
		},
		{
			DeviceID:     "SYSOPS00409043",
			IP:           "29.19.50.122",
			MachineType:  "qf-core601-flannel-2",
			Cluster:      "work",
			Role:         "x86",
			Arch:         "qf",
			IDC:          "601",
			Room:         "OF601-07P",
			Datacenter:   "private_cloud",
			Cabinet:      "central",
			Network:      "10703",
			AppID:        "",
			ResourcePool: "",
		},
		{
			DeviceID:     "SYSOPS00409042",
			IP:           "29.19.50.121",
			MachineType:  "qf-core601-flannel-2",
			Cluster:      "work",
			Role:         "x86",
			Arch:         "qf",
			IDC:          "601",
			Room:         "OF601-08P",
			Datacenter:   "private_cloud",
			Cabinet:      "central",
			Network:      "10703",
			AppID:        "",
			ResourcePool: "",
		},
		{
			DeviceID:     "EQUHST00093218",
			IP:           "29.20.50.24",
			MachineType:  "",
			Cluster:      "",
			Role:         "ARM",
			Arch:         "qf",
			IDC:          "203",
			Room:         "C203-10C",
			Datacenter:   "traditional",
			Cabinet:      "mgt",
			Network:      "85004",
			AppID:        "",
			ResourcePool: "",
		},
		{
			DeviceID:     "EQUHST00093215",
			IP:           "29.20.50.20",
			MachineType:  "",
			Cluster:      "",
			Role:         "ARM",
			Arch:         "qf",
			IDC:          "203",
			Room:         "C203-01C",
			Datacenter:   "traditional",
			Cabinet:      "mgt",
			Network:      "85004",
			AppID:        "",
			ResourcePool: "",
		},
		{
			DeviceID:     "EQUHST00093217",
			IP:           "29.20.50.22",
			MachineType:  "",
			Cluster:      "",
			Role:         "ARM",
			Arch:         "qf",
			IDC:          "203",
			Room:         "C203-05C",
			Datacenter:   "traditional",
			Cabinet:      "mgt",
			Network:      "85004",
			AppID:        "",
			ResourcePool: "",
		},
		{
			DeviceID:     "EQUHST00093216",
			IP:           "29.20.50.21",
			MachineType:  "",
			Cluster:      "",
			Role:         "ARM",
			Arch:         "qf",
			IDC:          "203",
			Room:         "C203-08C",
			Datacenter:   "traditional",
			Cabinet:      "mgt",
			Network:      "85004",
			AppID:        "",
			ResourcePool: "",
		},
		{
			DeviceID:     "SYSOPS00408553",
			IP:           "29.21.89.120",
			MachineType:  "orinoc-etcd",
			Cluster:      "etcd",
			Role:         "x86",
			Arch:         "qf",
			IDC:          "203",
			Room:         "C203-12H",
			Datacenter:   "private_cloud",
			Cabinet:      "central",
			Network:      "85004",
			AppID:        "",
			ResourcePool: "",
		},
		{
			DeviceID:     "SYSOPS00408552",
			IP:           "29.21.89.91",
			MachineType:  "orinoc-etcd",
			Cluster:      "etcd",
			Role:         "x86",
			Arch:         "qf",
			IDC:          "203",
			Room:         "C203-14H",
			Datacenter:   "private_cloud",
			Cabinet:      "central",
			Network:      "85004",
			AppID:        "",
			ResourcePool: "",
		},
	}

	// 插入数据
	for _, device := range mockDevices {
		if err := db.Create(&device).Error; err != nil {
			log.Printf("Warning: failed to create device %s: %v", device.DeviceID, err)
			return err
		}
	}

	log.Printf("成功插入 %d 条设备模拟数据", len(mockDevices))
	return nil
}

// ConfigureCORS 配置CORS中间件
func ConfigureCORS(r *gin.Engine) {
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
}
