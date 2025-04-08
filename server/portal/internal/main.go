package main

import (
	"log"
	"net/http"
	"time"

	"navy-ng/models/portal"
	"navy-ng/server/portal/internal/routers"
	"navy-ng/server/portal/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// @title           Navy-NG API
// @version         1.0
// @description     Navy-NG 管理平台 API 文档
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /fe-v1

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

func main() {
	// 初始化数据库连接
	db, err := gorm.Open(sqlite.Open("navy.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 先删除ops_job表，然后重新创建
	log.Println("删除ops_job表...")
	db.Exec("DROP TABLE IF EXISTS ops_job")

	// 手动创建ops_job表，确保ID是自增的
	log.Println("手动创建ops_job表...")
	createTableSQL := `
	CREATE TABLE ops_job (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at datetime,
		updated_at datetime,
		name varchar(255) NOT NULL,
		description text,
		status varchar(50) NOT NULL,
		progress integer DEFAULT 0,
		start_time datetime,
		end_time datetime,
		log_content text,
		deleted varchar(255)
	)
	`
	if err := db.Exec(createTableSQL).Error; err != nil {
		log.Fatalf("failed to create ops_job table: %v", err)
	}

	// 自动迁移其他表
	log.Println("创建其他数据库表...")
	err = db.AutoMigrate(&portal.K8sCluster{}, &portal.F5Info{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// 检查ops_job表结构
	var result map[string]interface{}
	db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='ops_job'").Scan(&result)
	log.Printf("ops_job表结构: %v", result)

	// 插入模拟数据
	log.Println("插入运维任务模拟数据...")
	insertMockOpsJobs(db)

	// 清空现有数据并插入样例数据
	clearAndSeedData(db)

	// 初始化服务
	f5Service := service.NewF5InfoService(db)
	opsService := service.NewOpsJobService(db)

	// 初始化路由处理器
	f5Handler := routers.NewF5InfoHandler(f5Service)
	opsHandler := routers.NewOpsJobHandler(opsService)

	// 创建 Gin 引擎
	r := gin.Default()

	// 配置 CORS 中间件
	configureCORS(r)

	// 注册路由
	api := r.Group("/fe-v1")
	f5Handler.RegisterRoutes(api)
	opsHandler.RegisterRoutes(api)

	// 注册 Swagger 路由
	routers.RegisterSwaggerRoutes(r)

	// 启动服务器
	port := ":8081" // 使用不同的端口
	log.Printf("Starting server on %s", port)
	if err := r.Run(port); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to run server: %v", err)
	}
}

// 插入运维任务模拟数据
func insertMockOpsJobs(db *gorm.DB) {
	// 获取当前时间
	now := time.Now()
	// 创建一些不同状态的任务
	mockJobs := []portal.OpsJob{
		{
			Name:        "部署Web应用到生产环境",
			Description: "将最新版本的Web应用部署到生产服务器",
			Status:      "completed",
			Progress:    100,
			StartTime:   now.Add(-24 * time.Hour),
			EndTime:     now.Add(-23 * time.Hour),
			LogContent:  `初始化部署...
连接到生产服务器...
拉取最新代码...
构建应用...
运行测试...
部署到生产服务器...
配置负载均衡...
验证部署...
部署完成！`,
		},
		{
			Name:        "数据库备份",
			Description: "执行所有生产数据库的完整备份",
			Status:      "running",
			Progress:    65,
			StartTime:   now.Add(-1 * time.Hour),
			EndTime:     now,
			LogContent:  `初始化备份任务...
连接到数据库服务器...
开始备份用户数据库...
用户数据库备份完成...
开始备份订单数据库...
订单数据库备份完成...
开始备份产品数据库...
正在进行中...`,
		},
		{
			Name:        "系统安全补丁更新",
			Description: "为所有服务器安装最新的安全补丁",
			Status:      "pending",
			Progress:    0,
			StartTime:   now,
			EndTime:     now,
			LogContent:  `任务创建，等待执行...`,
		},
		{
			Name:        "网络配置更新",
			Description: "更新负载均衡器和防火墙规则",
			Status:      "failed",
			Progress:    45,
			StartTime:   now.Add(-12 * time.Hour),
			EndTime:     now.Add(-11 * time.Hour),
			LogContent:  `初始化任务...
连接到网络设备...
备份当前配置...
应用新的负载均衡规则...
测试新配置...
错误：无法连接到主防火墙
回滚配置...
任务失败！`,
		},
		{
			Name:        "应用服务器扩容",
			Description: "增加3台新的应用服务器到集群",
			Status:      "pending",
			Progress:    0,
			StartTime:   now,
			EndTime:     now,
			LogContent:  `任务创建，等待执行...`,
		},
	}

	// 插入模拟数据
	for _, job := range mockJobs {
		result := db.Create(&job)
		if result.Error != nil {
			log.Printf("Warning: failed to insert mock ops job: %v", result.Error)
		}
	}

	log.Printf("成功插入 %d 条运维任务模拟数据", len(mockJobs))
}

func clearAndSeedData(db *gorm.DB) {
	// 清空现有数据
	if err := db.Exec("DELETE FROM k8s_cluster").Error; err != nil {
		log.Printf("Warning: failed to delete k8s_cluster data: %v", err)
	}
	if err := db.Exec("DELETE FROM f5_info").Error; err != nil {
		log.Printf("Warning: failed to delete f5_info data: %v", err)
	}

	// 插入样例 K8s 集群数据
	mockClusters := []portal.K8sCluster{
		{
			BaseModel: portal.BaseModel{ID: clusterIDProdEast},
			Name:      "生产集群-华东",
			Region:    "华东区域",
			Endpoint:  "https://k8s-prod-east.example.com:6443",
			Status:    statusRunning,
		},
		{
			BaseModel: portal.BaseModel{ID: clusterIDProdNorth},
			Name:      "生产集群-华北",
			Region:    "华北区域",
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
			log.Printf("Warning: failed to create cluster %d: %v", cluster.ID, err)
		}
	}

	// 创建时间和更新时间
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	lastWeek := now.AddDate(0, 0, -7)
	lastMonth := now.AddDate(0, -1, 0)

	// 插入mock F5数据
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
			PoolMembers:   "192.168.1.10:443 online,192.168.1.11:443 online,192.168.1.12:443 online",
			K8sClusterID:  clusterIDProdEast,
			Domains:       "finance.example.com,fin-api.example.com",
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
			K8sClusterID:  clusterIDProdEast,
			Domains:       "hr.example.com",
			GrafanaParams: "http://grafana.example.com/d/hr-prod",
			Ignored:       false,
			CreatedAt:     lastMonth,
			UpdatedAt:     lastWeek,
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
			Domains:       "marketing-test.example.com,mkt-test.example.com",
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
			UpdatedAt:     lastWeek,
		},
		{
			BaseModel:     portal.BaseModel{ID: f5IDPaymentProd},
			Name:          "支付网关-生产",
			VIP:           "10.100.2.1",
			Port:          port443,
			AppID:         "PAY-APP-001",
			InstanceGroup: "payment-prod",
			Status:        statusActive,
			PoolName:      "payment-pool-prod",
			PoolStatus:    statusHealthy,
			PoolMembers:   "192.168.5.10:443 online,192.168.5.11:443 online,192.168.5.12:443 online,192.168.5.13:443 online",
			K8sClusterID:  clusterIDProdNorth,
			Domains:       "pay.example.com,payment-api.example.com",
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

func configureCORS(r *gin.Engine) {
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
}
