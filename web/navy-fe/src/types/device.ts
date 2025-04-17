// 设备信息类型定义

export interface Device {
  id: number;           // ID
  ciCode: string;       // 设备编码
  ip: string;           // IP地址
  archType: string;     // CPU架构
  idc: string;          // IDC
  room: string;         // 机房
  cabinet: string;      // 所属机柜
  cabinetNO: string;    // 机柜编号
  infraType: string;    // 网络类型
  isLocalization: boolean; // 是否国产化
  netZone: string;      // 网络区域
  group: string;        // 机器类别
  appId: string;        // APPID
  appName?: string;     // 应用名称
  osCreateTime: string; // 操作系统创建时间
  cpu: number;          // CPU数量
  memory: number;       // 内存大小
  model: string;        // 型号
  kvmIp: string;        // KVM IP
  os: string;           // 操作系统
  company: string;      // 厂商
  osName: string;       // 操作系统名称
  osIssue: string;      // 操作系统版本
  osKernel: string;     // 操作系统内核
  status: string;       // 状态
  role: string;         // 角色
  cluster: string;      // 所属集群
  clusterId: number;    // 集群ID
  acceptanceTime: string; // 验收时间
  diskCount?: number;   // 磁盘数量
  diskDetail?: string;  // 磁盘详情
  networkSpeed?: string; // 网络速度
  createdAt: string;    // 创建时间
  updatedAt: string;    // 更新时间

  // 特性标记，用于前端显示
  isSpecial?: boolean;   // 是否为特殊设备
  featureCount?: number; // 特性数量
  featureDetails?: string[]; // 特性详情
}

export interface DeviceListResponse {
  list: Device[];
  page: number;
  size: number;
  total: number;
}

export interface DeviceQuery {
  page?: number;
  size?: number;
  keyword?: string;     // 全局搜索关键字
}
