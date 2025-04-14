package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"navy-ng/models/portal"
)

// SecurityCheckService 安全检查服务
type SecurityCheckService struct {
	db *gorm.DB
}

// NewSecurityCheckService 创建安全检查服务实例
func NewSecurityCheckService(db *gorm.DB) *SecurityCheckService {
	return &SecurityCheckService{db: db}
}

// GetSecurityChecks 获取安全检查结果
func (s *SecurityCheckService) GetSecurityChecks(ctx context.Context, clusterName string, nodeType string, checkType string) ([]portal.SecurityCheck, error) {
	var checks []portal.SecurityCheck
	query := s.db.WithContext(ctx)

	if clusterName != "" {
		query = query.Where("cluster_name = ?", clusterName)
	}
	if nodeType != "" {
		query = query.Where("node_type = ?", nodeType)
	}
	if checkType != "" {
		query = query.Where("check_type = ?", checkType)
	}

	if err := query.Find(&checks).Error; err != nil {
		return nil, fmt.Errorf("failed to get security checks: %w", err)
	}

	return checks, nil
}

// GetSecurityCheckItems 获取安全检查项
func (s *SecurityCheckService) GetSecurityCheckItems(ctx context.Context, securityCheckID int64) ([]portal.SecurityCheckItem, error) {
	var items []portal.SecurityCheckItem

	if err := s.db.WithContext(ctx).
		Where("security_check_id = ?", securityCheckID).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to get security check items: %w", err)
	}

	return items, nil
}
