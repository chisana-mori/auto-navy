package routers

import (
	"navy-ng/pkg/redis"
	"navy-ng/server/portal/internal/api"
	"navy-ng/server/portal/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ResourcePoolDeviceMatchingPolicyHandler 资源池设备匹配策略处理器
type ResourcePoolDeviceMatchingPolicyHandler struct {
	handler *api.ResourcePoolDeviceMatchingPolicyHandler
}

// NewResourcePoolDeviceMatchingPolicyHandler 创建资源池设备匹配策略处理器
func NewResourcePoolDeviceMatchingPolicyHandler(db *gorm.DB) *ResourcePoolDeviceMatchingPolicyHandler {
	// 创建 Redis 客户端和键构建器
	redisHandler := redis.NewRedisHandler("default")
	keyBuilder := redis.NewKeyBuilder("navy", service.CacheVersion)

	// 创建设备缓存服务
	deviceCache := service.NewDeviceCache(redisHandler, keyBuilder)

	// 创建资源池设备匹配策略服务
	policyService := service.NewResourcePoolDeviceMatchingPolicyService(db, deviceCache)

	// 创建API处理器
	handler := api.NewResourcePoolDeviceMatchingPolicyHandler(policyService)

	return &ResourcePoolDeviceMatchingPolicyHandler{
		handler: handler,
	}
}

// RegisterRoutes 注册路由
func (h *ResourcePoolDeviceMatchingPolicyHandler) RegisterRoutes(r *gin.RouterGroup) {
	h.handler.RegisterRoutes(r)
}
