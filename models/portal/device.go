/*
Package portal 提供数据模型定义.
*/
package portal

// Device 设备信息.
type Device struct {
	BaseModel
	DeviceID       int64   `gorm:"column:device_id;not null" json:"deviceId"`                      // 设备ID
	CICode         string  `gorm:"column:ci_code;type:varchar(255);index" json:"ciCode"`           // 设备编码
	IP             string  `gorm:"column:ip;type:varchar(50);index" json:"ip"`                     // IP地址
	ArchType       string  `gorm:"column:arch_type;type:varchar(50)" json:"archType" query:"like"` // CPU架构
	IDC            string  `gorm:"column:idc;type:varchar(100)" json:"idc"`                        // IDC
	Room           string  `gorm:"column:room;type:varchar(100)" json:"room"`                      // 机房
	Cabinet        string  `gorm:"column:cabinet;type:varchar(100)" json:"cabinet"`                // 所属机柜
	CabinetNO      string  `gorm:"column:cabinet_no;type:varchar(100)" json:"cabinetNo"`           // 机柜编号
	InfraType      string  `gorm:"column:infra_type;type:varchar(100)" json:"infraType"`           // 网络类型
	IsLocalization bool    `gorm:"column:is_localization;type:boolean" json:"isLocalization"`      // 是否国产化
	NetZone        string  `gorm:"column:net_zone;type:varchar(100)" json:"netZone" query:"like"`  // 网络区域
	Group          string  `gorm:"column:group;type:varchar(100)" json:"group"`                    // 机器类别
	AppID          string  `gorm:"column:appid;type:varchar(100)" json:"appId"`                    // APPID
	OsCreateTime   string  `gorm:"column:os_create_time;type:varchar(100)" json:"osCreateTime"`    // 操作系统创建时间
	CPU            float64 `gorm:"column:cpu;type:float" json:"cpu"`                               // CPU数量
	Memory         float64 `gorm:"column:memory;type:float" json:"memory"`                         // 内存大小
	Model          string  `gorm:"column:model;type:varchar(100)" json:"model"`                    // 型号
	KvmIP          string  `gorm:"column:kvm_ip;type:varchar(50)" json:"kvmIp"`                    // KVM IP
	OS             string  `gorm:"column:os;type:varchar(100)" json:"os"`                          // 操作系统
	Company        string  `gorm:"column:company;type:varchar(100)" json:"company"`                // 厂商
	OSName         string  `gorm:"column:os_name;type:varchar(100)" json:"osName"`                 // 操作系统名称
	OSIssue        string  `gorm:"column:os_issue;type:varchar(100)" json:"osIssue"`               // 操作系统版本
	OSKernel       string  `gorm:"column:os_kernel;type:varchar(100)" json:"osKernel"`             // 操作系统内核
	Status         string  `gorm:"column:status;type:varchar(50)" json:"status"`                   // 状态
	Role           string  `gorm:"column:role;type:varchar(100)" json:"role" query:"like"`         // 角色
	Cluster        string  `gorm:"column:cluster;type:varchar(255)" json:"cluster" query:"like"`   // 所属集群
	ClusterID      int     `gorm:"column:cluster_id;type:int" json:"clusterId"`                    // 集群ID
	AcceptanceTime string  `gorm:"column:acceptance_time;type:varchar(100)" json:"acceptanceTime"` // 验收时间
	DiskCount      int     `gorm:"column:disk_count" json:"diskCount"`                             // 磁盘数量
	DiskDetail     string  `gorm:"column:disk_detail" json:"diskDetail"`                           // 磁盘详情
	NetworkSpeed   string  `gorm:"column:network_speed" json:"networkSpeed"`                       // 网络速度

	// 特性标记，用于前端显示
	IsSpecial    bool `gorm:"column:is_special" json:"isSpecial"`       // 是否为特殊设备
	FeatureCount int  `gorm:"column:feature_count" json:"featureCount"` // 特性数量
}

// TableName 指定表名.
func (Device) TableName() string {
	return "device"
}
