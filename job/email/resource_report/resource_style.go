package resource_report

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
			Critical: 90.0,
			Warning:  80.0,
			Display:  80.0,
		},
		ClusterSmall: {
			Critical: 85.0,
			Warning:  75.0,
			Display:  70.0,
		},
	},
	EnvTest: {
		ClusterLarge: {
			Critical: 90.0,
			Warning:  85.0,
			Display:  85.0,
		},
		ClusterSmall: {
			Critical: 90.0,
			Warning:  80.0,
			Display:  75.0,
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

// GetCPUStyle 获取CPU分配率样式
// bmCount: 物理机数量
// cpuUsage: CPU使用率百分比
// environment: 环境类型 ("prd" 或 "test")
func GetCPUStyle(bmCount int, cpuUsage float64, environment string) ResourceStyleType {
	clusterType := getClusterType(bmCount)
	thresholds := getThresholds(environment, clusterType)

	// 应用样式规则
	if cpuUsage >= EmergencyThreshold {
		return StyleEmergency
	} else if cpuUsage >= thresholds.Critical {
		return StyleCritical
	} else if cpuUsage >= thresholds.Warning {
		return StyleWarning
	} else if cpuUsage < LowUtilizationThreshold && environment == EnvProduction {
		// 低利用率告警仅在生产环境下生效
		return StyleUnderutilized
	}

	return StyleNormal
}

// GetMemoryStyle 获取内存分配率样式
// bmCount: 物理机数量
// memUsage: 内存使用率百分比
// environment: 环境类型 ("prd" 或 "test")
func GetMemoryStyle(bmCount int, memUsage float64, environment string) ResourceStyleType {
	clusterType := getClusterType(bmCount)
	thresholds := getThresholds(environment, clusterType)

	// 应用样式规则
	if memUsage >= EmergencyThreshold {
		return StyleEmergency
	} else if memUsage >= thresholds.Critical {
		return StyleCritical
	} else if memUsage >= thresholds.Warning {
		return StyleWarning
	} else if memUsage < LowUtilizationThreshold && environment == EnvProduction {
		// 低利用率告警仅在生产环境下生效
		return StyleUnderutilized
	}

	return StyleNormal
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
