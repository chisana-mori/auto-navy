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

// isValidColumnName performs basic validation on potential column names.
// WARNING: This is a simplistic check and might not cover all injection vectors.
// A strict allowlist of known columns is generally safer.
func isValidColumnName(name string) bool {
	// Allow only alphanumeric characters and underscores
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	// Prevent overly long names
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	// Add more checks if needed (e.g., blacklist SQL keywords)
	return true
}


// applyDeviceFilter 应用设备字段筛选
func (s *DeviceQueryService) applyDeviceFilter(query *gorm.DB, block FilterBlock) *gorm.DB {
	column, ok := deviceFieldColumnMap[block.Key]
	if !ok {
		// Fallback for keys not in the map: convert camelCase to snake_case
		// Apply basic validation to the generated column name
		snakeCase := camelToSnake(block.Key)
		if !isValidColumnName(snakeCase) {
			fmt.Printf("Warning: Invalid or potentially unsafe column key detected: %s\n", block.Key)
			return query // Return unchanged query if the key is suspicious
		}
		column = fmt.Sprintf("d.%s", snakeCase)
		// Consider logging a warning here that a direct map entry is preferred
		fmt.Printf("Warning: Column key '%s' not found in deviceFieldColumnMap, using generated '%s'. Consider adding it to the map.\n", block.Key, column)
	}

	// Build and apply the condition using the helper function
	conditionScope := buildCondition(column, block.ConditionType, block.Value)
	return query.Scopes(conditionScope)
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
// deviceFieldColumnMap maps frontend keys to database column names for the device table
var deviceFieldColumnMap = map[string]string{
	"machineType":  "d.machine_type",
	"machine_type": "d.machine_type",
	"appId":        "d.app_id",
	"app_id":       "d.app_id",
	"resourcePool": "d.resource_pool",
	"resource_pool":"d.resource_pool",
	"deviceId":     "d.device_id",
	"device_id":    "d.device_id",
	"ip":           "d.ip",
	"cluster":      "d.cluster",
	"role":         "d.role",
	"arch":         "d.arch",
	"idc":          "d.idc",
	"room":         "d.room",
	"datacenter":   "d.datacenter",
	"cabinet":      "d.cabinet",
	"network":      "d.network",
	// Add other direct mappings if needed
}

// buildCondition creates a GORM scope function based on the condition type and value.
func buildCondition(column string, conditionType ConditionType, value string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// Escape value for LIKE conditions
		escapedValue := strings.ReplaceAll(value, "%", "\\%")
		escapedValue = strings.ReplaceAll(escapedValue, "_", "\\_")

		switch conditionType {
		case ConditionTypeEqual:
			return db.Where(column+" = ?", value)
		case ConditionTypeNotEqual:
			// Handle NULL properly for NOT EQUAL
			return db.Where(fmt.Sprintf("(%s != ? OR %s IS NULL)", column, column), value)
		case ConditionTypeContains:
			return db.Where(column+" LIKE ?", "%"+escapedValue+"%")
		case ConditionTypeNotContains:
			// Handle NULL properly for NOT CONTAINS
			return db.Where(fmt.Sprintf("(%s NOT LIKE ? OR %s IS NULL)", column, column), "%"+escapedValue+"%")
		case ConditionTypeExists:
			// Check for non-null and non-empty string
			return db.Where(fmt.Sprintf("%s IS NOT NULL AND %s != ''", column, column))
		case ConditionTypeNotExists:
			// Check for null or empty string
			return db.Where(fmt.Sprintf("%s IS NULL OR %s = ''", column, column))
		case ConditionTypeIn:
			values := strings.Split(value, ",")
			trimmedValues := make([]string, 0, len(values))
			for _, v := range values {
				trimmed := strings.TrimSpace(v)
				if trimmed != "" {
					trimmedValues = append(trimmedValues, trimmed)
				}
			}
			if len(trimmedValues) == 0 {
				// Avoid WHERE column IN (NULL) which might behave unexpectedly
				return db.Where("1 = 0") // Effectively returns no results
			}
			return db.Where(column+" IN (?)", trimmedValues)
		case ConditionTypeNotIn:
			values := strings.Split(value, ",")
			trimmedValues := make([]string, 0, len(values))
			for _, v := range values {
				trimmed := strings.TrimSpace(v)
				if trimmed != "" {
					trimmedValues = append(trimmedValues, trimmed)
				}
			}
			if len(trimmedValues) == 0 {
				// NOT IN empty set should return all non-null rows
				return db.Where(fmt.Sprintf("%s IS NOT NULL", column))
			}
			// Handle NULL properly for NOT IN
			return db.Where(fmt.Sprintf("(%s NOT IN (?) OR %s IS NULL)", column, column), trimmedValues)
		default:
			// Unknown condition type, return unchanged query
			return db
		}
	}
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

// determineRequiredJoins checks if the provided filter groups require joining the k8s_node table.
// It iterates through all blocks in all groups to see if any filter type is NodeLabel or Taint.
func determineRequiredJoins(groups []FilterGroup) bool {
	for _, group := range groups {
		for _, block := range group.Blocks {
			if block.Type == FilterTypeNodeLabel || block.Type == FilterTypeTaint {
				return true
			}
		}
	}
	return false
}

// buildDeviceQuery constructs the base GORM query for devices.
// It selects necessary device fields, filters out deleted devices,
// and optionally adds a LEFT JOIN to the k8s_node table if needed based on filter types.
func (s *DeviceQueryService) buildDeviceQuery(ctx context.Context, needJoinK8sNode bool) *gorm.DB {
	query := s.db.WithContext(ctx).Table("device d").
		Select("d.id, d.device_id, d.ip, d.machine_type, d.cluster, d.role, d.arch, d.idc, d.room, d.datacenter, d.cabinet, d.network, d.app_id, d.resource_pool, d.created_at, d.updated_at").
		Where("d.deleted = ?", "") // Base condition for not deleted devices

	if needJoinK8sNode {
		query = query.Joins("LEFT JOIN k8s_node kn ON LOWER(d.device_id) = LOWER(kn.nodename)")
	}
	return query
}

// applyFilterGroups applies a list of filter groups to the base query.
// It iterates through each group and applies its filter blocks using the correct
// logical operator (AND/OR) between groups.
func (s *DeviceQueryService) applyFilterGroups(query *gorm.DB, groups []FilterGroup, needJoinK8sNode bool) *gorm.DB {
	if len(groups) == 0 {
		return query
	}

	// Use a single transaction block for applying filters if needed, though GORM handles this generally.
	// We'll build the WHERE clause progressively.

	finalQuery := query // Start with the base query including joins

	for i, group := range groups {
		if len(group.Blocks) == 0 {
			continue
		}

		// Create a scope for the current group's conditions
		groupScope := func(db *gorm.DB) *gorm.DB {
			groupQuery := db // Start fresh for this group's AND conditions
			for _, block := range group.Blocks {
				// Apply each block's filter logic using the appropriate apply function
				// Note: applyFilterBlock handles the different FilterTypes internally
				groupQuery = s.applyFilterBlock(groupQuery, block)
			}
			return groupQuery
		}

		// Apply the group scope with the correct logical operator (AND/OR)
		if i == 0 {
			finalQuery = finalQuery.Scopes(groupScope) // First group is always ANDed initially
		} else if group.Operator == LogicalOperatorOr {
			// For OR, we need to group the previous conditions if they weren't already ORed
			// GORM's Or() method handles this correctly when chained.
			finalQuery = finalQuery.Or(s.db.Scopes(groupScope)) // Apply OR condition
		} else { // LogicalOperatorAnd or default
			finalQuery = finalQuery.Scopes(groupScope) // Apply AND condition
		}
	}

	return finalQuery
}


// executeCountQuery executes a COUNT query on the provided GORM query object.
// It uses a new session to avoid modifying the original query object which might be used for fetching data later.
func (s *DeviceQueryService) executeCountQuery(query *gorm.DB) (int64, error) {
	var total int64
	// Important: Create a new session for count to avoid modifying the original query object
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("failed to count devices: %w", err)
	}
	return total, nil
}

// executeListQuery executes the main data fetching query with pagination.
// It applies offset and limit based on the requested page and size,
// ensuring page and size are within valid ranges.
func (s *DeviceQueryService) executeListQuery(query *gorm.DB, page, size int) ([]portal.Device, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > MaxSize { // Assuming MaxSize is defined elsewhere or use a default
		size = DefaultSize // Assuming DefaultSize is defined
	}
	offset := (page - 1) * size

	var devices []portal.Device
	if err := query.Offset(offset).Limit(size).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}
	return devices, nil
}

// mapDevicesToResponse converts a slice of portal.Device models to a slice of DeviceResponse DTOs.
func mapDevicesToResponse(devices []portal.Device) []DeviceResponse {
	responses := make([]DeviceResponse, len(devices))
	for i, device := range devices {
		responses[i] = DeviceResponse{
			ID:           device.ID,
			DeviceID:     device.DeviceID, // Use direct field access
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
			CreatedAt:    time.Time(device.CreatedAt), // Ensure proper type conversion if needed
			UpdatedAt:    time.Time(device.UpdatedAt),
		}
	}
	return responses
}


// QueryDevices 查询设备 (Refactored)
func (s *DeviceQueryService) QueryDevices(ctx context.Context, req *DeviceQueryRequest) (*DeviceListResponse, error) {
	// 1. Determine if JOIN with k8s_node is needed
	needJoinK8sNode := determineRequiredJoins(req.Groups)

	// 2. Build the base query with necessary JOINs
	baseQuery := s.buildDeviceQuery(ctx, needJoinK8sNode)

	// 3. Apply filter groups to the query
	filteredQuery := s.applyFilterGroups(baseQuery, req.Groups, needJoinK8sNode)

	// 4. Execute count query (using a session to avoid modifying filteredQuery)
	total, err := s.executeCountQuery(filteredQuery.Session(&gorm.Session{})) // Use Session for count
	if err != nil {
		return nil, err
	}

	// 5. Execute list query with pagination
	devices, err := s.executeListQuery(filteredQuery, req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	// 6. Map results to response DTO
	responses := mapDevicesToResponse(devices)

	// 7. Return the final response
	return &DeviceListResponse{
		List:  responses,
		Total: total,
		Page:  req.Page, // Use request page/size or adjusted ones from executeListQuery
		Size:  req.Size,
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
