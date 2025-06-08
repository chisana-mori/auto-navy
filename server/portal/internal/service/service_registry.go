package service

import (
	"fmt"
	"navy-ng/models/portal"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// InitializeOrderServices 初始化并注册所有订单服务
func InitializeOrderServices(db *gorm.DB, logger *zap.Logger) {
	// 注册弹性伸缩订单服务
	// elasticScalingService := NewElasticScalingService(db, redisHandler, logger, deviceCache)
	// RegisterOrderService(string(portal.OrderTypeElasticScaling), elasticScalingService)

	// 注册维护订单服务
	maintenanceOrderService := NewMaintenanceOrderService(db, logger)
	RegisterOrderService(string(portal.OrderTypeMaintenance), maintenanceOrderService)

	// 可以继续注册其他类型的订单服务
	// deploymentService := NewDeploymentOrderService(db, logger)
	// RegisterOrderService(string(portal.OrderTypeDeployment), deploymentService)

	// generalOrderService := NewGeneralOrderService(db, logger)
	// RegisterOrderService(string(portal.OrderTypeGeneral), generalOrderService)
}

// GetMaintenanceOrderService 获取维护订单服务
func GetMaintenanceOrderService() (*MaintenanceOrderService, error) {
	service, found := GetOrderService(string(portal.OrderTypeMaintenance))
	if !found {
		return nil, fmt.Errorf("维护订单服务未注册")
	}

	maintenanceService, ok := service.(*MaintenanceOrderService)
	if !ok {
		return nil, fmt.Errorf("服务类型转换失败")
	}

	return maintenanceService, nil
}