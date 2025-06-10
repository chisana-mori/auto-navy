package portal

// PhoneDuty 值班电话信息
type PhoneDuty struct {
	BaseModel
	DutyDate string `gorm:"column:duty_date"`   // 值班日期
	WeekDay  string `gorm:"column:week_day"`    // 星期
	AUm      string `gorm:"column:a_um"`       // A班上午
	ACn      string `gorm:"column:a_cn"`       // A班下午
	BUm      string `gorm:"column:b_um"`       // B班上午
	BCn      string `gorm:"column:b_cn"`       // B班下午
	OmegaId  int64  `gorm:"column:omega_id"`   // Omega ID
	CUm      string `gorm:"column:c_um"`       // C班上午
	CCn      string `gorm:"column:c_cn"`       // C班下午
}

// TableName 指定表名
func (PhoneDuty) TableName() string {
	return "phone_duty"
}