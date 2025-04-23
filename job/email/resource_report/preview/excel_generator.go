package main

import (
	"fmt"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

// generateExcelReport 生成Excel报表预览
func generateExcelReport(data ReportTemplateData, date string) (string, error) {
	f := excelize.NewFile()

	// 创建两个sheet：概览(宽表)和详情(每资源池一行)
	f.SetSheetName("Sheet1", "集群概览")
	f.NewSheet("资源池详情")

	// -----------------------
	// 第一个sheet：集群概览(宽表格式)
	// -----------------------

	// 设置标题行样式
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  12,
			Color: "#FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create header style: %w", err)
	}

	// 设置数据行样式
	dataStyle, err := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create data style: %w", err)
	}

	// 设置警告样式 (70-75%)
	warningStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "#9C5700",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFEB9C"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create warning style: %w", err)
	}

	// 设置危险样式 (75-90%)
	dangerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "#974706",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#F8CBAD"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create danger style: %w", err)
	}

	// 设置紧急样式 (>90%)
	criticalStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "#9C0006",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFC7CE"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create critical style: %w", err)
	}

	// ---- 集群概览sheet (宽表) ----

	// 设置集群概览标题行
	overviewHeaders := []string{
		"集群", "总节点数", "物理节点", "虚拟节点",
		"总CPU容量(核)", "总CPU请求(核)", "总CPU使用率(%)",
		"总内存容量(GiB)", "总内存请求(GiB)", "总内存使用率(%)",
	}

	for i, header := range overviewHeaders {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue("集群概览", cell, header)
		f.SetCellStyle("集群概览", cell, cell, headerStyle)
	}

	// 写入集群概览数据
	row := 2
	for _, cluster := range data.Clusters {
		// 计算总体使用率
		cpuUsage := cluster.CPUUsagePercent
		memUsage := cluster.MemoryUsagePercent

		// 写入一行数据
		f.SetCellValue("集群概览", fmt.Sprintf("A%d", row), cluster.ClusterName)
		f.SetCellValue("集群概览", fmt.Sprintf("B%d", row), cluster.TotalNodes)
		f.SetCellValue("集群概览", fmt.Sprintf("C%d", row), cluster.PhysicalNodes)
		f.SetCellValue("集群概览", fmt.Sprintf("D%d", row), cluster.VirtualNodes)
		f.SetCellValue("集群概览", fmt.Sprintf("E%d", row), cluster.TotalCPUCapacity)
		f.SetCellValue("集群概览", fmt.Sprintf("F%d", row), cluster.RequestedCPU)
		f.SetCellValue("集群概览", fmt.Sprintf("G%d", row), fmt.Sprintf("%.1f%%", cpuUsage))
		f.SetCellValue("集群概览", fmt.Sprintf("H%d", row), cluster.TotalMemoryCapacity)
		f.SetCellValue("集群概览", fmt.Sprintf("I%d", row), cluster.RequestedMemory)
		f.SetCellValue("集群概览", fmt.Sprintf("J%d", row), fmt.Sprintf("%.1f%%", memUsage))

		// 为整行应用基本样式
		f.SetCellStyle("集群概览", fmt.Sprintf("A%d", row), fmt.Sprintf("J%d", row), dataStyle)

		// 根据使用率应用不同的样式
		// CPU使用率
		if cpuUsage >= 90.0 {
			f.SetCellStyle("集群概览", fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), criticalStyle)
		} else if cpuUsage >= 75.0 {
			f.SetCellStyle("集群概览", fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), dangerStyle)
		} else if cpuUsage >= 70.0 {
			f.SetCellStyle("集群概览", fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), warningStyle)
		}

		// 内存使用率
		if memUsage >= 90.0 {
			f.SetCellStyle("集群概览", fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), criticalStyle)
		} else if memUsage >= 75.0 {
			f.SetCellStyle("集群概览", fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), dangerStyle)
		} else if memUsage >= 70.0 {
			f.SetCellStyle("集群概览", fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), warningStyle)
		}

		row++
	}

	// 调整集群概览列宽
	f.SetColWidth("集群概览", "A", "A", 20)
	f.SetColWidth("集群概览", "B", "D", 10)
	f.SetColWidth("集群概览", "E", "J", 15)

	// -----------------------
	// 第二个sheet：资源池详情
	// -----------------------

	// 设置资源池详情标题行
	poolHeaders := []string{
		"集群", "资源池类型", "节点数", "物理节点", "虚拟节点",
		"CPU容量(核)", "CPU请求(核)", "CPU使用率(%)",
		"内存容量(GiB)", "内存请求(GiB)", "内存使用率(%)",
	}

	for i, header := range poolHeaders {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue("资源池详情", cell, header)
		f.SetCellStyle("资源池详情", cell, cell, headerStyle)
	}

	// 写入资源池详情数据
	row = 2
	for _, cluster := range data.Clusters {
		for _, pool := range cluster.ResourcePools {
			// 计算使用率
			cpuUsage := pool.CPUUsagePercent
			memUsage := pool.MemoryUsagePercent

			// 写入一行数据
			f.SetCellValue("资源池详情", fmt.Sprintf("A%d", row), cluster.ClusterName)
			f.SetCellValue("资源池详情", fmt.Sprintf("B%d", row), pool.NodeType)
			f.SetCellValue("资源池详情", fmt.Sprintf("C%d", row), pool.Nodes)
			f.SetCellValue("资源池详情", fmt.Sprintf("D%d", row), pool.PhysicalNodes)
			f.SetCellValue("资源池详情", fmt.Sprintf("E%d", row), pool.VirtualNodes)
			f.SetCellValue("资源池详情", fmt.Sprintf("F%d", row), pool.CPUCapacity)
			f.SetCellValue("资源池详情", fmt.Sprintf("G%d", row), pool.CPURequest)
			f.SetCellValue("资源池详情", fmt.Sprintf("H%d", row), fmt.Sprintf("%.1f%%", cpuUsage))
			f.SetCellValue("资源池详情", fmt.Sprintf("I%d", row), pool.MemoryCapacity)
			f.SetCellValue("资源池详情", fmt.Sprintf("J%d", row), pool.MemoryRequest)
			f.SetCellValue("资源池详情", fmt.Sprintf("K%d", row), fmt.Sprintf("%.1f%%", memUsage))

			// 为整行应用基本样式
			f.SetCellStyle("资源池详情", fmt.Sprintf("A%d", row), fmt.Sprintf("K%d", row), dataStyle)

			// 根据使用率应用不同的样式
			// CPU使用率
			if cpuUsage >= 90.0 {
				f.SetCellStyle("资源池详情", fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), criticalStyle)
			} else if cpuUsage >= 75.0 {
				f.SetCellStyle("资源池详情", fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), dangerStyle)
			} else if cpuUsage >= 70.0 {
				f.SetCellStyle("资源池详情", fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), warningStyle)
			}

			// 内存使用率
			if memUsage >= 90.0 {
				f.SetCellStyle("资源池详情", fmt.Sprintf("K%d", row), fmt.Sprintf("K%d", row), criticalStyle)
			} else if memUsage >= 75.0 {
				f.SetCellStyle("资源池详情", fmt.Sprintf("K%d", row), fmt.Sprintf("K%d", row), dangerStyle)
			} else if memUsage >= 70.0 {
				f.SetCellStyle("资源池详情", fmt.Sprintf("K%d", row), fmt.Sprintf("K%d", row), warningStyle)
			}

			row++
		}
	}

	// 调整资源池详情列宽
	f.SetColWidth("资源池详情", "A", "A", 20)
	f.SetColWidth("资源池详情", "B", "B", 15)
	f.SetColWidth("资源池详情", "C", "E", 10)
	f.SetColWidth("资源池详情", "F", "K", 15)

	// 默认激活集群概览sheet
	sheetIndex, _ := f.GetSheetIndex("集群概览")
	f.SetActiveSheet(sheetIndex)

	// 创建文件路径
	fileName := fmt.Sprintf("k8s_resource_report_%s.xlsx", date)
	filePath := filepath.Join(".", fileName)

	// 保存文件
	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("failed to save Excel file: %w", err)
	}

	return filePath, nil
}
