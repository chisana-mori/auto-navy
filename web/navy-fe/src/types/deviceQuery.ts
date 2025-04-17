// 筛选类型
export enum FilterType {
  NodeLabel = 'nodeLabel', // 节点标签
  Taint = 'taint',         // 污点
  Device = 'device',       // 设备字段
  NodeInfo = 'nodeInfo',   // 节点信息
}

// 设备字段类型
export enum DeviceFieldType {
  CICode = 'ciCode',           // 设备编码
  IP = 'ip',                   // IP地址
  ArchType = 'archType',       // CPU架构
  IDC = 'idc',                 // IDC
  Room = 'room',               // 机房
  Cabinet = 'cabinet',         // 所属机柜
  CabinetNO = 'cabinetNO',     // 机柜编号
  InfraType = 'infraType',     // 网络类型
  IsLocalization = 'isLocalization', // 是否国产化
  NetZone = 'netZone',         // 网络区域
  Group = 'group',             // 机器类别
  AppId = 'appId',             // APPID
  OsCreateTime = 'osCreateTime', // 操作系统创建时间
  CPU = 'cpu',                 // CPU数量
  Memory = 'memory',           // 内存大小
  Model = 'model',             // 型号
  KvmIP = 'kvmIp',             // KVM IP
  OS = 'os',                   // 操作系统
  Company = 'company',         // 厂商
  OSName = 'osName',           // 操作系统名称
  OSIssue = 'osIssue',         // 操作系统版本
  OSKernel = 'osKernel',       // 操作系统内核
  Status = 'status',           // 状态
  Role = 'role',               // 角色
  Cluster = 'cluster',         // 所属集群
  ClusterID = 'clusterId',     // 集群ID
  AcceptanceTime = 'acceptanceTime', // 验收时间
}

// 节点信息字段类型
export enum NodeInfoFieldType {
  DiskCount = 'diskCount',     // 磁盘数量
  DiskDetail = 'diskDetail',   // 磁盘详情
  NetworkSpeed = 'networkSpeed', // 网络速度
}

// 条件类型
export enum ConditionType {
  Equal = 'equal',               // 等于
  NotEqual = 'notEqual',         // 不等于
  Contains = 'contains',         // 包含
  NotContains = 'notContains',   // 不包含
  Exists = 'exists',             // 存在
  NotExists = 'notExists',       // 不存在
  In = 'in',                     // 在列表中
  NotIn = 'notIn',               // 不在列表中
  GreaterThan = 'greaterThan',   // 大于
  LessThan = 'lessThan',         // 小于
  IsEmpty = 'isEmpty',           // 为空
  IsNotEmpty = 'isNotEmpty',     // 不为空
}

// 逻辑运算符
export enum LogicalOperator {
  And = 'and', // 与
  Or = 'or',   // 或
}

// 筛选选项
export interface FilterOption {
  id: string;    // 选项ID
  label: string; // 选项标签
  value: string; // 选项值
}

// 筛选块
export interface FilterBlock {
  id: string;                  // 筛选块ID
  type: FilterType;            // 筛选类型
  conditionType: ConditionType; // 条件类型
  field?: string;              // 字段
  key?: string;                // 键
  value?: string | string[];   // 值（单选或多选）
  operator: LogicalOperator;   // 与下一个条件的逻辑关系
}

// 筛选组
export interface FilterGroup {
  id: string;                // 筛选组ID
  blocks: FilterBlock[];     // 筛选块列表
  operator: LogicalOperator; // 与下一个组的逻辑关系
}

// 查询模板
export interface QueryTemplate {
  id?: number;         // 模板ID，可选，创建时不需要提供
  name: string;        // 模板名称
  description: string; // 模板描述
  groups: FilterGroup[]; // 筛选组列表
}

// 设备查询请求
export interface DeviceQueryRequest {
  groups: FilterGroup[]; // 筛选组列表
  page?: number;         // 页码
  size?: number;         // 每页数量
}
