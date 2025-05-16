// 弹性伸缩策略类型定义
export interface Strategy {
  id: number;
  name: string;
  description: string;
  thresholdTriggerAction: 'pool_entry' | 'pool_exit';
  resourceTypes?: string | string[];
  cpuThresholdValue?: number;
  cpuThresholdType?: 'usage' | 'allocated';
  cpuTargetValue?: number;
  memoryThresholdValue?: number;
  memoryThresholdType?: 'usage' | 'allocated';
  memoryTargetValue?: number;
  conditionLogic: 'AND' | 'OR';
  durationMinutes?: number;
  cooldownMinutes?: number;
  deviceCount: number;
  nodeSelector: string;
  status: 'enabled' | 'disabled';
  createdBy: string;
  createdAt: string;
  updatedAt: string;
  clusters: string[];  // 策略列表视图使用
  clusterIds?: number[]; // 创建/编辑时使用
}

// 策略执行历史类型定义
export interface StrategyExecutionHistory {
  id: number;
  executionTime: string;
  triggeredValue: string;
  thresholdValue: string;
  result: 'order_created' | 'skipped' | 'failed_check';
  orderId?: number;
  reason: string;
}

// 策略详情类型定义
export interface StrategyDetail extends Strategy {
  executionHistory: StrategyExecutionHistory[];
  relatedOrders: OrderListItem[];
}

// 订单列表项类型定义
export interface OrderListItem {
  id: number;
  orderNumber: string;
  clusterId: number;
  clusterName: string;
  strategyId?: number;
  strategyName?: string;
  actionType: 'pool_entry' | 'pool_exit' | 'maintenance_request' | 'maintenance_uncordon';
  status: 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled' | 'pending_confirmation' | 'scheduled_for_maintenance' | 'maintenance_in_progress';
  deviceCount: number;
  createdBy: string;
  createdAt: string;
}

// 设备类型定义
export interface Device {
  id: number;
  ciCode: string;
  ip: string;
  archType: string;
  cpu: number;
  memory: number;
  status: string;
  role: string;
  cluster: string;
  clusterId: number;
  isSpecial: boolean;
  featureCount: number;
  orderStatus?: string; // 在订单中的状态
}

// 订单详情类型定义
export interface OrderDetail extends OrderListItem {
  deviceId?: number;
  deviceInfo?: Device;
  approver: string;
  executor: string;
  executionTime?: string;
  completionTime?: string;
  failureReason: string;
  maintenanceStartTime?: string;
  maintenanceEndTime?: string;
  externalTicketId?: string;
  devices: Device[];
}

// 工作台统计数据类型定义
export interface DashboardStats {
  strategyCount: number;
  triggeredTodayCount: number;
  enabledStrategyCount: number;
  clusterCount: number;
  abnormalClusterCount: number;
  pendingOrderCount: number;
  deviceCount: number;
  availableDeviceCount: number;
  inPoolDeviceCount: number;
}

// 资源类型数据
export interface ResourceTypeData {
  timestamps: string[];
  cpuUsageRatio: number[];
  cpuAllocationRatio: number[];
  memUsageRatio: number[];
  memAllocationRatio: number[];
}

// 资源分配趋势类型定义
export interface ResourceAllocationTrend {
  timestamps: string[];
  cpuUsageRatio: number[];
  cpuAllocationRatio: number[];
  memUsageRatio: number[];
  memAllocationRatio: number[];
  resourceTypes: string[];
  resourceTypeData: Record<string, ResourceTypeData>;
}

// 订单统计类型定义
export interface OrderStats {
  totalCount: number;
  pendingCount: number;
  processingCount: number;
  completedCount: number;
  failedCount: number;
  cancelledCount: number;
}

// 分页响应类型定义
export interface PaginatedResponse<T> {
  list: T[];
  total: number;
  page: number;
  size: number;
} 