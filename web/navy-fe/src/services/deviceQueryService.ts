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
      { id: 'ci_code', label: '设备编码', value: 'ci_code' },
      { id: 'ip', label: 'IP地址', value: 'ip' },
      { id: 'arch_type', label: 'CPU架构', value: 'arch_type' },
      { id: 'idc', label: 'IDC', value: 'idc' },
      { id: 'room', label: '机房', value: 'room' },
      { id: 'cabinet', label: '所属机柜', value: 'cabinet' },
      { id: 'cabinet_no', label: '机柜编号', value: 'cabinet_no' },
      { id: 'infra_type', label: '网络类型', value: 'infra_type' },
      { id: 'is_localization', label: '是否国产化', value: 'is_localization' },
      { id: 'net_zone', label: '网络区域', value: 'net_zone' },
      { id: 'group', label: '机器类别', value: 'group' },
      { id: 'appid', label: 'APPID', value: 'appid' },
      { id: 'os_create_time', label: '操作系统创建时间', value: 'os_create_time' },
      { id: 'cpu', label: 'CPU数量', value: 'cpu' },
      { id: 'memory', label: '内存大小', value: 'memory' },
      { id: 'model', label: '型号', value: 'model' },
      { id: 'kvm_ip', label: 'KVM IP', value: 'kvm_ip' },
      { id: 'os', label: '操作系统', value: 'os' },
      { id: 'company', label: '厂商', value: 'company' },
      { id: 'os_name', label: '操作系统名称', value: 'os_name' },
      { id: 'os_issue', label: '操作系统版本', value: 'os_issue' },
      { id: 'os_kernel', label: '操作系统内核', value: 'os_kernel' },
      { id: 'status', label: '状态', value: 'status' },
      { id: 'role', label: '角色', value: 'role' },
      { id: 'cluster', label: '所属集群', value: 'cluster' },
      { id: 'cluster_id', label: '集群ID', value: 'cluster_id' }
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

// 获取设备字段值
export async function getDeviceFieldValues(field: string): Promise<string[]> {
  return request(`${BASE_URL}/device-field-values`, {
    method: 'GET',
    params: { field },
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
