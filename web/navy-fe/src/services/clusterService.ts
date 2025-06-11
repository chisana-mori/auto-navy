import request from '../utils/request';

export interface Cluster {
  id: number;
  clusterID: string;
  clusterName: string;
  clusterNameCn: string;
  alias: string;
  idc: string;
  zone: string;
  room: string;
  status: string;
  clusterType: string;
  netType: string;
  architecture: string;
  createdAt: string;
  updatedAt: string;
}

export interface ClusterListResponse {
  list: Cluster[];
  total: number;
  page: number;
  size: number;
}

const BASE_URL = 'k8s-clusters';

// 获取集群列表
export async function getClusters(params?: {
  page?: number;
  size?: number;
  keyword?: string;
  status?: string;
}): Promise<ClusterListResponse> {
  return request(BASE_URL, {
    method: 'GET',
    params,
  });
}

// 获取集群详情
export async function getClusterDetail(id: number): Promise<Cluster> {
  return await request(`${BASE_URL}/${id}`, {
    method: 'GET',
  });
}

// 获取集群的IDC、Room和Zone信息
export async function getClusterLocationInfo(id: number): Promise<{
  idc: string;
  room: string;
  zone: string;
}> {
  return await request(`${BASE_URL}/${id}/location`, {
    method: 'GET',
  });
}

// 获取集群资源信息
export async function getClusterResources(params?: {
  page?: number;
  size?: number;
  keyword?: string;
  idc?: string;
  zone?: string;
}): Promise<any> {
  return request('cluster-resources', {
    method: 'GET',
    params,
  });
}

// 资源池分配率响应接口
export interface ResourcePoolAllocationRate {
  cluster_name: string;
  resource_pool: string;
  cpu_rate: number;
  memory_rate: number;
  cpu_request: number;
  cpu_capacity: number;
  memory_request: number;
  memory_capacity: number;
  query_date: string;
}

// 获取资源池分配率
export async function getResourcePoolAllocationRate(
  clusterName: string,
  resourcePool: string
): Promise<ResourcePoolAllocationRate | null> {
  try {
    return await request('cluster-resources/allocation-rate', {
      method: 'GET',
      params: {
        clusterName: clusterName,
        resourcePool: resourcePool,
      },
    });
  } catch (error) {
    // 如果返回404或其他错误，表示没有数据，返回null
    console.warn('Failed to get resource pool allocation rate:', error);
    return null;
  }
}

const clusterService = {
  getClusters,
  getClusterDetail,
  getClusterLocationInfo,
  getClusterResources,
  getResourcePoolAllocationRate,
};

export default clusterService;
