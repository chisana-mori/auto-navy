/*
Package main 是 Navy-NG 后端服务的入口点。
*/
package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"navy-ng/pkg/redis"
	"navy-ng/server/portal/internal/database"
	"navy-ng/server/portal/internal/routers"
	"navy-ng/server/portal/internal/service"
	// "navy-ng/server/portal/internal/service" // Service is no longer directly used here.
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
// @securityDefinitions.basic BasicAuth
// @in header
// @name Authorization

func main() {
	// 初始化数据库连接
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	// 确保Redis已初始化（使用项目现有Redis实现）
	initRedisIfNeeded()

	// 初始化路由处理器
	f5Handler := routers.NewF5InfoHandler(db)
	opsHandler := routers.NewOpsJobHandler(db)
	deviceHandler := routers.NewDeviceHandler(db)
	deviceQueryHandler := routers.NewDeviceQueryHandler(db)
	elasticScalingHandler := routers.NewElasticScalingHandler(db)
	maintenanceHandler := routers.NewMaintenanceHandler(db)
	resourcePoolDeviceMatchingPolicyHandler := routers.NewResourcePoolDeviceMatchingPolicyHandler(db)

	// 创建 Gin 引擎
	r := gin.Default()

	// 配置 CORS 中间件
	database.ConfigureCORS(r)

	// 注册路由
	api := r.Group("/fe-v1")
	f5Handler.RegisterRoutes(api)
	opsHandler.RegisterRoutes(api)
	deviceHandler.RegisterRoutes(api)
	deviceQueryHandler.RegisterRoutes(api)
	elasticScalingHandler.RegisterRoutes(api)
	maintenanceHandler.RegisterRoutes(api)
	resourcePoolDeviceMatchingPolicyHandler.RegisterRoutes(api)

	// 注册 Swagger 路由
	routers.RegisterSwaggerRoutes(r)

	// 启动弹性伸缩监控服务（仅在启用时）
	if os.Getenv("ENABLE_ELASTIC_SCALING_MONITOR") == "true" {
		monitorConfig := service.DefaultMonitorConfig()
		// 创建 logger for monitor
		logger, _ := zap.NewProduction()
		monitor := service.NewElasticScalingMonitor(db, monitorConfig, logger)
		monitor.Start()

		// 确保在应用退出时优雅地停止监控服务
		defer monitor.Stop()
	}

	// 启动服务器
	port := ":8081"
	log.Printf("Starting server on %s", port)
	if err := r.Run(port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Failed to run server: %v", err)
	}
}

// initRedisIfNeeded 确保Redis已初始化
func initRedisIfNeeded() {
	// 从环境变量获取Redis配置
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379" // 默认地址
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	// 使用项目现有Redis实现初始化连接
	if err := redis.Init("default", redisAddr, redisPassword); err != nil {
		log.Printf("Warning: Redis connection failed: %v. Distributed locking will not work properly.", err)
	} else {
		log.Println("Redis connection established successfully")
	}
}

// Removed unused import
