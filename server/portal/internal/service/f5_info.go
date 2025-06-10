// Package service provides the business logic for the portal module.
package service

import (
	"context"
	"errors"
	"fmt"
	"navy-ng/models/portal"
	"reflect"
	"time"

	"gorm.io/gorm"
)

// F5InfoService provides operations for F5Info.
type F5InfoService struct {
	db *gorm.DB
}

// NewF5InfoService creates a new F5InfoService.
func NewF5InfoService(db *gorm.DB) *F5InfoService {
	return &F5InfoService{db: db}
}

const (
	emptyString = ""
	zeroLength  = 0
	defaultPage = 1
	defaultSize = 10
	maxSize     = 100
	// f5InfoNotFoundMsg is the error message for F5 info record not found errors.
	f5InfoNotFoundMsg = "F5 info with id %d not found"
)

func (s *F5InfoService) GetF5Info(ctx context.Context, id int) (*F5InfoResponse, error) {
	var model portal.F5Info
	err := s.db.WithContext(ctx).Preload("K8sCluster").
		Where("id = ? AND deleted = ?", id, emptyString).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf(f5InfoNotFoundMsg, id)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return toF5InfoResponse(&model), nil
}

func (s *F5InfoService) ListF5Infos(ctx context.Context,
	query *F5InfoQuery) (*F5InfoListResponse, error) {
	var models []portal.F5Info
	var total int64

	db := s.db.WithContext(ctx).Model(&portal.F5Info{}).Where("deleted = ?", emptyString)

	// Apply filters
	if query.Name != emptyString {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.VIP != emptyString {
		db = db.Where("vip LIKE ?", "%"+query.VIP+"%")
	}
	if query.Port != emptyString {
		db = db.Where("port = ?", query.Port)
	}
	if query.AppID != emptyString {
		db = db.Where("appid LIKE ?", "%"+query.AppID+"%")
	}
	if query.InstanceGroup != emptyString {
		db = db.Where("instance_group LIKE ?", "%"+query.InstanceGroup+"%")
	}
	if query.Status != emptyString {
		db = db.Where("status = ?", query.Status)
	}
	if query.PoolName != emptyString {
		db = db.Where("pool_name LIKE ?", "%"+query.PoolName+"%")
	}
	if query.K8sClusterName != emptyString {
		db = db.Joins("JOIN k8s_cluster ON k8s_cluster.id = f5_info.k8s_cluster_id").
			Where("k8s_cluster.clustername LIKE ?", "%"+query.K8sClusterName+"%")
	}

	// Count total records
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count F5 infos: %w", err)
	}

	// Adjust pagination
	if query.Page <= 0 {
		query.Page = defaultPage
	}
	if query.Size <= 0 || query.Size > maxSize {
		query.Size = defaultSize
	}

	// Fetch data with pagination and preloading
	err := db.Preload("K8sCluster").
		Offset((query.Page - 1) * query.Size).Limit(query.Size).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list F5 infos: %w", err)
	}

	list := make([]*F5InfoResponse, zeroLength, len(models))
	for i := range models {
		list = append(list, toF5InfoResponse(&models[i]))
	}

	return &F5InfoListResponse{
		List:  list,
		Page:  query.Page,
		Size:  query.Size,
		Total: total,
	}, nil
}

func (s *F5InfoService) UpdateF5Info(ctx context.Context, id int,
	dto *F5InfoUpdateDTO) error {
	model := fromF5InfoUpdateDTO(dto)

	// Check if the record exists before updating
	var existing portal.F5Info
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf(f5InfoNotFoundMsg, id)
		}
		return fmt.Errorf("database error when checking existence: %w", err)
	}

	result := s.db.WithContext(ctx).Model(&portal.F5Info{}).Where("id = ?", id).
		Select("Name", "VIP", "Port", "AppID", "InstanceGroup", "Status", "PoolName", "PoolStatus",
			"PoolMembers", "K8sClusterID", "Domains", "GrafanaParams", "Ignored").
		Updates(model)

	if result.Error != nil {
		return fmt.Errorf("failed to update F5 info: %w", result.Error)
	}

	return nil
}

func (s *F5InfoService) DeleteF5Info(ctx context.Context, id int) error {
	result := s.db.WithContext(ctx).Model(&portal.F5Info{}).Where("id = ?", id).
		Update("deleted", "1") // Assuming "1" marks as deleted

	if result.Error != nil {
		return fmt.Errorf("failed to delete F5 info: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf(f5InfoNotFoundMsg, id)
	}
	return nil
}

// Helper function to convert portal.F5Info to F5InfoResponse
func toF5InfoResponse(m *portal.F5Info) *F5InfoResponse {
	if m == nil {
		return nil
	}
	resp := &F5InfoResponse{
		ID:            m.ID, // Removed unnecessary conversion int64(m.ID)
		Name:          m.Name,
		VIP:           m.VIP,
		Port:          m.Port,
		AppID:         m.AppID,
		InstanceGroup: m.InstanceGroup,
		Status:        m.Status,
		PoolName:      m.PoolName,
		PoolStatus:    m.PoolStatus,
		PoolMembers:   m.PoolMembers,
		K8sClusterID:  m.K8sClusterID,
		Domains:       m.Domains,
		GrafanaParams: m.GrafanaParams,
		Ignored:       m.Ignored,
		CreatedAt:     m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     m.UpdatedAt.Format(time.RFC3339),
	}

	// Check if K8sCluster relation is valid (not zero value) and has a valid ID
	if !reflect.DeepEqual(m.K8sCluster, portal.K8sCluster{}) && m.K8sCluster.ID > 0 {
		resp.K8sClusterName = m.K8sCluster.ClusterName
	}
	return resp
}

// Helper function to convert F5InfoUpdateDTO to portal.F5Info
func fromF5InfoUpdateDTO(dto *F5InfoUpdateDTO) *portal.F5Info {
	if dto == nil {
		return nil
	}
	return &portal.F5Info{
		Name:          dto.Name,
		VIP:           dto.VIP,
		Port:          dto.Port,
		AppID:         dto.AppID,
		InstanceGroup: dto.InstanceGroup,
		Status:        dto.Status,
		PoolName:      dto.PoolName,
		PoolStatus:    dto.PoolStatus,
		PoolMembers:   dto.PoolMembers,
		K8sClusterID:  dto.K8sClusterID,
		Domains:       dto.Domains,
		GrafanaParams: dto.GrafanaParams,
		Ignored:       dto.Ignored,
	}
}

// 注意：删除了重复的 ErrorResponse 和 SuccessResponse 结构体
