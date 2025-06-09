package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"navy-ng/models/portal"
	"strings"
	"time"

	"github.com/jinzhu/now"
	"gorm.io/gorm"
)

// snakeToCamel 将下划线命名转换为驼峰命名 (小驼峰)
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		// Handle potential empty strings after split (e.g., "__")
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(string(parts[i][0])) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

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

	// 特殊设备判断条件
	SpecialDeviceCondition = "device.`group` != '' OR " +
		"lf.id IS NOT NULL OR " +
		"tf.id IS NOT NULL OR " +
		"((device.`group` = '' OR device.`group` IS NULL) AND " +
		"(device.cluster = '' OR device.cluster IS NULL) AND " +
		"da.name IS NOT NULL AND da.name != '')"
	DeviceFieldNetZone      = "net_zone"
	DeviceFieldGroup        = "`group`"
	DeviceFieldAppID        = "appid"
	DeviceFieldOsCreateTime = "os_create_time"
	DeviceFieldCPU          = "cpu"
	DeviceFieldMemory       = "memory"
	DeviceFieldModel        = "model"
	DeviceFieldKvmIP        = "kvm_ip"
	DeviceFieldOS           = "os"
	DeviceFieldCompany      = "company"
	DeviceFieldOSName       = "os_name"
	DeviceFieldOSIssue      = "os_issue"
	DeviceFieldOSKernel     = "os_kernel"
	DeviceFieldStatus       = "status"
	DeviceFieldRole         = "role"
	DeviceFieldCluster      = "cluster"
	DeviceFieldClusterID    = "cluster_id"
)

// camelToSnake 将驼峰命名法转换为下划线命名法
// (Ensure this function exists and is correct)
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
	// 尝试将 key 转换为 camelCase (如果它是 snake_case)
	camelKey := block.Key
	if strings.Contains(camelKey, "_") {
		camelKey = snakeToCamel(camelKey)
	}

	// 使用新的 findDeviceFieldDefinition 函数查找字段定义
	fieldDef, ok := findDeviceFieldDefinition(camelKey)
	if !ok {
		// 如果找不到定义，记录警告并返回
		fmt.Printf("Warning: Field definition for key '%s' (camelCase '%s') not found. Skipping this filter.\n", block.Key, camelKey)
		return query
	}
	column := fieldDef.DbColumn // 获取数据库列名

	// 特殊处理 is_special 字段，使用与动态计算相同的条件
	// 使用转换后的 camelKey 进行比较
	if camelKey == "isSpecial" {
		// 使用常量定义的特殊设备判断条件

		switch block.ConditionType {
		case ConditionTypeEqual:
			// 将值转换为布尔值
			var boolValue bool
			if valStr, ok := block.Value.(string); ok { // Type assertion
				if strings.ToLower(valStr) == "true" || valStr == "1" {
					boolValue = true
				}
			} // Handle potential non-string value if necessary, default is false

			if boolValue {
				// 查询特殊设备
				return query.Where(SpecialDeviceCondition)
			} else {
				// 查询非特殊设备
				return query.Where("NOT (" + SpecialDeviceCondition + ")")
			}
		case ConditionTypeIn:
			// 特殊处理 isSpecial 字段
			if camelKey == "isSpecial" {
				// 处理 []string 类型
				if values, ok := block.Value.([]string); ok {
					if len(values) == 0 {
						return query.Where("1 = 0")
					}

					// 构建条件
					var conditions []string
					for _, v := range values {
						if strings.ToLower(v) == "true" || v == "1" {
							conditions = append(conditions, "("+SpecialDeviceCondition+")")
						} else if strings.ToLower(v) == "false" || v == "0" {
							conditions = append(conditions, "NOT ("+SpecialDeviceCondition+")")
						}
					}

					if len(conditions) > 0 {
						return query.Where(strings.Join(conditions, " OR "))
					}
					return query
				}

				// 处理 []interface{} 类型（JSON反序列化后的常见类型）
				if interfaceValues, ok := block.Value.([]interface{}); ok {
					if len(interfaceValues) == 0 {
						return query.Where("1 = 0")
					}

					// 构建条件
					var conditions []string
					for _, v := range interfaceValues {
						if str, ok := v.(string); ok {
							if strings.ToLower(str) == "true" || str == "1" {
								conditions = append(conditions, "("+SpecialDeviceCondition+")")
							} else if strings.ToLower(str) == "false" || str == "0" {
								conditions = append(conditions, "NOT ("+SpecialDeviceCondition+")")
							}
						} else if boolVal, ok := v.(bool); ok {
							if boolVal {
								conditions = append(conditions, "("+SpecialDeviceCondition+")")
							} else {
								conditions = append(conditions, "NOT ("+SpecialDeviceCondition+")")
							}
						} else {
							// 尝试将其他类型转换为字符串，然后判断
							strVal := fmt.Sprintf("%v", v)
							if strings.ToLower(strVal) == "true" || strVal == "1" {
								conditions = append(conditions, "("+SpecialDeviceCondition+")")
							} else if strings.ToLower(strVal) == "false" || strVal == "0" {
								conditions = append(conditions, "NOT ("+SpecialDeviceCondition+")")
							}
						}
					}

					if len(conditions) > 0 {
						return query.Where(strings.Join(conditions, " OR "))
					}
					return query
				}

				return query
			}

			// 处理其他字段
			// 处理 []string 类型
			if values, ok := block.Value.([]string); ok {
				if len(values) == 0 {
					return query.Where("1 = 0")
				}
				// 对ciCode字段使用大小写不敏感的查询
				if camelKey == "ciCode" {
					// 构建 UPPER(column) IN (UPPER(?), UPPER(?), ...) 条件
					placeholders := make([]string, len(values))
					args := make([]interface{}, len(values))
					for i, v := range values {
						placeholders[i] = "UPPER(?)"
						args[i] = v
					}
					return query.Where("UPPER("+column+") IN ("+strings.Join(placeholders, ", ")+")", args...)
				}
				return query.Where(column+" IN (?)", values)
			}

			// 处理 []interface{} 类型（JSON反序列化后的常见类型）
			if interfaceValues, ok := block.Value.([]interface{}); ok {
				if len(interfaceValues) == 0 {
					return query.Where("1 = 0")
				}

				// 将 []interface{} 转换为 []string
				values := make([]string, len(interfaceValues))
				for i, v := range interfaceValues {
					if str, ok := v.(string); ok {
						values[i] = str
					} else {
						// 尝试将非字符串值转换为字符串
						values[i] = fmt.Sprintf("%v", v)
					}
				}

				// 对ciCode字段使用大小写不敏感的查询
				if camelKey == "ciCode" {
					// 构建 UPPER(column) IN (UPPER(?), UPPER(?), ...) 条件
					placeholders := make([]string, len(values))
					args := make([]interface{}, len(values))
					for i, v := range values {
						placeholders[i] = "UPPER(?)"
						args[i] = v
					}
					return query.Where("UPPER("+column+") IN ("+strings.Join(placeholders, ", ")+")", args...)
				}
				return query.Where(column+" IN (?)", values)
			}

			// 处理其他类型
			fmt.Printf("Warning: Invalid value type for 'in' condition, expected []string or []interface{}, got %T. Skipping filter.\n", block.Value)
			return query
		case ConditionTypeNotEqual:
			// 将值转换为布尔值
			var boolValue bool
			if valStr, ok := block.Value.(string); ok { // Type assertion
				if strings.ToLower(valStr) == "true" || valStr == "1" {
					boolValue = true
				}
			} // Handle potential non-string value if necessary, default is false

			if boolValue {
				// 查询非特殊设备
				return query.Where("NOT (" + SpecialDeviceCondition + ")")
			} else {
				// 查询特殊设备
				return query.Where(SpecialDeviceCondition)
			}
		default:
			// 对于其他条件类型，使用默认处理
			return query
		}
	}

	// 直接构建条件，不使用Scopes
	switch block.ConditionType {
	case ConditionTypeEqual:
		if valStr, ok := block.Value.(string); ok {
			// 对ciCode字段使用大小写不敏感的查询
			if camelKey == "ciCode" {
				return query.Where("UPPER("+column+") = UPPER(?)", valStr)
			}
			return query.Where(column+" = ?", valStr)
		}
		return query // Or handle error/invalid type
	case ConditionTypeNotEqual:
		if valStr, ok := block.Value.(string); ok {
			// 对ciCode字段使用大小写不敏感的查询
			if camelKey == "ciCode" {
				return query.Where(fmt.Sprintf("(UPPER(%s) != UPPER(?) OR %s IS NULL)", column, column), valStr)
			}
			return query.Where(fmt.Sprintf("(%s != ? OR %s IS NULL)", column, column), valStr)
		}
		return query // Or handle error/invalid type
	case ConditionTypeContains:
		if valStr, ok := block.Value.(string); ok {
			escapedValue := strings.ReplaceAll(valStr, "%", "\\%")
			escapedValue = strings.ReplaceAll(escapedValue, "_", "\\_")
			// 对ciCode字段使用大小写不敏感的查询
			if camelKey == "ciCode" {
				return query.Where("UPPER("+column+") LIKE UPPER(?)", "%"+escapedValue+"%")
			}
			return query.Where(column+" LIKE ?", "%"+escapedValue+"%")
		}
		return query // Or handle error/invalid type
	case ConditionTypeNotContains:
		if valStr, ok := block.Value.(string); ok {
			escapedValue := strings.ReplaceAll(valStr, "%", "\\%")
			escapedValue = strings.ReplaceAll(escapedValue, "_", "\\_")
			// 对ciCode字段使用大小写不敏感的查询
			if camelKey == "ciCode" {
				return query.Where(fmt.Sprintf("(UPPER(%s) NOT LIKE UPPER(?) OR %s IS NULL)", column, column), "%"+escapedValue+"%")
			}
			return query.Where(fmt.Sprintf("(%s NOT LIKE ? OR %s IS NULL)", column, column), "%"+escapedValue+"%")
		}
		return query // Or handle error/invalid type
	case ConditionTypeExists:
		return query.Where(fmt.Sprintf("%s IS NOT NULL AND %s != ''", column, column))
	case ConditionTypeNotExists:
		return query.Where(fmt.Sprintf("%s IS NULL OR %s = ''", column, column))
	case ConditionTypeIn:
		// 特殊处理布尔类型字段
		if camelKey == "isLocalization" || camelKey == "isSpecial" {
			// 处理 []string 类型
			if values, ok := block.Value.([]string); ok {
				if len(values) == 0 {
					// 如果传入空数组，则不匹配任何记录
					return query.Where("1 = 0")
				}

				// 将布尔值字符串转换为 "1"/"0"
				convertedValues := make([]string, len(values))
				for i, v := range values {
					if strings.ToLower(v) == "true" || v == "1" {
						convertedValues[i] = "1"
					} else {
						convertedValues[i] = "0"
					}
				}

				return query.Where(column+" IN (?)", convertedValues)
			}

			// 处理 []interface{} 类型（JSON反序列化后的常见类型）
			if interfaceValues, ok := block.Value.([]interface{}); ok {
				if len(interfaceValues) == 0 {
					// 如果传入空数组，则不匹配任何记录
					return query.Where("1 = 0")
				}

				// 将 []interface{} 转换为 []string，并处理布尔值
				convertedValues := make([]string, len(interfaceValues))
				for i, v := range interfaceValues {
					if str, ok := v.(string); ok {
						if strings.ToLower(str) == "true" || str == "1" {
							convertedValues[i] = "1"
						} else {
							convertedValues[i] = "0"
						}
					} else if boolVal, ok := v.(bool); ok {
						if boolVal {
							convertedValues[i] = "1"
						} else {
							convertedValues[i] = "0"
						}
					} else {
						// 尝试将其他类型转换为字符串，然后判断
						strVal := fmt.Sprintf("%v", v)
						if strings.ToLower(strVal) == "true" || strVal == "1" {
							convertedValues[i] = "1"
						} else {
							convertedValues[i] = "0"
						}
					}
				}

				return query.Where(column+" IN (?)", convertedValues)
			}
		} else {
			// 处理非布尔类型字段
			// 处理 []string 类型
			if values, ok := block.Value.([]string); ok {
				if len(values) == 0 {
					// 如果传入空数组，则不匹配任何记录
					return query.Where("1 = 0")
				}
				// GORM handles slice arguments for IN clauses directly
				return query.Where(column+" IN (?)", values)
			}

			// 处理 []interface{} 类型（JSON反序列化后的常见类型）
			if interfaceValues, ok := block.Value.([]interface{}); ok {
				if len(interfaceValues) == 0 {
					// 如果传入空数组，则不匹配任何记录
					return query.Where("1 = 0")
				}

				// 将 []interface{} 转换为 []string
				values := make([]string, len(interfaceValues))
				for i, v := range interfaceValues {
					if str, ok := v.(string); ok {
						values[i] = str
					} else {
						// 尝试将非字符串值转换为字符串
						values[i] = fmt.Sprintf("%v", v)
					}
				}

				return query.Where(column+" IN (?)", values)
			}
		}

		// 处理其他类型
		fmt.Printf("Warning: Invalid value type for 'in' condition, expected []string or []interface{}, got %T. Skipping filter.\n", block.Value)
		return query
	case ConditionTypeNotIn:
		// 处理 []string 类型
		if values, ok := block.Value.([]string); ok {
			if len(values) == 0 {
				// 如果传入空数组，则匹配所有非 NULL 记录
				return query.Where(fmt.Sprintf("%s IS NOT NULL", column))
			}
			// GORM handles slice arguments for NOT IN clauses directly
			return query.Where(fmt.Sprintf("(%s NOT IN (?) OR %s IS NULL)", column, column), values)
		}

		// 处理 []interface{} 类型（JSON反序列化后的常见类型）
		if interfaceValues, ok := block.Value.([]interface{}); ok {
			if len(interfaceValues) == 0 {
				// 如果传入空数组，则匹配所有非 NULL 记录
				return query.Where(fmt.Sprintf("%s IS NOT NULL", column))
			}

			// 将 []interface{} 转换为 []string
			values := make([]string, len(interfaceValues))
			for i, v := range interfaceValues {
				if str, ok := v.(string); ok {
					values[i] = str
				} else {
					// 尝试将非字符串值转换为字符串
					values[i] = fmt.Sprintf("%v", v)
				}
			}

			return query.Where(fmt.Sprintf("(%s NOT IN (?) OR %s IS NULL)", column, column), values)
		}

		// 处理其他类型
		fmt.Printf("Warning: Invalid value type for 'notIn' condition, expected []string or []interface{}, got %T. Skipping filter.\n", block.Value)
		return query
	case ConditionTypeGreaterThan:
		if valStr, ok := block.Value.(string); ok {
			return query.Where(fmt.Sprintf("%s > ?", column), valStr)
		}
		return query // Or handle error/invalid type
	case ConditionTypeLessThan:
		if valStr, ok := block.Value.(string); ok {
			return query.Where(fmt.Sprintf("%s < ?", column), valStr)
		}
		return query // Or handle error/invalid type
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
	Value         interface{}     `json:"value"`         // 值 (可以是 string 或 []string)
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

// DeviceFieldDefinition defines the properties of a device field for filtering and display.
type DeviceFieldDefinition struct {
	CamelKey string // Key used internally and potentially by frontend JS (e.g., "ciCode")
	Label    string // User-friendly label (e.g., "设备编码")
	DbColumn string // Full database column name with table alias (e.g., "device.ci_code")
}

// deviceFieldDefinitions is the single source of truth for device fields.
var deviceFieldDefinitions = []DeviceFieldDefinition{
	{CamelKey: "ciCode", Label: "设备编码", DbColumn: "device.ci_code"},
	{CamelKey: "ip", Label: "IP地址", DbColumn: "device.ip"},
	{CamelKey: "archType", Label: "CPU架构", DbColumn: "device.arch_type"},
	{CamelKey: "idc", Label: "IDC", DbColumn: "device.idc"},
	{CamelKey: "room", Label: "机房", DbColumn: "device.room"},
	{CamelKey: "cabinet", Label: "机柜", DbColumn: "device.cabinet"},
	{CamelKey: "cabinetNo", Label: "机柜编号", DbColumn: "device.cabinet_no"},
	{CamelKey: "infraType", Label: "网络类型", DbColumn: "device.infra_type"},
	{CamelKey: "isLocalization", Label: "是否国产化", DbColumn: "device.is_localization"},
	{CamelKey: "netZone", Label: "网络区域", DbColumn: "device.net_zone"},
	{CamelKey: "group", Label: "机器类别", DbColumn: "device.`group`"}, // Note backticks
	{CamelKey: "appId", Label: "APPID", DbColumn: "device.appid"},
	{CamelKey: "osCreateTime", Label: "操作系统创建时间", DbColumn: "device.os_create_time"},
	{CamelKey: "cpu", Label: "CPU数量", DbColumn: "device.cpu"},
	{CamelKey: "memory", Label: "内存大小", DbColumn: "device.memory"},
	{CamelKey: "model", Label: "型号", DbColumn: "device.model"},
	{CamelKey: "kvmIp", Label: "KVM IP", DbColumn: "device.kvm_ip"},
	{CamelKey: "os", Label: "操作系统", DbColumn: "device.os"},
	{CamelKey: "company", Label: "厂商", DbColumn: "device.company"},
	{CamelKey: "osName", Label: "操作系统名称", DbColumn: "device.os_name"},
	{CamelKey: "osIssue", Label: "操作系统版本", DbColumn: "device.os_issue"},
	{CamelKey: "osKernel", Label: "操作系统内核", DbColumn: "device.os_kernel"},
	{CamelKey: "status", Label: "状态", DbColumn: "device.status"},
	{CamelKey: "role", Label: "角色", DbColumn: "device.role"},
	{CamelKey: "cluster", Label: "集群", DbColumn: "device.cluster"},
	{CamelKey: "clusterId", Label: "集群ID", DbColumn: "device.cluster_id"},
	{CamelKey: "isSpecial", Label: "特殊设备", DbColumn: "device.is_special"}, // Mapped for consistency
}

// Helper function to find a field definition by camelCase key
func findDeviceFieldDefinition(camelKey string) (*DeviceFieldDefinition, bool) {
	for _, def := range deviceFieldDefinitions {
		if def.CamelKey == camelKey {
			return &def, true
		}
	}
	return nil, false
}

// GetDbColumnForField 根据字段的驼峰键获取数据库列名
// 返回数据库列名和是否找到的布尔值
func (s *DeviceQueryService) GetDbColumnForField(camelKey string) (string, bool) {
	def, found := findDeviceFieldDefinition(camelKey)
	if !found {
		return "", false
	}
	return def.DbColumn, true
}

// DeviceQueryService 设备查询服务
type DeviceQueryService struct {
	db    *gorm.DB
	cache DeviceCacheInterface
}

// NewDeviceQueryService 创建设备查询服务
func NewDeviceQueryService(db *gorm.DB, cache DeviceCacheInterface) *DeviceQueryService {
	return &DeviceQueryService{db: db, cache: cache}
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
		// Use field.DbColumn directly (e.g., "device.appid", "device.`group`")
		// Ensure deviceFieldDefinitions has correct quoting (e.g., "device.`group`")
		dbColumnIdentifier := field.DbColumn

		var values []string

		// Build the query using the full DbColumn identifier
		query := s.db.WithContext(ctx).Table("device").
			Select(fmt.Sprintf("DISTINCT %s", dbColumnIdentifier)).
			Where(fmt.Sprintf("%s IS NOT NULL", dbColumnIdentifier)).
			Where(fmt.Sprintf("%s != ''", dbColumnIdentifier))

		// Pluck using the full DbColumn identifier
		if err := query.Order(dbColumnIdentifier).Pluck(dbColumnIdentifier, &values).Error; err != nil {
			fmt.Printf("Warning: failed to get device field values for %s (column: %s): %v\n", field.Label, dbColumnIdentifier, err)
			continue
		}

		if len(values) > 0 {
			fieldOptions := make([]FilterOption, 0, len(values))
			for _, value := range values {
				fieldOptions = append(fieldOptions, FilterOption{
					ID:    fmt.Sprintf("%s-%s", field.Value, value), // Keep using field.Value (snake_case) for ID consistency if needed
					Label: value,
					Value: value,
				})
			}

			deviceFieldValuesList = append(deviceFieldValuesList, DeviceFieldValues{
				Field:  field.Value, // Keep using field.Value (snake_case) for Field consistency if needed
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
	// 尝试从缓存获取
	if s.cache != nil {
		if cachedValues, err := s.cache.GetDeviceFieldValues(field); err == nil {
			return cachedValues, nil
		}
	}

	// 缓存未命中，从数据库查询
	// 尝试将 field 转换为 camelCase (如果它是 snake_case)
	camelField := field
	if strings.Contains(camelField, "_") {
		camelField = snakeToCamel(camelField)
	}

	// 使用新的 findDeviceFieldDefinition 函数查找字段定义
	fieldDef, ok := findDeviceFieldDefinition(camelField)
	if !ok {
		// 如果找不到定义，返回错误
		return nil, fmt.Errorf("invalid or unknown field name: %s", field)
	}
	column := fieldDef.DbColumn // 获取数据库列名

	// 从列名中提取字段名 (例如, 从 "device.ci_code" 提取 "ci_code")
	parts := strings.Split(column, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid column format: %s", column)
	}
	dbField := parts[1]

	// 查询不同的字段值
	var values []string
	var query *gorm.DB

	// 特殊处理布尔类型字段 (使用转换后的 camelField)
	if camelField == "isSpecial" {
		// 布尔类型字段只有 true 和 false 两个值
		// 缓存结果 (即使是硬编码的值也缓存，保持一致性)
		if s.cache != nil {
			s.cache.SetDeviceFieldValues(field, []string{"true", "false"}, false) // isLabelField is false
		}
		return []string{"true", "false"}, nil
	}

	// 判断是否是 SQL 关键字 (使用从 column 提取的 dbField)
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

	// 缓存结果
	if s.cache != nil {
		// 判断字段是否来自标签/污点表
		isLabelField := strings.HasPrefix(field, "label_") || strings.HasPrefix(field, "taint_")
		s.cache.SetDeviceFieldValues(field, values, isLabelField)
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

		SELECT 'label' as type, knl.` + "`key`" + ` as ` + "`key`" + `, knl.value as value, '' as effect
		FROM node_id n
		JOIN k8s_node_label knl ON n.id = knl.node_id
		JOIN label_feature lf ON knl.` + "`key`" + ` = lf.` + "`key`" + `
		WHERE knl.created_at BETWEEN ? AND ?

		UNION ALL

		SELECT 'taint' as type, knt.` + "`key`" + ` as ` + "`key`" + `, knt.value as value, knt.effect as effect
		FROM node_id n
		JOIN k8s_node_taint knt ON n.id = knt.node_id
		JOIN taint_feature tf ON knt.` + "`key`" + ` = tf.` + "`key`" + `
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
			"CASE WHEN " + SpecialDeviceCondition + " " +
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
	// 生成查询参数的哈希值
	queryHash := GenerateQueryHash(req)

	// 尝试从缓存获取
	if s.cache != nil {
		if cachedResponse, err := s.cache.GetDeviceList(queryHash); err == nil {
			return cachedResponse, nil
		}
	}

	// 缓存未命中，从数据库查询
	// 1. Build the base query with necessary JOINs
	baseQuery := s.buildDeviceQuery(ctx)

	// 2. Apply filter groups to the query
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

	// 7. 构建响应
	response := &DeviceListResponse{
		List:  responses,
		Total: total,
		Page:  req.Page, // Use request page/size or adjusted ones from executeListQuery
		Size:  req.Size,
	}

	// 8. 缓存查询结果
	if s.cache != nil {
		s.cache.SetDeviceList(queryHash, response)

		// 同时缓存单个设备
		for _, deviceResp := range responses {
			s.cache.SetDevice(deviceResp.ID, &deviceResp)
		}
	}

	return response, nil
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
		if valStr, ok := block.Value.(string); ok {
			return query.Where(fmt.Sprintf(SQLConditionEqual, tableAlias), valStr)
		}
		return query // Or handle error/invalid type
	case ConditionTypeNotEqual:
		if valStr, ok := block.Value.(string); ok {
			return query.Where(fmt.Sprintf(SQLConditionNotEqual, tableAlias, tableAlias), valStr)
		}
		return query // Or handle error/invalid type
	case ConditionTypeContains:
		if valStr, ok := block.Value.(string); ok {
			escapedValue := escapeValue(valStr)
			return query.Where(fmt.Sprintf(SQLConditionContains, tableAlias), "%"+escapedValue+"%")
		}
		return query // Or handle error/invalid type
	case ConditionTypeNotContains:
		if valStr, ok := block.Value.(string); ok {
			escapedValue := escapeValue(valStr)
			return query.Where(fmt.Sprintf(SQLConditionNotContains, tableAlias, tableAlias), "%"+escapedValue+"%")
		}
		return query // Or handle error/invalid type
	case ConditionTypeExists:
		return query.Where(fmt.Sprintf(SQLConditionExists, tableAlias))
	case ConditionTypeNotExists:
		return query.Where(fmt.Sprintf(SQLConditionNotExists, tableAlias))
	case ConditionTypeIn:
		// Expect Value to be []string for 'in'
		if values, ok := block.Value.([]string); ok {
			if len(values) == 0 {
				return query.Where("1 = 0") // No match if empty array
			}
			return query.Where(fmt.Sprintf(SQLConditionIn, tableAlias), values)
		}
		// Handle case where Value is string (split by comma) - legacy support?
		if valStr, ok := block.Value.(string); ok {
			values := splitAndTrimValues(valStr)
			if len(values) == 0 {
				return query.Where("1 = 0")
			}
			return query.Where(fmt.Sprintf(SQLConditionIn, tableAlias), values)
		}
		fmt.Printf("Warning: Invalid value type for 'in' condition in applyValueCondition, expected []string or string, got %T. Skipping filter.\n", block.Value)
		return query
	case ConditionTypeNotIn:
		// Expect Value to be []string for 'notIn'
		if values, ok := block.Value.([]string); ok {
			if len(values) == 0 {
				return query.Where(fmt.Sprintf("%s IS NOT NULL", tableAlias)) // Match all non-null if empty array
			}
			return query.Where(fmt.Sprintf(SQLConditionNotIn, tableAlias, tableAlias), values)
		}
		// Handle case where Value is string (split by comma) - legacy support?
		if valStr, ok := block.Value.(string); ok {
			values := splitAndTrimValues(valStr)
			if len(values) == 0 {
				return query.Where(fmt.Sprintf("%s IS NOT NULL", tableAlias))
			}
			return query.Where(fmt.Sprintf(SQLConditionNotIn, tableAlias, tableAlias), values)
		}
		fmt.Printf("Warning: Invalid value type for 'notIn' condition in applyValueCondition, expected []string or string, got %T. Skipping filter.\n", block.Value)
		return query
	default:
		return query
	}
}

// applyNodeLabelFilter 应用节点标签筛选
func (s *DeviceQueryService) applyNodeLabelFilter(query *gorm.DB, block FilterBlock) *gorm.DB {
	// 在 WHERE 子句中添加 key 条件
	// 使用结构化的方式添加条件，避免字符串拼接
	// 因为基础查询中已经包含了 LEFT JOIN k8s_node_label
	query = query.Where(TableAliasNodeLabel+".key = ?", block.Key)

	// 应用值条件
	return s.applyValueCondition(query, block, TableAliasNodeLabel)
}

// applyTaintFilter 应用污点筛选
func (s *DeviceQueryService) applyTaintFilter(query *gorm.DB, block FilterBlock) *gorm.DB {
	// 在 WHERE 子句中添加 key 条件
	// 使用结构化的方式添加条件，避免字符串拼接
	// 因为基础查询中已经包含了 LEFT JOIN k8s_node_taint
	query = query.Where(TableAliasNodeTaint+".key = ?", block.Key)

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

		// 清除相关缓存，因为模板可能会影响设备查询结果
		if s.cache != nil {
			s.cache.InvalidateDeviceLists()
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

	// 清除相关缓存，因为模板可能会影响设备查询结果
	if s.cache != nil {
		s.cache.InvalidateDeviceLists()
	}

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

// GetQueryTemplatesWithPagination 获取查询模板列表（支持分页）
func (s *DeviceQueryService) GetQueryTemplatesWithPagination(ctx context.Context, page, size int) (*QueryTemplateListResponse, error) {
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
	if err := s.db.WithContext(ctx).Model(&portal.QueryTemplate{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count templates: %w", err)
	}

	// 从数据库获取分页的模板
	var dbTemplates []portal.QueryTemplate
	if err := s.db.WithContext(ctx).
		Offset(offset).
		Limit(size).
		Order("id desc"). // 默认按ID降序排列，可以根据需要修改
		Find(&dbTemplates).Error; err != nil {
		return nil, fmt.Errorf("failed to get templates: %w", err)
	}

	// 转换为服务层模板格式
	templates := make([]QueryTemplateResponse, len(dbTemplates))
	for i, dbTemplate := range dbTemplates {
		var groups []FilterGroup
		if err := json.Unmarshal([]byte(dbTemplate.Groups), &groups); err != nil {
			return nil, fmt.Errorf("failed to unmarshal groups for template %d: %w", dbTemplate.ID, err)
		}

		// 转换为 QueryTemplateResponse 类型
		templates[i] = QueryTemplateResponse{
			ID:          dbTemplate.ID,
			Name:        dbTemplate.Name,
			Description: dbTemplate.Description,
			Groups:      convertGroupsToResponse(groups),
			CreatedAt:   time.Time(dbTemplate.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   time.Time(dbTemplate.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		}
	}

	// 构建响应
	response := &QueryTemplateListResponse{
		List:  templates,
		Total: total,
		Page:  page,
		Size:  size,
	}

	return response, nil
}

// convertGroupsToResponse 将内部的FilterGroup转换为响应对象中的FilterGroupRequest
func convertGroupsToResponse(groups []FilterGroup) []FilterGroupRequest {
	result := make([]FilterGroupRequest, len(groups))
	for i, group := range groups {
		blocks := make([]FilterBlockRequest, len(group.Blocks))
		for j, block := range group.Blocks {
			var valueStr string
			switch v := block.Value.(type) {
			case string:
				valueStr = v
			case []string:
				// Convert slice to comma-separated string for the response DTO
				valueStr = strings.Join(v, ",")
			default:
				// Handle other types or nil if necessary, default to empty string
				valueStr = ""
				if block.Value != nil {
					fmt.Printf("Warning: Unexpected type for FilterBlock.Value in convertGroupsToResponse: %T\n", block.Value)
				}
			}

			blocks[j] = FilterBlockRequest{
				ID:            block.ID,
				Type:          block.Type,
				ConditionType: block.ConditionType,
				Key:           block.Key,
				Value:         valueStr, // Assign the converted string value
				Operator:      block.Operator,
			}
		}

		result[i] = FilterGroupRequest{
			ID:       group.ID,
			Blocks:   blocks,
			Operator: group.Operator,
		}
	}
	return result
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

	// 检查是否有设备匹配策略引用此查询模板
	var policies []portal.ResourcePoolDeviceMatchingPolicy
	if err := s.db.WithContext(ctx).Where("query_template_id = ?", id).Find(&policies).Error; err != nil {
		return fmt.Errorf("failed to check policy references: %w", err)
	}

	if len(policies) > 0 {
		// 收集引用此模板的策略名称
		policyNames := make([]string, len(policies))
		for i, policy := range policies {
			policyNames[i] = policy.Name
		}
		return fmt.Errorf("无法删除查询模板，该模板正在被以下设备匹配策略引用：%s。请先在弹性扩容管理的设备管理策略中解绑该查询模板", strings.Join(policyNames, "、"))
	}

	// 删除模板
	if err := s.db.WithContext(ctx).Delete(&portal.QueryTemplate{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	// 清除相关缓存，因为模板可能会影响设备查询结果
	if s.cache != nil {
		s.cache.InvalidateDeviceLists()
	}

	return nil
}

// GetDeviceFields 获取设备字段列表 (动态生成)
func (s *DeviceQueryService) GetDeviceFields(ctx context.Context) ([]FilterOption, error) {
	options := make([]FilterOption, 0, len(deviceFieldDefinitions))
	for _, def := range deviceFieldDefinitions {
		// 使用 snake_case 作为 ID 和 Value，保持与旧代码和前端可能的期望一致
		snakeCaseKey := camelToSnake(def.CamelKey)
		options = append(options, FilterOption{
			ID:       snakeCaseKey,
			Label:    def.Label,
			Value:    snakeCaseKey, // Value 也使用 snake_case
			DbColumn: def.DbColumn,
		})
	}
	return options, nil
}
