package security_report

// SecurityReportData 安全报告数据
type SecurityReportData struct {
	ClusterName     string
	TotalChecks     int
	PassedChecks    int
	FailedChecks    int
	WarningChecks   int
	DetailedResults []SecurityCheckResult
}

// SecurityCheckResult 安全检查结果
type SecurityCheckResult struct {
	NodeType      string
	NodeName      string
	CheckType     string
	ItemName      string
	ItemValue     string
	Status        bool
	FixSuggestion string // 修复建议
}

// EmailTemplateData 邮件模板数据
type EmailTemplateData struct {
	// 汇总信息
	TotalClusters        int
	NormalClusters       int // 正常集群数
	UnscannedClusters    int // 未巡检集群数
	TotalNodes           int
	NormalNodes          int
	AbnormalNodes        int
	MissingNodesCount    int    // 未巡检节点总数
	NormalNodesPercent   string // 正常节点百分比
	AbnormalNodesPercent string // 异常节点百分比
	MissingNodesPercent  string // 未巡检节点百分比
	TotalChecks          int
	PassedChecks         int
	FailedChecks         int

	// 巡检失败节点
	MissingNodes []MissingNode

	// 异常节点详情
	AbnormalDetails []AbnormalDetail
	// 集群健康度概览
	ClusterHealthSummary []ClusterHealthInfo

	// 检查项失败概览 (模拟热力图)
	CheckItemFailureSummary []CheckItemFailureInfo
}

// MissingNode 巡检失败节点信息
type MissingNode struct {
	ClusterName string
	NodeType    string
	NodeName    string
}

// AbnormalDetail 异常节点详情
type AbnormalDetail struct {
	ClusterName string
	NodeType    string
	NodeName    string
	FailedItems []FailedItem
}

// FailedItem 失败的检查项
type FailedItem struct {
	ItemName      string
	ItemValue     string
	FixSuggestion string // 修复建议
}

// ClusterHealthInfo 集群健康信息
type ClusterHealthInfo struct {
	ClusterName   string
	StatusColor   string // e.g., "green", "yellow", "red"
	AbnormalNodes int
	NormalNodes   int // 集群中的正常节点数
	TotalNodes    int // 集群中的节点总数
	MissingNodes  int // 未巡检节点数量
	FailedChecks  int
	Exists        bool   // 集群在S3中是否存在
	AnchorID      string // 错误详情页面的锚点ID
}

// CheckItemFailureInfo 检查项失败信息 (用于模拟热力图)
type CheckItemFailureInfo struct {
	ItemName      string
	TotalFailures int
	HeatColor     string // e.g., "heat-level-1", "heat-level-2", "heat-level-high"
}
