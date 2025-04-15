import request from '../utils/request';
import { Device, DeviceListResponse, DeviceQuery } from '../types/device';

const BASE_URL = '/device';

// 获取设备列表
export async function getDeviceList(params: DeviceQuery): Promise<DeviceListResponse> {
  return request(`${BASE_URL}`, {
    method: 'GET',
    params,
  });
}

// 下载设备信息Excel
export async function downloadDeviceExcel(): Promise<Blob> {
  return request(`${BASE_URL}/export`, {
    method: 'GET',
    responseType: 'blob',
  }).then(response => response.data);
}

// 获取设备详情
export async function getDeviceDetail(id: string): Promise<Device> {
  return request(`${BASE_URL}/${id}`, {
    method: 'GET',
  });
}

// 更新设备角色
export async function updateDeviceRole(id: number, role: string): Promise<any> {
  return request(`${BASE_URL}/${id}/role`, {
    method: 'PATCH',
    data: { role },
  });
}

// 更新设备用途
export async function updateDeviceGroup(id: number, group: string): Promise<any> {
  return request(`${BASE_URL}/${id}/group`, {
    method: 'PATCH',
    data: { group },
  });
}
