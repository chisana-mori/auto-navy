package service

import (
	"time"
)

// ResourcePoolDeviceMatchingPolicy 资源池设备匹配策略DTO
type ResourcePoolDeviceMatchingPolicy struct {
	ID               int           `json:"id"`                                                       // 主键
	Name             string        `json:"name" binding:"required"`                                  // 策略名称
	Description      string        `json:"description"`                                              // 策略描述
	ResourcePoolType string        `json:"resourcePoolType" binding:"required"`                      // 资源池类型
	ActionType       string        `json:"actionType" binding:"required,oneof=pool_entry pool_exit"` // 动作类型：pool_entry 或 pool_exit
	QueryTemplateID  int           `json:"queryTemplateId" binding:"required"`                       // 关联的查询模板ID
	QueryGroups      []FilterGroup `json:"queryGroups,omitempty"`                                    // 查询条件组（从查询模板获取，非直接存储字段）
	Status           string        `json:"status" binding:"required,oneof=enabled disabled"`         // 状态：enabled 或 disabled
	AdditionConds    []string      `json:"additionConds,omitempty"`                                  // 额外动态条件，仅入池时有效
	CreatedBy        string        `json:"createdBy"`                                                // 创建者
	UpdatedBy        string        `json:"updatedBy"`                                                // 更新者
	CreatedAt        time.Time     `json:"createdAt"`                                                // 创建时间
	UpdatedAt        time.Time     `json:"updatedAt"`                                                // 更新时间

	// 关联的查询模板信息
	QueryTemplate *QueryTemplate `json:"queryTemplate,omitempty"` // 关联的查询模板
}

// ResourcePoolDeviceMatchingPolicyListResponse 资源池设备匹配策略列表响应
type ResourcePoolDeviceMatchingPolicyListResponse struct {
	List  []ResourcePoolDeviceMatchingPolicy `json:"list"`  // 策略列表
	Total int64                              `json:"total"` // 总数
	Page  int                                `json:"page"`  // 当前页码
	Size  int                                `json:"size"`  // 每页数量
}

// CreateResourcePoolDeviceMatchingPolicyRequest 创建资源池设备匹配策略请求
type CreateResourcePoolDeviceMatchingPolicyRequest struct {
	Name             string   `json:"name" binding:"required"`                                  // 策略名称
	Description      string   `json:"description"`                                              // 策略描述
	ResourcePoolType string   `json:"resourcePoolType" binding:"required"`                      // 资源池类型
	ActionType       string   `json:"actionType" binding:"required,oneof=pool_entry pool_exit"` // 动作类型：pool_entry 或 pool_exit
	QueryTemplateID  int      `json:"queryTemplateId" binding:"required"`                       // 关联的查询模板ID
	Status           string   `json:"status" binding:"required,oneof=enabled disabled"`         // 状态：enabled 或 disabled
	AdditionConds    []string `json:"additionConds,omitempty"`                                  // 额外动态条件，仅入池时有效
}

// UpdateResourcePoolDeviceMatchingPolicyRequest 更新资源池设备匹配策略请求
type UpdateResourcePoolDeviceMatchingPolicyRequest struct {
	Name             string   `json:"name" binding:"required"`                                  // 策略名称
	Description      string   `json:"description"`                                              // 策略描述
	ResourcePoolType string   `json:"resourcePoolType" binding:"required"`                      // 资源池类型
	ActionType       string   `json:"actionType" binding:"required,oneof=pool_entry pool_exit"` // 动作类型：pool_entry 或 pool_exit
	QueryTemplateID  int      `json:"queryTemplateId" binding:"required"`                       // 关联的查询模板ID
	Status           string   `json:"status" binding:"required,oneof=enabled disabled"`         // 状态：enabled 或 disabled
	AdditionConds    []string `json:"additionConds,omitempty"`                                  // 额外动态条件，仅入池时有效
}

// UpdateResourcePoolDeviceMatchingPolicyStatusRequest 更新资源池设备匹配策略状态请求
type UpdateResourcePoolDeviceMatchingPolicyStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=enabled disabled"` // 状态：enabled 或 disabled
}

// ToServiceModel 将创建请求转换为服务模型
func (req *CreateResourcePoolDeviceMatchingPolicyRequest) ToServiceModel(username string) *ResourcePoolDeviceMatchingPolicy {
	return &ResourcePoolDeviceMatchingPolicy{
		Name:             req.Name,
		Description:      req.Description,
		ResourcePoolType: req.ResourcePoolType,
		ActionType:       req.ActionType,
		QueryTemplateID:  req.QueryTemplateID,
		Status:           req.Status,
		AdditionConds:    req.AdditionConds,
		CreatedBy:        username,
		UpdatedBy:        username,
	}
}

// ToServiceModel 将更新请求转换为服务模型
func (req *UpdateResourcePoolDeviceMatchingPolicyRequest) ToServiceModel(id int, username string) *ResourcePoolDeviceMatchingPolicy {
	return &ResourcePoolDeviceMatchingPolicy{
		ID:               id,
		Name:             req.Name,
		Description:      req.Description,
		ResourcePoolType: req.ResourcePoolType,
		ActionType:       req.ActionType,
		QueryTemplateID:  req.QueryTemplateID,
		Status:           req.Status,
		AdditionConds:    req.AdditionConds,
		UpdatedBy:        username,
	}
}

// ToResponse 将服务模型转换为响应模型
func (p *ResourcePoolDeviceMatchingPolicy) ToResponse() *ResourcePoolDeviceMatchingPolicy {
	return &ResourcePoolDeviceMatchingPolicy{
		ID:               p.ID,
		Name:             p.Name,
		Description:      p.Description,
		ResourcePoolType: p.ResourcePoolType,
		ActionType:       p.ActionType,
		QueryTemplateID:  p.QueryTemplateID,
		QueryGroups:      p.QueryGroups,
		QueryTemplate:    p.QueryTemplate,
		Status:           p.Status,
		CreatedBy:        p.CreatedBy,
		UpdatedBy:        p.UpdatedBy,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

// 使用 device_query.go 中定义的 QueryTemplate 类型

// 使用 device_query.go 中定义的 FilterBlock 和 FilterGroup 类型
