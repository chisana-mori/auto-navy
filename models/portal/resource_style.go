// Package portal 包含资源样式和处理相关的常量与函数
package portal

// Style 表示资源使用率的样式类型
type Style string

// 资源使用率样式常量定义
const (
	StyleEmergency     Style = "emergency"     // 紧急（最高告警级别，通常为深红色）
	StyleCritical      Style = "critical"      // 危险（高告警级别，通常为红色）
	StyleWarning       Style = "warning"       // 警告（中等告警级别，通常为黄色）
	StyleNormal        Style = "normal"        // 正常（无告警，通常为绿色）
	StyleUnderutilized Style = "underutilized" // 低利用率（特殊告警，通常为蓝色）
)

// 集群类型常量
const (
	ClusterTypeLarge     = "large" // 大型集群（物理机节点数 > 150）
	ClusterTypeSmall     = "small" // 小型/中型集群（物理机节点数 <= 150）
	ClusterSizeThreshold = 150     // 大小集群的划分阈值
)

// 环境类型常量
const (
	EnvProduction = "prd"  // 生产环境
	EnvTest       = "test" // 测试环境
)

// 资源使用率阈值常量
const (
	// 低利用率阈值（通用）
	LowUtilizationThreshold = 55.0

	// 警告级别阈值（根据集群大小和环境有所不同）
	EmergencyThresholdLarge = 95.0 // 大型集群紧急阈值
	EmergencyThresholdSmall = 90.0 // 小型集群紧急阈值
)

// StyleThresholds 存储不同环境和集群规模的样式阈值
type StyleThresholds struct {
	Emergency      float64 // 紧急告警阈值
	Critical       float64 // 危险告警阈值
	Warning        float64 // 警告告警阈值
	LowUtilization float64 // 低利用率告警阈值
}

// ResourcePoolStatus 资源池状态
type ResourcePoolStatus struct {
	Style         Style // 样式类型
	ShouldDisplay bool  // 是否应该显示
}

// getClusterType 根据物理机节点数判断集群类型
func getClusterType(bmCount int) string {
	if bmCount > ClusterSizeThreshold {
		return ClusterTypeLarge
	}
	return ClusterTypeSmall
}

// getThresholds 获取指定环境和集群类型的阈值
func getThresholds(clusterType, environment string) StyleThresholds {
	// 初始化基础阈值
	thresholds := StyleThresholds{
		LowUtilization: LowUtilizationThreshold,
	}

	// 大型集群的基础阈值（生产环境）
	if clusterType == ClusterTypeLarge {
		thresholds.Emergency = EmergencyThresholdLarge
		thresholds.Critical = 85.0
		thresholds.Warning = 80.0
	} else {
		// 小型集群的基础阈值（生产环境）
		thresholds.Emergency = EmergencyThresholdSmall
		thresholds.Critical = 75.0
		thresholds.Warning = 70.0
	}

	// 测试环境调整（阈值上调5%，不计算低利用率）
	if environment == EnvTest {
		thresholds.Warning += 5.0
		thresholds.Critical += 5.0
		thresholds.LowUtilization = 0.0 // 测试环境不考虑低利用率
	}

	return thresholds
}

// GetCPUStyle 根据CPU使用率获取样式
func GetCPUStyle(bmCount int, cpuUsage float64, environment string) Style {
	clusterType := getClusterType(bmCount)
	thresholds := getThresholds(clusterType, environment)

	// 高使用率部分
	if cpuUsage >= thresholds.Emergency {
		return StyleEmergency
	} else if cpuUsage >= thresholds.Critical {
		return StyleCritical
	} else if cpuUsage >= thresholds.Warning {
		return StyleWarning
	}

	// 低使用率部分（仅在生产环境且有配置阈值时判断）
	if thresholds.LowUtilization > 0 && cpuUsage < thresholds.LowUtilization {
		return StyleUnderutilized
	}

	// 默认正常
	return StyleNormal
}

// GetMemoryStyle 根据内存使用率获取样式
func GetMemoryStyle(bmCount int, memUsage float64, environment string) Style {
	clusterType := getClusterType(bmCount)
	thresholds := getThresholds(clusterType, environment)

	// 高使用率部分
	if memUsage >= thresholds.Emergency {
		return StyleEmergency
	} else if memUsage >= thresholds.Critical {
		return StyleCritical
	} else if memUsage >= thresholds.Warning {
		return StyleWarning
	}

	// 低使用率部分（仅在生产环境且有配置阈值时判断）
	if thresholds.LowUtilization > 0 && memUsage < thresholds.LowUtilization {
		return StyleUnderutilized
	}

	// 默认正常
	return StyleNormal
}

// GetResourcePoolStatus 获取资源池状态
func GetResourcePoolStatus(bmCount int, cpuUsage, memUsage float64, environment string) ResourcePoolStatus {
	cpuStyle := GetCPUStyle(bmCount, cpuUsage, environment)
	memStyle := GetMemoryStyle(bmCount, memUsage, environment)

	// 如果CPU或内存样式不是"normal"，则应该显示该资源池
	shouldDisplay := cpuStyle != StyleNormal || memStyle != StyleNormal

	// 确定整体样式（取两者中告警级别更高的）
	var style Style
	if cpuStyle == StyleEmergency || memStyle == StyleEmergency {
		style = StyleEmergency
	} else if cpuStyle == StyleCritical || memStyle == StyleCritical {
		style = StyleCritical
	} else if cpuStyle == StyleWarning || memStyle == StyleWarning {
		style = StyleWarning
	} else if cpuStyle == StyleUnderutilized || memStyle == StyleUnderutilized {
		style = StyleUnderutilized
	} else {
		style = StyleNormal
	}

	return ResourcePoolStatus{
		Style:         style,
		ShouldDisplay: shouldDisplay,
	}
}

// IsResourcePoolAbnormal 判断资源池是否异常（高使用率或低使用率）
func IsResourcePoolAbnormal(bmCount int, cpuUsage, memUsage float64, environment string) bool {
	cpuStyle := GetCPUStyle(bmCount, cpuUsage, environment)
	memStyle := GetMemoryStyle(bmCount, memUsage, environment)

	// 如果CPU或内存样式不是"normal"，则认为资源池异常
	return cpuStyle != StyleNormal || memStyle != StyleNormal
}

// IsClusterAbnormal 判断集群是否异常（高使用率或低使用率）
func IsClusterAbnormal(bmCount int, cpuUsage, memUsage float64, environment string) bool {
	// 复用资源池异常判断逻辑
	return IsResourcePoolAbnormal(bmCount, cpuUsage, memUsage, environment)
}
