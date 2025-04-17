/*
Package main 是 Navy-NG 后端服务的入口点。
*/
package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"navy-ng/server/portal/internal/database"
	"navy-ng/server/portal/internal/routers"
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

	// 初始化路由处理器 (Service 实例化移至 Handler 构造函数)
	f5Handler := routers.NewF5InfoHandler(db)
	opsHandler := routers.NewOpsJobHandler(db)
	deviceHandler := routers.NewDeviceHandler(db)
	deviceQueryHandler := routers.NewDeviceQueryHandler(db)

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

	// 注册 Swagger 路由
	routers.RegisterSwaggerRoutes(r)

	// 启动服务器
	port := ":8081" // 使用不同的端口
	log.Printf("Starting server on %s", port)
	if err := r.Run(port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Failed to run server: %v", err)
	}
}

// Removed unused import
