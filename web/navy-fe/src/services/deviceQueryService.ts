import request from '../utils/request';
import { DeviceListResponse } from '../types/device';
import {
  FilterOption,
  DeviceQueryRequest,
  QueryTemplate
} from '../types/deviceQuery';

const BASE_URL = '/device-query';

// 获取筛选选项
export async function getFilterOptions(): Promise<Record<string, FilterOption[]>> {
  const response = await request<Record<string, FilterOption[]>>(`${BASE_URL}/filter-options`, {
    method: 'GET',
  });

  // 直接使用响应数据，因为响应拦截器已经处理过了
  const result = (response as unknown) as Record<string, FilterOption[]>;

  // 确保设备字段选项存在
  if (!result.deviceFields) {
    result.deviceFields = [
      { id: 'ip', label: 'IP地址', value: 'ip' },
      { id: 'machineType', label: '机器类型', value: 'machineType' },
      { id: 'role', label: '集群角色', value: 'role' },
      { id: 'arch', label: '架构', value: 'arch' },
      { id: 'idc', label: 'IDC', value: 'idc' },
      { id: 'room', label: 'Room', value: 'room' },
      { id: 'datacenter', label: '机房', value: 'datacenter' },
      { id: 'cabinet', label: '机柜号', value: 'cabinet' },
      { id: 'network', label: '网络区域', value: 'network' },
      { id: 'appId', label: 'APPID', value: 'appId' },
      { id: 'resourcePool', label: '资源池/产品', value: 'resourcePool' }
    ];
  }

  console.log('处理后的筛选选项:', result);
  return result;
}

// 获取标签值
export async function getLabelValues(key: string): Promise<string[]> {
  return request(`${BASE_URL}/label-values`, {
    method: 'GET',
    params: { key },
  });
}

// 获取污点值
export async function getTaintValues(key: string): Promise<string[]> {
  return request(`${BASE_URL}/taint-values`, {
    method: 'GET',
    params: { key },
  });
}

// 查询设备
export async function queryDevices(params: DeviceQueryRequest): Promise<DeviceListResponse> {
  return request(`${BASE_URL}/query`, {
    method: 'POST',
    data: params,
  });
}

// 保存查询模板
export async function saveQueryTemplate(template: QueryTemplate): Promise<any> {
  return request(`${BASE_URL}/templates`, {
    method: 'POST',
    data: template,
  });
}

// 获取查询模板列表
export async function getQueryTemplates(): Promise<QueryTemplate[]> {
  return request(`${BASE_URL}/templates`, {
    method: 'GET',
  });
}

// 获取查询模板
export async function getQueryTemplate(id: number | string): Promise<QueryTemplate> {
  return request(`${BASE_URL}/templates/${id}`, {
    method: 'GET',
  });
}

// 删除查询模板
export async function deleteQueryTemplate(id: number | string): Promise<any> {
  return request(`${BASE_URL}/templates/${id}`, {
    method: 'DELETE',
  });
}
