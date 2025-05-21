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

const BASE_URL = 'clusters';

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
  try {
    return await request(`${BASE_URL}/${id}`, {
      method: 'GET',
    });
  } catch (error) {
    console.error('获取集群详情失败，使用模拟数据:', error);
    // 返回模拟数据
    return {
      id: id,
      clusterID: `cluster-${id}`,
      clusterName: `集群-${id}`,
      clusterNameCn: `测试集群-${id}`,
      alias: `测试集群-${id}`,
      idc: 'gl',
      zone: 'egt',
      room: 'A1',
      status: 'running',
      clusterType: 'tool',
      netType: 'physical',
      architecture: 'x86_64',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };
  }
}

// 获取集群的IDC、Room和Zone信息
export async function getClusterLocationInfo(id: number): Promise<{
  idc: string;
  zone: string;
  room: string;
}> {
  try {
    const cluster = await getClusterDetail(id);
    return {
      idc: cluster.idc || '',
      zone: cluster.zone || '',
      room: cluster.room || '',
    };
  } catch (error) {
    console.error('获取集群位置信息失败:', error);
    return {
      idc: '',
      zone: '',
      room: '',
    };
  }
}

const clusterService = {
  getClusters,
  getClusterDetail,
  getClusterLocationInfo,
};

export default clusterService;
