// 筛选类型
export enum FilterType {
  NodeLabel = 'nodeLabel', // 节点标签
  Taint = 'taint',         // 污点
  Device = 'device',       // 设备字段
}

// 设备字段类型
export enum DeviceFieldType {
  IP = 'ip',                   // IP地址
  MachineType = 'machineType', // 机器类型
  Role = 'role',               // 集群角色
  Arch = 'arch',               // 架构
  IDC = 'idc',                 // IDC
  Room = 'room',               // Room
  Datacenter = 'datacenter',   // 机房
  Cabinet = 'cabinet',         // 机柜号
  Network = 'network',         // 网络区域
  AppId = 'appId',             // APPID
  ResourcePool = 'resourcePool' // 资源池
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
