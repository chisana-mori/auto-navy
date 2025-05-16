package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"navy-ng/models/portal"
)

// ResourcePoolDeviceMatchingPolicyService 资源池设备匹配策略服务
type ResourcePoolDeviceMatchingPolicyService struct {
	db    *gorm.DB
	cache *DeviceCache
}

// NewResourcePoolDeviceMatchingPolicyService 创建资源池设备匹配策略服务
func NewResourcePoolDeviceMatchingPolicyService(db *gorm.DB, cache *DeviceCache) *ResourcePoolDeviceMatchingPolicyService {
	return &ResourcePoolDeviceMatchingPolicyService{
		db:    db,
		cache: cache,
	}
}

// GetResourcePoolDeviceMatchingPolicies 获取资源池设备匹配策略列表
func (s *ResourcePoolDeviceMatchingPolicyService) GetResourcePoolDeviceMatchingPolicies(ctx context.Context, page, size int) (*ResourcePoolDeviceMatchingPolicyListResponse, error) {
	// 验证分页参数
	if page <= 0 {
		page = DefaultPage
	}
	if size <= 0 || size > MaxSize {
		size = DefaultSize
	}

	// 计算数据库偏移量
	offset := (page - 1) * size

	// 查询总数
	var total int64
	if err := s.db.WithContext(ctx).Model(&portal.ResourcePoolDeviceMatchingPolicy{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count policies: %w", err)
	}

	// 从数据库获取分页的策略
	var dbPolicies []portal.ResourcePoolDeviceMatchingPolicy
	if err := s.db.WithContext(ctx).
		Offset(offset).
		Limit(size).
		Order("id desc"). // 默认按ID降序排列
		Find(&dbPolicies).Error; err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}

	// 转换为服务层策略格式
	policies := make([]ResourcePoolDeviceMatchingPolicy, len(dbPolicies))
	for i, dbPolicy := range dbPolicies {
		// 解析查询条件组
		var queryGroups []FilterGroup
		if err := json.Unmarshal([]byte(dbPolicy.QueryGroups), &queryGroups); err != nil {
			return nil, fmt.Errorf("failed to unmarshal query groups for policy %d: %w", dbPolicy.ID, err)
		}

		policies[i] = ResourcePoolDeviceMatchingPolicy{
			ID:               dbPolicy.ID,
			Name:             dbPolicy.Name,
			Description:      dbPolicy.Description,
			ResourcePoolType: dbPolicy.ResourcePoolType,
			ActionType:       dbPolicy.ActionType,
			QueryGroups:      queryGroups,
			Status:           dbPolicy.Status,
			CreatedBy:        dbPolicy.CreatedBy,
			UpdatedBy:        dbPolicy.UpdatedBy,
			CreatedAt:        time.Time(dbPolicy.CreatedAt),
			UpdatedAt:        time.Time(dbPolicy.UpdatedAt),
		}
	}

	// 构建响应
	response := &ResourcePoolDeviceMatchingPolicyListResponse{
		List:  policies,
		Total: total,
		Page:  page,
		Size:  size,
	}

	return response, nil
}

// GetResourcePoolDeviceMatchingPolicy 获取资源池设备匹配策略详情
func (s *ResourcePoolDeviceMatchingPolicyService) GetResourcePoolDeviceMatchingPolicy(ctx context.Context, id int64) (*ResourcePoolDeviceMatchingPolicy, error) {
	// 从数据库获取指定策略
	var dbPolicy portal.ResourcePoolDeviceMatchingPolicy
	if err := s.db.WithContext(ctx).First(&dbPolicy, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("policy not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	// 解析查询条件组
	var queryGroups []FilterGroup
	if err := json.Unmarshal([]byte(dbPolicy.QueryGroups), &queryGroups); err != nil {
		return nil, fmt.Errorf("failed to unmarshal query groups for policy %d: %w", dbPolicy.ID, err)
	}

	// 转换为服务层策略格式
	policy := &ResourcePoolDeviceMatchingPolicy{
		ID:               dbPolicy.ID,
		Name:             dbPolicy.Name,
		Description:      dbPolicy.Description,
		ResourcePoolType: dbPolicy.ResourcePoolType,
		ActionType:       dbPolicy.ActionType,
		QueryGroups:      queryGroups,
		Status:           dbPolicy.Status,
		CreatedBy:        dbPolicy.CreatedBy,
		UpdatedBy:        dbPolicy.UpdatedBy,
		CreatedAt:        time.Time(dbPolicy.CreatedAt),
		UpdatedAt:        time.Time(dbPolicy.UpdatedAt),
	}

	return policy, nil
}

// CreateResourcePoolDeviceMatchingPolicy 创建资源池设备匹配策略
func (s *ResourcePoolDeviceMatchingPolicyService) CreateResourcePoolDeviceMatchingPolicy(ctx context.Context, policy *ResourcePoolDeviceMatchingPolicy) error {
	// 验证输入参数
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	if policy.ResourcePoolType == "" {
		return fmt.Errorf("resource pool type is required")
	}

	if policy.ActionType == "" {
		return fmt.Errorf("action type is required")
	}

	if len(policy.QueryGroups) == 0 {
		return fmt.Errorf("query groups are required")
	}

	// 将查询条件组转换为JSON字符串
	queryGroupsJSON, err := json.Marshal(policy.QueryGroups)
	if err != nil {
		return fmt.Errorf("failed to marshal query groups: %w", err)
	}

	// 将策略数据转换为数据库模型
	dbPolicy := &portal.ResourcePoolDeviceMatchingPolicy{
		Name:             policy.Name,
		Description:      policy.Description,
		ResourcePoolType: policy.ResourcePoolType,
		ActionType:       policy.ActionType,
		QueryGroups:      string(queryGroupsJSON),
		Status:           policy.Status,
		CreatedBy:        policy.CreatedBy,
		UpdatedBy:        policy.UpdatedBy,
	}

	// 创建新策略
	result := s.db.WithContext(ctx).Create(dbPolicy)
	if result.Error != nil {
		return fmt.Errorf("failed to create policy: %w", result.Error)
	}

	// 更新返回的策略ID
	policy.ID = dbPolicy.ID
	policy.CreatedAt = time.Time(dbPolicy.CreatedAt)
	policy.UpdatedAt = time.Time(dbPolicy.UpdatedAt)

	// 清除相关缓存（如果有）
	if s.cache != nil {
		// 这里可以添加清除缓存的逻辑，如果需要的话
		// 例如: s.cache.InvalidateResourcePolicies()
	}

	return nil
}

// UpdateResourcePoolDeviceMatchingPolicy 更新资源池设备匹配策略
func (s *ResourcePoolDeviceMatchingPolicyService) UpdateResourcePoolDeviceMatchingPolicy(ctx context.Context, policy *ResourcePoolDeviceMatchingPolicy) error {
	// 验证输入参数
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	if policy.ID <= 0 {
		return fmt.Errorf("invalid policy ID: %d", policy.ID)
	}

	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	if policy.ResourcePoolType == "" {
		return fmt.Errorf("resource pool type is required")
	}

	if policy.ActionType == "" {
		return fmt.Errorf("action type is required")
	}

	if len(policy.QueryGroups) == 0 {
		return fmt.Errorf("query groups are required")
	}

	// 检查策略是否存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&portal.ResourcePoolDeviceMatchingPolicy{}).Where("id = ?", policy.ID).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check policy existence: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("policy not found: %d", policy.ID)
	}

	// 将查询条件组转换为JSON字符串
	queryGroupsJSON, err := json.Marshal(policy.QueryGroups)
	if err != nil {
		return fmt.Errorf("failed to marshal query groups: %w", err)
	}

	// 更新策略
	result := s.db.WithContext(ctx).Model(&portal.ResourcePoolDeviceMatchingPolicy{}).
		Where("id = ?", policy.ID).
		Updates(map[string]interface{}{
			"name":               policy.Name,
			"description":        policy.Description,
			"resource_pool_type": policy.ResourcePoolType,
			"action_type":        policy.ActionType,
			"query_groups":       string(queryGroupsJSON),
			"status":             policy.Status,
			"updated_by":         policy.UpdatedBy,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update policy: %w", result.Error)
	}

	// 清除相关缓存（如果有）
	if s.cache != nil {
		// 这里可以添加清除缓存的逻辑，如果需要的话
		// 例如: s.cache.InvalidateResourcePolicies()
	}

	return nil
}

// DeleteResourcePoolDeviceMatchingPolicy 删除资源池设备匹配策略
func (s *ResourcePoolDeviceMatchingPolicyService) DeleteResourcePoolDeviceMatchingPolicy(ctx context.Context, id int64) error {
	// 验证ID参数
	if id <= 0 {
		return fmt.Errorf("invalid policy ID: %d", id)
	}

	// 检查策略是否存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&portal.ResourcePoolDeviceMatchingPolicy{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check policy existence: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("policy not found: %d", id)
	}

	// 删除策略
	if err := s.db.WithContext(ctx).Delete(&portal.ResourcePoolDeviceMatchingPolicy{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	// 清除相关缓存（如果有）
	if s.cache != nil {
		// 这里可以添加清除缓存的逻辑，如果需要的话
		// 例如: s.cache.InvalidateResourcePolicies()
	}

	return nil
}

// UpdateResourcePoolDeviceMatchingPolicyStatus 更新资源池设备匹配策略状态
func (s *ResourcePoolDeviceMatchingPolicyService) UpdateResourcePoolDeviceMatchingPolicyStatus(ctx context.Context, id int64, status string) error {
	// 验证参数
	if id <= 0 {
		return fmt.Errorf("invalid policy ID: %d", id)
	}

	// 验证状态值
	if status != "enabled" && status != "disabled" {
		return fmt.Errorf("invalid status: %s, must be 'enabled' or 'disabled'", status)
	}

	// 检查策略是否存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&portal.ResourcePoolDeviceMatchingPolicy{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check policy existence: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("policy not found: %d", id)
	}

	// 更新状态
	if err := s.db.WithContext(ctx).Model(&portal.ResourcePoolDeviceMatchingPolicy{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update policy status: %w", err)
	}

	// 清除相关缓存（如果有）
	if s.cache != nil {
		// 这里可以添加清除缓存的逻辑，如果需要的话
		// 例如: s.cache.InvalidateResourcePolicies()
	}

	return nil
}

// GetResourcePoolDeviceMatchingPoliciesByType 根据资源池类型和动作类型获取匹配策略
func (s *ResourcePoolDeviceMatchingPolicyService) GetResourcePoolDeviceMatchingPoliciesByType(ctx context.Context, resourcePoolType, actionType string) ([]ResourcePoolDeviceMatchingPolicy, error) {
	// 验证参数
	if resourcePoolType == "" {
		return nil, fmt.Errorf("resource pool type is required")
	}

	if actionType == "" {
		return nil, fmt.Errorf("action type is required")
	}

	// 从数据库获取指定类型的策略
	var dbPolicies []portal.ResourcePoolDeviceMatchingPolicy
	if err := s.db.WithContext(ctx).
		Where("resource_pool_type = ? AND action_type = ? AND status = 'enabled'", resourcePoolType, actionType).
		Find(&dbPolicies).Error; err != nil {
		return nil, fmt.Errorf("failed to get policies by type: %w", err)
	}

	// 转换为服务层策略格式
	policies := make([]ResourcePoolDeviceMatchingPolicy, len(dbPolicies))
	for i, dbPolicy := range dbPolicies {
		// 解析查询条件组
		var queryGroups []FilterGroup
		if err := json.Unmarshal([]byte(dbPolicy.QueryGroups), &queryGroups); err != nil {
			return nil, fmt.Errorf("failed to unmarshal query groups for policy %d: %w", dbPolicy.ID, err)
		}

		policies[i] = ResourcePoolDeviceMatchingPolicy{
			ID:               dbPolicy.ID,
			Name:             dbPolicy.Name,
			Description:      dbPolicy.Description,
			ResourcePoolType: dbPolicy.ResourcePoolType,
			ActionType:       dbPolicy.ActionType,
			QueryGroups:      queryGroups,
			Status:           dbPolicy.Status,
			CreatedBy:        dbPolicy.CreatedBy,
			UpdatedBy:        dbPolicy.UpdatedBy,
			CreatedAt:        time.Time(dbPolicy.CreatedAt),
			UpdatedAt:        time.Time(dbPolicy.UpdatedAt),
		}
	}

	return policies, nil
}
