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
	NodeType  string
	NodeName  string
	CheckType string
	ItemName  string
	ItemValue string
	Status    bool
}

// EmailTemplateData 邮件模板数据
type EmailTemplateData struct {
	// 汇总信息
	TotalClusters int
	TotalNodes    int
	NormalNodes   int
	AbnormalNodes int
	TotalChecks   int
	PassedChecks  int
	FailedChecks  int

	// 巡检失败节点
	MissingNodes []MissingNode

	// 异常节点详情
	AbnormalDetails []AbnormalDetail
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
	ItemName  string
	ItemValue string
}
