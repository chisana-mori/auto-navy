package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/now"
	"gorm.io/gorm"

	"navy-ng/models/portal"
)

// camelToSnake 将驼峰命名法转换为下划线命名法
func camelToSnake(s string) string {
	// 在大写字母前添加下划线，然后将所有字母转换为小写
	var result string
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result += "_"
		}
		result += string(r)
	}
	return strings.ToLower(result)
}

// FilterType 筛选类型
type FilterType string

const (
	FilterTypeNodeLabel FilterType = "nodeLabel" // 节点标签
	FilterTypeTaint     FilterType = "taint"     // 污点
	FilterTypeDevice    FilterType = "device"    // 设备字段
)

// applyFilterBlock 应用筛选块
func (s *DeviceQueryService) applyFilterBlock(query *gorm.DB, block FilterBlock) *gorm.DB {
	switch block.Type {
	case FilterTypeNodeLabel:
		return s.applyNodeLabelFilter(query, block)
	case FilterTypeTaint:
		return s.applyTaintFilter(query, block)
	case FilterTypeDevice:
		return s.applyDeviceFilter(query, block)
	default:
		return query
	}
}

// applyDeviceFilter 应用设备字段筛选
func (s *DeviceQueryService) applyDeviceFilter(query *gorm.DB, block FilterBlock) *gorm.DB {
	// 预处理LIKE查询的值，转义特殊字符
	escapedValue := strings.ReplaceAll(block.Value, "%", "\\%")
	escapedValue = strings.ReplaceAll(escapedValue, "_", "\\_")

	// 根据字段名构建查询条件
	var column string
	switch block.Key {
	case "machineType", "machine_type":
		column = "d.machine_type"
	case "appId", "app_id":
		column = "d.app_id"
	case "resourcePool", "resource_pool":
		column = "d.resource_pool"
	case "deviceId", "device_id":
		column = "d.device_id"
	case "ip":
		column = "d.ip"
	case "cluster":
		column = "d.cluster"
	case "role":
		column = "d.role"
	case "arch":
		column = "d.arch"
	case "idc":
		column = "d.idc"
	case "room":
		column = "d.room"
	case "datacenter":
		column = "d.datacenter"
	case "cabinet":
		column = "d.cabinet"
	case "network":
		column = "d.network"
	default:
		// 尝试将驼峰命名法转换为下划线命名法
		snakeCase := camelToSnake(block.Key)
		column = fmt.Sprintf("d.%s", snakeCase)
	}

	switch block.ConditionType {
	case ConditionTypeEqual:
		return query.Where(column+" = ?", block.Value)
	case ConditionTypeNotEqual:
		return query.Where(column+" != ?", block.Value)
	case ConditionTypeContains:
		return query.Where(column+" LIKE ?", "%"+escapedValue+"%")
	case ConditionTypeNotContains:
		return query.Where(column+" NOT LIKE ?", "%"+escapedValue+"%")
	case ConditionTypeIn:
		// 处理IN查询，将逗号分隔的值转换为切片
		values := strings.Split(block.Value, ",")
		// 对每个值进行转义
		for i, v := range values {
			values[i] = strings.TrimSpace(v)
		}
		return query.Where(column+" IN (?)", values)
	case ConditionTypeNotIn:
		// 处理NOT IN查询
		values := strings.Split(block.Value, ",")
		// 对每个值进行转义
		for i, v := range values {
			values[i] = strings.TrimSpace(v)
		}
		return query.Where(column+" NOT IN (?)", values)
	default:
		return query
	}
}

// ConditionType 条件类型
type ConditionType string

const (
	ConditionTypeEqual       ConditionType = "equal"       // 等于
	ConditionTypeNotEqual    ConditionType = "notEqual"    // 不等于
	ConditionTypeContains    ConditionType = "contains"    // 包含
	ConditionTypeNotContains ConditionType = "notContains" // 不包含
	ConditionTypeExists      ConditionType = "exists"      // 存在
	ConditionTypeNotExists   ConditionType = "notExists"   // 不存在
	ConditionTypeIn          ConditionType = "in"          // 在列表中
	ConditionTypeNotIn       ConditionType = "notIn"       // 不在列表中
)

// LogicalOperator 逻辑运算符
type LogicalOperator string

const (
	LogicalOperatorAnd LogicalOperator = "and" // 与
	LogicalOperatorOr  LogicalOperator = "or"  // 或
)

// FilterOption 筛选选项
type FilterOption struct {
	ID       string `json:"id"`       // 选项ID
	Label    string `json:"label"`    // 选项标签
	Value    string `json:"value"`    // 选项值
	DbColumn string `json:"dbColumn"` // 数据库列名
}

// FilterBlock 筛选块
type FilterBlock struct {
	ID            string          `json:"id"`            // 筛选块ID
	Type          FilterType      `json:"type"`          // 筛选类型
	ConditionType ConditionType   `json:"conditionType"` // 条件类型
	Key           string          `json:"key"`           // 键
	Value         string          `json:"value"`         // 值
	Operator      LogicalOperator `json:"operator"`      // 与下一个条件的逻辑关系
}

// FilterGroup 筛选组
type FilterGroup struct {
	ID       string          `json:"id"`       // 筛选组ID
	Blocks   []FilterBlock   `json:"blocks"`   // 筛选块列表
	Operator LogicalOperator `json:"operator"` // 与下一个组的逻辑关系
}

// QueryTemplate 查询模板
type QueryTemplate struct {
	ID          int64         `json:"id"`          // 模板ID
	Name        string        `json:"name"`        // 模板名称
	Description string        `json:"description"` // 模板描述
	Groups      []FilterGroup `json:"groups"`      // 筛选组列表
}

// DeviceQueryRequest 设备查询请求
type DeviceQueryRequest struct {
	Groups []FilterGroup `json:"groups"` // 筛选组列表
	Page   int           `json:"page"`   // 页码
	Size   int           `json:"size"`   // 每页数量
}

// DeviceFieldValues 设备字段值
type DeviceFieldValues struct {
	Field  string         `json:"field"`
	Values []FilterOption `json:"values"`
}

// DeviceQueryService 设备查询服务
type DeviceQueryService struct {
	db *gorm.DB
}

// NewDeviceQueryService 创建设备查询服务
func NewDeviceQueryService(db *gorm.DB) *DeviceQueryService {
	return &DeviceQueryService{db: db}
}

// GetFilterOptions 获取筛选选项
func (s *DeviceQueryService) GetFilterOptions(ctx context.Context) (map[string]interface{}, error) {
	options := make(map[string]interface{})

	todayStart := now.BeginningOfDay()
	todayEnd := now.EndOfDay()

	// 获取节点标签的 key 列表
	var labelKeys []string
	if err := s.db.WithContext(ctx).Model(&portal.K8sNodeLabel{}).
		Select("DISTINCT `key`").
		Where("`key` != ?", "").
		Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).
		Pluck("key", &labelKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get node label keys: %w", err)
	}

	var labelKeyOptions []FilterOption
	for _, key := range labelKeys {
		labelKeyOptions = append(labelKeyOptions, FilterOption{
			ID:    key,
			Label: key,
			Value: key,
		})
	}
	options["nodeLabelKeys"] = labelKeyOptions

	// 获取节点污点的 key 列表
	var taintKeys []string
	if err := s.db.WithContext(ctx).Model(&portal.K8sNodeTaint{}).
		Select("DISTINCT `key`").
		Where("`key` != ?", "").
		Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).
		Pluck("key", &taintKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get node taint keys: %w", err)
	}

	var taintKeyOptions []FilterOption
	for _, key := range taintKeys {
		taintKeyOptions = append(taintKeyOptions, FilterOption{
			ID:    key,
			Label: key,
			Value: key,
		})
	}
	options["nodeTaintKeys"] = taintKeyOptions

	// 获取设备字段列表
	deviceFields, err := s.GetDeviceFields(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device fields: %w", err)
	}

	// 获取设备字段的可选值
	var deviceFieldValuesList []DeviceFieldValues
	for _, field := range deviceFields {
		var values []string
		query := s.db.WithContext(ctx).Table("device").
			Select(fmt.Sprintf("DISTINCT %s", field.Value)).
			Where(fmt.Sprintf("%s IS NOT NULL", field.Value)).
			Where(fmt.Sprintf("%s != ''", field.Value)).
			Where("deleted = ''")

		if err := query.Pluck(field.Value, &values).Error; err != nil {
			return nil, fmt.Errorf("failed to get device field values for %s: %w", field.Value, err)
		}

		if len(values) > 0 {
			fieldOptions := make([]FilterOption, 0, len(values))
			for _, value := range values {
				fieldOptions = append(fieldOptions, FilterOption{
					ID:    fmt.Sprintf("%s-%s", field.Value, value),
					Label: value,
					Value: value,
				})
			}

			deviceFieldValuesList = append(deviceFieldValuesList, DeviceFieldValues{
				Field:  field.Value,
				Values: fieldOptions,
			})
		}
	}
	options["deviceFieldValues"] = deviceFieldValuesList

	return options, nil
}

// GetLabelValues 获取标签值
func (s *DeviceQueryService) GetLabelValues(ctx context.Context, key string) ([]FilterOption, error) {
	var values []string

	// 获取今天的开始和结束时间
	todayStart := now.BeginningOfDay()
	todayEnd := now.EndOfDay()

	if err := s.db.WithContext(ctx).Model(&portal.K8sNodeLabel{}).
		Select("DISTINCT value").
		Where("`key` = ?", key).
		Where("value != ''").
		Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).
		Order("value").
		Pluck("value", &values).Error; err != nil {
		return nil, fmt.Errorf("failed to get label values: %w", err)
	}

	options := make([]FilterOption, 0, len(values))
	for _, value := range values {
		options = append(options, FilterOption{
			ID:    fmt.Sprintf("%s-%s", key, value),
			Label: value,
			Value: value,
		})
	}
	return options, nil
}

// GetTaintValues 获取污点值
func (s *DeviceQueryService) GetTaintValues(ctx context.Context, key string) ([]FilterOption, error) {
	var values []string
	// 获取今天的开始和结束时间
	todayStart := now.BeginningOfDay()
	todayEnd := now.EndOfDay()

	if err := s.db.WithContext(ctx).Model(&portal.K8sNodeTaint{}).
		Select("DISTINCT value").
		Where("`key` = ?", key).
		Where("value != ''").
		Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).
		Order("value").
		Pluck("value", &values).Error; err != nil {
		return nil, fmt.Errorf("failed to get taint values: %w", err)
	}

	options := make([]FilterOption, 0, len(values))
	for _, value := range values {
		options = append(options, FilterOption{
			ID:    fmt.Sprintf("%s-%s", key, value),
			Label: value,
			Value: value,
		})
	}
	return options, nil
}

// QueryDevices 查询设备
func (s *DeviceQueryService) QueryDevices(ctx context.Context, req *DeviceQueryRequest) (*DeviceListResponse, error) {
	// 构建基础查询
	query := s.db.WithContext(ctx).Table("device d").
		Select("d.id, d.device_id as deviceId, d.ip, d.machine_type as machineType, d.cluster, d.role, d.arch, d.idc, d.room, d.datacenter, d.cabinet, d.network, d.app_id as appId, d.resource_pool as resourcePool, d.created_at as createdAt, d.updated_at as updatedAt").
		Where("d.deleted = ?", "")

	// 检查是否需要关联k8s_node表
	needJoinK8sNode := false
	if len(req.Groups) > 0 {
		for _, group := range req.Groups {
			for _, block := range group.Blocks {
				if block.Type == FilterTypeNodeLabel || block.Type == FilterTypeTaint {
					needJoinK8sNode = true
					break
				}
			}
			if needJoinK8sNode {
				break
			}
		}
	}

	// 仅在需要时添加JOIN
	if needJoinK8sNode {
		query = query.Joins("LEFT JOIN k8s_node kn ON LOWER(d.device_id) = LOWER(kn.nodename)")
	}

	// 应用筛选条件
	if len(req.Groups) > 0 {
		for i, group := range req.Groups {
			if len(group.Blocks) == 0 {
				continue
			}

			// 为每个组创建一个子查询
			subQuery := s.db.WithContext(ctx).Table("device d")
			if needJoinK8sNode {
				subQuery = subQuery.Joins("LEFT JOIN k8s_node kn ON LOWER(d.device_id) = LOWER(kn.nodename)")
			}
			subQuery = subQuery.Select("d.id")

			for _, block := range group.Blocks {
				subQuery = s.applyFilterBlock(subQuery, block)
			}

			// 将子查询应用到主查询
			if i == 0 {
				for _, block := range group.Blocks {
					query = s.applyFilterBlock(query, block)
				}
			} else if req.Groups[i-1].Operator == LogicalOperatorOr {
				subQuery := s.db.WithContext(ctx).Table("device d")
				if needJoinK8sNode {
					subQuery = subQuery.Joins("LEFT JOIN k8s_node kn ON LOWER(d.device_id) = LOWER(kn.nodename)")
				}
				for _, block := range group.Blocks {
					subQuery = s.applyFilterBlock(subQuery, block)
				}
				subQuery = subQuery.Select("d.id")
				query = query.Or("d.id IN (?)", subQuery)
			} else {
				for _, block := range group.Blocks {
					query = s.applyFilterBlock(query, block)
				}
			}
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count devices: %w", err)
	}

	// 分页
	page := req.Page
	if page <= 0 {
		page = 1
	}

	size := req.Size
	if size <= 0 {
		size = 10
	}

	offset := (page - 1) * size

	// 获取设备列表
	var devices []portal.Device
	if err := query.Offset(offset).Limit(size).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}

	// 转换为响应格式
	responses := make([]DeviceResponse, len(devices))
	for i, device := range devices {
		responses[i] = DeviceResponse{
			ID:           device.ID,
			DeviceID:     device.DeviceID,
			IP:           device.IP,
			MachineType:  device.MachineType,
			Cluster:      device.Cluster,
			Role:         device.Role,
			Arch:         device.Arch,
			IDC:          device.IDC,
			Room:         device.Room,
			Datacenter:   device.Datacenter,
			Cabinet:      device.Cabinet,
			Network:      device.Network,
			AppID:        device.AppID,
			ResourcePool: device.ResourcePool,
			CreatedAt:    time.Time(device.CreatedAt),
			UpdatedAt:    time.Time(device.UpdatedAt),
		}
	}

	return &DeviceListResponse{
		List:  responses,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// applyNodeLabelFilter 应用节点标签筛选
func (s *DeviceQueryService) applyNodeLabelFilter(query *gorm.DB, block FilterBlock) *gorm.DB {
	// 预处理LIKE查询的值，转义特殊字符
	escapedValue := strings.ReplaceAll(block.Value, "%", "\\%")
	escapedValue = strings.ReplaceAll(escapedValue, "_", "\\_")

	// 获取当天的开始和结束时间
	todayStart := now.BeginningOfDay()
	todayEnd := now.EndOfDay()

	// 使用 INNER JOIN 而不是 LEFT JOIN，确保只返回匹配所有条件的记录
	// 并且只查询当天的标签数据
	query = query.Joins("INNER JOIN k8s_node_label nl ON kn.id = nl.node_id AND nl.key = ? AND nl.created_at BETWEEN ? AND ?",
		block.Key, todayStart, todayEnd)

	switch block.ConditionType {
	case ConditionTypeEqual:
		return query.Where("nl.value = ?", block.Value)
	case ConditionTypeNotEqual:
		return query.Where("nl.value != ? OR nl.value IS NULL", block.Value)
	case ConditionTypeContains:
		return query.Where("nl.value LIKE ?", "%"+escapedValue+"%")
	case ConditionTypeNotContains:
		return query.Where("nl.value NOT LIKE ? OR nl.value IS NULL", "%"+escapedValue+"%")
	case ConditionTypeExists:
		return query.Where("nl.value IS NOT NULL")
	case ConditionTypeNotExists:
		return query.Where("nl.value IS NULL")
	case ConditionTypeIn:
		values := strings.Split(block.Value, ",")
		for i, v := range values {
			values[i] = strings.TrimSpace(v)
		}
		return query.Where("nl.value IN (?)", values)
	case ConditionTypeNotIn:
		values := strings.Split(block.Value, ",")
		for i, v := range values {
			values[i] = strings.TrimSpace(v)
		}
		return query.Where("nl.value NOT IN (?) OR nl.value IS NULL", values)
	default:
		return query
	}
}

// applyTaintFilter 应用污点筛选
func (s *DeviceQueryService) applyTaintFilter(query *gorm.DB, block FilterBlock) *gorm.DB {
	// 预处理LIKE查询的值，转义特殊字符
	escapedValue := strings.ReplaceAll(block.Value, "%", "\\%")
	escapedValue = strings.ReplaceAll(escapedValue, "_", "\\_")

	// 获取当天的开始和结束时间
	todayStart := now.BeginningOfDay()
	todayEnd := now.EndOfDay()

	// 使用 INNER JOIN 而不是 LEFT JOIN，确保只返回匹配所有条件的记录
	// 并且只查询当天的污点数据
	query = query.Joins("INNER JOIN k8s_node_taint nt ON kn.id = nt.node_id AND nt.key = ? AND nt.created_at BETWEEN ? AND ?",
		block.Key, todayStart, todayEnd)

	switch block.ConditionType {
	case ConditionTypeEqual:
		return query.Where("nt.value = ?", block.Value)
	case ConditionTypeNotEqual:
		return query.Where("nt.value != ? OR nt.value IS NULL", block.Value)
	case ConditionTypeContains:
		return query.Where("nt.value LIKE ?", "%"+escapedValue+"%")
	case ConditionTypeNotContains:
		return query.Where("nt.value NOT LIKE ? OR nt.value IS NULL", "%"+escapedValue+"%")
	case ConditionTypeExists:
		return query.Where("nt.value IS NOT NULL")
	case ConditionTypeNotExists:
		return query.Where("nt.value IS NULL")
	case ConditionTypeIn:
		values := strings.Split(block.Value, ",")
		for i, v := range values {
			values[i] = strings.TrimSpace(v)
		}
		return query.Where("nt.value IN (?)", values)
	case ConditionTypeNotIn:
		values := strings.Split(block.Value, ",")
		for i, v := range values {
			values[i] = strings.TrimSpace(v)
		}
		return query.Where("nt.value NOT IN (?) OR nt.value IS NULL", values)
	default:
		return query
	}
}

// SaveQueryTemplate 保存查询模板
func (s *DeviceQueryService) SaveQueryTemplate(ctx context.Context, template *QueryTemplate) error {
	// 将模板数据转换为数据库模型
	groupsJSON, err := json.Marshal(template.Groups)
	if err != nil {
		return fmt.Errorf("failed to marshal groups: %w", err)
	}

	dbTemplate := &portal.QueryTemplate{
		Name:        template.Name,
		Description: template.Description,
		Groups:      string(groupsJSON),
		CreatedBy:   "system", // TODO: 从上下文获取当前用户
		UpdatedBy:   "system", // TODO: 从上下文获取当前用户
	}

	// 如果模板ID存在，则更新现有模板
	if template.ID != 0 {
		dbTemplate.ID = template.ID
		result := s.db.WithContext(ctx).Save(dbTemplate)
		if result.Error != nil {
			return fmt.Errorf("failed to update template: %w", result.Error)
		}
		return nil
	}

	// 创建新模板
	result := s.db.WithContext(ctx).Create(dbTemplate)
	if result.Error != nil {
		return fmt.Errorf("failed to create template: %w", result.Error)
	}

	// 更新返回的模板ID
	template.ID = dbTemplate.ID
	return nil
}

// GetQueryTemplates 获取查询模板列表
func (s *DeviceQueryService) GetQueryTemplates(ctx context.Context) ([]QueryTemplate, error) {
	// 从数据库获取所有模板
	var dbTemplates []portal.QueryTemplate
	if err := s.db.WithContext(ctx).Find(&dbTemplates).Error; err != nil {
		return nil, fmt.Errorf("failed to get templates: %w", err)
	}

	// 转换为服务层模板格式
	templates := make([]QueryTemplate, len(dbTemplates))
	for i, dbTemplate := range dbTemplates {
		var groups []FilterGroup
		if err := json.Unmarshal([]byte(dbTemplate.Groups), &groups); err != nil {
			return nil, fmt.Errorf("failed to unmarshal groups for template %d: %w", dbTemplate.ID, err)
		}

		templates[i] = QueryTemplate{
			ID:          dbTemplate.ID,
			Name:        dbTemplate.Name,
			Description: dbTemplate.Description,
			Groups:      groups,
		}
	}

	return templates, nil
}

// GetQueryTemplate 获取查询模板
func (s *DeviceQueryService) GetQueryTemplate(ctx context.Context, id int64) (*QueryTemplate, error) {
	// 从数据库获取指定模板
	var dbTemplate portal.QueryTemplate
	if err := s.db.WithContext(ctx).First(&dbTemplate, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// 解析筛选组
	var groups []FilterGroup
	if err := json.Unmarshal([]byte(dbTemplate.Groups), &groups); err != nil {
		return nil, fmt.Errorf("failed to unmarshal groups: %w", err)
	}

	// 转换为服务层模板格式
	template := &QueryTemplate{
		ID:          dbTemplate.ID,
		Name:        dbTemplate.Name,
		Description: dbTemplate.Description,
		Groups:      groups,
	}

	return template, nil
}

// DeleteQueryTemplate 删除查询模板
func (s *DeviceQueryService) DeleteQueryTemplate(ctx context.Context, id int64) error {
	// 检查模板是否存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&portal.QueryTemplate{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check template existence: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("template not found: %d", id)
	}

	// 删除模板
	if err := s.db.WithContext(ctx).Delete(&portal.QueryTemplate{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// GetDeviceFields 获取设备字段列表
func (s *DeviceQueryService) GetDeviceFields(ctx context.Context) ([]FilterOption, error) {
	return []FilterOption{
		{
			ID:       "ip",
			Label:    "IP地址",
			Value:    "ip",
			DbColumn: "d.ip",
		},
		{
			ID:       "machine_type",
			Label:    "机器类型",
			Value:    "machine_type",
			DbColumn: "d.machine_type",
		},
		{
			ID:       "role",
			Label:    "集群角色",
			Value:    "role",
			DbColumn: "d.role",
		},
		{
			ID:       "arch",
			Label:    "架构",
			Value:    "arch",
			DbColumn: "d.arch",
		},
		{
			ID:       "idc",
			Label:    "IDC",
			Value:    "idc",
			DbColumn: "d.idc",
		},
		{
			ID:       "room",
			Label:    "Room",
			Value:    "room",
			DbColumn: "d.room",
		},
		{
			ID:       "datacenter",
			Label:    "机房",
			Value:    "datacenter",
			DbColumn: "d.datacenter",
		},
		{
			ID:       "cabinet",
			Label:    "机柜号",
			Value:    "cabinet",
			DbColumn: "d.cabinet",
		},
		{
			ID:       "network",
			Label:    "网络区域",
			Value:    "network",
			DbColumn: "d.network",
		},
		{
			ID:       "app_id",
			Label:    "APPID",
			Value:    "app_id",
			DbColumn: "d.app_id",
		},
		{
			ID:       "resource_pool",
			Label:    "资源池",
			Value:    "resource_pool",
			DbColumn: "d.resource_pool",
		},
	}, nil
}
