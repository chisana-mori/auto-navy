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

// SQL 常量定义
const (
	// 通用SQL片段
	SQLCreatedAtBetween = "created_at BETWEEN ? AND ?"
	SQLDistinct         = "DISTINCT"
	SQLIsNotNull        = "IS NOT NULL"
	SQLIsNull           = "IS NULL"
	SQLNotEmpty         = "!= ''"
	SQLValueField       = "value"
	SQLKeyField         = "`key`"
	// 表别名
	TableAliasDevice    = "device"
	TableAliasK8sNode   = "kn"
	TableAliasNodeLabel = "knl"
	TableAliasNodeTaint = "knt"

	// 表连接SQL
	SQLJoinK8sNode   = "INNER JOIN k8s_node %s ON LOWER(%s.device_id) = LOWER(%s.nodename)"
	SQLJoinNodeLabel = "INNER JOIN k8s_node_label %s ON %s.id = %s.node_id AND %s.key = ? AND %s.created_at BETWEEN ? AND ?"
	SQLJoinNodeTaint = "INNER JOIN k8s_node_taint %s ON %s.id = %s.node_id AND %s.key = ? AND %s.created_at BETWEEN ? AND ?"

	// 查询条件SQL模板
	SQLConditionEqual       = "%s.value = ?"
	SQLConditionNotEqual    = "%s.value != ? OR %s.value IS NULL"
	SQLConditionContains    = "%s.value LIKE ?"
	SQLConditionNotContains = "%s.value NOT LIKE ? OR %s.value IS NULL"
	SQLConditionExists      = "%s.value IS NOT NULL"
	SQLConditionNotExists   = "%s.value IS NULL"
	SQLConditionIn          = "%s.value IN (?)"
	SQLConditionNotIn       = "%s.value NOT IN (?) OR %s.value IS NULL"

	// 设备字段映射
	DeviceFieldCICode         = "ci_code"
	DeviceFieldIP             = "ip"
	DeviceFieldArchType       = "arch_type"
	DeviceFieldIDC            = "idc"
	DeviceFieldRoom           = "room"
	DeviceFieldCabinet        = "cabinet"
	DeviceFieldCabinetNO      = "cabinet_no"
	DeviceFieldInfraType      = "infra_type"
	DeviceFieldIsLocalization = "is_localization"
	DeviceFieldNetZone        = "net_zone"
	DeviceFieldGroup          = "`group`"
	DeviceFieldAppID          = "appid"
	DeviceFieldOsCreateTime   = "os_create_time"
	DeviceFieldCPU            = "cpu"
	DeviceFieldMemory         = "memory"
	DeviceFieldModel          = "model"
	DeviceFieldKvmIP          = "kvm_ip"
	DeviceFieldOS             = "os"
	DeviceFieldCompany        = "company"
	DeviceFieldOSName         = "os_name"
	DeviceFieldOSIssue        = "os_issue"
	DeviceFieldOSKernel       = "os_kernel"
	DeviceFieldStatus         = "status"
	DeviceFieldRole           = "role"
	DeviceFieldCluster        = "cluster"
	DeviceFieldClusterID      = "cluster_id"
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
		column = fmt.Sprintf("device.%s", snakeCase)
		// Consider logging a warning here that a direct map entry is preferred
		fmt.Printf("Warning: Column key '%s' not found in deviceFieldColumnMap, using generated '%s'. Consider adding it to the map.\n", block.Key, column)
	}

	// 直接构建条件，不使用Scopes
	switch block.ConditionType {
	case ConditionTypeEqual:
		return query.Where(column+" = ?", block.Value)
	case ConditionTypeNotEqual:
		return query.Where(fmt.Sprintf("(%s != ? OR %s IS NULL)", column, column), block.Value)
	case ConditionTypeContains:
		escapedValue := strings.ReplaceAll(block.Value, "%", "\\%")
		escapedValue = strings.ReplaceAll(escapedValue, "_", "\\_")
		return query.Where(column+" LIKE ?", "%"+escapedValue+"%")
	case ConditionTypeNotContains:
		escapedValue := strings.ReplaceAll(block.Value, "%", "\\%")
		escapedValue = strings.ReplaceAll(escapedValue, "_", "\\_")
		return query.Where(fmt.Sprintf("(%s NOT LIKE ? OR %s IS NULL)", column, column), "%"+escapedValue+"%")
	case ConditionTypeExists:
		return query.Where(fmt.Sprintf("%s IS NOT NULL AND %s != ''", column, column))
	case ConditionTypeNotExists:
		return query.Where(fmt.Sprintf("%s IS NULL OR %s = ''", column, column))
	case ConditionTypeIn:
		values := strings.Split(block.Value, ",")
		trimmedValues := make([]string, 0, len(values))
		for _, v := range values {
			trimmed := strings.TrimSpace(v)
			if trimmed != "" {
				trimmedValues = append(trimmedValues, trimmed)
			}
		}
		if len(trimmedValues) == 0 {
			return query.Where("1 = 0")
		}
		return query.Where(column+" IN (?)", trimmedValues)
	case ConditionTypeNotIn:
		values := strings.Split(block.Value, ",")
		trimmedValues := make([]string, 0, len(values))
		for _, v := range values {
			trimmed := strings.TrimSpace(v)
			if trimmed != "" {
				trimmedValues = append(trimmedValues, trimmed)
			}
		}
		if len(trimmedValues) == 0 {
			return query.Where(fmt.Sprintf("%s IS NOT NULL", column))
		}
		return query.Where(fmt.Sprintf("(%s NOT IN (?) OR %s IS NULL)", column, column), trimmedValues)
	case ConditionTypeGreaterThan:
		return query.Where(fmt.Sprintf("%s > ?", column), block.Value)
	case ConditionTypeLessThan:
		return query.Where(fmt.Sprintf("%s < ?", column), block.Value)
	case ConditionTypeIsEmpty:
		return query.Where(fmt.Sprintf("(%s IS NULL OR %s = '')", column, column))
	case ConditionTypeIsNotEmpty:
		return query.Where(fmt.Sprintf("%s IS NOT NULL AND %s != ''", column, column))
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
	ConditionTypeGreaterThan ConditionType = "greaterThan" // 大于
	ConditionTypeLessThan    ConditionType = "lessThan"    // 小于
	ConditionTypeIsEmpty     ConditionType = "isEmpty"     // 为空
	ConditionTypeIsNotEmpty  ConditionType = "isNotEmpty"  // 不为空
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
	"ciCode":          "device.ci_code",
	"ci_code":         "device.ci_code",
	"ip":              "device.ip",
	"archType":        "device.arch_type",
	"arch_type":       "device.arch_type",
	"idc":             "device.idc",
	"room":            "device.room",
	"cabinet":         "device.cabinet",
	"cabinetNo":       "device.cabinet_no",
	"cabinet_no":      "device.cabinet_no",
	"infraType":       "device.infra_type",
	"infra_type":      "device.infra_type",
	"isLocalization":  "device.is_localization",
	"is_localization": "device.is_localization",
	"netZone":         "device.net_zone",
	"net_zone":        "device.net_zone",
	"group":           "device.`group`",
	"appId":           "device.appid",
	"appid":           "device.appid",
	"osCreateTime":    "device.os_create_time",
	"os_create_time":  "device.os_create_time",
	"cpu":             "device.cpu",
	"memory":          "device.memory",
	"model":           "device.model",
	"kvmIp":           "device.kvm_ip",
	"kvm_ip":          "device.kvm_ip",
	"os":              "device.os",
	"company":         "device.company",
	"osName":          "device.os_name",
	"os_name":         "device.os_name",
	"osIssue":         "device.os_issue",
	"os_issue":        "device.os_issue",
	"osKernel":        "device.os_kernel",
	"os_kernel":       "device.os_kernel",
	"status":          "device.status",
	"role":            "device.role",
	"cluster":         "device.cluster",
	"clusterId":       "device.cluster_id",
	"cluster_id":      "device.cluster_id",
	// Add other direct mappings if needed
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
func (s *DeviceQueryService) GetFilterOptions(ctx context.Context) (map[string]any, error) {
	options := make(map[string]any)

	// 使用 time.Now() 获取当前时间，然后使用 now 包处理
	currentTime := time.Now()
	todayStart := now.New(currentTime).BeginningOfDay()
	todayEnd := now.New(currentTime).EndOfDay()

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
		var query *gorm.DB

		// 特殊处理 group 字段
		if field.Value == "group" {
			query = s.db.WithContext(ctx).Table("device").
				Select("DISTINCT `group`").
				Where("`group` IS NOT NULL").
				Where("`group` != ''")
		} else {
			query = s.db.WithContext(ctx).Table("device").
				Select(fmt.Sprintf("DISTINCT %s", field.Value)).
				Where(fmt.Sprintf("%s IS NOT NULL", field.Value)).
				Where(fmt.Sprintf("%s != ''", field.Value))
		}

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
	currentTime := time.Now()
	todayStart := now.New(currentTime).BeginningOfDay()
	todayEnd := now.New(currentTime).EndOfDay()

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
	currentTime := time.Now()
	todayStart := now.New(currentTime).BeginningOfDay()
	todayEnd := now.New(currentTime).EndOfDay()

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

// GetDeviceFieldValues 获取设备字段值
func (s *DeviceQueryService) GetDeviceFieldValues(ctx context.Context, field string) ([]string, error) {
	// 检查字段是否存在于映射中
	column, ok := deviceFieldColumnMap[field]
	if !ok {
		// 如果不存在，尝试将驼峰命名转换为下划线命名
		snakeCase := camelToSnake(field)
		if !isValidColumnName(snakeCase) {
			return nil, fmt.Errorf("invalid field name: %s", field)
		}
		column = fmt.Sprintf("device.%s", snakeCase)
	}

	// 从列名中提取字段名
	parts := strings.Split(column, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid column format: %s", column)
	}
	dbField := parts[1]

	// 查询不同的字段值
	var values []string
	var query *gorm.DB

	// 判断是否是 SQL 关键字
	isReservedKeyword := dbField == "`group`" || dbField == "group"

	// 构建查询
	query = s.db.WithContext(ctx).Table("device")

	// 处理选择字段
	if isReservedKeyword {
		query = query.Select("DISTINCT `group`")
	} else {
		query = query.Select(fmt.Sprintf("DISTINCT %s", dbField))
	}

	// 不再添加非空过滤条件，允许返回空值

	// 添加排序和限制
	if isReservedKeyword {
		query = query.Order("`group`")
	} else {
		query = query.Order(dbField)
	}

	// 执行查询
	var pluckField string
	if isReservedKeyword {
		pluckField = "`group`"
	} else {
		pluckField = dbField
	}
	if err := query.Limit(100).Pluck(pluckField, &values).Error; err != nil {
		return nil, fmt.Errorf("failed to get device field values for %s: %w", field, err)
	}

	return values, nil
}

// DeviceFeatureDetails 设备特性详情
type DeviceFeatureDetails struct {
	Labels []LabelDetail `json:"labels"` // 标签详情
	Taints []TaintDetail `json:"taints"` // 污点详情
}

// LabelDetail 标签详情
type LabelDetail struct {
	Key   string `json:"key"`   // 标签键
	Value string `json:"value"` // 标签值
}

// TaintDetail 污点详情
type TaintDetail struct {
	Key    string `json:"key"`    // 污点键
	Value  string `json:"value"`  // 污点值
	Effect string `json:"effect"` // 效果
}

// GetDeviceFeatureDetails 获取设备的特性详情（标签和污点）- 使用UNION ALL优化
func (s *DeviceQueryService) GetDeviceFeatureDetails(ctx context.Context, ciCode string) (*DeviceFeatureDetails, error) {
	// 获取今天的开始和结束时间
	currentTime := time.Now()
	todayStart := now.New(currentTime).BeginningOfDay()
	todayEnd := now.New(currentTime).EndOfDay()

	// 定义结果结构体，用于UNION ALL查询
	type FeatureResult struct {
		Type   string `gorm:"column:type"`   // 标记是label还是taint
		Key    string `gorm:"column:key"`    // 键
		Value  string `gorm:"column:value"`  // 值
		Effect string `gorm:"column:effect"` // 效果(仅用于taint)
	}

	var results []FeatureResult

	// 构建UNION ALL查询
	query := `
		WITH node_id AS (
			SELECT id FROM k8s_node
			WHERE LOWER(nodename) = LOWER(?)
			AND (status != 'Offline' OR status IS NULL)
			LIMIT 1
		)

		SELECT 'label' as type, knl.key as key, knl.value as value, '' as effect
		FROM node_id n
		JOIN k8s_node_label knl ON n.id = knl.node_id
		JOIN label_feature lf ON knl.key = lf.key
		WHERE knl.created_at BETWEEN ? AND ?

		UNION ALL

		SELECT 'taint' as type, knt.key as key, knt.value as value, knt.effect as effect
		FROM node_id n
		JOIN k8s_node_taint knt ON n.id = knt.node_id
		JOIN taint_feature tf ON knt.key = tf.key
		WHERE knt.created_at BETWEEN ? AND ?
	`

	// 执行查询
	if err := s.db.WithContext(ctx).Raw(
		query,
		ciCode,
		todayStart,
		todayEnd,
		todayStart,
		todayEnd,
	).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get device features: %w", err)
	}

	// 将结果分类
	var labels []LabelDetail
	var taints []TaintDetail

	for _, r := range results {
		if r.Type == "label" {
			labels = append(labels, LabelDetail{
				Key:   r.Key,
				Value: r.Value,
			})
		} else {
			taints = append(taints, TaintDetail{
				Key:    r.Key,
				Value:  r.Value,
				Effect: r.Effect,
			})
		}
	}

	return &DeviceFeatureDetails{Labels: labels, Taints: taints}, nil
}

// buildDeviceQuery constructs the base GORM query for devices.
// It selects necessary device fields, filters out deleted devices,
// and adds a LEFT JOIN to the k8s_node table to determine if the device is special.
func (s *DeviceQueryService) buildDeviceQuery(ctx context.Context) *gorm.DB {
	// 获取今天的开始和结束时间
	currentTime := time.Now()
	todayStart := now.New(currentTime).BeginningOfDay()
	todayEnd := now.New(currentTime).EndOfDay()

	// 构建查询，添加 isSpecial 字段用于标识特殊设备
	query := s.db.WithContext(ctx).Table("device").
		Select("device.*, " +
			// 添加 isSpecial 字段，当满足以下条件之一时为 true：
			// 1. 机器用途不为空
			// 2. 可以关联到 label_feature
			// 3. 可以关联到 taint_feature
			// 4. 当 device.group 和 cluster 为空，但可以关联到 device_app 时
			"CASE WHEN device.`group` != '' OR lf.id IS NOT NULL OR tf.id IS NOT NULL OR " +
			"     ((device.`group` = '' OR device.`group` IS NULL) AND (device.cluster = '' OR device.cluster IS NULL) AND da.name IS NOT NULL AND da.name != '') " +
			"THEN TRUE ELSE FALSE END AS is_special, " +
			// 添加特性计数字段，用于前端显示
			"(CASE WHEN device.`group` != '' THEN 1 ELSE 0 END + " +
			" CASE WHEN lf.id IS NOT NULL THEN 1 ELSE 0 END + " +
			" CASE WHEN tf.id IS NOT NULL THEN 1 ELSE 0 END + " +
			" CASE WHEN (device.`group` = '' OR device.`group` IS NULL) AND (device.cluster = '' OR device.cluster IS NULL) AND da.name IS NOT NULL AND da.name != '' THEN 1 ELSE 0 END" +
			") AS feature_count, " +
			// 只在 device.cluster 为空时获取 device_app.name
			"CASE WHEN device.cluster = '' OR device.cluster IS NULL THEN da.name ELSE NULL END AS app_name")

	// 默认关联 k8s_node 表，并添加 status != Offline 筛选条件
	query = query.Joins("LEFT JOIN k8s_node kn ON LOWER(device.ci_code) = LOWER(kn.nodename) AND (kn.status != 'Offline' OR kn.status IS NULL)")

	// 关联 k8s_node_label 表和 label_feature 表，用于判断是否为特殊设备
	query = query.Joins("LEFT JOIN k8s_node_label knl ON kn.id = knl.node_id AND knl.created_at BETWEEN ? AND ?", todayStart, todayEnd)
	query = query.Joins("LEFT JOIN label_feature lf ON knl.key = lf.key")

	// 关联 k8s_node_taint 表和 taint_feature 表，用于判断是否为特殊设备
	query = query.Joins("LEFT JOIN k8s_node_taint knt ON kn.id = knt.node_id AND knt.created_at BETWEEN ? AND ?", todayStart, todayEnd)
	query = query.Joins("LEFT JOIN taint_feature tf ON knt.key = tf.key")

	// 关联 device_app 表，用于获取设备的来源信息
	query = query.Joins("LEFT JOIN device_app da ON device.appid = da.app_id")

	// 使用 GROUP BY 确保每个设备只返回一行
	query = query.Group("device.id")

	return query
}

// applyFilterGroups applies a list of filter groups to the base query.
// It iterates through each group and applies its filter blocks using the correct
// logical operator (AND/OR) between groups.
func (s *DeviceQueryService) applyFilterGroups(query *gorm.DB, groups []FilterGroup) *gorm.DB {
	if len(groups) == 0 {
		return query
	}

	// Use a single transaction block for applying filters if needed, though GORM handles this generally.
	// We'll build the WHERE clause progressively.

	finalQuery := query // Start with the base query including joins

	// 处理所有组
	for i, group := range groups {
		if len(group.Blocks) == 0 {
			continue
		}

		// 处理组内的块
		groupQuery := s.db.Session(&gorm.Session{})

		// 处理第一个块
		firstBlock := group.Blocks[0]

		// 直接应用第一个块的条件
		groupQuery = s.applyFilterBlock(groupQuery, firstBlock)

		// 处理其余块，根据前一个块的操作符决定使用 AND 还是 OR
		for j := 1; j < len(group.Blocks); j++ {
			block := group.Blocks[j]
			prevBlock := group.Blocks[j-1]

			// 根据前一个块的操作符决定使用 AND 还是 OR
			if prevBlock.Operator == LogicalOperatorOr {
				// 使用 OR 操作符
				blockQuery := s.db.Session(&gorm.Session{})
				blockQuery = s.applyFilterBlock(blockQuery, block)
				groupQuery = groupQuery.Or(blockQuery)
			} else { // LogicalOperatorAnd 或默认
				// 使用 AND 操作符
				groupQuery = s.applyFilterBlock(groupQuery, block)
			}
		}

		// 将组的条件应用到最终查询
		if i == 0 {
			// 第一个组直接应用
			finalQuery = finalQuery.Where(groupQuery)
		} else {
			// 获取前一个组
			prevGroup := groups[i-1]
			if prevGroup.Operator == LogicalOperatorOr {
				// 使用 OR 操作符
				finalQuery = finalQuery.Or(groupQuery)
			} else { // LogicalOperatorAnd 或默认
				// 使用 AND 操作符
				finalQuery = finalQuery.Where(groupQuery)
			}
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
			ID:             device.ID,
			CICode:         device.CICode,
			IP:             device.IP,
			ArchType:       device.ArchType,
			IDC:            device.IDC,
			Room:           device.Room,
			Cabinet:        device.Cabinet,
			CabinetNO:      device.CabinetNO,
			InfraType:      device.InfraType,
			IsLocalization: device.IsLocalization,
			NetZone:        device.NetZone,
			Group:          device.Group,
			AppID:          device.AppID,
			AppName:        device.AppName,
			OsCreateTime:   device.OsCreateTime,
			CPU:            device.CPU,
			Memory:         device.Memory,
			Model:          device.Model,
			KvmIP:          device.KvmIP,
			OS:             device.OS,
			Company:        device.Company,
			OSName:         device.OSName,
			OSIssue:        device.OSIssue,
			OSKernel:       device.OSKernel,
			Status:         device.Status,
			Role:           device.Role,
			Cluster:        device.Cluster,
			AcceptanceTime: device.AcceptanceTime,
			DiskCount:      device.DiskCount,
			DiskDetail:     device.DiskDetail,
			NetworkSpeed:   device.NetworkSpeed,
			IsSpecial:      device.IsSpecial,
			FeatureCount:   device.FeatureCount,
			CreatedAt:      time.Time(device.CreatedAt),
			UpdatedAt:      time.Time(device.UpdatedAt),
		}
	}
	return responses
}

// QueryDevices 查询设备 (Refactored)
func (s *DeviceQueryService) QueryDevices(ctx context.Context, req *DeviceQueryRequest) (*DeviceListResponse, error) {

	// 2. Build the base query with necessary JOINs
	baseQuery := s.buildDeviceQuery(ctx)

	// 3. Apply filter groups to the query
	filteredQuery := s.applyFilterGroups(baseQuery, req.Groups)

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

// escapeValue 转义特殊字符
func escapeValue(value string) string {
	escaped := strings.ReplaceAll(value, "%", "\\%")
	return strings.ReplaceAll(escaped, "_", "\\_")
}

// splitAndTrimValues 将逗号分隔的字符串分割并去除空格
func splitAndTrimValues(value string) []string {
	values := strings.Split(value, ",")
	for i, v := range values {
		values[i] = strings.TrimSpace(v)
	}
	return values
}

// applyValueCondition 应用值条件
func (s *DeviceQueryService) applyValueCondition(query *gorm.DB, block FilterBlock, tableAlias string) *gorm.DB {
	switch block.ConditionType {
	case ConditionTypeEqual:
		return query.Where(fmt.Sprintf(SQLConditionEqual, tableAlias), block.Value)
	case ConditionTypeNotEqual:
		return query.Where(fmt.Sprintf(SQLConditionNotEqual, tableAlias, tableAlias), block.Value)
	case ConditionTypeContains:
		escapedValue := escapeValue(block.Value)
		return query.Where(fmt.Sprintf(SQLConditionContains, tableAlias), "%"+escapedValue+"%")
	case ConditionTypeNotContains:
		escapedValue := escapeValue(block.Value)
		return query.Where(fmt.Sprintf(SQLConditionNotContains, tableAlias, tableAlias), "%"+escapedValue+"%")
	case ConditionTypeExists:
		return query.Where(fmt.Sprintf(SQLConditionExists, tableAlias))
	case ConditionTypeNotExists:
		return query.Where(fmt.Sprintf(SQLConditionNotExists, tableAlias))
	case ConditionTypeIn:
		values := splitAndTrimValues(block.Value)
		return query.Where(fmt.Sprintf(SQLConditionIn, tableAlias), values)
	case ConditionTypeNotIn:
		values := splitAndTrimValues(block.Value)
		return query.Where(fmt.Sprintf(SQLConditionNotIn, tableAlias, tableAlias), values)
	default:
		return query
	}
}

// applyNodeLabelFilter 应用节点标签筛选
func (s *DeviceQueryService) applyNodeLabelFilter(query *gorm.DB, block FilterBlock) *gorm.DB {
	// 获取当天的开始和结束时间
	currentTime := time.Now()
	todayStart := now.New(currentTime).BeginningOfDay()
	todayEnd := now.New(currentTime).EndOfDay()

	// 使用 INNER JOIN 而不是 LEFT JOIN，确保只返回匹配所有条件的记录
	// 并且只查询当天的标签数据
	query = query.Joins(
		fmt.Sprintf(SQLJoinNodeLabel, TableAliasNodeLabel, TableAliasK8sNode, TableAliasNodeLabel, TableAliasNodeLabel, TableAliasNodeLabel),
		block.Key, todayStart, todayEnd,
	)

	// 应用值条件
	return s.applyValueCondition(query, block, TableAliasNodeLabel)
}

// applyTaintFilter 应用污点筛选
func (s *DeviceQueryService) applyTaintFilter(query *gorm.DB, block FilterBlock) *gorm.DB {
	// 获取当天的开始和结束时间
	currentTime := time.Now()
	todayStart := now.New(currentTime).BeginningOfDay()
	todayEnd := now.New(currentTime).EndOfDay()

	// 使用 INNER JOIN 而不是 LEFT JOIN，确保只返回匹配所有条件的记录
	// 并且只查询当天的污点数据
	query = query.Joins(
		fmt.Sprintf(SQLJoinNodeTaint, TableAliasNodeTaint, TableAliasK8sNode, TableAliasNodeTaint, TableAliasNodeTaint, TableAliasNodeTaint),
		block.Key, todayStart, todayEnd,
	)

	// 应用值条件
	return s.applyValueCondition(query, block, TableAliasNodeTaint)
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
			ID:       "ci_code",
			Label:    "设备编码",
			Value:    "ci_code",
			DbColumn: "device.ci_code",
		},
		{
			ID:       "ip",
			Label:    "IP地址",
			Value:    "ip",
			DbColumn: "device.ip",
		},
		{
			ID:       "arch_type",
			Label:    "CPU架构",
			Value:    "arch_type",
			DbColumn: "device.arch_type",
		},
		{
			ID:       "idc",
			Label:    "IDC",
			Value:    "idc",
			DbColumn: "device.idc",
		},
		{
			ID:       "room",
			Label:    "机房",
			Value:    "room",
			DbColumn: "device.room",
		},
		{
			ID:       "cabinet",
			Label:    "机柜",
			Value:    "cabinet",
			DbColumn: "device.cabinet",
		},
		{
			ID:       "cabinet_no",
			Label:    "机柜编号",
			Value:    "cabinet_no",
			DbColumn: "device.cabinet_no",
		},
		{
			ID:       "infra_type",
			Label:    "网络类型",
			Value:    "infra_type",
			DbColumn: "device.infra_type",
		},
		{
			ID:       "is_localization",
			Label:    "是否国产化",
			Value:    "is_localization",
			DbColumn: "device.is_localization",
		},
		{
			ID:       "net_zone",
			Label:    "网络区域",
			Value:    "net_zone",
			DbColumn: "device.net_zone",
		},
		{
			ID:       "group",
			Label:    "机器类别",
			Value:    "group",
			DbColumn: "device.`group`",
		},
		{
			ID:       "appid",
			Label:    "APPID",
			Value:    "appid",
			DbColumn: "device.appid",
		},
		{
			ID:       "os_create_time",
			Label:    "操作系统创建时间",
			Value:    "os_create_time",
			DbColumn: "device.os_create_time",
		},
		{
			ID:       "cpu",
			Label:    "CPU数量",
			Value:    "cpu",
			DbColumn: "device.cpu",
		},
		{
			ID:       "memory",
			Label:    "内存大小",
			Value:    "memory",
			DbColumn: "device.memory",
		},
		{
			ID:       "model",
			Label:    "型号",
			Value:    "model",
			DbColumn: "device.model",
		},
		{
			ID:       "kvm_ip",
			Label:    "KVM IP",
			Value:    "kvm_ip",
			DbColumn: "device.kvm_ip",
		},
		{
			ID:       "os",
			Label:    "操作系统",
			Value:    "os",
			DbColumn: "device.os",
		},
		{
			ID:       "company",
			Label:    "厂商",
			Value:    "company",
			DbColumn: "device.company",
		},
		{
			ID:       "os_name",
			Label:    "操作系统名称",
			Value:    "os_name",
			DbColumn: "device.os_name",
		},
		{
			ID:       "os_issue",
			Label:    "操作系统版本",
			Value:    "os_issue",
			DbColumn: "device.os_issue",
		},
		{
			ID:       "os_kernel",
			Label:    "操作系统内核",
			Value:    "os_kernel",
			DbColumn: "device.os_kernel",
		},
		{
			ID:       "status",
			Label:    "状态",
			Value:    "status",
			DbColumn: "device.status",
		},
		{
			ID:       "role",
			Label:    "角色",
			Value:    "role",
			DbColumn: "device.role",
		},
		{
			ID:       "cluster",
			Label:    "集群",
			Value:    "cluster",
			DbColumn: "device.cluster",
		},
		{
			ID:       "cluster_id",
			Label:    "集群ID",
			Value:    "cluster_id",
			DbColumn: "device.cluster_id",
		},
	}, nil
}
