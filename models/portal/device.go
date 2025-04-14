/*
Package portal 提供数据模型定义.
*/
package portal

// Device 设备信息.
type Device struct {
	BaseModel
	DeviceID     string    `gorm:"column:device_id;type:varchar(255);not null;index" json:"deviceId"`     // 设备ID
	IP           string    `gorm:"column:ip;type:varchar(50);index" json:"ip"`                             // IP地址
	MachineType  string    `gorm:"column:machine_type;type:varchar(100)" json:"machineType"`              // 机器类型
	Cluster      string    `gorm:"column:cluster;type:varchar(255)" json:"cluster"`                        // 所属集群
	Role         string    `gorm:"column:role;type:varchar(100)" json:"role"`                              // 集群角色
	Arch         string    `gorm:"column:arch;type:varchar(50)" json:"arch"`                               // 架构
	IDC          string    `gorm:"column:idc;type:varchar(100)" json:"idc"`                                // IDC
	Room         string    `gorm:"column:room;type:varchar(100)" json:"room"`                              // Room
	Datacenter   string    `gorm:"column:datacenter;type:varchar(100)" json:"datacenter"`                  // 机房
	Cabinet      string    `gorm:"column:cabinet;type:varchar(100)" json:"cabinet"`                        // 机柜号
	Network      string    `gorm:"column:network;type:varchar(100)" json:"network"`                        // 网络区域
	AppID        string    `gorm:"column:app_id;type:varchar(100)" json:"appId"`                          // APPID
	ResourcePool string    `gorm:"column:resource_pool;type:varchar(255)" json:"resourcePool"`            // 资源池/产品
	Deleted      string    `gorm:"column:deleted;type:varchar(255)" json:"deleted,omitempty"`              // 软删除标记
}

// TableName 指定表名.
func (Device) TableName() string {
	return "device"
}
