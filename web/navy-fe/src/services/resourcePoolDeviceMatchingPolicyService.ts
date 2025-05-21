import axios from 'axios';
import { FilterGroup } from '../types/deviceQuery';

// 查询模板类型
export interface QueryTemplate {
  id: number;
  name: string;
  description: string;
  groups: FilterGroup[];
}

// 资源池设备匹配策略类型
export interface ResourcePoolDeviceMatchingPolicy {
  id?: number;
  name: string;
  description: string;
  resourcePoolType: string;
  actionType: 'pool_entry' | 'pool_exit';
  queryTemplateId: number;
  queryGroups?: FilterGroup[];  // 从查询模板获取，非直接存储字段
  queryTemplate?: QueryTemplate; // 关联的查询模板
  status: 'enabled' | 'disabled';
  additionConds?: string[];     // 额外动态条件，仅入池时有效
  createdBy?: string;
  updatedBy?: string;
  createdAt?: string;
  updatedAt?: string;
}

// 分页响应类型
export interface PaginatedResponse<T> {
  list: T[];
  total: number;
  page: number;
  size: number;
}

// 获取资源池设备匹配策略列表
export const getResourcePoolDeviceMatchingPolicies = async (page = 1, size = 10): Promise<PaginatedResponse<ResourcePoolDeviceMatchingPolicy>> => {
  try {
    const response = await axios.get('/api/v1/resource-pool/matching-policies', {
      params: { page, size }
    });
    return response.data.data;
  } catch (error) {
    console.error('获取资源池设备匹配策略列表失败:', error);
    throw error;
  }
};

// 获取资源池设备匹配策略详情
export const getResourcePoolDeviceMatchingPolicy = async (id: number): Promise<ResourcePoolDeviceMatchingPolicy> => {
  try {
    const response = await axios.get(`/api/v1/resource-pool/matching-policies/${id}`);
    return response.data.data;
  } catch (error) {
    console.error('获取资源池设备匹配策略详情失败:', error);
    throw error;
  }
};

// 创建资源池设备匹配策略
export const createResourcePoolDeviceMatchingPolicy = async (policy: ResourcePoolDeviceMatchingPolicy): Promise<ResourcePoolDeviceMatchingPolicy> => {
  try {
    const response = await axios.post('/api/v1/resource-pool/matching-policies', policy);
    return response.data.data;
  } catch (error) {
    console.error('创建资源池设备匹配策略失败:', error);
    throw error;
  }
};

// 更新资源池设备匹配策略
export const updateResourcePoolDeviceMatchingPolicy = async (policy: ResourcePoolDeviceMatchingPolicy): Promise<ResourcePoolDeviceMatchingPolicy> => {
  try {
    const response = await axios.put(`/api/v1/resource-pool/matching-policies/${policy.id}`, policy);
    return response.data.data;
  } catch (error) {
    console.error('更新资源池设备匹配策略失败:', error);
    throw error;
  }
};

// 更新资源池设备匹配策略状态
export const updateResourcePoolDeviceMatchingPolicyStatus = async (id: number, status: string): Promise<void> => {
  try {
    await axios.put(`/api/v1/resource-pool/matching-policies/${id}/status`, { status });
  } catch (error) {
    console.error('更新资源池设备匹配策略状态失败:', error);
    throw error;
  }
};

// 删除资源池设备匹配策略
export const deleteResourcePoolDeviceMatchingPolicy = async (id: number): Promise<void> => {
  try {
    await axios.delete(`/api/v1/resource-pool/matching-policies/${id}`);
  } catch (error) {
    console.error('删除资源池设备匹配策略失败:', error);
    throw error;
  }
};

// 根据资源池类型和动作类型获取匹配策略
export const getResourcePoolDeviceMatchingPoliciesByType = async (resourcePoolType: string, actionType: string): Promise<ResourcePoolDeviceMatchingPolicy[]> => {
  try {
    const response = await axios.get('/api/v1/resource-pool/matching-policies/by-type', {
      params: { resourcePoolType, actionType }
    });
    return response.data.data;
  } catch (error) {
    console.error('根据类型获取资源池设备匹配策略失败:', error);
    throw error;
  }
};
