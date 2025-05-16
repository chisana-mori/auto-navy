package resource_report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strconv"
)

// ResourceStyleType 表示资源使用率样式类型
type ResourceStyleType string

const (
	// StyleEmergency 表示紧急级别(红色)
	StyleEmergency ResourceStyleType = "emergency"
	// StyleCritical 表示危险级别(橙红色)
	StyleCritical ResourceStyleType = "critical"
	// StyleWarning 表示警告级别(黄色)
	StyleWarning ResourceStyleType = "warning"
	// StyleNormal 表示正常级别(绿色)
	StyleNormal ResourceStyleType = "normal"
	// StyleUnderutilized 表示低利用率(蓝色)
	StyleUnderutilized ResourceStyleType = "underutilized"
)

// ResourcePoolDisplayStatus 定义是否应该显示资源池
type ResourcePoolDisplayStatus struct {
	ShouldDisplay bool              // 是否应该显示该资源池
	CpuStyle      ResourceStyleType // CPU分配率样式
	MemStyle      ResourceStyleType // 内存分配率样式
}

// 集群类型
const (
	ClusterLarge        = "large" // 大型集群(>150物理机)
	ClusterSmall        = "small" // 小型/中型集群(≤150物理机)
	ClusterSizeBoundary = 150     // 大小集群的边界值
)

// 环境类型
const (
	EnvProduction = "prd"  // 生产环境
	EnvTest       = "test" // 测试环境
)

// 使用率阈值常量定义
const (
	// 低利用率阈值(所有环境和集群类型相同)
	LowUtilizationThreshold = 55.0

	// 紧急级别阈值(所有环境和集群类型相同)
	EmergencyThreshold = 95.0
)

// StyleThresholds 定义不同环境和集群规模下的样式阈值
type StyleThresholds struct {
	Critical float64 // 危险阈值
	Warning  float64 // 警告阈值
	Display  float64 // 是否显示阈值
}

// 样式阈值规则表
var styleRuleMatrix = map[string]map[string]*StyleThresholds{
	EnvProduction: {
		ClusterLarge: {
			Critical: 90.0, // 90-95% 危险
			Warning:  80.0, // 80-90% 警告
			Display:  80.0, // >=80% 需要显示
		},
		ClusterSmall: {
			Critical: 85.0, // 85-95% 危险
			Warning:  75.0, // 75-85% 警告
			Display:  75.0, // >=75% 需要显示
		},
	},
	EnvTest: {
		ClusterLarge: {
			Critical: 90.0, // 90-95% 危险
			Warning:  85.0, // 85-90% 警告 (测试环境比生产高5%)
			Display:  85.0, // >=85% 需要显示
		},
		ClusterSmall: {
			Critical: 90.0, // 90-95% 危险
			Warning:  80.0, // 80-90% 警告 (测试环境比生产高5%)
			Display:  80.0, // >=80% 需要显示
		},
	},
}

// getClusterType 根据物理机数量判断集群类型
func getClusterType(bmCount int) string {
	if bmCount > ClusterSizeBoundary {
		return ClusterLarge
	}
	return ClusterSmall
}

// getThresholds 获取指定环境和集群类型的阈值规则
func getThresholds(environment, clusterType string) *StyleThresholds {
	// 获取环境规则
	envRules, ok := styleRuleMatrix[environment]
	if !ok {
		// 如果找不到指定环境，默认使用生产环境规则
		envRules = styleRuleMatrix[EnvProduction]
	}

	// 获取集群规模规则
	thresholds, ok := envRules[clusterType]
	if !ok {
		// 如果找不到指定集群类型，默认使用小型集群规则
		thresholds = envRules[ClusterSmall]
	}

	return thresholds
}

// getCPUStyleForEnvironment returns the CPU style based on environment and cluster size
func getCPUStyleForEnvironment(environment string, cpuUsage float64, clusterType string) ResourceStyleType {
	// Get the appropriate thresholds
	thresholds := getThresholds(environment, clusterType)

	// Always check emergency threshold first (same for all environments and cluster sizes)
	if cpuUsage >= EmergencyThreshold { // 95.0%
		return StyleEmergency
	}

	// Check critical threshold (varies by environment and cluster size)
	if cpuUsage >= thresholds.Critical {
		return StyleCritical
	}

	// Check warning threshold (varies by environment and cluster size)
	if cpuUsage >= thresholds.Warning {
		return StyleWarning
	}

	// Check for underutilization (only in production)
	// Low utilization alerts only apply in production environments
	if environment == EnvProduction && cpuUsage < LowUtilizationThreshold {
		return StyleUnderutilized
	}

	// Default case - normal usage
	return StyleNormal
}

// GetCPUStyle gets the style for CPU usage based on environment, cluster size, and usage percentage
func GetCPUStyle(bmCount int, cpuUsage float64, environment string) ResourceStyleType {
	// Default to production if not specified
	if environment == "" {
		environment = EnvProduction
	}

	// Determine cluster type based on BM count
	clusterType := getClusterType(bmCount)

	// Get style based on environment and cluster type
	return getCPUStyleForEnvironment(environment, cpuUsage, clusterType)
}

// getMemoryStyleForEnvironment returns the memory style based on environment and cluster size
func getMemoryStyleForEnvironment(environment string, memUsage float64, clusterType string) ResourceStyleType {
	// Get the appropriate thresholds
	thresholds := getThresholds(environment, clusterType)

	// Always check emergency threshold first (same for all environments and cluster sizes)
	if memUsage >= EmergencyThreshold { // 95.0%
		return StyleEmergency
	}

	// Check critical threshold (varies by environment and cluster size)
	if memUsage >= thresholds.Critical {
		return StyleCritical
	}

	// Check warning threshold (varies by environment and cluster size)
	if memUsage >= thresholds.Warning {
		return StyleWarning
	}

	// Check for underutilization (only in production)
	// Low utilization alerts only apply in production environments
	if environment == EnvProduction && memUsage < LowUtilizationThreshold {
		return StyleUnderutilized
	}

	// Default case - normal usage
	return StyleNormal
}

// GetMemoryStyle gets the style for memory usage based on environment, cluster size, and usage percentage
func GetMemoryStyle(bmCount int, memUsage float64, environment string) ResourceStyleType {
	// Default to production if not specified
	if environment == "" {
		environment = EnvProduction
	}

	// Determine cluster type based on BM count
	clusterType := getClusterType(bmCount)

	// Get style based on environment and cluster type
	return getMemoryStyleForEnvironment(environment, memUsage, clusterType)
}

// GetResourcePoolStatus 判断资源池是否应该显示及其样式
// bmCount: 物理机数量
// cpuUsage: CPU使用率百分比
// memUsage: 内存使用率百分比
// environment: 环境类型 ("prd" 或 "test")
func GetResourcePoolStatus(bmCount int, cpuUsage, memUsage float64, environment string) ResourcePoolDisplayStatus {
	clusterType := getClusterType(bmCount)
	thresholds := getThresholds(environment, clusterType)

	// 计算CPU和内存样式
	cpuStyle := GetCPUStyle(bmCount, cpuUsage, environment)
	memStyle := GetMemoryStyle(bmCount, memUsage, environment)

	// 判断是否应该显示
	shouldDisplay := false

	// 高使用率判断
	if cpuUsage >= thresholds.Display || memUsage >= thresholds.Display {
		shouldDisplay = true
	}

	// 低使用率判断(仅生产环境)
	if environment == EnvProduction && (cpuUsage < LowUtilizationThreshold || memUsage < LowUtilizationThreshold) {
		shouldDisplay = true
	}

	return ResourcePoolDisplayStatus{
		ShouldDisplay: shouldDisplay,
		CpuStyle:      cpuStyle,
		MemStyle:      memStyle,
	}
}

// IsResourcePoolAbnormal 判断资源池是否异常
// bmCount: 物理机数量
// cpuUsage: CPU使用率百分比
// memUsage: 内存使用率百分比
// environment: 环境类型 ("prd" 或 "test")
func IsResourcePoolAbnormal(bmCount int, cpuUsage, memUsage float64, environment string) bool {
	status := GetResourcePoolStatus(bmCount, cpuUsage, memUsage, environment)
	return status.ShouldDisplay
}

// IsClusterAbnormal 判断集群是否异常
// bmCount: 物理机数量
// cpuUsage: CPU使用率百分比
// memUsage: 内存使用率百分比
// environment: 环境类型 ("prd" 或 "test")
func IsClusterAbnormal(bmCount int, cpuUsage, memUsage float64, environment string) bool {
	return IsResourcePoolAbnormal(bmCount, cpuUsage, memUsage, environment)
}

// toFloat converts any numeric type to float64
func toFloat(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case int16:
		return float64(v)
	case int8:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint16:
		return float64(v)
	case uint8:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

// GetCPUStyleName returns the string representation of CPU style.
func GetCPUStyleName(bmCount int, cpuUsage float64, environment string) string {
	return string(GetCPUStyle(bmCount, cpuUsage, environment))
}

// GetMemStyleName returns the string representation of Memory style.
func GetMemStyleName(bmCount int, memUsage float64, environment string) string {
	return string(GetMemoryStyle(bmCount, memUsage, environment))
}

// GetBarClassName returns the CSS class name for the usage bar background color.
func GetBarClassName(style ResourceStyleType) string {
	switch style {
	case StyleEmergency:
		return "emergency-usage"
	case StyleCritical:
		return "danger-usage"
	case StyleWarning:
		return "high-usage"
	case StyleUnderutilized:
		return "underutilized-usage"
	default:
		return "normal-usage"
	}
}

// GetBarWidthClassName returns the CSS class name for the usage bar width.
func GetBarWidthClassName(usage float64) string {
	if usage > 100.0 {
		return "width-100"
	}
	thresholds := []float64{95, 90, 85, 80, 75, 70, 65, 60, 55, 50, 45, 40, 35, 30, 25, 20, 15, 10, 5}
	for _, t := range thresholds {
		if usage >= t {
			return fmt.Sprintf("width-%.0f", t)
		}
	}
	return "width-0"
}

// GetCPUTooltipMessage generates the tooltip message for CPU usage.
func GetCPUTooltipMessage(cpuUsage float64, bmCount int, environment string, cpuRequest float64, cpuCapacity float64) string {
	clusterType := getClusterType(bmCount)
	thresholds := getThresholds(environment, clusterType)
	currentValueStr := fmt.Sprintf("%.1f/%.1f 核 (%.1f%%)", cpuRequest, cpuCapacity, cpuUsage)
	var statusDesc string

	if environment == EnvTest {
		if cpuUsage >= EmergencyThreshold { // 95% for test, same as prod emergency
			statusDesc = "<b>紧急</b> - 资源池CPU分配率超过95%，需要立即关注并采取措施。"
		} else if cpuUsage >= thresholds.Critical { // Test Large: 90, Test Small: 90
			statusDesc = fmt.Sprintf("<b>危险</b> - 资源池CPU分配率在%.0f%%-95%%之间，需要尽快关注。", thresholds.Critical)
		} else if cpuUsage >= thresholds.Warning { // Test Large: 85, Test Small: 80
			statusDesc = fmt.Sprintf("<b>警告</b> - 资源池CPU分配率在%.0f%%-%.0f%%之间，建议关注资源趋势。", thresholds.Warning, thresholds.Critical)
		} else {
			statusDesc = "<b>正常</b> - 资源池CPU分配率在安全范围内。"
		}
	} else { // EnvProduction
		if cpuUsage >= EmergencyThreshold { // 95%
			statusDesc = "<b>紧急</b> - 资源池CPU分配率超过95%，需要立即关注并采取措施。"
		} else if cpuUsage >= thresholds.Critical { // Prod Large: 90, Prod Small: 85
			statusDesc = fmt.Sprintf("<b>危险</b> - 资源池CPU分配率在%.0f%%-95%%之间，需要尽快关注。", thresholds.Critical)
		} else if cpuUsage >= thresholds.Warning { // Prod Large: 80, Prod Small: 75
			statusDesc = fmt.Sprintf("<b>警告</b> - 资源池CPU分配率在%.0f%%-%.0f%%之间，建议关注资源趋势。", thresholds.Warning, thresholds.Critical)
		} else if cpuUsage < LowUtilizationThreshold { // 55%
			statusDesc = "<b>低分配率</b> - 资源池CPU分配率低于55%，资源利用率不足，建议考虑资源整合。"
		} else {
			statusDesc = "<b>正常</b> - 资源池CPU分配率在安全范围内。"
		}
	}
	return fmt.Sprintf("当前值：%s<br><br>状态：%s", currentValueStr, statusDesc)
}

// GetMemTooltipMessage generates the tooltip message for Memory usage.
func GetMemTooltipMessage(memUsage float64, bmCount int, environment string, memRequest float64, memCapacity float64) string {
	clusterType := getClusterType(bmCount)
	thresholds := getThresholds(environment, clusterType)
	currentValueStr := fmt.Sprintf("%.1f/%.1f GiB (%.1f%%)", memRequest, memCapacity, memUsage)
	var statusDesc string

	if environment == EnvTest {
		if memUsage >= EmergencyThreshold { // 95%
			statusDesc = "<b>紧急</b> - 资源池内存分配率超过95%，需要立即关注并采取措施。"
		} else if memUsage >= thresholds.Critical { // Test Large: 90, Test Small: 90
			statusDesc = fmt.Sprintf("<b>危险</b> - 资源池内存分配率在%.0f%%-95%%之间，需要尽快关注。", thresholds.Critical)
		} else if memUsage >= thresholds.Warning { // Test Large: 85, Test Small: 80
			statusDesc = fmt.Sprintf("<b>警告</b> - 资源池内存分配率在%.0f%%-%.0f%%之间，建议关注资源趋势。", thresholds.Warning, thresholds.Critical)
		} else {
			statusDesc = "<b>正常</b> - 资源池内存分配率在安全范围内。"
		}
	} else { // EnvProduction
		if memUsage >= EmergencyThreshold { // 95%
			statusDesc = "<b>紧急</b> - 资源池内存分配率超过95%，需要立即关注并采取措施。"
		} else if memUsage >= thresholds.Critical { // Prod Large: 90, Prod Small: 85
			statusDesc = fmt.Sprintf("<b>危险</b> - 资源池内存分配率在%.0f%%-95%%之间，需要尽快关注。", thresholds.Critical)
		} else if memUsage >= thresholds.Warning { // Prod Large: 80, Prod Small: 75
			statusDesc = fmt.Sprintf("<b>警告</b> - 资源池内存分配率在%.0f%%-%.0f%%之间，建议关注资源趋势。", thresholds.Warning, thresholds.Critical)
		} else if memUsage < LowUtilizationThreshold { // 55%
			statusDesc = "<b>低分配率</b> - 资源池内存分配率低于55%，资源利用率不足，建议考虑资源整合。"
		} else {
			statusDesc = "<b>正常</b> - 资源池内存分配率在安全范围内。"
		}
	}
	return fmt.Sprintf("当前值：%s<br/>状态：%s", currentValueStr, statusDesc)
}

// GetCPUTrendStyleClass returns the CSS class for CPU trend history item.
func GetCPUTrendStyleClass(trendUsage float64, bmCount int, environment string) string {
	style := GetCPUStyle(bmCount, trendUsage, environment)
	switch style {
	case StyleEmergency:
		return "bar-emergency" // Using the same classes as the main chart for stronger colors
	case StyleCritical:
		return "bar-critical" // Using the same classes as the main chart for stronger colors
	case StyleWarning:
		return "bar-warning" // Using the same classes as the main chart for stronger colors
	case StyleUnderutilized:
		return "bar-underutilized"
	default:
		return "bar-normal"
	}
}

// GetMemTrendStyleClass returns the CSS class for Memory trend history item.
func GetMemTrendStyleClass(trendUsage float64, bmCount int, environment string) string {
	style := GetMemoryStyle(bmCount, trendUsage, environment)
	switch style {
	case StyleEmergency:
		return "bar-emergency" // Using the same classes as the main chart for stronger colors
	case StyleCritical:
		return "bar-critical" // Using the same classes as the main chart for stronger colors
	case StyleWarning:
		return "bar-warning" // Using the same classes as the main chart for stronger colors
	case StyleUnderutilized:
		return "bar-underutilized"
	default:
		return "bar-normal"
	}
}

// GetPoolHeaderClassName returns the CSS class for the resource pool header.
func GetPoolHeaderClassName(resourceType string) string {
	switch resourceType {
	case "total":
		return "resource-header-total"
	case "total_intel", "intel_common", "intel_gpu", "intel_taint", "intel_non_gpu", "aplus_intel", "dplus_intel":
		return "resource-header-intel"
	case "total_arm", "arm_common", "arm_gpu", "arm_taint", "aplus_arm", "dplus_arm":
		return "resource-header-arm"
	case "total_hg", "hg_common", "hg_taint", "aplus_hg", "dplus_hg":
		return "resource-header-hg"
	case "total_taint": // This case might be redundant if specific arch_taint is used
		return "resource-header-taint"
	case "total_common": // This case might be redundant if specific arch_common is used
		return "resource-header-common"
	case "total_gpu": // This case might be redundant if specific arch_gpu is used
		return "resource-header-gpu"
	case "aplus_total":
		return "resource-header-aplus"
	case "dplus_total":
		return "resource-header-dplus"
	default:
		// Fallback for any other types, or if a more generic class is preferred
		return "resource-header-common" // Or an empty string if no default
	}
}

// CustomTemplateFuncs returns a map of custom template functions
func CustomTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"toFloat": func(val interface{}) float64 {
			return toFloat(val)
		},
		"mul": func(a, b interface{}) float64 {
			af := toFloat(a)
			bf := toFloat(b)
			return af * bf
		},
		"div": func(a, b interface{}) float64 {
			af := toFloat(a)
			bf := toFloat(b)
			if bf == 0 {
				return 0
			}
			return af / bf
		},
		"ge": func(a, b interface{}) bool {
			af := toFloat(a)
			bf := toFloat(b)
			return af >= bf
		},
		"gt": func(a, b interface{}) bool {
			af := toFloat(a)
			bf := toFloat(b)
			return af > bf
		},
		"lt": func(a, b interface{}) bool {
			af := toFloat(a)
			bf := toFloat(b)
			return af < bf
		},
		"eq": func(a, b interface{}) bool {
			af := toFloat(a)
			bf := toFloat(b)
			return af == bf
		},
		"add": func(a, b interface{}) int {
			af := int(toFloat(a))
			bf := int(toFloat(b))
			return af + bf
		},
		"sub": func(a, b interface{}) int {
			af := int(toFloat(a))
			bf := int(toFloat(b))
			return af - bf
		},
		"len": func(a interface{}) int {
			switch v := a.(type) {
			case []string:
				return len(v)
			case []float64:
				return len(v)
			case []int:
				return len(v)
			case string:
				return len(v)
			default:
				return 0
			}
		},
		"formatFloat": func(f float64, precision int) string {
			format := "%." + strconv.Itoa(precision) + "f"
			return fmt.Sprintf(format, f)
		},
		"formatBytes": func(bytes float64) string {
			// 输入已经是GB单位，直接格式化输出
			if bytes >= 1024 {
				return fmt.Sprintf("%.2f TB", bytes/1024)
			} else {
				return fmt.Sprintf("%.2f GB", bytes)
			}
		},
		// 获取CPU样式类的函数
		"getCpuStyleClass": func(bmCount int, cpuUsage float64, environment string) string {
			return string(GetCPUStyle(bmCount, cpuUsage, environment))
		},
		// 获取内存样式类的函数
		"getMemStyleClass": func(bmCount int, memUsage float64, environment string) string {
			return string(GetMemoryStyle(bmCount, memUsage, environment))
		},
		// 判断资源池是否应该显示的函数
		"shouldShowResourcePool": func(bmCount int, cpuUsage, memUsage float64, environment string) bool {
			status := GetResourcePoolStatus(bmCount, cpuUsage, memUsage, environment)
			return status.ShouldDisplay
		},
		// 判断资源池是否异常的函数
		"isResourcePoolAbnormal": func(bmCount int, cpuUsage, memUsage float64, environment string) bool {
			return IsResourcePoolAbnormal(bmCount, cpuUsage, memUsage, environment)
		},
		// 获取CPU颜色类的函数 - 使用统一样式库
		"getCpuColorClass": func(pool *ResourcePool, environment string) string {
			if pool == nil {
				return ""
			}
			return string(GetCPUStyle(pool.BMCount, pool.CPUUsagePercent, environment))
		},
		// 获取内存颜色类的函数
		"getMemColorClass": func(pool *ResourcePool, environment string) string {
			if pool == nil {
				return ""
			}
			return string(GetMemoryStyle(pool.BMCount, pool.MemoryUsagePercent, environment))
		},
		// 判断资源池是否需要显示的函数 - 使用统一样式库
		"shouldShowPool": func(cpuUsage, memoryUsage float64, bmCount int, environment string) bool {
			// 使用统一样式库判断资源池是否应显示
			return GetResourcePoolStatus(bmCount, cpuUsage, memoryUsage, environment).ShouldDisplay
		},
		// 获取资源池类型的颜色
		"getPoolTypeColor": func(poolType string) string {
			switch poolType {
			case "total":
				return "#00188F" // 深蓝色
			case "intel_common":
				return "#0078D7" // 蓝色
			case "intel_gpu":
				return "#2B579A" // 深蓝色
			case "amd_common":
				return "#D83B01" // 红色
			case "arm_common":
				return "#107C10" // 绿色
			case "hg_common":
				return "#5C2D91" // 紫色
			default:
				return "#000000" // 黑色
			}
		},
		// 获取CPU趋势样式类
		"getCPUTrendStyleClass": func(trendUsage float64, bmCount int, environment string) string {
			return GetCPUTrendStyleClass(trendUsage, bmCount, environment)
		},
		// 获取内存趋势样式类
		"getMemTrendStyleClass": func(trendUsage float64, bmCount int, environment string) string {
			return GetMemTrendStyleClass(trendUsage, bmCount, environment)
		},
		// 获取使用条样式类
		"getBarClassName": func(style ResourceStyleType) string {
			return GetBarClassName(style)
		},
		// 获取使用条宽度类
		"getBarWidthClassName": func(usage float64) string {
			return GetBarWidthClassName(usage)
		},
		// 获取CPU工具提示
		"getCPUTooltipMessage": func(cpuUsage float64, bmCount int, environment string, cpuRequest float64, cpuCapacity float64) string {
			return GetCPUTooltipMessage(cpuUsage, bmCount, environment, cpuRequest, cpuCapacity)
		},
		// 获取内存工具提示
		"getMemTooltipMessage": func(memUsage float64, bmCount int, environment string, memRequest float64, memCapacity float64) string {
			return GetMemTooltipMessage(memUsage, bmCount, environment, memRequest, memCapacity)
		},
		// 获取资源池头部样式类
		"getPoolHeaderClassName": func(resourceType string) string {
			return GetPoolHeaderClassName(resourceType)
		},
		// 获取阈值函数
		"getThresholdWarning": func(environment, clusterType string) float64 {
			thresholds := getThresholds(environment, clusterType)
			return thresholds.Warning
		},
		"getThresholdCritical": func(environment, clusterType string) float64 {
			thresholds := getThresholds(environment, clusterType)
			return thresholds.Critical
		},
		"getThresholdEmergency": func() float64 {
			return EmergencyThreshold
		},
		"getThresholdLow": func() float64 {
			return LowUtilizationThreshold
		},
		"getThresholdDisplay": func(environment, clusterType string) float64 {
			thresholds := getThresholds(environment, clusterType)
			return thresholds.Display
		},
		// Add direct access to GetMemoryStyle and GetCPUStyle
		"getMemoryStyle": func(bmCount int, memUsage float64, environment string) ResourceStyleType {
			return GetMemoryStyle(bmCount, memUsage, environment)
		},
		"getCPUStyle": func(bmCount int, cpuUsage float64, environment string) ResourceStyleType {
			return GetCPUStyle(bmCount, cpuUsage, environment)
		},
		// Add the safeHTML function
		"safeHTML": func(s interface{}) template.HTML {
			return template.HTML(fmt.Sprint(s))
		},
	}
}
