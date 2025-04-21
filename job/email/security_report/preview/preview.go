package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/Masterminds/sprig/v3"
)

// 创建模拟数据
func createMockData() map[string]interface{} {
	// 基础节点数据
	normalNodes := float64(9)
	abnormalNodes := float64(3)
	totalNodes := normalNodes + abnormalNodes
	totalClusters := 3
	normalClusters := 1    // 正常集群数
	unscannedClusters := 0 // 未巡检集群数

	// 检查项数据
	passedChecks := 16
	failedChecks := 4
	totalChecks := passedChecks + failedChecks

	// 计算百分比
	normalNodesPercent := fmt.Sprintf("%.0f", normalNodes*100/totalNodes)
	abnormalNodesPercent := fmt.Sprintf("%.0f", abnormalNodes*100/totalNodes)

	// 集群健康状态摘要
	clusterHealthSummary := []map[string]interface{}{
		{
			"ClusterName":   "集群A",
			"StatusColor":   "red",
			"AbnormalNodes": 1,
			"TotalNodes":    4, // 总节点数
			"FailedChecks":  2,
			"Exists":        true,
			"AnchorID":      "集群A",
		},
		{
			"ClusterName":   "集群B",
			"StatusColor":   "red",
			"AbnormalNodes": 2,
			"TotalNodes":    5, // 总节点数
			"FailedChecks":  2,
			"Exists":        true,
			"AnchorID":      "集群B",
		},
		{
			"ClusterName":   "集群C",
			"StatusColor":   "green",
			"AbnormalNodes": 0,
			"TotalNodes":    3, // 总节点数
			"FailedChecks":  0,
			"Exists":        true,
			"AnchorID":      "集群C",
		},
	}

	// 检查项失败摘要
	checkItemFailureSummary := []map[string]interface{}{}

	// 仅用于测试空状态图形的场景
	emptyTest := false
	if emptyTest {
		// 测试场景：无异常情况
		checkItemFailureSummary = []map[string]interface{}{}
		abnormalNodes = 0
		normalNodes = 12
		totalNodes = normalNodes
		normalNodesPercent = "100"
	} else {
		// 正常场景：有异常情况
		checkItemFailureSummary = []map[string]interface{}{
			{
				"ItemName":      "未授权访问风险",
				"TotalFailures": 3,
				"HeatColor":     "heat-level-high",
			},
			{
				"ItemName":      "密码强度不足",
				"TotalFailures": 2,
				"HeatColor":     "heat-level-2",
			},
			{
				"ItemName":      "防火墙配置不当",
				"TotalFailures": 1,
				"HeatColor":     "heat-level-1",
			},
			{
				"ItemName":      "日志审计缺失",
				"TotalFailures": 1,
				"HeatColor":     "heat-level-1",
			},
		}
	}

	// 缺失节点数据
	missingNodes := []map[string]interface{}{
		{
			"ClusterName": "集群A",
			"NodeType":    "数据库节点",
			"NodeName":    "db-slave-02",
		},
	}

	// 异常节点详情
	abnormalDetails := []map[string]interface{}{
		{
			"ClusterName": "集群A",
			"NodeType":    "应用服务器",
			"NodeName":    "app-server-01",
			"FailedItems": []map[string]interface{}{
				{
					"ItemName":      "未授权访问风险",
					"ItemValue":     "发现8个敏感端口开放",
					"FixSuggestion": "关闭不必要的端口，配置IP白名单",
				},
				{
					"ItemName":      "密码强度不足",
					"ItemValue":     "管理员密码仅8位，未包含特殊字符",
					"FixSuggestion": "更新密码策略，要求至少12位且包含大小写字母、数字和特殊字符",
				},
			},
		},
		{
			"ClusterName": "集群B",
			"NodeType":    "负载均衡器",
			"NodeName":    "lb-master-01",
			"FailedItems": []map[string]interface{}{
				{
					"ItemName":      "防火墙配置不当",
					"ItemValue":     "过于宽松的入站规则",
					"FixSuggestion": "限制入站流量仅来自已知IP地址",
				},
			},
		},
		{
			"ClusterName": "集群B",
			"NodeType":    "应用服务器",
			"NodeName":    "app-server-03",
			"FailedItems": []map[string]interface{}{
				{
					"ItemName":      "日志审计缺失",
					"ItemValue":     "未启用关键操作审计日志",
					"FixSuggestion": "配置审计日志，并将日志发送到中央日志服务器",
				},
			},
		},
	}

	// 组合所有数据
	return map[string]interface{}{
		"NormalNodes":             normalNodes,
		"AbnormalNodes":           abnormalNodes,
		"TotalNodes":              totalNodes,
		"NormalNodesPercent":      normalNodesPercent,
		"AbnormalNodesPercent":    abnormalNodesPercent,
		"TotalClusters":           totalClusters,
		"NormalClusters":          normalClusters,
		"UnscannedClusters":       unscannedClusters,
		"TotalChecks":             totalChecks,
		"PassedChecks":            passedChecks,
		"FailedChecks":            failedChecks,
		"ClusterHealthSummary":    clusterHealthSummary,
		"CheckItemFailureSummary": checkItemFailureSummary,
		"MissingNodes":            missingNodes,
		"AbnormalDetails":         abnormalDetails,
	}
}

func main() {
	// 创建模拟数据
	data := createMockData()

	// 将模拟数据保存到文件中，方便查看
	mockDataBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("序列化模拟数据失败: %v", err)
	}
	err = ioutil.WriteFile("../mock_data.json", mockDataBytes, 0644)
	if err != nil {
		log.Fatalf("保存模拟数据失败: %v", err)
	}
	log.Println("模拟数据已保存到: ../mock_data.json")

	// 扩展函数映射
	funcMap := template.FuncMap{
		"toFloat64": toFloat64,
		"sin": func(a interface{}) float64 {
			return math.Sin(toFloat64(a))
		},
		"cos": func(a interface{}) float64 {
			return math.Cos(toFloat64(a))
		},
		"negate": func(a interface{}) float64 {
			return -toFloat64(a)
		},
		"gt": func(a, b interface{}) bool {
			return toFloat64(a) > toFloat64(b)
		},
		"lt": func(a, b interface{}) bool {
			return toFloat64(a) < toFloat64(b)
		},
		"eq": func(a, b interface{}) bool {
			return toFloat64(a) == toFloat64(b)
		},
		// 添加一个专门用于比较字符串的函数
		"strEq": func(a, b string) bool {
			return a == b
		},
		"printf": func(format string, a ...interface{}) string {
			return fmt.Sprintf(format, a...)
		},
		"safeHTML": func(s interface{}) template.HTML {
			return template.HTML(fmt.Sprint(s))
		},
		// 基本算术函数
		"add": func(a, b interface{}) float64 {
			return toFloat64(a) + toFloat64(b)
		},
		"sub": func(a, b interface{}) float64 {
			return toFloat64(a) - toFloat64(b)
		},
		"mul": func(a, b interface{}) float64 {
			return toFloat64(a) * toFloat64(b)
		},
		"div": func(a, b interface{}) float64 {
			bb := toFloat64(b)
			if bb == 0 {
				return 0
			}
			return toFloat64(a) / bb
		},
		// 计算饼图坐标
		"svgArcX": func(angle float64, radius float64) float64 {
			return radius * math.Sin(angle*math.Pi/180)
		},
		"svgArcY": func(angle float64, radius float64) float64 {
			return -radius * math.Cos(angle*math.Pi/180)
		},
		// 日期格式化
		"now": func() time.Time {
			return time.Now()
		},
		"date": func(layout string, t time.Time) string {
			return t.Format(layout)
		},
	}

	// 创建模板，使用Sprig函数库并添加自定义函数
	tmpl, err := template.New("template.html").Funcs(sprig.FuncMap()).Funcs(funcMap).ParseFiles("../template.html")
	if err != nil {
		log.Fatalf("解析 template.html 失败: %v", err)
	}

	// 输出到 preview.html
	out, err := os.Create("../preview.html")
	if err != nil {
		log.Fatalf("创建 preview.html 失败: %v", err)
	}
	defer out.Close()
	if err := tmpl.Execute(out, data); err != nil {
		log.Fatalf("渲染模板失败: %v", err)
	}
	log.Println("预览文件已生成: ../preview.html")
}

func toFloat64(a interface{}) float64 {
	switch v := a.(type) {
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
