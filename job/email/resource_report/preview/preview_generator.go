package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"navy-ng/job/email/resource_report"

	"github.com/Masterminds/sprig/v3"
)

// ResourcePool 资源池结构 - Preview generator local version with additional fields
type ResourcePool struct {
	// Core fields
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

	// Formatted display fields
	CPURequestFormatted     string // 格式化的CPU请求量
	CPUCapacityFormatted    string // 格式化的CPU容量
	MemoryRequestFormatted  string // 格式化的内存请求量
	MemoryCapacityFormatted string // 格式化的内存容量

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

	// Template specific fields
	ShowPoolDetails bool   // 是否显示资源池详情
	HeaderClass     string // 资源池头部CSS类

	// Resource Display fields - these would be populated by backend logic
	CPUDisplay    CPUDisplay    // CPU显示相关信息
	MemoryDisplay MemoryDisplay // 内存显示相关信息
	HasCPUStats   bool          // 是否有CPU统计信息

	// CPU和内存波动历史样式类
	CPUHistoryTrendClasses    []string // 历史CPU趋势样式类
	MemoryHistoryTrendClasses []string // 历史内存趋势样式类
}

// CPUDisplay holds all display info for CPU
type CPUDisplay struct {
	Class         string       // CSS class
	UsageText     string       // Formatted usage text
	TooltipText   string       // Text for tooltip
	BarClass      string       // CSS class for usage bar
	BarWidthClass string       // CSS class for bar width
	History       []TrendPoint // Historical trend points
}

// MemoryDisplay holds all display info for Memory
type MemoryDisplay struct {
	Class         string       // CSS class
	UsageText     string       // Formatted usage text
	TooltipText   string       // Text for tooltip
	BarClass      string       // CSS class for usage bar
	BarWidthClass string       // CSS class for bar width
	History       []TrendPoint // Historical trend points
}

// TrendPoint represents a single point in the trend history
type TrendPoint struct {
	Value     float64 // Actual usage value
	ValueText string  // Formatted value text
	Class     string  // CSS class for styling
	Label     string  // Day label (e.g., "Today", "Yesterday")
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
	IsAbnormalOverall   bool                     // 标识集群是否异常（可能来自于API或外部指标）
	ShowDetails         bool                     // 是否在报告中显示详情部分
	OverviewPools       []OverviewPool           // 用于概览表格的资源池数据
}

// OverviewPool 用于概览表格的简化资源池结构
type OverviewPool struct {
	ResourceType            string
	NodeType                string
	HasData                 bool
	Nodes                   int
	BMCount                 int
	VMCount                 int
	CPUCapacity             float64
	CPURequest              float64
	CPUUsagePercent         float64
	CPUUsageText            string
	CPUIndicatorClass       string
	CPURequestFormatted     string
	CPUCapacityFormatted    string
	MemoryCapacity          float64
	MemoryRequest           float64
	MemoryUsagePercent      float64
	MemoryUsageText         string
	MemoryIndicatorClass    string
	MemoryRequestFormatted  string
	MemoryCapacityFormatted string
	Desc                    string
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
	Environment          string       // 环境类型："test" 或 "test"
	ShowResourcePoolDesc bool         // 是否显示资源池描述
}

// 将资源报告的标准ResourcePool转换为预览本地的ResourcePool
func convertResourcePool(pool resource_report.ResourcePool, environment string) ResourcePool {
	// 计算CPU和内存使用率
	cpuUsage := 0.0
	if pool.CPUCapacity > 0 {
		cpuUsage = pool.CPURequest / pool.CPUCapacity * 100
	}
	memUsage := 0.0
	if pool.MemoryCapacity > 0 {
		memUsage = pool.MemoryRequest / pool.MemoryCapacity * 100
	}

	// Format request and capacity values
	cpuRequestFormatted := fmt.Sprintf("%.1f", pool.CPURequest)
	cpuCapacityFormatted := fmt.Sprintf("%.1f", pool.CPUCapacity)
	memoryRequestFormatted := fmt.Sprintf("%.1f", pool.MemoryRequest)
	memoryCapacityFormatted := fmt.Sprintf("%.1f", pool.MemoryCapacity)

	// For preview, we'll show details for most pools
	// In real implementation, this would be set based on some criteria
	showPoolDetails := pool.IsAbnormal // Only show details for abnormal pools

	// Set appropriate header class based on resource type
	headerClass := resource_report.GetPoolHeaderClassName(pool.ResourceType)
	hasCPUStats := pool.CPUCapacity > 0

	// Create CPU trend history for preview
	cpuHistory := make([]TrendPoint, 0)
	for i, value := range pool.CPUHistory {
		dayLabel := "今天"
		if i < len(pool.CPUHistory)-1 {
			daysAgo := len(pool.CPUHistory) - i - 1
			if daysAgo == 1 {
				dayLabel = "昨天"
			} else {
				dayLabel = fmt.Sprintf("%d天前", daysAgo)
			}
		}

		// Get appropriate CPU style class for this historical point
		cpuStyleClass := resource_report.GetCPUTrendStyleClass(value, pool.BMCount, environment)

		cpuHistory = append(cpuHistory, TrendPoint{
			Value:     value,
			ValueText: fmt.Sprintf("%.1f%%", value),
			Class:     cpuStyleClass,
			Label:     dayLabel,
		})
	}

	// Create memory trend history for preview
	memHistory := make([]TrendPoint, 0)
	for i, value := range pool.MemoryHistory {
		dayLabel := "今天"
		if i < len(pool.MemoryHistory)-1 {
			daysAgo := len(pool.MemoryHistory) - i - 1
			if daysAgo == 1 {
				dayLabel = "昨天"
			} else {
				dayLabel = fmt.Sprintf("%d天前", daysAgo)
			}
		}

		// Get appropriate memory style class for this historical point
		memStyleClass := resource_report.GetMemTrendStyleClass(value, pool.BMCount, environment)

		memHistory = append(memHistory, TrendPoint{
			Value:     value,
			ValueText: fmt.Sprintf("%.1f%%", value),
			Class:     memStyleClass,
			Label:     dayLabel,
		})
	}

	// For preview version, all styles are computed with proper environment
	cpuClass := string(resource_report.GetCPUStyle(pool.BMCount, cpuUsage, environment))
	memClass := string(resource_report.GetMemoryStyle(pool.BMCount, memUsage, environment))

	// Get tooltip text using the environment parameter
	cpuTooltip := resource_report.GetCPUTooltipMessage(cpuUsage, pool.BMCount, environment, pool.CPURequest, pool.CPUCapacity)
	memTooltip := resource_report.GetMemTooltipMessage(memUsage, pool.BMCount, environment, pool.MemoryRequest, pool.MemoryCapacity)

	// Get bar styles using the environment parameter
	cpuBarClass := resource_report.GetBarClassName(resource_report.GetCPUStyle(pool.BMCount, cpuUsage, environment))
	memBarClass := resource_report.GetBarClassName(resource_report.GetMemoryStyle(pool.BMCount, memUsage, environment))

	// Create basic CPU display info with appropriate styling
	cpuDisplay := CPUDisplay{
		Class:         cpuClass,
		UsageText:     fmt.Sprintf("%.1f%%", cpuUsage),
		TooltipText:   cpuTooltip,
		BarClass:      cpuBarClass,
		BarWidthClass: resource_report.GetBarWidthClassName(cpuUsage),
		History:       cpuHistory,
	}

	// Create memory display info with appropriate styling
	memDisplay := MemoryDisplay{
		Class:         memClass,
		UsageText:     fmt.Sprintf("%.1f%%", memUsage),
		TooltipText:   memTooltip,
		BarClass:      memBarClass,
		BarWidthClass: resource_report.GetBarWidthClassName(memUsage),
		History:       memHistory,
	}

	return ResourcePool{
		ResourceType:              pool.ResourceType,
		NodeType:                  pool.NodeType,
		Nodes:                     pool.Nodes,
		NodeCount:                 pool.Nodes,
		Type:                      pool.ResourceType,
		CPUCapacity:               pool.CPUCapacity,
		MemoryCapacity:            pool.MemoryCapacity,
		CPURequest:                pool.CPURequest,
		MemoryRequest:             pool.MemoryRequest,
		CPUUsagePercent:           cpuUsage,
		MemoryUsagePercent:        memUsage,
		PhysicalNodes:             pool.BMCount,
		VirtualNodes:              pool.VMCount,
		BMCount:                   pool.BMCount,
		VMCount:                   pool.VMCount,
		PodCount:                  pool.PodCount,
		PerNodeCpuRequest:         pool.PerNodeCpuRequest,
		PerNodeMemRequest:         pool.PerNodeMemRequest,
		CPURequestFormatted:       cpuRequestFormatted,
		CPUCapacityFormatted:      cpuCapacityFormatted,
		MemoryRequestFormatted:    memoryRequestFormatted,
		MemoryCapacityFormatted:   memoryCapacityFormatted,
		CPUHistory:                pool.CPUHistory,
		MemoryHistory:             pool.MemoryHistory,
		TotalCPU:                  pool.CPUCapacity,
		TotalMemory:               pool.MemoryCapacity,
		RequestedCPU:              pool.CPURequest,
		RequestedMemory:           pool.MemoryRequest,
		TooltipText:               pool.TooltipText,
		MaxCpuUsageRatio:          pool.MaxCpuUsageRatio,
		MaxMemoryUsageRatio:       pool.MaxMemoryUsageRatio,
		Desc:                      "", // 资源池描述默认为空字符串
		IsAbnormal:                pool.IsAbnormal,
		ShowPoolDetails:           showPoolDetails,
		HeaderClass:               headerClass,
		CPUDisplay:                cpuDisplay,
		MemoryDisplay:             memDisplay,
		HasCPUStats:               hasCPUStats,
		CPUHistoryTrendClasses:    pool.CPUHistoryTrendClasses,
		MemoryHistoryTrendClasses: pool.MemoryHistoryTrendClasses,
	}
}

// 将资源报告的标准ClusterResourceSummary转换为预览本地的ClusterResourceSummary
func convertClusterSummary(cluster resource_report.ClusterResourceSummary, environment string) ClusterResourceSummary {
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
		localPools[i] = convertResourcePool(pool, environment)
		poolsByType[pool.ResourceType] = &localPools[i]
	}

	// Make sure isAbnormalOverall and showDetails are set correctly
	isAbnormalOverall := cluster.IsAbnormal
	showDetails := cluster.IsAbnormal

	// Create simple overview pools for the preview
	overviewPoolsMap := map[string]OverviewPool{
		"total_common": {ResourceType: "total_common", NodeType: "通用资源", HasData: false},
		"intel_common": {ResourceType: "intel_common", NodeType: "Intel通用", HasData: false},
		"hg_common":    {ResourceType: "hg_common", NodeType: "海光通用", HasData: false},
		"arm_common":   {ResourceType: "arm_common", NodeType: "ARM通用", HasData: false},
	}

	// Fill in data for each pool type if available
	for _, pool := range localPools {
		if op, exists := overviewPoolsMap[pool.ResourceType]; exists {
			// Get appropriate style classes based on usage percentages
			cpuClass := string(resource_report.GetCPUStyle(pool.BMCount, pool.CPUUsagePercent, environment))
			memClass := string(resource_report.GetMemoryStyle(pool.BMCount, pool.MemoryUsagePercent, environment))

			op.HasData = true
			op.Nodes = pool.Nodes
			op.BMCount = pool.BMCount
			op.VMCount = pool.VMCount
			op.CPUCapacity = pool.CPUCapacity
			op.CPURequest = pool.CPURequest
			op.CPUUsagePercent = pool.CPUUsagePercent
			op.CPUUsageText = fmt.Sprintf("%.1f%%", pool.CPUUsagePercent)
			op.CPUIndicatorClass = cpuClass
			op.CPURequestFormatted = fmt.Sprintf("%.1f", pool.CPURequest)
			op.CPUCapacityFormatted = fmt.Sprintf("%.1f", pool.CPUCapacity)
			op.MemoryCapacity = pool.MemoryCapacity
			op.MemoryRequest = pool.MemoryRequest
			op.MemoryUsagePercent = pool.MemoryUsagePercent
			op.MemoryUsageText = fmt.Sprintf("%.1f%%", pool.MemoryUsagePercent)
			op.MemoryIndicatorClass = memClass
			op.MemoryRequestFormatted = fmt.Sprintf("%.1f", pool.MemoryRequest)
			op.MemoryCapacityFormatted = fmt.Sprintf("%.1f", pool.MemoryCapacity)
			op.Desc = pool.Desc
			overviewPoolsMap[pool.ResourceType] = op
		}
	}

	// Convert map to slice in the expected order
	overviewPools := []OverviewPool{
		overviewPoolsMap["total_common"],
		overviewPoolsMap["intel_common"],
		overviewPoolsMap["hg_common"],
		overviewPoolsMap["arm_common"],
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
		IsAbnormalOverall:   isAbnormalOverall,
		ShowDetails:         showDetails,
		OverviewPools:       overviewPools,
	}
}

// 将资源报告的标准ReportTemplateData转换为预览本地的ReportTemplateData
func convertTemplateData(data resource_report.ReportTemplateData) ReportTemplateData {
	// 转换集群列表
	localClusters := make([]ClusterResourceSummary, len(data.Clusters))
	for i, cluster := range data.Clusters {
		localClusters[i] = convertClusterSummary(cluster, data.Environment)
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

// Create示例数据
// Change return type to resource_report.ReportTemplateData
func createMockData(reportDate string, environment string) resource_report.ReportTemplateData {
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

	// Return the resource_report.ReportTemplateData directly
	return standardData
}

// GeneratePreview 生成预览文件
func generatePreview(environment string) error {
	// 生成当前日期字符串
	reportDate := time.Now().Format(time.DateOnly)

	// 读取模板文件
	templatePath := "../template.html"
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("读取模板文件失败: %v", err)
	}

	// 创建模板，合并sprig和自定义函数
	funcMap := sprig.FuncMap()
	for k, v := range resource_report.CustomTemplateFuncs() {
		funcMap[k] = v
	}

	// Add safeHTML function to allow HTML in tooltips
	funcMap["safeHTML"] = func(s interface{}) template.HTML {
		return template.HTML(fmt.Sprint(s))
	}

	tmpl, err := template.New("resourceReport").Funcs(funcMap).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("解析模板失败: %v", err)
	}

	// createMockData now returns resource_report.ReportTemplateData
	reportPackageData := createMockData(reportDate, environment)

	// For HTML template, convert to local ReportTemplateData
	htmlPreviewData := convertTemplateData(reportPackageData)

	// 渲染模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, htmlPreviewData); err != nil { // Use converted data for HTML
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
	// GenerateExcelReport expects resource_report.ReportTemplateData
	excelFilePath, err := resource_report.GenerateExcelReport(reportPackageData, reportDate) // Use original data for Excel
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
	env := "test"
	if len(os.Args) > 1 {
		env = os.Args[1]
	}

	if err := generatePreview(env); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating preview: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Preview generated successfully!")
}
