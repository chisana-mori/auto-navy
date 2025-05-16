import axios from 'axios';
import {
  Strategy,
  StrategyDetail,
  OrderListItem,
  OrderDetail,
  Device,
  DashboardStats,
  ResourceAllocationTrend,
  OrderStats,
  PaginatedResponse
} from '../types/elastic-scaling';

const API_BASE = '/fe-v1/elastic-scaling';

// 策略相关API
export const strategyApi = {
  // 获取策略列表
  getStrategies: async (params: {
    name?: string;
    status?: string;
    action?: string;
    page?: number;
    pageSize?: number;
  }): Promise<PaginatedResponse<Strategy>> => {
    const response = await axios.get(`${API_BASE}/strategies`, { params });
    return response.data.data;
  },

  // 获取策略详情
  getStrategy: async (id: number): Promise<StrategyDetail> => {
    const response = await axios.get(`${API_BASE}/strategies/${id}`);
    return response.data.data;
  },

  // 创建策略
  createStrategy: async (strategy: Omit<Strategy, 'id' | 'createdAt' | 'updatedAt'>): Promise<{ id: number }> => {
    const response = await axios.post(`${API_BASE}/strategies`, strategy);
    return response.data.data;
  },

  // 更新策略
  updateStrategy: async (id: number, strategy: Omit<Strategy, 'id' | 'createdAt' | 'updatedAt'>): Promise<void> => {
    await axios.put(`${API_BASE}/strategies/${id}`, strategy);
  },

  // 删除策略
  deleteStrategy: async (id: number): Promise<void> => {
    await axios.delete(`${API_BASE}/strategies/${id}`);
  },

  // 更新策略状态
  updateStrategyStatus: async (id: number, status: 'enabled' | 'disabled'): Promise<void> => {
    await axios.put(`${API_BASE}/strategies/${id}/status`, { status });
  },

  // 获取策略执行历史
  getStrategyExecutionHistory: async (id: number): Promise<any[]> => {
    const response = await axios.get(`${API_BASE}/strategies/${id}/execution-history`);
    return response.data.data;
  }
};

// 订单相关API
export const orderApi = {
  // 获取订单列表
  getOrders: async (params: {
    clusterId?: number;
    strategyId?: number;
    actionType?: string;
    status?: string;
    page?: number;
    pageSize?: number;
  }): Promise<PaginatedResponse<OrderListItem>> => {
    const response = await axios.get(`${API_BASE}/orders`, { params });
    return response.data.data;
  },

  // 获取订单详情
  getOrder: async (id: number): Promise<OrderDetail> => {
    const response = await axios.get(`${API_BASE}/orders/${id}`);
    return response.data.data;
  },

  // 创建订单
  createOrder: async (order: any): Promise<{ id: number }> => {
    const response = await axios.post(`${API_BASE}/orders`, order);
    return response.data.data;
  },

  // 更新订单状态
  updateOrderStatus: async (id: number, status: string, reason?: string): Promise<void> => {
    await axios.put(`${API_BASE}/orders/${id}/status`, { status, reason });
  },

  // 获取订单设备
  getOrderDevices: async (id: number): Promise<Device[]> => {
    const response = await axios.get(`${API_BASE}/orders/${id}/devices`);
    return response.data.data;
  },

  // 更新订单设备状态
  updateOrderDeviceStatus: async (orderId: number, deviceId: number, status: string): Promise<void> => {
    await axios.put(`${API_BASE}/orders/${orderId}/devices/${deviceId}/status`, { status });
  }
};

// 统计相关API
export const statsApi = {
  // 获取工作台统计数据
  getDashboardStats: async (): Promise<DashboardStats> => {
    const response = await fetch('/fe-v1/elastic-scaling/stats/dashboard');
    if (!response.ok) {
      throw new Error(`HTTP error: ${response.status}`);
    }
    const result = await response.json();
    return result.data;
  },

  // 获取资源分配趋势
  getResourceAllocationTrend: async (clusterId: number, timeRange: string, resourceTypes: string | string[] = 'total'): Promise<ResourceAllocationTrend> => {
    // Convert array to comma-separated string if necessary
    const resourceTypesParam = Array.isArray(resourceTypes) ? resourceTypes.join(',') : resourceTypes;
    
    const response = await fetch(`/fe-v1/elastic-scaling/stats/resource-trend?clusterId=${clusterId}&timeRange=${timeRange}&resourceTypes=${resourceTypesParam}`);
    if (!response.ok) {
      throw new Error(`HTTP error: ${response.status}`);
    }
    const result = await response.json();
    return result.data;
  },

  // 获取订单统计
  getOrderStats: async (timeRange: string): Promise<OrderStats> => {
    const response = await fetch(`/fe-v1/elastic-scaling/stats/orders?timeRange=${timeRange}`);
    if (!response.ok) {
      throw new Error(`HTTP error: ${response.status}`);
    }
    const result = await response.json();
    return result.data;
  }
}; 