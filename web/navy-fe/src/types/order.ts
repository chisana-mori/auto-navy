// 通用订单类型定义

// 订单类型枚举
export type OrderType = 'elastic_scaling' | 'maintenance' | 'deployment';

// 订单状态枚举
export type OrderStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled' | 'ignored';

// 基础订单接口
export interface Order {
  id: number;
  orderNumber: string;
  name: string;
  description: string;
  type: OrderType;
  status: OrderStatus;
  executor: string;
  executionTime?: string;
  createdBy: string;
  completionTime?: string;
  failureReason: string;
  createdAt: string;
  updatedAt: string;
}

// 订单查询参数
export interface OrderQuery {
  type?: OrderType;
  status?: OrderStatus;
  createdBy?: string;
  page?: number;
  pageSize?: number;
  startTime?: string;
  endTime?: string;
}

// 分页响应
export interface PaginatedOrderResponse {
  list: Order[];
  total: number;
}

// 订单状态更新请求
export interface OrderStatusUpdateRequest {
  status: OrderStatus;
  reason?: string;
}

// 订单创建请求
export interface OrderCreateRequest {
  name: string;
  description: string;
  type: OrderType;
  createdBy?: string;
}

// 订单统计
export interface OrderStatistics {
  totalCount: number;
  pendingCount: number;
  processingCount: number;
  completedCount: number;
  failedCount: number;
  cancelledCount: number;
  ignoredCount: number;
}

// 订单状态选项
export const ORDER_STATUS_OPTIONS = [
  { value: 'pending', label: '待处理', color: '#faad14' },
  { value: 'processing', label: '处理中', color: '#1890ff' },
  { value: 'completed', label: '已完成', color: '#52c41a' },
  { value: 'failed', label: '失败', color: '#f5222d' },
  { value: 'cancelled', label: '已取消', color: '#d9d9d9' },
  { value: 'ignored', label: '已忽略', color: '#d9d9d9' },
] as const;

// 订单类型选项
export const ORDER_TYPE_OPTIONS = [
  { value: 'elastic_scaling', label: '弹性伸缩' },
  { value: 'maintenance', label: '设备维护' },
  { value: 'deployment', label: '应用部署' },
] as const;

// 获取订单状态显示信息
export function getOrderStatusInfo(status: OrderStatus) {
  return ORDER_STATUS_OPTIONS.find(option => option.value === status) || {
    value: status,
    label: status,
    color: '#d9d9d9'
  };
}

// 获取订单类型显示信息
export function getOrderTypeInfo(type: OrderType) {
  return ORDER_TYPE_OPTIONS.find(option => option.value === type) || {
    value: type,
    label: type
  };
}

// 订单操作权限检查
export function canUpdateOrderStatus(status: OrderStatus, targetStatus: OrderStatus): boolean {
  const statusFlow: Record<OrderStatus, OrderStatus[]> = {
    pending: ['processing', 'cancelled', 'ignored'],
    processing: ['completed', 'failed', 'cancelled'],
    completed: [],
    failed: ['pending'],
    cancelled: ['pending'],
    ignored: ['pending'],
  };

  return statusFlow[status]?.includes(targetStatus) || false;
}

// 订单状态是否为最终状态
export function isFinalStatus(status: OrderStatus): boolean {
  return ['completed', 'failed', 'cancelled', 'ignored'].includes(status);
}

// 订单状态是否为活跃状态
export function isActiveStatus(status: OrderStatus): boolean {
  return ['pending', 'processing'].includes(status);
}
