package portal

type LabelManagement struct {
	BaseModel
	Name        string       `gorm:"column:name"`
	Key         string       `gorm:"column:key"`
	Source      int          `gorm:"int:2;column:source"`      // 0内部 1外部 2其他
	IsControl   bool         `gorm:"column:is_control;char:1"` // 0纳管 1不纳管
	Range       string       `gorm:"column:range"`
	IDC         string       `gorm:"column:idc"`
	Status      int          `gorm:"column:status"` // 0正常 1停用
	IsDenyList  bool         `gorm:"column:is_deny_list"`
	LabelValues []LabelValue `gorm:"foreignkey:LabelID"`
	Nodes       []K8sNode    `gorm:"many2many:k8s_node_label_features"`
	Color       string       `gorm:"column:color"`
}

func (l LabelManagement) TableName() string {
	return "label_feature"
}

type LabelValue struct {
	BaseModel
	LabelID int  `gorm:"column:label_id"`
	Value   string `gorm:"column:value"`
}

func (l LabelValue) TableName() string {
	return "label_feature_value"
}
