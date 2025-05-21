import request from '../utils/request';

// 获取查询模板列表
export async function getQueryTemplates(page = 1, size = 10) {
  const response = await request(`/device-query/templates?page=${page}&size=${size}`, {
    method: 'GET',
  });
  return response.data || response;
}

// 获取查询模板详情
export async function getQueryTemplate(id: number) {
  const response = await request(`/device-query/templates/${id}`, {
    method: 'GET',
  });
  return response.data || response;
}

// 创建查询模板
export async function createQueryTemplate(template: any) {
  const response = await request('/device-query/templates', {
    method: 'POST',
    data: template,
  });
  return response.data || response;
}

// 更新查询模板
export async function updateQueryTemplate(template: any) {
  const response = await request(`/device-query/templates/${template.id}`, {
    method: 'PUT',
    data: template,
  });
  return response.data || response;
}

// 删除查询模板
export async function deleteQueryTemplate(id: number) {
  const response = await request(`/device-query/templates/${id}`, {
    method: 'DELETE',
  });
  return response.data || response;
}
