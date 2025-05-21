import request from '../utils/request';
import { DeviceListResponse } from '../types/device';
import {
  FilterOption,
  DeviceQueryRequest,
  QueryTemplate,
  QueryTemplateListResponse
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
      { id: 'cluster_id', label: '集群ID', value: 'cluster_id' },
      { id: 'is_special', label: '特殊设备', value: 'is_special' }
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

// 标签详情
export interface LabelDetail {
  key: string;   // 标签键
  value: string; // 标签值
}

// 污点详情
export interface TaintDetail {
  key: string;    // 污点键
  value: string;  // 污点值
  effect: string; // 效果
}

// 设备特性详情
export interface DeviceFeatureDetails {
  labels: LabelDetail[]; // 标签详情
  taints: TaintDetail[]; // 污点详情
}

// 获取设备特性详情
export async function getDeviceFeatureDetails(ciCode: string): Promise<DeviceFeatureDetails> {
  return request(`${BASE_URL}/device-feature-details`, {
    method: 'GET',
    params: { ci_code: ciCode },
  });
}

// 查询设备
export async function queryDevices(params: DeviceQueryRequest): Promise<DeviceListResponse> {
  console.log('发送设备查询请求, 参数:', JSON.stringify(params));

  try {
    // 确保参数有效
    if (!params.groups || !Array.isArray(params.groups)) {
      console.error('查询参数无效: groups 不是数组');
      throw new Error('查询参数无效');
    }

    // 检查每个组和块的有效性
    params.groups.forEach((group, groupIndex) => {
      if (!group || typeof group !== 'object') {
        console.error(`查询参数无效: 组 ${groupIndex} 不是有效对象`);
      } else if (!Array.isArray(group.blocks)) {
        console.error(`查询参数无效: 组 ${groupIndex} 的 blocks 不是数组`);
      } else {
        group.blocks.forEach((block, blockIndex) => {
          if (!block || typeof block !== 'object') {
            console.error(`查询参数无效: 组 ${groupIndex} 的块 ${blockIndex} 不是有效对象`);
          } else {
            // 检查关键字段
            if (!block.type) {
              console.error(`查询参数无效: 组 ${groupIndex} 的块 ${blockIndex} 缺少 type 字段`);
            }
            if (!block.conditionType) {
              console.error(`查询参数无效: 组 ${groupIndex} 的块 ${blockIndex} 缺少 conditionType 字段`);
            }
            if (!block.field && !block.key) {
              console.error(`查询参数无效: 组 ${groupIndex} 的块 ${blockIndex} 缺少 field 和 key 字段`);
            }

            // 确保 key 和 field 字段同步
            if (block.field && !block.key) {
              console.log(`修复查询参数: 组 ${groupIndex} 的块 ${blockIndex} 设置 key = field`);
              block.key = block.field;
            } else if (block.key && !block.field) {
              console.log(`修复查询参数: 组 ${groupIndex} 的块 ${blockIndex} 设置 field = key`);
              block.field = block.key;
            }
          }
        });
      }
    });

    // 清理参数中的无效数据
    const cleanedParams = {
      ...params,
      groups: params.groups.filter(group =>
        group && typeof group === 'object' && Array.isArray(group.blocks) && group.blocks.length > 0
      )
    };

    if (cleanedParams.groups.length === 0) {
      console.error('清理后的查询参数无效: 没有有效的筛选组');
      throw new Error('查询参数无效: 没有有效的筛选组');
    }

    console.log('清理后的查询参数:', JSON.stringify(cleanedParams));

    const response = await request(`${BASE_URL}/query`, {
      method: 'POST',
      data: cleanedParams,
    });

    console.log('查询响应:', response);

    // 将响应转换为正确的类型
    const responseData = response as any;

    // 确保响应符合 DeviceListResponse 类型
    const result: DeviceListResponse = {
      list: Array.isArray(responseData.list) ? responseData.list : [],
      total: typeof responseData.total === 'number' ? responseData.total : 0,
      page: typeof responseData.page === 'number' ? responseData.page : 1,
      size: typeof responseData.size === 'number' ? responseData.size : 10
    };

    return result;
  } catch (error) {
    console.error('查询设备失败:', error);
    throw error;
  }
}

// 保存查询模板
export async function saveQueryTemplate(template: QueryTemplate): Promise<any> {
  return request(`${BASE_URL}/templates`, {
    method: 'POST',
    data: template,
  });
}

// 获取查询模板列表
export async function getQueryTemplates(params?: { page?: number; size?: number }): Promise<QueryTemplateListResponse> {
  try {
    console.log('获取模板列表', params);

    // 构建查询参数
    const queryParams: Record<string, string | number> = {};
    if (params?.page) {
      queryParams.page = params.page;
    }
    if (params?.size) {
      queryParams.size = params.size;
    }

    const response = await request(`${BASE_URL}/templates`, {
      method: 'GET',
      params: queryParams,
    });

    console.log('模板列表原始响应:', response);

    // 如果返回的是带分页结构的响应
    if (response && typeof response === 'object' && 'list' in response) {
      // 使用类型断言告诉TypeScript这个对象有QueryTemplateListResponse的结构
      const responseData = response as unknown as Partial<QueryTemplateListResponse>;

      // 处理模板列表
      const list = responseData.list || [];
      const processedTemplates = Array.isArray(list) ? list.map(processTemplate) : [];

      return {
        list: processedTemplates,
        total: typeof responseData.total === 'number' ? responseData.total : 0,
        page: typeof responseData.page === 'number' ? responseData.page : 1,
        size: typeof responseData.size === 'number' ? responseData.size : 10
      };
    }

    // 兼容旧版返回格式（直接返回数组）
    if (Array.isArray(response)) {
      const processedTemplates = response.map(processTemplate);

      return {
        list: processedTemplates,
        total: processedTemplates.length,
        page: params?.page || 1,
        size: params?.size || 10
      };
    }

    console.warn('模板列表响应格式不符合预期');
    return { list: [], total: 0, page: 1, size: 10 };
  } catch (error) {
    console.error('获取模板列表失败:', error);
    throw error;
  }
}

// 处理模板数据，确保格式正确
function processTemplate(template: any): QueryTemplate {
  // 如果 groups 是字符串，尝试解析为 JSON
  if (typeof template.groups === 'string') {
    try {
      template.groups = JSON.parse(template.groups);
    } catch (error) {
      console.error(`Failed to parse template groups for template ${template.id}:`, error);
      template.groups = [];
    }
  }

  // 如果 groups 不是数组，初始化为空数组
  if (!Array.isArray(template.groups)) {
    console.warn(`Template ${template.id} groups is not an array, initializing as empty array`);
    template.groups = [];
  }

  // 确保每个组和块都有有效的 ID
  template.groups = template.groups.map((group: any) => {
    // 如果组没有 ID，生成一个
    if (!group.id) {
      group.id = generateUUID();
    }

    // 确保 blocks 是数组
    if (!Array.isArray(group.blocks)) {
      group.blocks = [];
    } else {
      // 处理每个块
      group.blocks = group.blocks.map((block: any) => {
        // 如果块没有 ID，生成一个
        if (!block.id) {
          block.id = generateUUID();
        }

        // 确保 key 和 field 字段同步
        if (block.field && !block.key) {
          block.key = block.field;
        } else if (block.key && !block.field) {
          block.field = block.key;
        }

        return block;
      });
    }
    return group;
  });

  console.log(`Processed template ${template.id}:`, template);
  return template;
}

// 获取查询模板
export async function getQueryTemplate(id: number | string): Promise<QueryTemplate> {
  try {
    console.log(`获取模板数据, id: ${id}`);
    const response = await request(`${BASE_URL}/templates/${id}`, {
      method: 'GET',
    });

    console.log(`模板原始响应:`, response);

    // 将响应转换为 QueryTemplate 类型
    const template = response as unknown as QueryTemplate;

    // 如果 groups 是字符串，尝试解析为 JSON
    if (typeof template.groups === 'string') {
      try {
        template.groups = JSON.parse(template.groups);
        console.log('解析后的 groups:', template.groups);
      } catch (error) {
        console.error('Failed to parse template groups:', error);
        template.groups = [];
      }
    }

    // 如果 groups 不是数组，初始化为空数组
    if (!Array.isArray(template.groups)) {
      console.warn('Template groups is not an array, initializing as empty array');
      template.groups = [];
    }

    // 确保每个组和块都有有效的 ID
    template.groups = template.groups.map((group: any) => {
      // 如果组没有 ID，生成一个
      if (!group.id) {
        group.id = generateUUID();
      }

      // 确保 blocks 是数组
      if (!Array.isArray(group.blocks)) {
        group.blocks = [];
      } else {
        // 处理每个块
        group.blocks = group.blocks.map((block: any) => {
          // 如果块没有 ID，生成一个
          if (!block.id) {
            block.id = generateUUID();
          }

          // 确保 key 和 field 字段同步
          if (block.field && !block.key) {
            block.key = block.field;
          } else if (block.key && !block.field) {
            block.field = block.key;
          }

          return block;
        });
      }
      return group;
    });

    console.log('Processed single template:', template);
    return template;
  } catch (error) {
    console.error(`获取模板失败, id: ${id}`, error);
    throw error;
  }
}

// 生成 UUID
function generateUUID() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : ((r & 0x3) | 0x8);
    return v.toString(16);
  });
}

// 删除查询模板
export async function deleteQueryTemplate(id: number | string): Promise<any> {
  return request(`${BASE_URL}/templates/${id}`, {
    method: 'DELETE',
  });
}
