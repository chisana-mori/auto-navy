package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"navy-ng/job/email/resource_report"

	"github.com/Masterminds/sprig/v3"
)

// ResourcePool 资源池详情 - Preview generator's local version with additional fields
type ResourcePool struct {
	ResourceType       string
	NodeType           string
	Nodes              int
	NodeCount          int    // 与Nodes相同
	Type               string // 与ResourceType相同
	CPUCapacity        float64
	MemoryCapacity     float64
	CPURequest         float64
	MemoryRequest      float64
	CPUUsagePercent    float64
	MemoryUsagePercent float64
	PhysicalNodes      int     // 物理节点数量
	VirtualNodes       int     // 虚拟节点数量
	BMCount            int     // 兼容旧字段
	VMCount            int     // 兼容旧字段
	PodCount           int     // 新增Pod数量字段
	PerNodeCpuRequest  float64 // 新增节点平均CPU分配
	PerNodeMemRequest  float64 // 新增节点平均内存分配
	// 7天资源波动历史数据
	CPUHistory    []float64 // CPU使用率历史数据
	MemoryHistory []float64 // 内存使用率历史数据
	// 模板需要的字段
	TotalCPU        float64 // 与CPUCapacity相同
	TotalMemory     float64 // 与MemoryCapacity相同
	RequestedCPU    float64 // 与CPURequest相同
	RequestedMemory float64 // 与MemoryRequest相同
	TooltipText     string  // 资源池类型的tooltip文本
	// 过去24小时平均CPU和内存最大使用率
	MaxCpuUsageRatio    float64 // 平均CPU最大使用率，存储为小数，如0.36代表百分之36
	MaxMemoryUsageRatio float64 // 平均内存最大使用率，存储为小数，如0.36代表百分之36
	Desc                string  // 资源池描述字段
	IsAbnormal          bool    // 添加IsAbnormal字段，标识资源池是否异常
}

// ClusterResourceSummary 集群资源摘要 - Preview generator's local version with additional fields
type ClusterResourceSummary struct {
	ClusterName         string
	Name                string // 添加Name字段以兼容模板
	Desc                string // 集群分组描述字段，用于将集群按组分类和排序
	TotalNodes          int
	PhysicalNodes       int // 物理节点数量
	VirtualNodes        int // 虚拟节点数量
	TotalCPUCapacity    float64
	TotalMemoryCapacity float64
	TotalCPU            float64 // 与TotalCPUCapacity相同
	TotalMemory         float64 // 与TotalMemoryCapacity相同
	RequestedCPU        float64
	RequestedMemory     float64
	CPUUsagePercent     float64
	MemoryUsagePercent  float64
	ResourcePools       []ResourcePool
	ResourcePoolsByType map[string]*ResourcePool // 根据资源池类型快速查找资源池
	IsAbnormal          bool                     // 添加IsAbnormal字段，标识集群是否异常
}

// ClusterStats 集群统计信息 - Preview generator's local version
type ClusterStats struct {
	TotalClusters     int     // 总已巡检集群数
	NormalClusters    int     // 正常集群数
	AbnormalClusters  int     // 异常集群数
	GeneralPodDensity float64 // 通用集群Pod密度
}

// ReportTemplateData 报告模板数据 - Preview generator's local version
type ReportTemplateData struct {
	ReportDate           string
	Clusters             []ClusterResourceSummary
	Stats                ClusterStats // 添加集群统计信息
	HasHighUsageClusters bool         // 是否存在高使用率集群（CPU或内存使用率>=70%）
	Environment          string       // 环境类型："prd" 或 "test"
	ShowResourcePoolDesc bool         // 是否显示资源池描述
}

// 将资源报告的标准ResourcePool转换为预览本地的ResourcePool
func convertResourcePool(pool resource_report.ResourcePool) ResourcePool {
	// 计算CPU和内存使用率
	cpuUsage := 0.0
	if pool.CPUCapacity > 0 {
		cpuUsage = pool.CPURequest / pool.CPUCapacity * 100
	}
	memUsage := 0.0
	if pool.MemoryCapacity > 0 {
		memUsage = pool.MemoryRequest / pool.MemoryCapacity * 100
	}

	// 根据资源池类型设置tooltip文本
	var tooltipText string
	switch pool.ResourceType {
	case "total":
		tooltipText = "集群所有物理机资源总和，包含集群中所有类型的节点。"
	case "total_intel":
		tooltipText = "Intel架构物理机节点资源，使用Intel CPU的所有节点。"
	case "intel_common":
		tooltipText = "Intel物理机通用应用节点资源，没有特殊标记或污点的Intel节点。"
	case "intel_gpu":
		tooltipText = "Intel架构GPU物理机节点，配备了GPU的Intel节点。"
	case "intel_taint":
		tooltipText = "Intel架构带污点物理机节点，带有特殊污点标记的Intel节点。"
	case "intel_non_gpu":
		tooltipText = "Intel架构无GPU物理机节点，不包含GPU的Intel节点。"
	case "total_arm":
		tooltipText = "ARM架构物理机节点资源，使用ARM CPU的所有节点。"
	case "arm_common":
		tooltipText = "ARM物理机节点通用应用资源，没有特殊标记或污点的ARM节点。"
	case "arm_gpu":
		tooltipText = "ARM架构GPU物理机节点，配备了GPU的ARM节点。"
	case "arm_taint":
		tooltipText = "ARM架构带污点物理机节点，带有特殊污点标记的ARM节点。"
	case "total_hg":
		tooltipText = "海光架构物理机节点资源，使用海光CPU的所有节点。"
	case "hg_common":
		tooltipText = "海光物理机通用应用节点资源，没有特殊标记或污点的海光节点。"
	case "hg_taint":
		tooltipText = "海光架构带污点物理机节点，带有特殊污点标记的海光节点。"
	case "total_taint":
		tooltipText = "带污点的物理机节点资源，所有带有特殊污点标记的节点。"
	case "total_common":
		tooltipText = "物理机节点通用应用资源总和，所有没有特殊标记或污点的普通节点。"
	case "total_gpu":
		tooltipText = "包含GPU的物理机节点资源，所有配备了GPU的节点。"
	case "aplus_total":
		tooltipText = "A+物理机资源总和，所有高性能计算节点。"
	case "aplus_intel":
		tooltipText = "A+Intel架构物理机节点，高性能计算的Intel节点。"
	case "aplus_arm":
		tooltipText = "A+ARM架构物理机节点，高性能计算的ARM节点。"
	case "aplus_hg":
		tooltipText = "A+海光架构物理机节点，高性能计算的海光节点。"
	case "dplus_total":
		tooltipText = "D+物理机资源总和，所有高存储容量节点。"
	case "dplus_intel":
		tooltipText = "D+Intel架构物理机节点，高存储容量的Intel节点。"
	case "dplus_arm":
		tooltipText = "D+ARM架构物理机节点，高存储容量的ARM节点。"
	case "dplus_hg":
		tooltipText = "D+海光架构物理机节点，高存储容量的海光节点。"
	default:
		tooltipText = pool.NodeType + "资源池，类型: " + pool.ResourceType
	}

	return ResourcePool{
		ResourceType:        pool.ResourceType,
		NodeType:            pool.NodeType,
		Nodes:               pool.Nodes,
		NodeCount:           pool.Nodes,
		Type:                pool.ResourceType,
		CPUCapacity:         pool.CPUCapacity,
		MemoryCapacity:      pool.MemoryCapacity,
		CPURequest:          pool.CPURequest,
		MemoryRequest:       pool.MemoryRequest,
		CPUUsagePercent:     cpuUsage,
		MemoryUsagePercent:  memUsage,
		BMCount:             pool.BMCount,
		VMCount:             pool.VMCount,
		PodCount:            pool.PodCount,
		PerNodeCpuRequest:   pool.PerNodeCpuRequest,
		PerNodeMemRequest:   pool.PerNodeMemRequest,
		CPUHistory:          pool.CPUHistory,
		MemoryHistory:       pool.MemoryHistory,
		TotalCPU:            pool.CPUCapacity,
		TotalMemory:         pool.MemoryCapacity,
		RequestedCPU:        pool.CPURequest,
		RequestedMemory:     pool.MemoryRequest,
		TooltipText:         tooltipText,
		MaxCpuUsageRatio:    pool.MaxCpuUsageRatio,
		MaxMemoryUsageRatio: pool.MaxMemoryUsageRatio,
		Desc:                "", // 资源池描述默认为空字符串
		IsAbnormal:          pool.IsAbnormal,
	}
}

// 将资源报告的标准ClusterResourceSummary转换为预览本地的ClusterResourceSummary
func convertClusterSummary(cluster resource_report.ClusterResourceSummary) ClusterResourceSummary {
	// 计算物理和虚拟节点数量
	physicalNodes := 0
	virtualNodes := 0
	for _, pool := range cluster.ResourcePools {
		if pool.ResourceType == "total" {
			physicalNodes = pool.BMCount
			virtualNodes = pool.VMCount
			break
		}
	}

	// 计算CPU和内存使用率
	cpuUsage := 0.0
	if cluster.TotalCPUCapacity > 0 {
		cpuUsage = cluster.TotalCPURequest / cluster.TotalCPUCapacity * 100
	}
	memUsage := 0.0
	if cluster.TotalMemoryCapacity > 0 {
		memUsage = cluster.TotalMemoryRequest / cluster.TotalMemoryCapacity * 100
	}

	// 转换资源池
	localPools := make([]ResourcePool, len(cluster.ResourcePools))
	poolsByType := make(map[string]*ResourcePool)
	for i, pool := range cluster.ResourcePools {
		localPools[i] = convertResourcePool(pool)
		poolsByType[pool.ResourceType] = &localPools[i]
	}

	return ClusterResourceSummary{
		ClusterName:         cluster.ClusterName,
		Name:                cluster.ClusterName,
		Desc:                cluster.Desc, // 复制集群描述字段
		TotalNodes:          cluster.TotalNodes,
		PhysicalNodes:       physicalNodes,
		VirtualNodes:        virtualNodes,
		TotalCPUCapacity:    cluster.TotalCPUCapacity,
		TotalMemoryCapacity: cluster.TotalMemoryCapacity,
		TotalCPU:            cluster.TotalCPUCapacity,
		TotalMemory:         cluster.TotalMemoryCapacity,
		RequestedCPU:        cluster.TotalCPURequest,
		RequestedMemory:     cluster.TotalMemoryRequest,
		CPUUsagePercent:     cpuUsage,
		MemoryUsagePercent:  memUsage,
		ResourcePools:       localPools,
		ResourcePoolsByType: poolsByType,
		IsAbnormal:          cluster.IsAbnormal,
	}
}

// 将资源报告的标准ReportTemplateData转换为预览本地的ReportTemplateData
func convertTemplateData(data resource_report.ReportTemplateData) ReportTemplateData {
	// 转换集群列表
	localClusters := make([]ClusterResourceSummary, len(data.Clusters))
	for i, cluster := range data.Clusters {
		localClusters[i] = convertClusterSummary(cluster)
	}

	return ReportTemplateData{
		ReportDate: data.ReportDate,
		Clusters:   localClusters,
		Stats: ClusterStats{
			TotalClusters:     data.Stats.TotalClusters,
			NormalClusters:    data.Stats.NormalClusters,
			AbnormalClusters:  data.Stats.AbnormalClusters,
			GeneralPodDensity: data.Stats.GeneralPodDensity,
		},
		HasHighUsageClusters: data.HasHighUsageClusters,
		Environment:          data.Environment,
		ShowResourcePoolDesc: data.ShowResourcePoolDesc,
	}
}

// 辅助函数，用于将接口转换为浮点数
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
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

// 添加自定义模板函数
func customTemplateFuncs() template.FuncMap {
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
			case []interface{}:
				return len(v)
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
			return string(resource_report.GetCPUStyle(bmCount, cpuUsage, environment))
		},
		// 获取内存样式类的函数
		"getMemStyleClass": func(bmCount int, memUsage float64, environment string) string {
			return string(resource_report.GetMemoryStyle(bmCount, memUsage, environment))
		},
		// 判断资源池是否应该显示的函数
		"shouldShowResourcePool": func(bmCount int, cpuUsage, memUsage float64, environment string) bool {
			status := resource_report.GetResourcePoolStatus(bmCount, cpuUsage, memUsage, environment)
			return status.ShouldDisplay
		},
		// 判断资源池是否异常的函数
		"isResourcePoolAbnormal": func(bmCount int, cpuUsage, memUsage float64, environment string) bool {
			return resource_report.IsResourcePoolAbnormal(bmCount, cpuUsage, memUsage, environment)
		},
		// 新增函数，用于判断资源池是否需要显示
		"shouldShowPool": func(cpuUsage, memoryUsage float64, bmCount int, environment string) bool {
			// 使用统一样式库判断资源池是否应显示
			return resource_report.GetResourcePoolStatus(bmCount, cpuUsage, memoryUsage, environment).ShouldDisplay
		},
		// 获取CPU颜色类的函数 - 使用统一样式库
		"getCpuColorClass": func(pool *ResourcePool, environment string) string {
			if pool == nil {
				return ""
			}
			return string(resource_report.GetCPUStyle(pool.BMCount, pool.CPUUsagePercent, environment))
		},
		// 获取内存颜色类的函数 - 使用统一样式库
		"getMemColorClass": func(pool *ResourcePool, environment string) string {
			if pool == nil {
				return ""
			}
			return string(resource_report.GetMemoryStyle(pool.BMCount, pool.MemoryUsagePercent, environment))
		},
		// 获取资源使用率对应的颜色类
		"getResourceUsageColorClass": func(usage float64, isLargePool bool, environment string) string {
			if environment == "test" {
				// 测试环境规则
				if isLargePool {
					if usage >= 95.0 {
						return "emergency-usage"
					} else if usage >= 90.0 {
						return "critical-usage"
					} else if usage >= 85.0 {
						return "warning-usage"
					}
					return "normal"
				} else {
					if usage >= 90.0 {
						return "emergency-usage"
					} else if usage >= 80.0 {
						return "critical-usage"
					} else if usage >= 75.0 {
						return "warning-usage"
					}
					return "normal"
				}
			} else {
				// 生产环境规则
				if isLargePool {
					if usage >= 95.0 {
						return "emergency-usage"
					} else if usage >= 85.0 {
						return "critical-usage"
					} else if usage >= 80.0 {
						return "warning-usage"
					} else if usage < 55.0 {
						return "underutilized"
					}
					return "normal"
				} else {
					if usage >= 90.0 {
						return "emergency-usage"
					} else if usage >= 75.0 {
						return "critical-usage"
					} else if usage >= 70.0 {
						return "warning-usage"
					} else if usage < 55.0 {
						return "underutilized"
					}
					return "normal"
				}
			}
		},
		// 获取波动图颜色类的函数
		"getTrendColorClass": func(usage float64, isLargePool bool, resourceType, environment string) string {
			baseClass := ""
			if resourceType == "cpu" {
				baseClass = "cpu-trend-"
			} else if resourceType == "memory" {
				baseClass = "memory-trend-"
			} else {
				return "" // Unknown resource type
			}

			if environment == "test" {
				// 测试环境规则
				if isLargePool {
					if usage >= 95.0 {
						return baseClass + "emergency"
					} else if usage >= 90.0 {
						return baseClass + "critical"
					} else if usage >= 85.0 {
						return baseClass + "high"
					}
					return baseClass + "normal"
				} else {
					if usage >= 90.0 {
						return baseClass + "emergency"
					} else if usage >= 80.0 {
						return baseClass + "critical"
					} else if usage >= 75.0 {
						return baseClass + "high"
					}
					return baseClass + "normal"
				}
			} else {
				// 生产环境规则
				if isLargePool {
					if usage >= 95.0 {
						return baseClass + "emergency"
					} else if usage >= 85.0 {
						return baseClass + "critical"
					} else if usage >= 80.0 {
						return baseClass + "high"
					} else if usage < 55.0 {
						return baseClass + "underutilized"
					}
					return baseClass + "normal"
				} else {
					if usage >= 90.0 {
						return baseClass + "emergency"
					} else if usage >= 75.0 {
						return baseClass + "critical"
					} else if usage >= 70.0 {
						return baseClass + "high"
					} else if usage < 55.0 {
						return baseClass + "underutilized"
					}
					return baseClass + "normal"
				}
			}
		},
		// 新增函数，用于获取资源池类型的颜色
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
		// 创建字符串切片函数
		"slice": func(args ...interface{}) []interface{} {
			return args
		},
	}
}

// 创建示例数据
func createMockData(reportDate string, environment string) ReportTemplateData {
	// 创建生产环境集群（高负载）
	cluster1 := resource_report.ClusterResourceSummary{
		ClusterName:         "production-cluster",
		Desc:                "生产环境", // 添加集群描述
		TotalNodes:          30,
		TotalCPUCapacity:    480.0,
		TotalMemoryCapacity: 960.0,
		TotalCPURequest:     408.0,
		TotalMemoryRequest:  864.0,
		IsAbnormal:          true,
		ResourcePools: []resource_report.ResourcePool{
			{
				ResourceType:      "total",
				NodeType:          "总资源",
				Nodes:             30,
				CPUCapacity:       480.0,
				MemoryCapacity:    960.0,
				CPURequest:        408.0,
				MemoryRequest:     864.0,
				BMCount:           20,
				VMCount:           10,
				PodCount:          300,
				PerNodeCpuRequest: 13.6,
				PerNodeMemRequest: 28.8,
				CPUHistory:        []float64{78.0, 80.0, 82.0, 85.0, 83.0, 84.0, 85.0},
				MemoryHistory:     []float64{82.0, 85.0, 87.0, 90.0, 88.0, 89.0, 90.0},
				IsAbnormal:        true,
			},
			{
				ResourceType:      "intel_common",
				NodeType:          "Intel通用节点",
				Nodes:             15,
				CPUCapacity:       240.0,
				MemoryCapacity:    480.0,
				CPURequest:        216.0,
				MemoryRequest:     456.0,
				BMCount:           10,
				VMCount:           5,
				PodCount:          160,
				PerNodeCpuRequest: 14.4,
				PerNodeMemRequest: 30.4,
				CPUHistory:        []float64{85.0, 87.0, 88.0, 90.0, 92.0, 91.0, 90.0},
				MemoryHistory:     []float64{90.0, 92.0, 93.0, 94.0, 96.0, 95.0, 95.0},
				IsAbnormal:        true,
			},
		},
		NodesData: []resource_report.NodeResourceDetail{
			{
				NodeName:          "node-1",
				CPURequest:        14.2,
				MemoryRequest:     28.4,
				CPULimit:          16.0,
				MemoryLimit:       32.0,
				CPUUsage:          12.1,
				MemoryUsage:       26.8,
				CPUAllocatable:    16.0,
				MemoryAllocatable: 32.0,
			},
		},
	}

	// 创建测试环境集群（中等负载）
	cluster2 := resource_report.ClusterResourceSummary{
		ClusterName:         "test-cluster",
		Desc:                "测试环境", // 添加集群描述
		TotalNodes:          15,
		TotalCPUCapacity:    240.0,
		TotalMemoryCapacity: 480.0,
		TotalCPURequest:     168.0,
		TotalMemoryRequest:  360.0,
		IsAbnormal:          true,
		ResourcePools: []resource_report.ResourcePool{
			{
				ResourceType:      "total",
				NodeType:          "总资源",
				Nodes:             15,
				CPUCapacity:       240.0,
				MemoryCapacity:    480.0,
				CPURequest:        168.0,
				MemoryRequest:     360.0,
				BMCount:           8,
				VMCount:           7,
				PodCount:          120,
				PerNodeCpuRequest: 11.2,
				PerNodeMemRequest: 24.0,
				CPUHistory:        []float64{65.0, 67.0, 68.0, 70.0, 72.0, 71.0, 70.0},
				MemoryHistory:     []float64{70.0, 72.0, 73.0, 75.0, 76.0, 74.0, 75.0},
			},
		},
	}

	// 创建低使用率集群示例
	cluster3 := resource_report.ClusterResourceSummary{
		ClusterName:         "underutilized-cluster",
		Desc:                "低利用率集群", // 添加集群描述
		TotalNodes:          20,
		TotalCPUCapacity:    320.0,
		TotalMemoryCapacity: 640.0,
		TotalCPURequest:     160.0,
		TotalMemoryRequest:  320.0,
		IsAbnormal:          true,
		ResourcePools: []resource_report.ResourcePool{
			{
				ResourceType:        "total",
				NodeType:            "总资源",
				Nodes:               20,
				CPUCapacity:         320.0,
				MemoryCapacity:      640.0,
				CPURequest:          160.0,
				MemoryRequest:       320.0,
				BMCount:             12,
				VMCount:             8,
				PodCount:            80,
				PerNodeCpuRequest:   8.0,
				PerNodeMemRequest:   16.0,
				CPUHistory:          []float64{48.0, 47.0, 46.0, 45.0, 44.0, 45.0, 50.0},
				MemoryHistory:       []float64{52.0, 51.0, 50.0, 49.0, 48.0, 47.0, 50.0},
				MaxCpuUsageRatio:    0.65,
				MaxMemoryUsageRatio: 0.70,
			},
			{
				ResourceType:        "intel_common",
				NodeType:            "Intel通用节点",
				Nodes:               10,
				CPUCapacity:         160.0,
				MemoryCapacity:      320.0,
				CPURequest:          80.0,
				MemoryRequest:       160.0,
				BMCount:             6,
				VMCount:             4,
				PodCount:            40,
				PerNodeCpuRequest:   8.0,
				PerNodeMemRequest:   16.0,
				CPUHistory:          []float64{45.0, 44.0, 43.0, 42.0, 41.0, 40.0, 50.0},
				MemoryHistory:       []float64{50.0, 49.0, 48.0, 47.0, 46.0, 45.0, 50.0},
				MaxCpuUsageRatio:    0.60,
				MaxMemoryUsageRatio: 0.65,
			},
			{
				ResourceType:        "hg_common",
				NodeType:            "海光通用节点",
				Nodes:               5,
				CPUCapacity:         80.0,
				MemoryCapacity:      160.0,
				CPURequest:          40.0,
				MemoryRequest:       80.0,
				BMCount:             3,
				VMCount:             2,
				PodCount:            20,
				PerNodeCpuRequest:   8.0,
				PerNodeMemRequest:   16.0,
				CPUHistory:          []float64{48.0, 47.0, 46.0, 45.0, 44.0, 45.0, 50.0},
				MemoryHistory:       []float64{52.0, 51.0, 50.0, 49.0, 48.0, 47.0, 50.0},
				MaxCpuUsageRatio:    0.55,
				MaxMemoryUsageRatio: 0.60,
			},
			{
				ResourceType:        "arm_common",
				NodeType:            "ARM通用节点",
				Nodes:               5,
				CPUCapacity:         80.0,
				MemoryCapacity:      160.0,
				CPURequest:          40.0,
				MemoryRequest:       80.0,
				BMCount:             3,
				VMCount:             2,
				PodCount:            20,
				PerNodeCpuRequest:   8.0,
				PerNodeMemRequest:   16.0,
				CPUHistory:          []float64{52.0, 51.0, 50.0, 49.0, 48.0, 47.0, 50.0},
				MemoryHistory:       []float64{54.0, 53.0, 52.0, 51.0, 50.0, 49.0, 50.0},
				MaxCpuUsageRatio:    0.70,
				MaxMemoryUsageRatio: 0.75,
			},
		},
	}

	// 创建极低使用率集群示例
	cluster4 := resource_report.ClusterResourceSummary{
		ClusterName:         "very-underutilized-cluster",
		Desc:                "极低利用率集群", // 添加集群描述
		TotalNodes:          10,
		TotalCPUCapacity:    160.0,
		TotalMemoryCapacity: 320.0,
		TotalCPURequest:     40.0,
		TotalMemoryRequest:  80.0,
		IsAbnormal:          true,
		ResourcePools: []resource_report.ResourcePool{
			{
				ResourceType:      "total",
				NodeType:          "总资源",
				Nodes:             10,
				CPUCapacity:       160.0,
				MemoryCapacity:    320.0,
				CPURequest:        40.0,
				MemoryRequest:     80.0,
				BMCount:           6,
				VMCount:           4,
				PodCount:          20,
				PerNodeCpuRequest: 4.0,
				PerNodeMemRequest: 8.0,
				CPUHistory:        []float64{28.0, 27.0, 26.0, 25.0, 24.0, 25.0, 25.0},
				MemoryHistory:     []float64{27.0, 26.0, 25.0, 24.0, 23.0, 22.0, 25.0},
			},
		},
	}

	// 创建统计信息
	stats := resource_report.ClusterStats{
		TotalClusters:     4, // 修正为实际的集群数量，与上面创建的cluster1/2/3/4一致
		NormalClusters:    2, // 正常集群数量
		AbnormalClusters:  2, // 异常集群数量
		GeneralPodDensity: 10.5,
	}

	// 创建标准的模板数据
	standardData := resource_report.ReportTemplateData{
		ReportDate:           reportDate,
		Clusters:             []resource_report.ClusterResourceSummary{cluster1, cluster2, cluster3, cluster4},
		Stats:                stats,
		HasHighUsageClusters: true,        // 表示有异常使用率（高或低）的集群
		Environment:          environment, // 使用传入的环境类型
		ShowResourcePoolDesc: true,        // 设置ShowResourcePoolDesc为true
	}

	// 转换为本地格式
	return convertTemplateData(standardData)
}

// GeneratePreview 生成预览文件
func generatePreview(environment string) error {
	// 生成当前日期字符串
	reportDate := time.Now().Format("2006-01-02")

	// 读取模板文件
	templatePath := "../template.html"
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("读取模板文件失败: %v", err)
	}

	// 创建模板，合并sprig和自定义函数
	funcMap := sprig.FuncMap()
	for k, v := range customTemplateFuncs() {
		funcMap[k] = v
	}

	tmpl, err := template.New("resourceReport").Funcs(funcMap).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("解析模板失败: %v", err)
	}

	// 创建模拟数据
	// 这里我们创建一个简单的模拟数据集
	data := createMockData(reportDate, environment)

	// 渲染模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("渲染模板失败: %v", err)
	}

	// 确保输出目录存在
	outputDir := "."
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %v", err)
		}
	}

	// 生成Excel报告
	excelFilePath, err := generateExcelReport(data, reportDate)
	if err != nil {
		fmt.Printf("生成Excel报告失败: %v\n", err)
	} else {
		fmt.Printf("Excel报告已生成: %s\n", excelFilePath)
	}

	// 将结果保存到文件
	outputPath := filepath.Join(outputDir, "preview.html")
	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("保存预览文件失败: %v", err)
	}

	// 获取当前目录的绝对路径
	absPath, _ := filepath.Abs(outputPath)
	fmt.Printf("预览HTML文件已生成: %s\n", absPath)

	return nil
}

// 添加main函数
func main() {
	// 默认使用生产环境配置生成预览
	env := "prd"
	if len(os.Args) > 1 {
		env = os.Args[1]
	}

	if err := generatePreview(env); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating preview: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Preview generated successfully!")
}
