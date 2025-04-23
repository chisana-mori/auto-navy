package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Masterminds/sprig/v3"
)

// ResourcePool 资源池详情
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
	PhysicalNodes      int // 物理节点数量
	VirtualNodes       int // 虚拟节点数量
	BMCount            int // 兼容旧字段
	VMCount            int // 兼容旧字段
	// 模板需要的字段
	TotalCPU        float64 // 与CPUCapacity相同
	TotalMemory     float64 // 与MemoryCapacity相同
	RequestedCPU    float64 // 与CPURequest相同
	RequestedMemory float64 // 与MemoryRequest相同
}

// ClusterResourceSummary 集群资源摘要
type ClusterResourceSummary struct {
	ClusterName         string
	Name                string // 添加Name字段以兼容模板
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
}

// ReportTemplateData 报告模板数据
type ReportTemplateData struct {
	ReportDate           string
	Clusters             []ClusterResourceSummary
	HasHighUsageClusters bool
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
		"formatFloat": func(f float64, precision int) string {
			format := "%." + strconv.Itoa(precision) + "f"
			return fmt.Sprintf(format, f)
		},
		"formatBytes": func(bytes float64) string {
			const (
				KB = 1024
				MB = 1024 * KB
				GB = 1024 * MB
				TB = 1024 * GB
			)

			switch {
			case bytes >= TB:
				return fmt.Sprintf("%.2f TB", bytes/TB)
			case bytes >= GB:
				return fmt.Sprintf("%.2f GB", bytes/GB)
			case bytes >= MB:
				return fmt.Sprintf("%.2f MB", bytes/MB)
			case bytes >= KB:
				return fmt.Sprintf("%.2f KB", bytes/KB)
			default:
				return fmt.Sprintf("%.0f B", bytes)
			}
		},
		// 新增函数，用于判断资源池是否需要显示
		"shouldShowPool": func(cpuUsage, memoryUsage float64) bool {
			// 根据新模板，只显示 CPU 或内存使用率 >= 70% 的资源池
			return cpuUsage >= 70.0 || memoryUsage >= 70.0
		},
		// 新增函数，用于获取资源池类型的颜色
		"getPoolTypeColor": func(poolType string) string {
			switch poolType {
			case "total":
				return "#3498db" // 蓝色
			case "intel_common":
				return "#2ecc71" // 绿色
			case "intel_gpu":
				return "#e74c3c" // 红色
			case "arm_common":
				return "#f39c12" // 橙色
			case "total_common":
				return "#9b59b6" // 紫色
			default:
				return "#95a5a6" // 灰色
			}
		},
		// 新增函数，用于获取资源使用率的颜色
		"getUsageColor": func(usagePercent float64) string {
			// 根据新模板，使用 70%、75%、90% 作为警告、危险和紧急的阈值
			switch {
			case usagePercent >= 90.0:
				return "#e74c3c" // 红色（紧急）
			case usagePercent >= 75.0:
				return "#f39c12" // 橙色（危险）
			case usagePercent >= 70.0:
				return "#f1c40f" // 黄色（警告）
			default:
				return "#2ecc71" // 绿色（正常）
			}
		},
	}
}

// createMockData 创建模拟数据
func createMockData(reportDate string) ReportTemplateData {
	// 创建生产环境集群（高负载）
	cluster1 := ClusterResourceSummary{
		ClusterName:         "production-cluster",
		Name:                "production-cluster",
		TotalNodes:          30,
		PhysicalNodes:       20,
		VirtualNodes:        10,
		TotalCPUCapacity:    480.0,
		TotalMemoryCapacity: 960.0,
		TotalCPU:            480.0,
		TotalMemory:         960.0,
		RequestedCPU:        408.0,
		RequestedMemory:     864.0,
		CPUUsagePercent:     85.0,
		MemoryUsagePercent:  90.0,
		ResourcePools: []ResourcePool{
			{
				ResourceType:       "total",
				NodeType:           "总资源",
				Nodes:              30,
				NodeCount:          30,
				Type:               "total",
				CPUCapacity:        480.0,
				MemoryCapacity:     960.0,
				CPURequest:         408.0,
				MemoryRequest:      864.0,
				CPUUsagePercent:    85.0,
				MemoryUsagePercent: 90.0,
				PhysicalNodes:      20,
				VirtualNodes:       10,
				BMCount:            20,
				VMCount:            10,
				TotalCPU:           480.0,
				TotalMemory:        960.0,
				RequestedCPU:       408.0,
				RequestedMemory:    864.0,
			},
			{
				ResourceType:       "intel_common",
				NodeType:           "Intel通用节点",
				Nodes:              15,
				NodeCount:          15,
				Type:               "intel_common",
				CPUCapacity:        240.0,
				MemoryCapacity:     480.0,
				CPURequest:         216.0,
				MemoryRequest:      456.0,
				CPUUsagePercent:    90.0,
				MemoryUsagePercent: 95.0,
				PhysicalNodes:      10,
				VirtualNodes:       5,
				BMCount:            10,
				VMCount:            5,
				TotalCPU:           240.0,
				TotalMemory:        480.0,
				RequestedCPU:       216.0,
				RequestedMemory:    456.0,
			},
			{
				ResourceType:       "intel_gpu",
				NodeType:           "Intel GPU节点",
				Nodes:              5,
				NodeCount:          5,
				Type:               "intel_gpu",
				CPUCapacity:        80.0,
				MemoryCapacity:     160.0,
				CPURequest:         76.0,
				MemoryRequest:      152.0,
				CPUUsagePercent:    95.0,
				MemoryUsagePercent: 95.0,
				PhysicalNodes:      5,
				VirtualNodes:       0,
				BMCount:            5,
				VMCount:            0,
				TotalCPU:           80.0,
				TotalMemory:        160.0,
				RequestedCPU:       76.0,
				RequestedMemory:    152.0,
			},
		},
	}

	// 创建测试环境集群（中等负载）
	cluster2 := ClusterResourceSummary{
		ClusterName:         "test-cluster",
		Name:                "test-cluster",
		TotalNodes:          15,
		PhysicalNodes:       8,
		VirtualNodes:        7,
		TotalCPUCapacity:    240.0,
		TotalMemoryCapacity: 480.0,
		TotalCPU:            240.0,
		TotalMemory:         480.0,
		RequestedCPU:        168.0,
		RequestedMemory:     360.0,
		CPUUsagePercent:     70.0,
		MemoryUsagePercent:  75.0,
		ResourcePools: []ResourcePool{
			{
				ResourceType:       "total",
				NodeType:           "总资源",
				Nodes:              15,
				NodeCount:          15,
				Type:               "total",
				CPUCapacity:        240.0,
				MemoryCapacity:     480.0,
				CPURequest:         168.0,
				MemoryRequest:      360.0,
				CPUUsagePercent:    70.0,
				MemoryUsagePercent: 75.0,
				PhysicalNodes:      8,
				VirtualNodes:       7,
				BMCount:            8,
				VMCount:            7,
				TotalCPU:           240.0,
				TotalMemory:        480.0,
				RequestedCPU:       168.0,
				RequestedMemory:    360.0,
			},
			{
				ResourceType:       "intel_common",
				NodeType:           "Intel通用节点",
				Nodes:              10,
				NodeCount:          10,
				Type:               "intel_common",
				CPUCapacity:        160.0,
				MemoryCapacity:     320.0,
				CPURequest:         128.0,
				MemoryRequest:      272.0,
				CPUUsagePercent:    80.0,
				MemoryUsagePercent: 85.0,
				PhysicalNodes:      6,
				VirtualNodes:       4,
				BMCount:            6,
				VMCount:            4,
				TotalCPU:           160.0,
				TotalMemory:        320.0,
				RequestedCPU:       128.0,
				RequestedMemory:    272.0,
			},
		},
	}

	// 创建开发环境集群（低负载）
	cluster3 := ClusterResourceSummary{
		ClusterName:         "dev-cluster",
		Name:                "dev-cluster",
		TotalNodes:          10,
		PhysicalNodes:       4,
		VirtualNodes:        6,
		TotalCPUCapacity:    160.0,
		TotalMemoryCapacity: 320.0,
		TotalCPU:            160.0,
		TotalMemory:         320.0,
		RequestedCPU:        96.0,
		RequestedMemory:     192.0,
		CPUUsagePercent:     60.0,
		MemoryUsagePercent:  60.0,
		ResourcePools: []ResourcePool{
			{
				ResourceType:       "total",
				NodeType:           "总资源",
				Nodes:              10,
				NodeCount:          10,
				Type:               "total",
				CPUCapacity:        160.0,
				MemoryCapacity:     320.0,
				CPURequest:         96.0,
				MemoryRequest:      192.0,
				CPUUsagePercent:    60.0,
				MemoryUsagePercent: 60.0,
				PhysicalNodes:      4,
				VirtualNodes:       6,
				BMCount:            4,
				VMCount:            6,
				TotalCPU:           160.0,
				TotalMemory:        320.0,
				RequestedCPU:       96.0,
				RequestedMemory:    192.0,
			},
		},
	}

	// 检查是否有高负载集群
	hasHighUsageClusters := false
	for _, cluster := range []ClusterResourceSummary{cluster1, cluster2, cluster3} {
		// 根据新模板，使用 70% 作为高负载的阈值
		if cluster.CPUUsagePercent >= 70.0 || cluster.MemoryUsagePercent >= 70.0 {
			hasHighUsageClusters = true
			break
		}
	}

	return ReportTemplateData{
		ReportDate:           reportDate,
		Clusters:             []ClusterResourceSummary{cluster1, cluster2, cluster3},
		HasHighUsageClusters: hasHighUsageClusters,
	}
}

// GeneratePreview 生成预览文件
func generatePreview() error {
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
	data := createMockData(reportDate)

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

	// 将结果保存到文件
	outputPath := filepath.Join(outputDir, "preview.html")
	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("保存预览文件失败: %v", err)
	}

	// 获取当前目录的绝对路径
	absPath, _ := filepath.Abs(outputPath)
	fmt.Printf("预览HTML文件已生成: %s\n", absPath)

	// 生成Excel预览文件
	excelPath, err := generateExcelReport(data, reportDate)
	if err != nil {
		return fmt.Errorf("生成Excel预览文件失败: %v", err)
	}

	// 获取Excel文件的绝对路径
	excelAbsPath, _ := filepath.Abs(excelPath)
	fmt.Printf("预览Excel文件已生成: %s\n", excelAbsPath)

	return nil
}
