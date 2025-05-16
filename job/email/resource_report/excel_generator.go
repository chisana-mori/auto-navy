package resource_report

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Column Name Constants
const (
	// Common sheet elements
	sheetNameOverview         = "信息概览"
	sheetNameClusterOverview  = "集群概览"
	sheetNameResourcePoolDetail = "资源池详情"

	// Common column names
	colClusterName = "集群"
	colClusterDesc = "集群用途"

	// Cluster Overview Sheet - Pool Prefixes
	prefixTotalCommon = "总资源-"
	prefixIntelCommon = "Intel-"
	prefixHGCommon    = "海光-"
	prefixARMCommon   = "ARM-"

	// Cluster Overview Sheet - Pool Metric Suffixes
	suffixCPUCapacity     = "CPU容量(核)"
	suffixCPURequest      = "CPU请求(核)"
	suffixCPUAllocRate    = "CPU分配率(%)"
	suffixCPUMaxUsageRate = "CPU最大使用率(%)"
	suffixMemCapacity     = "内存容量(GiB)"
	suffixMemRequest      = "内存请求(GiB)"
	suffixMemAllocRate    = "内存分配率(%)"
	suffixMemMaxUsageRate = "内存最大使用率(%)"
	suffixBMNodeCount     = "物理机节点数"
	suffixVMNodeCount     = "虚拟机节点数"
	suffixPodCount        = "Pod数量" // Also used in Detail sheet
	suffixPodDensity      = "Pod密度" // Also used in Detail sheet
	suffixAvgNodeCPU      = "节点平均CPU(核)"
	suffixAvgNodeMem      = "节点平均内存(GiB)"

	// Resource Pool Detail Sheet Specific Column Names
	colDetailResourceType       = "资源池"
	colDetailCPUCapacity        = "CPU容量"
	colDetailCPUAllocRate       = "CPU分配率"
	colDetailAvgCPUMaxUsageRate = "平均CPU最大使用率"
	colDetailMemCapacity        = "内存容量"
	colDetailMemAllocRate       = "内存分配率"
	colDetailAvgMemMaxUsageRate = "平均内存最大使用率"
)

// Dynamically generated column definitions and indices
var (
	clusterOverviewColumns    []ColumnDefinition // Column definitions for "集群概览"
	resourcePoolDetailColumns []ColumnDefinition // Column definitions for "资源池详情"

	// Indices for styled cells in Cluster Overview, relative to start of a pool's data block (0-indexed within the 14 pool metrics)
	poolColIdxCPUAllocRate    int
	poolColIdxCPUMaxUsageRate int
	poolColIdxMemAllocRate    int
	poolColIdxMemMaxUsageRate int

	// Indices for styled cells in Resource Pool Detail, relative to start of row (0-indexed column index)
	detailColIdxCPUAllocRate       int
	detailColIdxAvgCPUMaxUsageRate int
	detailColIdxMemAllocRate       int
	detailColIdxAvgMemMaxUsageRate int
)

func init() {
	// Define suffixes for the 14 metrics per pool in Cluster Overview
	poolMetricSuffixes := []string{
		suffixCPUCapacity, suffixCPURequest, suffixCPUAllocRate, suffixCPUMaxUsageRate,
		suffixMemCapacity, suffixMemRequest, suffixMemAllocRate, suffixMemMaxUsageRate,
		suffixBMNodeCount, suffixVMNodeCount, suffixPodCount, suffixPodDensity,
		suffixAvgNodeCPU, suffixAvgNodeMem,
	}

	// Calculate indices for styled cells within a pool's data block
	for i, suffix := range poolMetricSuffixes {
		switch suffix {
		case suffixCPUAllocRate:
			poolColIdxCPUAllocRate = i
		case suffixCPUMaxUsageRate:
			poolColIdxCPUMaxUsageRate = i
		case suffixMemAllocRate:
			poolColIdxMemAllocRate = i
		case suffixMemMaxUsageRate:
			poolColIdxMemMaxUsageRate = i
		}
	}

	// Define columns for "集群概览" sheet
	clusterOverviewColumns = []ColumnDefinition{
		{Header: colClusterName, GetValue: func(data interface{}) interface{} {
			// Data is now a struct containing Cluster and AllPoolDataMaps
			rowData := data.(struct {
				Cluster         ClusterResourceSummary
				AllPoolDataMaps map[string]map[string]interface{}
			})
			return rowData.Cluster.ClusterName
		}},
		{Header: colClusterDesc, GetValue: func(data interface{}) interface{} {
			// Data is now a struct containing Cluster and AllPoolDataMaps
			rowData := data.(struct {
				Cluster         ClusterResourceSummary
				AllPoolDataMaps map[string]map[string]interface{}
			})
			return rowData.Cluster.Desc
		}},
	}

	// Define pool types and their prefixes in a fixed order
	poolTypes := []struct {
		Type   string
		Prefix string
	}{
		{Type: "total", Prefix: prefixTotalCommon},
		{Type: "intel_common", Prefix: prefixIntelCommon},
		{Type: "hg_common", Prefix: prefixHGCommon},
		{Type: "arm_common", Prefix: prefixARMCommon},
	}

	// Add pool-specific columns to "集群概览"
	for _, poolTypeInfo := range poolTypes {
		currentPoolType := poolTypeInfo.Type   // Capture loop variable
		currentPrefix := poolTypeInfo.Prefix // Capture loop variable
		for _, suffix := range poolMetricSuffixes {
			header := currentPrefix + suffix
			currentSuffix := suffix // Capture loop variable
			clusterOverviewColumns = append(clusterOverviewColumns, ColumnDefinition{
				Header: header,
				GetValue: func(data interface{}) interface{} {
					// Data is now a struct containing Cluster and AllPoolDataMaps
					rowData := data.(struct {
						Cluster         ClusterResourceSummary
						AllPoolDataMaps map[string]map[string]interface{}
					})
					// Retrieve the specific pool's data map and then the metric value
					if poolDataMap, ok := rowData.AllPoolDataMaps[currentPoolType]; ok {
						if value, ok := poolDataMap[currentSuffix]; ok {
							return value
						}
					}
					return "" // Return empty string if data not found
				},
			})
		}
	}

	// Define columns for "资源池详情" sheet
	resourcePoolDetailColumns = []ColumnDefinition{
		{Header: colClusterName, GetValue: func(data interface{}) interface{} { return data.(struct { Cluster ClusterResourceSummary; Pool ResourcePool }).Cluster.ClusterName }},
		{Header: colClusterDesc, GetValue: func(data interface{}) interface{} { return data.(struct { Cluster ClusterResourceSummary; Pool ResourcePool }).Cluster.Desc }},
		{Header: colDetailResourceType, GetValue: func(data interface{}) interface{} { return data.(struct { Cluster ClusterResourceSummary; Pool ResourcePool }).Pool.ResourceType }},
		{Header: colDetailCPUCapacity, GetValue: func(data interface{}) interface{} { return fmt.Sprintf("%.1f", data.(struct { Cluster ClusterResourceSummary; Pool ResourcePool }).Pool.CPUCapacity) }},
		{Header: colDetailCPUAllocRate, GetValue: func(data interface{}) interface{} { return fmt.Sprintf("%.1f%%", data.(struct { Cluster ClusterResourceSummary; Pool ResourcePool }).Pool.CPUUsagePercent) }},
		{Header: colDetailAvgCPUMaxUsageRate, GetValue: func(data interface{}) interface{} { return fmt.Sprintf("%.1f%%", data.(struct { Cluster ClusterResourceSummary; Pool ResourcePool }).Pool.MaxCpuUsageRatio*100) }},
		{Header: colDetailMemCapacity, GetValue: func(data interface{}) interface{} { return fmt.Sprintf("%.1f", data.(struct { Cluster ClusterResourceSummary; Pool ResourcePool }).Pool.MemoryCapacity) }},
		{Header: colDetailMemAllocRate, GetValue: func(data interface{}) interface{} { return fmt.Sprintf("%.1f%%", data.(struct { Cluster ClusterResourceSummary; Pool ResourcePool }).Pool.MemoryUsagePercent) }},
		{Header: colDetailAvgMemMaxUsageRate, GetValue: func(data interface{}) interface{} { return fmt.Sprintf("%.1f%%", data.(struct { Cluster ClusterResourceSummary; Pool ResourcePool }).Pool.MaxMemoryUsageRatio*100) }},
		{Header: suffixPodCount, GetValue: func(data interface{}) interface{} { return data.(struct { Cluster ClusterResourceSummary; Pool ResourcePool }).Pool.PodCount }},
		{Header: suffixPodDensity, GetValue: func(data interface{}) interface{} {
			pool := data.(struct { Cluster ClusterResourceSummary; Pool ResourcePool }).Pool
			var podDensity float64
			if pool.BMCount > 0 {
				podDensity = float64(pool.PodCount) / float64(pool.BMCount)
			}
			return fmt.Sprintf("%.2f", podDensity)
		}},
	}

	// Calculate indices for styled cells in Resource Pool Detail sheet
	for i, colDef := range resourcePoolDetailColumns {
		switch colDef.Header {
		case colDetailCPUAllocRate:
			detailColIdxCPUAllocRate = i
		case colDetailAvgCPUMaxUsageRate:
			detailColIdxAvgCPUMaxUsageRate = i
		case colDetailMemAllocRate:
			detailColIdxMemAllocRate = i
		case colDetailAvgMemMaxUsageRate:
			detailColIdxAvgMemMaxUsageRate = i
		}
	}
}

// ColumnDefinition defines the metadata for an Excel column.
type ColumnDefinition struct {
	Header string
	// GetValue is a function that takes a data struct (e.g., ClusterResourceSummary or ResourcePool)
	// and returns the value for this column. Using interface{} allows flexibility.
	GetValue func(data interface{}) interface{}
	// Optional: Add formatting or styling information here later if needed
}

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

// GenerateExcelReport generates Excel report
func GenerateExcelReport(data ReportTemplateData, date string) (string, error) {
	generator := NewExcelReportGenerator(data, date)
	return generator.Generate()
}

// Generate creates the Excel report
func (g *ExcelReportGenerator) Generate() (string, error) {
	g.workbook = excelize.NewFile()
	g.initializeSheets()

	if err := g.createStyles(); err != nil {
		return "", err
	}
	if err := g.generateOverviewSheet(); err != nil {
		return "", err
	}
	if err := g.generateClustersSheet(); err != nil {
		return "", err
	}
	if err := g.generateResourcePoolsSheet(); err != nil {
		return "", err
	}

	sheetIndex, _ := g.workbook.GetSheetIndex(sheetNameOverview)
	g.workbook.SetActiveSheet(sheetIndex)

	fileName := fmt.Sprintf("k8s_resource_report_%s.xlsx", g.date)
	filePath := filepath.Join(".", fileName)
	if err := g.workbook.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("failed to save Excel file: %w", err)
	}
	return filePath, nil
}

func (g *ExcelReportGenerator) initializeSheets() {
	g.workbook.SetSheetName("Sheet1", sheetNameOverview)
	g.workbook.NewSheet(sheetNameClusterOverview)
	g.workbook.NewSheet(sheetNameResourcePoolDetail)
}

func (g *ExcelReportGenerator) createStyles() error {
	if err := g.createBasicStyles(); err != nil {
		return err
	}
	return g.createUsageStyles()
}

func (g *ExcelReportGenerator) createBasicStyles() error {
	titleStyle, err := g.workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "#000000"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return fmt.Errorf("failed to create title style: %w", err)
	}
	g.styles["title"] = titleStyle

	labelStyle, err := g.workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "#000000"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return fmt.Errorf("failed to create label style: %w", err)
	}
	g.styles["label"] = labelStyle

	valueStyle, err := g.workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 12, Color: "#000000"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return fmt.Errorf("failed to create value style: %w", err)
	}
	g.styles["value"] = valueStyle

	abnormalStyle, err := g.workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "#FF0000"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return fmt.Errorf("failed to create abnormal style: %w", err)
	}
	g.styles["abnormal"] = abnormalStyle

	headerStyle, err := g.workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create header style: %w", err)
	}
	g.styles["header"] = headerStyle

	dataStyle, err := g.workbook.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create data style: %w", err)
	}
	g.styles["data"] = dataStyle
	return nil
}

func (g *ExcelReportGenerator) createUsageStyles() error {
	createStyle := func(fontColor, fillColor string) int {
		style, _ := g.workbook.NewStyle(&excelize.Style{
			Font:   &excelize.Font{Bold: true, Color: fontColor},
			Fill:   excelize.Fill{Type: "pattern", Color: []string{fillColor}, Pattern: 1},
			Border: []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
		})
		return style
	}
	g.styles["normal"] = createStyle("#006100", "#C6EFCE")
	g.styles["warning"] = createStyle("#9C5700", "#FFEB9C")
	g.styles["danger"] = createStyle("#974706", "#F8CBAD")
	g.styles["critical"] = createStyle("#9C0006", "#FFC7CE")
	g.styles["lowUsage"] = createStyle("#0070C0", "#DDEBF7")
	return nil
}

func (g *ExcelReportGenerator) generateOverviewSheet() error {
	sheet := sheetNameOverview
	g.workbook.SetCellValue(sheet, "A1", fmt.Sprintf("Kubernetes 集群资源报告 - %s", g.date))
	g.workbook.MergeCell(sheet, "A1", "C1")
	g.workbook.SetCellStyle(sheet, "A1", "C1", g.styles["title"])
	g.workbook.SetRowHeight(sheet, 1, 30)

	g.addClusterStatsSection(sheet)
	g.addGeneralClusterSection(sheet)

	g.workbook.SetColWidth(sheet, "A", "A", 20)
	g.workbook.SetColWidth(sheet, "B", "B", 30)
	g.workbook.SetColWidth(sheet, "C", "C", 20)
	return nil
}

func (g *ExcelReportGenerator) addClusterStatsSection(sheet string) {
	g.workbook.SetCellValue(sheet, "A3", "集群统计信息")
	g.workbook.MergeCell(sheet, "A3", "C3")
	g.workbook.SetCellStyle(sheet, "A3", "C3", g.styles["label"])
	g.workbook.SetCellValue(sheet, "A4", "总已巡检集群数：")
	g.workbook.SetCellValue(sheet, "B4", g.data.Stats.TotalClusters)
	g.workbook.SetCellStyle(sheet, "A4", "A4", g.styles["label"])
	g.workbook.SetCellStyle(sheet, "B4", "B4", g.styles["value"])
	g.workbook.SetCellValue(sheet, "A5", "正常集群数：")
	g.workbook.SetCellValue(sheet, "B5", g.data.Stats.NormalClusters)
	g.workbook.SetCellStyle(sheet, "A5", "A5", g.styles["label"])
	g.workbook.SetCellStyle(sheet, "B5", "B5", g.styles["value"])
	g.workbook.SetCellValue(sheet, "A6", "异常集群数：")
	g.workbook.SetCellValue(sheet, "B6", g.data.Stats.AbnormalClusters)
	g.workbook.SetCellStyle(sheet, "A6", "A6", g.styles["label"])
	if g.data.Stats.AbnormalClusters > 0 {
		g.workbook.SetCellStyle(sheet, "B6", "B6", g.styles["abnormal"])
	} else {
		g.workbook.SetCellStyle(sheet, "B6", "B6", g.styles["value"])
	}
}

func (g *ExcelReportGenerator) addGeneralClusterSection(sheet string) {
	g.workbook.SetCellValue(sheet, "A8", "通用集群信息")
	g.workbook.MergeCell(sheet, "A8", "C8")
	g.workbook.SetCellStyle(sheet, "A8", "C8", g.styles["label"])
	g.workbook.SetCellValue(sheet, "A9", "通用集群Pod密度：")
	g.workbook.SetCellValue(sheet, "B9", fmt.Sprintf("%.2f", g.data.Stats.GeneralPodDensity))
	g.workbook.SetCellStyle(sheet, "A9", "A9", g.styles["label"])
	g.workbook.SetCellStyle(sheet, "B9", "B9", g.styles["value"])
	g.workbook.SetCellValue(sheet, "A10", "Pod密度说明：")
	g.workbook.SetCellValue(sheet, "B10", "Pod密度 = Pod总数 / 物理机节点数，表示平均每个物理机节点上运行的Pod数量")
	g.workbook.MergeCell(sheet, "B10", "C10")
	g.workbook.SetCellStyle(sheet, "A10", "A10", g.styles["label"])
	g.workbook.SetCellStyle(sheet, "B10", "C10", g.styles["value"])
}

func (g *ExcelReportGenerator) generateClustersSheet() error {
	sheet := sheetNameClusterOverview
	g.setupClusterSheetHeaders(sheet)

	rowNum := 2
	for _, cluster := range g.data.Clusters {
		pools := g.initializeClusterPools(cluster)
		g.writeClusterRow(sheet, rowNum, cluster, pools)
		rowNum++
	}
	return nil
}

func (g *ExcelReportGenerator) setupClusterSheetHeaders(sheet string) {
	for i, colDef := range clusterOverviewColumns {
		colExcelNum := i + 1
		colName, _ := excelize.ColumnNumberToName(colExcelNum)
		cellID := fmt.Sprintf("%s1", colName)
		g.workbook.SetCellValue(sheet, cellID, colDef.Header)

		width := calculateColumnWidth(colDef.Header)
		if colDef.Header == colClusterName { // Make "集群" column wider
			width *= 3
		}
		g.workbook.SetColWidth(sheet, colName, colName, width)
		g.workbook.SetCellStyle(sheet, cellID, cellID, g.styles["header"])
	}
}

func (g *ExcelReportGenerator) initializeClusterPools(cluster ClusterResourceSummary) map[string]ResourcePool {
	pools := map[string]ResourcePool{
		"total": {}, "intel_common": {}, "hg_common": {}, "arm_common": {},
	}
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

// preparePoolDataMap creates a map of metric_suffix -> value for a given resource pool.
func (g *ExcelReportGenerator) preparePoolDataMap(pool ResourcePool) map[string]interface{} {
	var podDensity float64
	if pool.BMCount > 0 {
		podDensity = float64(pool.PodCount) / float64(pool.BMCount)
	}
	return map[string]interface{}{
		suffixCPUCapacity:     pool.CPUCapacity,
		suffixCPURequest:      pool.CPURequest,
		suffixCPUAllocRate:    fmt.Sprintf("%.1f%%", pool.CPUUsagePercent),
		suffixCPUMaxUsageRate: fmt.Sprintf("%.1f%%", pool.MaxCpuUsageRatio*100),
		suffixMemCapacity:     pool.MemoryCapacity,
		suffixMemRequest:      pool.MemoryRequest,
		suffixMemAllocRate:    fmt.Sprintf("%.1f%%", pool.MemoryUsagePercent),
		suffixMemMaxUsageRate: fmt.Sprintf("%.1f%%", pool.MaxMemoryUsageRatio*100),
		suffixBMNodeCount:     pool.BMCount,
		suffixVMNodeCount:     pool.VMCount,
		suffixPodCount:        pool.PodCount,
		suffixPodDensity:      fmt.Sprintf("%.2f", podDensity),
		suffixAvgNodeCPU:      pool.PerNodeCpuRequest,
		suffixAvgNodeMem:      pool.PerNodeMemRequest,
	}
}

func (g *ExcelReportGenerator) writeClusterRow(sheet string, rowNum int, cluster ClusterResourceSummary, poolsData map[string]ResourcePool) {
	// Prepare data maps for all pools in this cluster once
	totalDataMap := g.preparePoolDataMap(poolsData["total"])
	intelDataMap := g.preparePoolDataMap(poolsData["intel_common"])
	hgDataMap := g.preparePoolDataMap(poolsData["hg_common"])
	armDataMap := g.preparePoolDataMap(poolsData["arm_common"])

	// Combine all pool data maps into a single map for easier lookup in GetValue
	allPoolDataMaps := map[string]map[string]interface{}{
		"total":        totalDataMap,
		"intel_common": intelDataMap,
		"hg_common":    hgDataMap,
		"arm_common":   armDataMap,
	}

	for i, colDef := range clusterOverviewColumns {
		colExcelNum := i + 1
		cell, _ := excelize.CoordinatesToCellName(colExcelNum, rowNum)
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["data"]) // Apply base data style

		// Use the GetValue function from the column definition to get the cell value
		// Pass the cluster and the pre-calculated pool data maps
		value := colDef.GetValue(struct {
			Cluster         ClusterResourceSummary
			AllPoolDataMaps map[string]map[string]interface{}
		}{
			Cluster:         cluster,
			AllPoolDataMaps: allPoolDataMaps,
		})
		g.workbook.SetCellValue(sheet, cell, value)
	}

	// Calculate start column indices for styling each pool block (0-based)
	// These indices are now based on the position of the pool columns in the clusterOverviewColumns slice
	totalPoolStartColIdx := -1
	intelPoolStartColIdx := -1
	hgPoolStartColIdx := -1
	armPoolStartColIdx := -1

	// Find the starting index for each pool type by checking the header prefix
	for i, colDef := range clusterOverviewColumns {
		if totalPoolStartColIdx == -1 && strings.HasPrefix(colDef.Header, prefixTotalCommon) {
			totalPoolStartColIdx = i
		} else if intelPoolStartColIdx == -1 && strings.HasPrefix(colDef.Header, prefixIntelCommon) {
			intelPoolStartColIdx = i
		} else if hgPoolStartColIdx == -1 && strings.HasPrefix(colDef.Header, prefixHGCommon) {
			hgPoolStartColIdx = i
		} else if armPoolStartColIdx == -1 && strings.HasPrefix(colDef.Header, prefixARMCommon) {
			armPoolStartColIdx = i
		}
	}

	g.applyPoolStyles(sheet, rowNum,
		poolsData["total"], poolsData["intel_common"], poolsData["hg_common"], poolsData["arm_common"],
		totalPoolStartColIdx, intelPoolStartColIdx, hgPoolStartColIdx, armPoolStartColIdx)
}

func (g *ExcelReportGenerator) applyPoolStyles(sheet string, rowNum int,
	totalPool, intelPool, hgPool, armPool ResourcePool,
	totalStartIdx, intelStartIdx, hgStartIdx, armStartIdx int) {

	applyStylesForSpecificPool := func(pool ResourcePool, poolDataStartColIdx int) {
		if pool.Nodes == 0 {
			return
		}
		// CPU Allocation Rate
		cpuAllocCellCol := poolDataStartColIdx + poolColIdxCPUAllocRate + 1
		cpuAllocCell, _ := excelize.CoordinatesToCellName(cpuAllocCellCol, rowNum)
		g.applyCPUUsageStyle(sheet, cpuAllocCell, pool.CPUUsagePercent)

		// CPU Max Usage Rate
		cpuMaxUsageCellCol := poolDataStartColIdx + poolColIdxCPUMaxUsageRate + 1
		cpuMaxUsageCell, _ := excelize.CoordinatesToCellName(cpuMaxUsageCellCol, rowNum)
		g.applyCPUUsageStyle(sheet, cpuMaxUsageCell, pool.MaxCpuUsageRatio*100)

		// Memory Allocation Rate
		memAllocCellCol := poolDataStartColIdx + poolColIdxMemAllocRate + 1
		memAllocCell, _ := excelize.CoordinatesToCellName(memAllocCellCol, rowNum)
		g.applyMemUsageStyle(sheet, memAllocCell, pool.MemoryUsagePercent)

		// Memory Max Usage Rate
		memMaxUsageCellCol := poolDataStartColIdx + poolColIdxMemMaxUsageRate + 1
		memMaxUsageCell, _ := excelize.CoordinatesToCellName(memMaxUsageCellCol, rowNum)
		g.applyMemUsageStyle(sheet, memMaxUsageCell, pool.MaxMemoryUsageRatio*100)
	}

	applyStylesForSpecificPool(totalPool, totalStartIdx)
	applyStylesForSpecificPool(intelPool, intelStartIdx)
	applyStylesForSpecificPool(hgPool, hgStartIdx)
	applyStylesForSpecificPool(armPool, armStartIdx)
}

func (g *ExcelReportGenerator) applyCPUUsageStyle(sheet, cell string, cpuUsage float64) {
	styleKey := GetCPUStyle(150, cpuUsage, g.data.Environment)
	excelStyleKey := ""
	switch styleKey {
	case "emergency":
		excelStyleKey = "critical"
	case "critical":
		excelStyleKey = "danger"
	case "warning":
		excelStyleKey = "warning"
	case "normal":
		excelStyleKey = "normal"
	case "underutilized":
		excelStyleKey = "lowUsage"
	}
	if excelStyleKey != "" {
		if styleID, ok := g.styles[excelStyleKey]; ok {
			g.workbook.SetCellStyle(sheet, cell, cell, styleID)
		}
	}
}

func (g *ExcelReportGenerator) applyMemUsageStyle(sheet, cell string, memUsage float64) {
	styleKey := GetMemoryStyle(150, memUsage, g.data.Environment)
	excelStyleKey := ""
	switch styleKey {
	case "emergency":
		excelStyleKey = "critical"
	case "critical":
		excelStyleKey = "danger"
	case "warning":
		excelStyleKey = "warning"
	case "normal":
		excelStyleKey = "normal"
	case "underutilized":
		excelStyleKey = "lowUsage"
	}
	if excelStyleKey != "" {
		if styleID, ok := g.styles[excelStyleKey]; ok {
			g.workbook.SetCellStyle(sheet, cell, cell, styleID)
		}
	}
}

func (g *ExcelReportGenerator) generateResourcePoolsSheet() error {
	sheet := sheetNameResourcePoolDetail
	g.setupResourcePoolHeaders(sheet)

	rowNum := 2
	for _, cluster := range g.data.Clusters {
		for _, pool := range cluster.ResourcePools {
			if pool.Nodes == 0 {
				continue
			}
			// Apply base data style to all cells in the row for this sheet
			for i := range resourcePoolDetailColumns {
				colExcelNum := i + 1
				cell, _ := excelize.CoordinatesToCellName(colExcelNum, rowNum)
				g.workbook.SetCellStyle(sheet, cell, cell, g.styles["data"])
			}
			g.writeResourcePoolRow(sheet, rowNum, cluster, pool)
			rowNum++
		}
	}
	return nil
}

func (g *ExcelReportGenerator) setupResourcePoolHeaders(sheet string) {
	for i, colDef := range resourcePoolDetailColumns {
		colExcelNum := i + 1
		colName, _ := excelize.ColumnNumberToName(colExcelNum)
		cellID := fmt.Sprintf("%s1", colName)
		g.workbook.SetCellValue(sheet, cellID, colDef.Header)

		width := calculateColumnWidth(colDef.Header)
		if colDef.Header == colClusterName {
			width *= 3
		}
		g.workbook.SetColWidth(sheet, colName, colName, width)
		g.workbook.SetCellStyle(sheet, cellID, cellID, g.styles["header"])
	}
}

func (g *ExcelReportGenerator) writeResourcePoolRow(sheet string, rowNum int, cluster ClusterResourceSummary, pool ResourcePool) {
	// The data for each row comes directly from the cluster and pool objects via the GetValue function in ColumnDefinition
	// No need to prepare a separate rowData map here anymore.

	// We need to pass both cluster and pool to the GetValue function for this sheet.
	// Create a struct to hold both for the GetValue function.
	data := struct {
		Cluster ClusterResourceSummary
		Pool    ResourcePool
	}{
		Cluster: cluster,
		Pool:    pool,
	}

	for i, colDef := range resourcePoolDetailColumns {
		colExcelNum := i + 1
		cell, _ := excelize.CoordinatesToCellName(colExcelNum, rowNum)
		g.workbook.SetCellStyle(sheet, cell, cell, g.styles["data"]) // Apply base data style

		// Use the GetValue function from the column definition to get the cell value
		value := colDef.GetValue(data)
		g.workbook.SetCellValue(sheet, cell, value)
	}

	// Apply styles based on the calculated indices
	cellCPUAlloc, _ := excelize.CoordinatesToCellName(detailColIdxCPUAllocRate+1, rowNum)
	g.applyResourcePoolCPUStyle(sheet, cellCPUAlloc, pool.CPUUsagePercent)
	cellCPUMaxUsage, _ := excelize.CoordinatesToCellName(detailColIdxAvgCPUMaxUsageRate+1, rowNum)
	g.applyResourcePoolCPUStyle(sheet, cellCPUMaxUsage, pool.MaxCpuUsageRatio*100)
	cellMemAlloc, _ := excelize.CoordinatesToCellName(detailColIdxMemAllocRate+1, rowNum)
	g.applyResourcePoolMemStyle(sheet, cellMemAlloc, pool.MemoryUsagePercent)
	cellMemMaxUsage, _ := excelize.CoordinatesToCellName(detailColIdxAvgMemMaxUsageRate+1, rowNum)
	g.applyResourcePoolMemStyle(sheet, cellMemMaxUsage, pool.MaxMemoryUsageRatio*100)
}

func (g *ExcelReportGenerator) applyResourcePoolCPUStyle(sheet, cell string, cpuUsage float64) {
	g.applyCPUUsageStyle(sheet, cell, cpuUsage)
}

func (g *ExcelReportGenerator) applyResourcePoolMemStyle(sheet, cell string, memUsage float64) {
	g.applyMemUsageStyle(sheet, cell, memUsage)
}

func calculateColumnWidth(text string) float64 {
	baseWidth := 1.2
	width := 0.0
	for _, r := range text {
		if r < 128 {
			width += baseWidth
		} else {
			width += baseWidth * 2
		}
	}
	width += 2
	if width < 8 {
		width = 8
	}
	if width > 50 {
		width = 50
	}
	return width
}