// 设备信息类型定义

export interface Device {
  id: number;           // ID
  deviceId: string;     // 设备ID
  ip: string;           // IP地址
  machineType: string;  // 机器类型
  cluster: string;      // 所属集群
  role: string;         // 集群角色
  arch: string;         // 架构
  idc: string;          // IDC
  room: string;         // Room
  datacenter: string;   // 机房
  cabinet: string;      // 机柜号
  network: string;      // 网络区域
  appId: string;        // APPID
  resourcePool: string;  // 资源池/产品
  createdAt: string;    // 创建时间
  updatedAt: string;    // 更新时间
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
