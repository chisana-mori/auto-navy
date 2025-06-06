import request from '../utils/request';
import type {
  Order,
  OrderQuery,
  PaginatedOrderResponse,
  OrderStatusUpdateRequest,
  OrderCreateRequest,
  OrderStatistics
} from '../types/order';

// 通用订单服务类
export class OrderService {
  private baseUrl = '/fe-v1/orders';

  /**
   * 获取订单列表
   */
  async getOrders(query: OrderQuery = {}): Promise<PaginatedOrderResponse> {
    const params = new URLSearchParams();
    
    if (query.type) params.append('type', query.type);
    if (query.status) params.append('status', query.status);
    if (query.createdBy) params.append('createdBy', query.createdBy);
    if (query.page) params.append('page', query.page.toString());
    if (query.pageSize) params.append('pageSize', query.pageSize.toString());
    if (query.startTime) params.append('startTime', query.startTime);
    if (query.endTime) params.append('endTime', query.endTime);

    const response = await request.get(`${this.baseUrl}?${params.toString()}`);
    return response.data;
  }

  /**
   * 根据ID获取订单详情
   */
  async getOrderById(id: number): Promise<Order> {
    const response = await request.get(`${this.baseUrl}/${id}`);
    return response.data;
  }

  /**
   * 创建订单
   */
  async createOrder(orderData: OrderCreateRequest): Promise<{ id: number }> {
    const response = await request.post(this.baseUrl, orderData);
    return response.data;
  }

  /**
   * 更新订单状态
   */
  async updateOrderStatus(id: number, statusUpdate: OrderStatusUpdateRequest): Promise<void> {
    await request.put(`${this.baseUrl}/${id}/status`, statusUpdate);
  }

  /**
   * 删除订单
   */
  async deleteOrder(id: number): Promise<void> {
    await request.delete(`${this.baseUrl}/${id}`);
  }

  /**
   * 批量更新订单状态
   */
  async batchUpdateOrderStatus(ids: number[], statusUpdate: OrderStatusUpdateRequest): Promise<void> {
    const promises = ids.map(id => this.updateOrderStatus(id, statusUpdate));
    await Promise.all(promises);
  }

  /**
   * 获取订单统计信息
   */
  async getOrderStatistics(startTime?: string, endTime?: string): Promise<OrderStatistics> {
    const params = new URLSearchParams();
    if (startTime) params.append('startTime', startTime);
    if (endTime) params.append('endTime', endTime);

    const response = await request.get(`${this.baseUrl}/statistics?${params.toString()}`);
    return response.data;
  }

  /**
   * 导出订单数据
   */
  async exportOrders(query: OrderQuery = {}): Promise<Blob> {
    const params = new URLSearchParams();
    
    if (query.type) params.append('type', query.type);
    if (query.status) params.append('status', query.status);
    if (query.createdBy) params.append('createdBy', query.createdBy);
    if (query.startTime) params.append('startTime', query.startTime);
    if (query.endTime) params.append('endTime', query.endTime);

    const response = await request.get(`${this.baseUrl}/export?${params.toString()}`, {
      responseType: 'blob'
    });
    return response.data;
  }

  /**
   * 获取订单操作日志
   */
  async getOrderLogs(id: number): Promise<any[]> {
    const response = await request.get(`${this.baseUrl}/${id}/logs`);
    return response.data;
  }
}

// 创建单例实例
export const orderService = new OrderService();

// 导出默认实例
export default orderService;
