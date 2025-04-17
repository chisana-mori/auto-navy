package portal

type TaintManagement struct {
	BaseModel
	Key         string
	Value       string
	Effect      string
	Description string
	Type        string
	Status      int
	Color       string
}

func (t TaintManagement) TableName() string {
	return "taint_feature"
}
