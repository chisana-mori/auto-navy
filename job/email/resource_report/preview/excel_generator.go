package main

import (
	"fmt"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

// ExcelReportGenerator handles Excel report generation
type ExcelReportGenerator struct {
	data     ReportTemplateData
	date     string
	workbook *excelize.File
	styles   map[string]int
}

// NewExcelReportGenerator creates a new Excel report generator
func NewExcelReportGenerator(data ReportTemplateData, date string) *ExcelReportGenerator {
	return &ExcelReportGenerator{
		data:   data,
		date:   date,
		styles: make(map[string]int),
	}
}

// generateExcelReport 生成Excel报表预览
func generateExcelReport(data ReportTemplateData, date string) (string, error) {
	generator := NewExcelReportGenerator(data, date)
	return generator.Generate()
}

// Generate creates the Excel report
func (g *ExcelReportGenerator) Generate() (string, error) {
	// Initialize workbook
	g.workbook = excelize.NewFile()

	// Set up sheets
	g.initializeSheets()

	// Create styles
	if err := g.createStyles(); err != nil {
		return "", err
	}

	// Generate content for each sheet
	if err := g.generateOverviewSheet(); err != nil {
		return "", err
	}

	if err := g.generateClustersSheet(); err != nil {
		return "", err
	}

	if err := g.generateResourcePoolsSheet(); err != nil {
		return "", err
	}

	// Set active sheet
	sheetIndex, _ := g.workbook.GetSheetIndex("信息概览")
	g.workbook.SetActiveSheet(sheetIndex)

	// Save file
	fileName := fmt.Sprintf("k8s_resource_report_%s.xlsx", g.date)
	filePath := filepath.Join(".", fileName)

	if err := g.workbook.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("failed to save Excel file: %w", err)
	}

	return filePath, nil
}

// initializeSheets sets up the three worksheet tabs
func (g *ExcelReportGenerator) initializeSheets() {
	// Create three sheets: information overview, cluster overview (wide table), and details (one row per resource pool)
	g.workbook.SetSheetName("Sheet1", "信息概览")
	g.workbook.NewSheet("集群概览")
	g.workbook.NewSheet("资源池详情")
}

// createStyles creates all the styles needed for the report
func (g *ExcelReportGenerator) createStyles() error {
	if err := g.createBasicStyles(); err != nil {
		return err
	}

	if err := g.createUsageStyles(); err != nil {
		return err
	}

	return nil
}

// createBasicStyles creates title, label, and value styles
func (g *ExcelReportGenerator) createBasicStyles() error {
	// Title style (bold, centered)
	titleStyle, err := g.workbook.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  14,
			Color: "#000000",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create title style: %w", err)
	}
	g.styles["title"] = titleStyle

	// Label style (bold, left-aligned)
	labelStyle, err := g.workbook.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  12,
			Color: "#000000",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create label style: %w", err)
	}
	g.styles["label"] = labelStyle

	// Value style (normal, left-aligned)
	valueStyle, err := g.workbook.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:  12,
			Color: "#000000",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create value style: %w", err)
	}
	g.styles["value"] = valueStyle

	// Abnormal value style (bold, red)
	abnormalStyle, err := g.workbook.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  12,
			Color: "#FF0000",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create abnormal style: %w", err)
	}
	g.styles["abnormal"] = abnormalStyle

	// Header style for tables (white text on blue background)
	headerStyle, err := g.workbook.NewStyle(&excelize.Style{
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
		return fmt.Errorf("failed to create header style: %w", err)
	}
	g.styles["header"] = headerStyle

	// Data style (bordered cells)
	dataStyle, err := g.workbook.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create data style: %w", err)
	}
	g.styles["data"] = dataStyle

	return nil
}

// createUsageStyles creates styles for different usage levels
func (g *ExcelReportGenerator) createUsageStyles() error {
	// Normal style (55-70%, green)
	normalStyle, err := g.workbook.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "#006100",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#C6EFCE"},
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
		return fmt.Errorf("failed to create normal style: %w", err)
	}
	g.styles["normal"] = normalStyle

	// Warning style (70-75%, yellow)
	warningStyle, err := g.workbook.NewStyle(&excelize.Style{
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
		return fmt.Errorf("failed to create warning style: %w", err)
	}
	g.styles["warning"] = warningStyle

	// Danger style (75-90%, orange)
	dangerStyle, err := g.workbook.NewStyle(&excelize.Style{
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
		return fmt.Errorf("failed to create danger style: %w", err)
	}
	g.styles["danger"] = dangerStyle

	// Critical style (>90%, red)
	criticalStyle, err := g.workbook.NewStyle(&excelize.Style{
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
		return fmt.Errorf("failed to create critical style: %w", err)
	}
	g.styles["critical"] = criticalStyle

	// Low usage style (<55%, blue)
	lowUsageStyle, err := g.workbook.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "#0070C0",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#DDEBF7"},
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
		return fmt.Errorf("failed to create low usage style: %w", err)
	}
	g.styles["lowUsage"] = lowUsageStyle

	return nil
}

// generateOverviewSheet creates the overview information sheet
func (g *ExcelReportGenerator) generateOverviewSheet() error {
	// Set up overview sheet
	sheet := "信息概览"

	// Add title
	g.workbook.SetCellValue(sheet, "A1", fmt.Sprintf("Kubernetes 集群资源报告 - %s", g.date))
	g.workbook.MergeCell(sheet, "A1", "C1")
	g.workbook.SetCellStyle(sheet, "A1", "C1", g.styles["title"])
	g.workbook.SetRowHeight(sheet, 1, 30)

	// Add cluster statistics section
	g.addClusterStatsSection(sheet)

	// Add general cluster information section
	g.addGeneralClusterSection(sheet)

	// Adjust column widths
	g.workbook.SetColWidth(sheet, "A", "A", 20)
	g.workbook.SetColWidth(sheet, "B", "B", 30)
	g.workbook.SetColWidth(sheet, "C", "C", 20)

	return nil
}

// addClusterStatsSection adds the cluster statistics to the overview sheet
func (g *ExcelReportGenerator) addClusterStatsSection(sheet string) {
	// Section title
	g.workbook.SetCellValue(sheet, "A3", "集群统计信息")
	g.workbook.MergeCell(sheet, "A3", "C3")
	g.workbook.SetCellStyle(sheet, "A3", "C3", g.styles["label"])

	// Total clusters
	g.workbook.SetCellValue(sheet, "A4", "总已巡检集群数：")
	g.workbook.SetCellValue(sheet, "B4", g.data.Stats.TotalClusters)
	g.workbook.SetCellStyle(sheet, "A4", "A4", g.styles["label"])
	g.workbook.SetCellStyle(sheet, "B4", "B4", g.styles["value"])

	// Normal clusters
	g.workbook.SetCellValue(sheet, "A5", "正常集群数：")
	g.workbook.SetCellValue(sheet, "B5", g.data.Stats.NormalClusters)
	g.workbook.SetCellStyle(sheet, "A5", "A5", g.styles["label"])
	g.workbook.SetCellStyle(sheet, "B5", "B5", g.styles["value"])

	// Abnormal clusters
	g.workbook.SetCellValue(sheet, "A6", "异常集群数：")
	g.workbook.SetCellValue(sheet, "B6", g.data.Stats.AbnormalClusters)
	g.workbook.SetCellStyle(sheet, "A6", "A6", g.styles["label"])

	// Apply style based on abnormal count
	if g.data.Stats.AbnormalClusters > 0 {
		g.workbook.SetCellStyle(sheet, "B6", "B6", g.styles["abnormal"])
	} else {
		g.workbook.SetCellStyle(sheet, "B6", "B6", g.styles["value"])
	}
}

// addGeneralClusterSection adds the general cluster info to the overview sheet
func (g *ExcelReportGenerator) addGeneralClusterSection(sheet string) {
	// Section title
	g.workbook.SetCellValue(sheet, "A8", "通用集群信息")
	g.workbook.MergeCell(sheet, "A8", "C8")
	g.workbook.SetCellStyle(sheet, "A8", "C8", g.styles["label"])

	// Pod density
	g.workbook.SetCellValue(sheet, "A9", "通用集群Pod密度：")
	g.workbook.SetCellValue(sheet, "B9", fmt.Sprintf("%.2f", g.data.Stats.GeneralPodDensity))
	g.workbook.SetCellStyle(sheet, "A9", "A9", g.styles["label"])
	g.workbook.SetCellStyle(sheet, "B9", "B9", g.styles["value"])

	// Add explanation for Pod density
	g.workbook.SetCellValue(sheet, "A10", "Pod密度说明：")
	g.workbook.SetCellValue(sheet, "B10", "Pod密度 = Pod总数 / 物理机节点数，表示平均每个物理机节点上运行的Pod数量")
	g.workbook.MergeCell(sheet, "B10", "C10")
	g.workbook.SetCellStyle(sheet, "A10", "A10", g.styles["label"])
	g.workbook.SetCellStyle(sheet, "B10", "C10", g.styles["value"])
}

// generateClustersSheet creates the clusters overview sheet with wide table format
func (g *ExcelReportGenerator) generateClustersSheet() error {
	sheet := "集群概览"

	// Set up column headers
	g.setupClusterSheetHeaders(sheet)

	// Add data for each cluster
	row := 2
	for _, cluster := range g.data.Clusters {
		// Find pools for the cluster
		pools := g.initializeClusterPools(cluster)

		// Write cluster data
		g.writeClusterRow(sheet, row, cluster, pools)

		row++
	}

	return nil
}

// setupClusterSheetHeaders adds all column headers to the clusters sheet
func (g *ExcelReportGenerator) setupClusterSheetHeaders(sheet string) {
	// Define column headers
	columnHeaders := []string{
		"集群",
		"总通用资源-CPU容量(核)",
		"总通用资源-CPU请求(核)",
		"总通用资源-CPU分配率(%)",
		"总通用资源-CPU最大使用率(%)",
		"总通用资源-内存容量(GiB)",
		"总通用资源-内存请求(GiB)",
		"总通用资源-内存分配率(%)",
		"总通用资源-内存最大使用率(%)",
		"总通用资源-Pod数量",
		"总通用资源-Pod密度",
		"总通用资源-节点平均CPU(核)",
		"总通用资源-节点平均内存(GiB)",
	}

	// Set column headers and calculate widths
	for i, header := range columnHeaders {
		colName := getColumnName(i)
		cellID := colName + "1"
		g.workbook.SetCellValue(sheet, cellID, header)

		// Set column width based on header text
		width := calculateColumnWidth(header)
		g.workbook.SetColWidth(sheet, colName, colName, width)
	}

	// Intel resource pool headers
	intelHeaders := []string{
		"Intel通用-CPU容量(核)",
		"Intel通用-CPU请求(核)",
		"Intel通用-CPU分配率(%)",
		"Intel通用-CPU最大使用率(%)",
		"Intel通用-内存容量(GiB)",
		"Intel通用-内存请求(GiB)",
		"Intel通用-内存分配率(%)",
		"Intel通用-内存最大使用率(%)",
		"Intel通用-Pod数量",
		"Intel通用-Pod密度",
		"Intel通用-节点平均CPU(核)",
		"Intel通用-节点平均内存(GiB)",
	}

	// Set Intel headers starting from column N (index 13)
	for i, header := range intelHeaders {
		colName := getColumnName(i + 13)
		cellID := colName + "1"
		g.workbook.SetCellValue(sheet, cellID, header)

		// Set column width based on header text
		width := calculateColumnWidth(header)
		g.workbook.SetColWidth(sheet, colName, colName, width)
	}

	// HG resource pool headers
	hgHeaders := []string{
		"海光通用-CPU容量(核)",
		"海光通用-CPU请求(核)",
		"海光通用-CPU分配率(%)",
		"海光通用-CPU最大使用率(%)",
		"海光通用-内存容量(GiB)",
		"海光通用-内存请求(GiB)",
		"海光通用-内存分配率(%)",
		"海光通用-内存最大使用率(%)",
		"海光通用-Pod数量",
		"海光通用-Pod密度",
		"海光通用-节点平均CPU(核)",
		"海光通用-节点平均内存(GiB)",
	}

	// Set HG headers starting from column Z (index 25)
	for i, header := range hgHeaders {
		colName := getColumnName(i + 25)
		cellID := colName + "1"
		g.workbook.SetCellValue(sheet, cellID, header)

		// Set column width based on header text
		width := calculateColumnWidth(header)
		g.workbook.SetColWidth(sheet, colName, colName, width)
	}

	// ARM resource pool headers
	armHeaders := []string{
		"ARM通用-CPU容量(核)",
		"ARM通用-CPU请求(核)",
		"ARM通用-CPU分配率(%)",
		"ARM通用-CPU最大使用率(%)",
		"ARM通用-内存容量(GiB)",
		"ARM通用-内存请求(GiB)",
		"ARM通用-内存分配率(%)",
		"ARM通用-内存最大使用率(%)",
		"ARM通用-Pod数量",
		"ARM通用-Pod密度",
		"ARM通用-节点平均CPU(核)",
		"ARM通用-节点平均内存(GiB)",
	}

	// Set ARM headers starting from column AL (index 37)
	for i, header := range armHeaders {
		colName := getColumnName(i + 37)
		cellID := colName + "1"
		g.workbook.SetCellValue(sheet, cellID, header)

		// Set column width based on header text
		width := calculateColumnWidth(header)
		g.workbook.SetColWidth(sheet, colName, colName, width)
	}

	// Apply header style to all header cells
	// Calculate the total number of columns we need to style
	totalColumns := len(columnHeaders) + len(intelHeaders) + len(hgHeaders) + len(armHeaders)
	for i := 0; i < totalColumns; i++ {
		cell := getColumnName(i) + "1"
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["header"])
	}
}

// initializeClusterPools finds the four main resource pools for a cluster
func (g *ExcelReportGenerator) initializeClusterPools(cluster ClusterResourceSummary) map[string]ResourcePool {
	// Initialize empty pools
	pools := map[string]ResourcePool{
		"total":        {},
		"intel_common": {},
		"hg_common":    {},
		"arm_common":   {},
	}

	// Find actual pools from cluster data
	for _, pool := range cluster.ResourcePools {
		switch pool.ResourceType {
		case "total", "total_common":
			pools["total"] = pool
		case "intel_common":
			pools["intel_common"] = pool
		case "hg_common":
			pools["hg_common"] = pool
		case "arm_common":
			pools["arm_common"] = pool
		}
	}

	return pools
}

// writeClusterRow writes a row of data for a cluster
func (g *ExcelReportGenerator) writeClusterRow(sheet string, row int, cluster ClusterResourceSummary, pools map[string]ResourcePool) {
	totalPool := pools["total"]
	intelPool := pools["intel_common"]
	hgPool := pools["hg_common"]
	armPool := pools["arm_common"]

	// Apply base data style to entire row
	// Calculate the total number of columns we need to style
	totalColumns := 13 + 12 + 12 + 12 // Total + Intel + HG + ARM columns
	for i := 0; i < totalColumns; i++ {
		cell := getColumnName(i) + fmt.Sprintf("%d", row)
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["data"])
	}

	// Write basic cluster info
	g.workbook.SetCellValue(sheet, getColumnName(0)+fmt.Sprintf("%d", row), cluster.ClusterName)

	// Write total pool data
	g.writePoolData(sheet, row, getColumnName(1), totalPool)

	// Write Intel pool data
	g.writePoolData(sheet, row, getColumnName(13), intelPool)

	// Write HG pool data
	g.writePoolData(sheet, row, getColumnName(25), hgPool)

	// Write ARM pool data
	g.writePoolData(sheet, row, getColumnName(37), armPool)

	// Apply usage-based styles
	g.applyPoolStyles(sheet, row, totalPool, intelPool, hgPool, armPool)
}

// writePoolData writes data for a resource pool starting at the specified column
func (g *ExcelReportGenerator) writePoolData(sheet string, row int, startCol string, pool ResourcePool) {
	// Get the starting column index
	var startColIndex int
	if len(startCol) == 1 {
		startColIndex = int(startCol[0] - 'A')
	} else if len(startCol) == 2 {
		startColIndex = 26 + int(startCol[1]-'A')
	}

	// CPU metrics
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex)+fmt.Sprintf("%d", row), pool.CPUCapacity)
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex+1)+fmt.Sprintf("%d", row), pool.CPURequest)
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex+2)+fmt.Sprintf("%d", row), fmt.Sprintf("%.1f%%", pool.CPUUsagePercent))
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex+3)+fmt.Sprintf("%d", row), fmt.Sprintf("%.1f%%", pool.MaxCpuUsageRatio*100))

	// Memory metrics
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex+4)+fmt.Sprintf("%d", row), pool.MemoryCapacity)
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex+5)+fmt.Sprintf("%d", row), pool.MemoryRequest)
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex+6)+fmt.Sprintf("%d", row), fmt.Sprintf("%.1f%%", pool.MemoryUsagePercent))
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex+7)+fmt.Sprintf("%d", row), fmt.Sprintf("%.1f%%", pool.MaxMemoryUsageRatio*100))

	// Other metrics
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex+8)+fmt.Sprintf("%d", row), pool.PodCount)

	// Calculate pod density
	var podDensity float64
	if pool.BMCount > 0 {
		podDensity = float64(pool.PodCount) / float64(pool.BMCount)
	}
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex+9)+fmt.Sprintf("%d", row), fmt.Sprintf("%.2f", podDensity))

	// Node average metrics
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex+10)+fmt.Sprintf("%d", row), pool.PerNodeCpuRequest)
	g.workbook.SetCellValue(sheet, getColumnName(startColIndex+11)+fmt.Sprintf("%d", row), pool.PerNodeMemRequest)
}

// applyPoolStyles applies usage-based styles to resource usage cells
func (g *ExcelReportGenerator) applyPoolStyles(sheet string, row int, totalPool, intelPool, hgPool, armPool ResourcePool) {
	// Define column indices for each pool's CPU and memory usage cells
	totalPoolIndices := map[string]int{
		"cpuUsage":    3, // D
		"cpuMaxUsage": 4, // E
		"memUsage":    8, // I
		"memMaxUsage": 9, // J
	}

	intelPoolIndices := map[string]int{
		"cpuUsage":    15, // P
		"cpuMaxUsage": 16, // Q
		"memUsage":    19, // T
		"memMaxUsage": 20, // U
	}

	hgPoolIndices := map[string]int{
		"cpuUsage":    27, // AB
		"cpuMaxUsage": 28, // AC
		"memUsage":    31, // AF
		"memMaxUsage": 32, // AG
	}

	armPoolIndices := map[string]int{
		"cpuUsage":    39, // AN
		"cpuMaxUsage": 40, // AO
		"memUsage":    43, // AR
		"memMaxUsage": 44, // AS
	}

	// Apply styles to total pool
	if totalPool.Nodes > 0 {
		g.applyCPUUsageStyle(sheet, getColumnName(totalPoolIndices["cpuUsage"])+fmt.Sprintf("%d", row), totalPool.CPUUsagePercent)
		g.applyCPUUsageStyle(sheet, getColumnName(totalPoolIndices["cpuMaxUsage"])+fmt.Sprintf("%d", row), totalPool.MaxCpuUsageRatio*100)
		g.applyMemUsageStyle(sheet, getColumnName(totalPoolIndices["memUsage"])+fmt.Sprintf("%d", row), totalPool.MemoryUsagePercent)
		g.applyMemUsageStyle(sheet, getColumnName(totalPoolIndices["memMaxUsage"])+fmt.Sprintf("%d", row), totalPool.MaxMemoryUsageRatio*100)
	}

	// Apply styles to Intel pool
	if intelPool.Nodes > 0 {
		g.applyCPUUsageStyle(sheet, getColumnName(intelPoolIndices["cpuUsage"])+fmt.Sprintf("%d", row), intelPool.CPUUsagePercent)
		g.applyCPUUsageStyle(sheet, getColumnName(intelPoolIndices["cpuMaxUsage"])+fmt.Sprintf("%d", row), intelPool.MaxCpuUsageRatio*100)
		g.applyMemUsageStyle(sheet, getColumnName(intelPoolIndices["memUsage"])+fmt.Sprintf("%d", row), intelPool.MemoryUsagePercent)
		g.applyMemUsageStyle(sheet, getColumnName(intelPoolIndices["memMaxUsage"])+fmt.Sprintf("%d", row), intelPool.MaxMemoryUsageRatio*100)
	}

	// Apply styles to HG pool
	if hgPool.Nodes > 0 {
		g.applyCPUUsageStyle(sheet, getColumnName(hgPoolIndices["cpuUsage"])+fmt.Sprintf("%d", row), hgPool.CPUUsagePercent)
		g.applyCPUUsageStyle(sheet, getColumnName(hgPoolIndices["cpuMaxUsage"])+fmt.Sprintf("%d", row), hgPool.MaxCpuUsageRatio*100)
		g.applyMemUsageStyle(sheet, getColumnName(hgPoolIndices["memUsage"])+fmt.Sprintf("%d", row), hgPool.MemoryUsagePercent)
		g.applyMemUsageStyle(sheet, getColumnName(hgPoolIndices["memMaxUsage"])+fmt.Sprintf("%d", row), hgPool.MaxMemoryUsageRatio*100)
	}

	// Apply styles to ARM pool
	if armPool.Nodes > 0 {
		g.applyCPUUsageStyle(sheet, getColumnName(armPoolIndices["cpuUsage"])+fmt.Sprintf("%d", row), armPool.CPUUsagePercent)
		g.applyCPUUsageStyle(sheet, getColumnName(armPoolIndices["cpuMaxUsage"])+fmt.Sprintf("%d", row), armPool.MaxCpuUsageRatio*100)
		g.applyMemUsageStyle(sheet, getColumnName(armPoolIndices["memUsage"])+fmt.Sprintf("%d", row), armPool.MemoryUsagePercent)
		g.applyMemUsageStyle(sheet, getColumnName(armPoolIndices["memMaxUsage"])+fmt.Sprintf("%d", row), armPool.MaxMemoryUsageRatio*100)
	}
}

// applyCPUUsageStyle applies appropriate style based on CPU usage percentage
func (g *ExcelReportGenerator) applyCPUUsageStyle(sheet, cell string, cpuUsage float64) {
	if cpuUsage >= 90.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["critical"])
	} else if cpuUsage >= 75.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["danger"])
	} else if cpuUsage >= 70.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["warning"])
	} else if cpuUsage >= 55.0 && cpuUsage < 70.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["normal"])
	} else if cpuUsage < 55.0 && cpuUsage > 0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["lowUsage"])
	}
}

// applyMemUsageStyle applies appropriate style based on memory usage percentage
func (g *ExcelReportGenerator) applyMemUsageStyle(sheet, cell string, memUsage float64) {
	if memUsage >= 90.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["critical"])
	} else if memUsage >= 75.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["danger"])
	} else if memUsage >= 70.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["warning"])
	} else if memUsage >= 55.0 && memUsage < 70.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["normal"])
	} else if memUsage < 55.0 && memUsage > 0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["lowUsage"])
	}
}

// generateResourcePoolsSheet creates the resource pools detail sheet
func (g *ExcelReportGenerator) generateResourcePoolsSheet() error {
	sheet := "资源池详情"

	// Set up headers
	g.setupResourcePoolHeaders(sheet)

	// Add data for each cluster's resource pools
	row := 2
	for _, cluster := range g.data.Clusters {
		for _, pool := range cluster.ResourcePools {
			// Apply base data style to entire row
			for i := 0; i < 10; i++ { // 10 columns (A-J)
				cell := getColumnName(i) + fmt.Sprintf("%d", row)
				g.workbook.SetCellStyle(sheet, cell, cell, g.styles["data"])
			}

			g.writeResourcePoolRow(sheet, row, cluster, pool)
			row++
		}
	}

	return nil
}

// setupResourcePoolHeaders adds headers to the resource pools sheet
func (g *ExcelReportGenerator) setupResourcePoolHeaders(sheet string) {
	// Define column headers
	columnHeaders := []string{
		"集群",         // A - Cluster name
		"资源池",        // B - Resource type
		"CPU容量",      // C - CPU capacity
		"CPU分配率",     // D - CPU allocation rate
		"平均CPU最大使用率", // E - Average CPU max usage
		"内存容量",       // F - Memory capacity
		"内存分配率",      // G - Memory allocation rate
		"平均内存最大使用率",  // H - Average memory max usage
		"Pod数量",      // I - Pod count
		"Pod密度",      // J - Pod density
	}

	// Set column headers and calculate widths
	for i, header := range columnHeaders {
		// Get column name
		colName := getColumnName(i)

		// Set header value
		g.workbook.SetCellValue(sheet, colName+"1", header)

		// Calculate and set column width based on header text
		width := calculateColumnWidth(header)
		g.workbook.SetColWidth(sheet, colName, colName, width)

		// Apply header style
		g.workbook.SetCellStyle(sheet, colName+"1", colName+"1", g.styles["header"])
	}
}

// writeResourcePoolRow writes a single row of data for a resource pool
func (g *ExcelReportGenerator) writeResourcePoolRow(sheet string, row int, cluster ClusterResourceSummary, pool ResourcePool) {
	// Skip empty pools
	if pool.Nodes == 0 {
		return
	}

	// Calculate pod density
	var podDensity float64
	if pool.BMCount > 0 {
		podDensity = float64(pool.PodCount) / float64(pool.BMCount)
	}

	// Define column indices
	columnIndices := map[string]int{
		"cluster":      0, // A
		"resourceType": 1, // B
		"cpuCapacity":  2, // C
		"cpuUsage":     3, // D
		"cpuMaxUsage":  4, // E
		"memCapacity":  5, // F
		"memUsage":     6, // G
		"memMaxUsage":  7, // H
		"podCount":     8, // I
		"podDensity":   9, // J
	}

	// Basic info
	g.workbook.SetCellValue(sheet, getColumnName(columnIndices["cluster"])+fmt.Sprintf("%d", row), cluster.ClusterName)
	g.workbook.SetCellValue(sheet, getColumnName(columnIndices["resourceType"])+fmt.Sprintf("%d", row), pool.ResourceType)

	// CPU metrics
	g.workbook.SetCellValue(sheet, getColumnName(columnIndices["cpuCapacity"])+fmt.Sprintf("%d", row), fmt.Sprintf("%.1f", pool.CPUCapacity))
	g.workbook.SetCellValue(sheet, getColumnName(columnIndices["cpuUsage"])+fmt.Sprintf("%d", row), fmt.Sprintf("%.1f%%", pool.CPUUsagePercent))
	g.workbook.SetCellValue(sheet, getColumnName(columnIndices["cpuMaxUsage"])+fmt.Sprintf("%d", row), fmt.Sprintf("%.1f%%", pool.MaxCpuUsageRatio*100))

	// Memory metrics
	g.workbook.SetCellValue(sheet, getColumnName(columnIndices["memCapacity"])+fmt.Sprintf("%d", row), fmt.Sprintf("%.1f", pool.MemoryCapacity))
	g.workbook.SetCellValue(sheet, getColumnName(columnIndices["memUsage"])+fmt.Sprintf("%d", row), fmt.Sprintf("%.1f%%", pool.MemoryUsagePercent))
	g.workbook.SetCellValue(sheet, getColumnName(columnIndices["memMaxUsage"])+fmt.Sprintf("%d", row), fmt.Sprintf("%.1f%%", pool.MaxMemoryUsageRatio*100))

	// Pod metrics
	g.workbook.SetCellValue(sheet, getColumnName(columnIndices["podCount"])+fmt.Sprintf("%d", row), pool.PodCount)
	g.workbook.SetCellValue(sheet, getColumnName(columnIndices["podDensity"])+fmt.Sprintf("%d", row), fmt.Sprintf("%.2f", podDensity))

	// Apply styles
	g.applyResourcePoolCPUStyle(sheet, getColumnName(columnIndices["cpuUsage"])+fmt.Sprintf("%d", row), pool.CPUUsagePercent)
	g.applyResourcePoolCPUStyle(sheet, getColumnName(columnIndices["cpuMaxUsage"])+fmt.Sprintf("%d", row), pool.MaxCpuUsageRatio*100)
	g.applyResourcePoolMemStyle(sheet, getColumnName(columnIndices["memUsage"])+fmt.Sprintf("%d", row), pool.MemoryUsagePercent)
	g.applyResourcePoolMemStyle(sheet, getColumnName(columnIndices["memMaxUsage"])+fmt.Sprintf("%d", row), pool.MaxMemoryUsageRatio*100)
}

// applyResourcePoolCPUStyle applies style to CPU usage cells based on usage percentage
func (g *ExcelReportGenerator) applyResourcePoolCPUStyle(sheet, cell string, cpuUsage float64) {
	if cpuUsage >= 90.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["critical"])
	} else if cpuUsage >= 75.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["danger"])
	} else if cpuUsage >= 70.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["warning"])
	} else if cpuUsage >= 55.0 && cpuUsage < 70.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["normal"])
	} else if cpuUsage < 55.0 && cpuUsage > 0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["lowUsage"])
	} else {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["data"])
	}
}

// applyResourcePoolMemStyle applies style to memory usage cells based on usage percentage
func (g *ExcelReportGenerator) applyResourcePoolMemStyle(sheet, cell string, memUsage float64) {
	if memUsage >= 90.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["critical"])
	} else if memUsage >= 75.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["danger"])
	} else if memUsage >= 70.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["warning"])
	} else if memUsage >= 55.0 && memUsage < 70.0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["normal"])
	} else if memUsage < 55.0 && memUsage > 0 {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["lowUsage"])
	} else {
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["data"])
	}
}

// getColumnName converts a 0-based index to Excel column name (A, B, C, ..., Z, AA, AB, ...)
func getColumnName(index int) string {
	result := ""
	for index >= 0 {
		remainder := index % 26
		result = string('A'+remainder) + result
		index = index/26 - 1
	}
	return result
}

// calculateColumnWidth calculates the appropriate width for a column based on its content
func calculateColumnWidth(text string) float64 {
	// Base width for a character (in Excel units)
	baseWidth := 1.2

	// Calculate width based on text length
	// Chinese characters need more width
	width := 0.0
	for _, r := range text {
		if r < 128 { // ASCII character
			width += baseWidth
		} else { // Chinese or other wide character
			width += baseWidth * 2
		}
	}

	// Add some padding
	width += 2

	// Ensure minimum width
	if width < 8 {
		width = 8
	}

	// Cap maximum width
	if width > 50 {
		width = 50
	}

	return width
}

// The rest of the functions will be added in subsequent edits
