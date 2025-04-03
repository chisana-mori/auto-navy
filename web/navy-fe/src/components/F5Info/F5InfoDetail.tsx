import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, Descriptions, Button, message, List, Tag, Badge, Space, Divider } from 'antd';
import { 
  CheckCircleFilled, 
  CloseCircleFilled, 
  WarningFilled, 
  CloudServerOutlined, 
  RollbackOutlined, 
  DeleteOutlined,
  InfoCircleOutlined,
  GlobalOutlined,
  ApiOutlined
} from '@ant-design/icons';
import request from '../../utils/request';
import type { F5Info } from '../../types/f5';

// 格式化时间为 yyyy-MM-dd HH:mm:ss
const formatDateTime = (dateString: string) => {
  if (!dateString) return '';
  
  try {
    const date = new Date(dateString);
    
    // 检查日期是否有效
    if (isNaN(date.getTime())) {
      return dateString;
    }
    
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');
    
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
  } catch (error) {
    console.error('日期格式化错误:', error);
    return dateString;
  }
};

// 获取状态对应的图标和颜色
const getStatusInfo = (status: string) => {
  const lowerStatus = status.toLowerCase();
  if (lowerStatus === 'active' || lowerStatus === 'online' || lowerStatus === 'running' || lowerStatus === 'healthy') {
    return {
      icon: <CheckCircleFilled style={{ color: '#52c41a' }} />,
      color: '#f6ffed',
      textColor: '#52c41a',
      tagColor: 'success',
      badgeStatus: 'success' as const
    };
  } else if (lowerStatus === 'inactive' || lowerStatus === 'offline' || lowerStatus === 'stopped') {
    return {
      icon: <CloseCircleFilled style={{ color: '#ff4d4f' }} />,
      color: '#fff1f0',
      textColor: '#ff4d4f',
      tagColor: 'error',
      badgeStatus: 'error' as const
    };
  } else if (lowerStatus === 'degraded' || lowerStatus === 'warning') {
    return {
      icon: <WarningFilled style={{ color: '#faad14' }} />,
      color: '#fffbe6',
      textColor: '#faad14',
      tagColor: 'warning',
      badgeStatus: 'warning' as const
    };
  }
  
  return {
    icon: null,
    color: 'transparent',
    textColor: 'inherit',
    tagColor: 'default' as const,
    badgeStatus: 'default' as const
  };
};

// 定义Pool成员的接口
interface PoolMember {
  ip: string;
  status: string;
}

const F5InfoDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<F5Info | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      if (!id) return;
      setLoading(true);
      try {
        const response = await request.get<any, F5Info>(`/f5/${id}`);
        setData(response);
      } catch (error) {
        console.error('获取详情失败:', error);
        message.error('获取详情失败');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [id]);

  const handleDelete = async () => {
    if (!id) return;
    try {
      await request.delete(`/f5/${id}`);
      message.success('删除成功');
      navigate('/f5');
    } catch (error) {
      console.error('删除失败:', error);
      message.error('删除失败');
    }
  };

  // 处理忽略/取消忽略操作
  const handleToggleIgnore = async (ignore: boolean) => {
    if (!id || !data) return;
    try {
      // 构建更新数据，保留所有必填字段
      const updateData = {
        ignored: ignore,
        name: data.name,
        vip: data.vip,
        port: data.port,
        appid: data.appid,
        // 保留其他原始值，避免后端验证失败
        instance_group: data.instance_group,
        status: data.status,
        pool_name: data.pool_name,
        pool_status: data.pool_status,
        pool_members: data.pool_members,
        k8s_cluster_id: data.k8s_cluster_id,
        domains: data.domains,
        grafana_params: data.grafana_params
      };

      await request.put(`/f5/${id}`, updateData);
      message.success(ignore ? '已忽略' : '已取消忽略');
      
      // 重新获取数据
      const response = await request.get<any, F5Info>(`/f5/${id}`);
      setData(response);
    } catch (error) {
      console.error(ignore ? '忽略失败:' : '取消忽略失败:', error);
      message.error(ignore ? '忽略失败' : '取消忽略失败');
    }
  };

  // 将逗号分隔的字符串转换为数组，并解析每个成员的IP和状态
  const parsePoolMembers = (poolMembers: string): PoolMember[] => {
    if (!poolMembers) return [];
    // 先按逗号分隔，然后处理每项
    return poolMembers.split(',').filter(item => item.trim()).map(item => {
      // 假设格式为 "IP online" 或 "IP offline"
      const parts = item.trim().split(' ');
      const ip = parts[0];
      const status = parts.length > 1 ? parts[1].toLowerCase() : 'unknown';
      return { ip, status };
    });
  };

  // 将逗号分隔的域名转换为数组
  const parseDomains = (domains: string): string[] => {
    if (!domains) return [];
    return domains.split(',').filter(item => item.trim());
  };

  if (loading) {
    return (
      <Card className="f5-info-detail-loading-card">
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
          加载中...
        </div>
      </Card>
    );
  }

  if (!data) {
    return (
      <Card className="f5-info-detail-empty-card">
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
          未找到数据
        </div>
      </Card>
    );
  }

  const poolMembers = parsePoolMembers(data.pool_members);
  const domains = parseDomains(data.domains);
  const statusInfo = getStatusInfo(data.status);
  const poolStatusInfo = getStatusInfo(data.pool_status);

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <CloudServerOutlined style={{ fontSize: '20px', color: '#1677ff', marginRight: '12px' }} />
          <span>F5 详情</span>
          <Badge status={statusInfo.badgeStatus} text={data.name} style={{ marginLeft: 12 }} />
        </div>
      }
      extra={
        <Space>
          <Button 
            icon={<RollbackOutlined />} 
            onClick={() => navigate('/f5')}
          >
            返回列表
          </Button>
          {data.ignored ? (
            <Button 
              type="default"
              onClick={() => handleToggleIgnore(false)}
              icon={<CheckCircleFilled />}
            >
              取消忽略
            </Button>
          ) : (
            <Button 
              type="default"
              onClick={() => handleToggleIgnore(true)}
              icon={<CloseCircleFilled />}
            >
              忽略
            </Button>
          )}
          <Button 
            type="primary" 
            danger 
            icon={<DeleteOutlined />} 
            onClick={handleDelete}
          >
            删除
          </Button>
        </Space>
      }
      className={data.ignored ? 'ignored-detail-card' : ''}
      style={{ 
        backgroundColor: data.ignored ? '#fffbe6' : 'white',
        borderTop: `2px solid ${statusInfo.textColor}`
      }}
    >
      <Divider orientation="left">
        <Space>
          <InfoCircleOutlined />
          <span>基本信息</span>
        </Space>
      </Divider>
      
      <div style={{ display: 'table', tableLayout: 'fixed', width: '100%' }}>
        <Descriptions 
          bordered 
          column={3}
          size="middle"
          style={{ width: '100%' }}
          labelStyle={{ width: '15%', textAlign: 'right' }}
          contentStyle={{ width: '18.33%' }}
        >
          <Descriptions.Item label="名称">
            <span className="field-highlight">{data.name}</span>
          </Descriptions.Item>
          <Descriptions.Item label="VIP">{data.vip}</Descriptions.Item>
          <Descriptions.Item label="端口">{data.port}</Descriptions.Item>
          <Descriptions.Item label="appid">{data.appid}</Descriptions.Item>
          <Descriptions.Item label="实例组">{data.instance_group}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={statusInfo.tagColor} icon={statusInfo.icon}>
              {data.status}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="忽略状态">
            <Tag color={data.ignored ? 'default' : 'green'}>
              {data.ignored ? '是' : '否'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="创建时间">{formatDateTime(data.created_at)}</Descriptions.Item>
          <Descriptions.Item label="更新时间">{formatDateTime(data.updated_at)}</Descriptions.Item>
          <Descriptions.Item label="域名" span={3}>
            {domains.length > 0 ? (
              <List
                size="small"
                bordered
                dataSource={domains}
                renderItem={(domain) => (
                  <List.Item>
                    <Badge status="processing" text={domain} />
                  </List.Item>
                )}
                style={{ backgroundColor: 'white' }}
              />
            ) : (
              '无域名'
            )}
          </Descriptions.Item>
        </Descriptions>
      </div>
      
      <Divider orientation="left">
        <Space>
          <GlobalOutlined />
          <span>Pool 信息</span>
        </Space>
      </Divider>
      
      <div style={{ display: 'table', tableLayout: 'fixed', width: '100%' }}>
        <Descriptions 
          bordered 
          column={3}
          size="middle"
          style={{ width: '100%' }}
          labelStyle={{ width: '15%', textAlign: 'right' }}
          contentStyle={{ width: '18.33%' }}
        >
          <Descriptions.Item label="Pool名称">{data.pool_name}</Descriptions.Item>
          <Descriptions.Item label="Pool状态" span={2}>
            <Tag color={poolStatusInfo.tagColor} icon={poolStatusInfo.icon}>
              {data.pool_status}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Pool成员" span={3}>
            {poolMembers.length > 0 ? (
              <List
                size="small"
                bordered
                dataSource={poolMembers}
                renderItem={(member) => (
                  <List.Item>
                    <Badge 
                      status={member.status === 'online' ? 'success' : 'error'} 
                      text={`${member.ip} ${member.status}`} 
                    />
                  </List.Item>
                )}
                style={{ backgroundColor: 'white' }}
              />
            ) : (
              '无成员'
            )}
          </Descriptions.Item>
        </Descriptions>
      </div>
      
      <Divider orientation="left">
        <Space>
          <ApiOutlined />
          <span>关联信息</span>
        </Space>
      </Divider>
      
      <div style={{ display: 'table', tableLayout: 'fixed', width: '100%' }}>
        <Descriptions 
          bordered 
          column={2}
          size="middle"
          style={{ width: '100%' }}
          labelStyle={{ width: '30%', textAlign: 'right' }}
          contentStyle={{ width: '70%' }}
        >
          <Descriptions.Item label="K8s集群名称">{data.k8s_cluster_name || '未知集群'}</Descriptions.Item>
        </Descriptions>
      </div>
    </Card>
  );
};

export default F5InfoDetail; 