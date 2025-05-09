package security_check

// ConfigCheck 配置检查项
type ConfigCheck struct {
	Name          string
	Value         string
	Status        bool
	FixSuggestion string // 修复建议
}

// ConfigResult 配置检查结果
type ConfigResult struct {
	ClusterName string
	NodeType    string
	NodeName    string
	CheckType   string
	Checks      []ConfigCheck
}
