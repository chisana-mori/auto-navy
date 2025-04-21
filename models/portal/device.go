/*
Package portal 提供数据模型定义.
*/
package portal

import (
	"fmt"
	"gorm.io/gorm" // 添加 gorm 包导入
)

// PublishDeviceChangeEventFunc 定义了发布设备变更事件的函数类型
// 参数是设备 ID
type PublishDeviceChangeEventFunc func(deviceID int64)

// deviceChangeEventPublisher 是一个包级变量，用于持有实际的事件发布函数
// 这个变量将在 service 包初始化时被赋值
var deviceChangeEventPublisher PublishDeviceChangeEventFunc

// RegisterDeviceChangeEventPublisher 允许 service 包注册事件发布函数
func RegisterDeviceChangeEventPublisher(publisher PublishDeviceChangeEventFunc) {
	if deviceChangeEventPublisher != nil {
		// 防止重复注册，或者根据需要处理
		fmt.Println("Warning: Device change event publisher already registered.")
		return
	}
	deviceChangeEventPublisher = publisher
}

// Device 设备信息.
type Device struct {
	BaseModel
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

	// 特性标记，用于前端显示，这些字段是计算得出的，设置为只读
	IsSpecial    bool   `gorm:"column:is_special;->" json:"isSpecial"`       // 是否为特殊设备，只读
	FeatureCount int    `gorm:"column:feature_count;->" json:"featureCount"` // 特性数量，只读
	AppName      string `gorm:"column:app_name;->" json:"appName"`           // 应用名称，只读
}

// TableName 指定表名.
func (Device) TableName() string {
	return "device"
}

// AfterSave GORM Hook: 在创建或更新设备后触发
func (d *Device) AfterSave(tx *gorm.DB) (err error) {
	// 调用已注册的事件发布函数
	if deviceChangeEventPublisher != nil {
		fmt.Printf("Hook AfterSave triggered for Device ID: %d, publishing event...\n", d.ID)
		deviceChangeEventPublisher(d.ID)
	} else {
		fmt.Printf("Warning: Hook AfterSave triggered for Device ID: %d, but no event publisher registered.\n", d.ID)
	}
	return nil
}

// AfterDelete GORM Hook: 在删除设备后触发
func (d *Device) AfterDelete(tx *gorm.DB) (err error) {
	// 调用已注册的事件发布函数
	if deviceChangeEventPublisher != nil {
		fmt.Printf("Hook AfterDelete triggered for Device ID: %d, publishing event...\n", d.ID)
		deviceChangeEventPublisher(d.ID)
	} else {
		fmt.Printf("Warning: Hook AfterDelete triggered for Device ID: %d, but no event publisher registered.\n", d.ID)
	}
	return nil
}
