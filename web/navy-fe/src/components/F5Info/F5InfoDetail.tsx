import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, Button, message, Tag } from 'antd';
import request from '../../utils/request';
import type { F5Info } from '../../types/f5';

// 图标导入
import { 
  CloudServerOutlined, 
  InfoCircleOutlined, 
  RollbackOutlined, 
  GlobalOutlined, 
  ApiOutlined, 
  CheckCircleFilled, 
  CloseCircleFilled, 
  WarningFilled, 
  DeleteOutlined
} from '@ant-design/icons';

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
      color: 'green',
      tagColor: 'success'
    };
  } else if (lowerStatus === 'inactive' || lowerStatus === 'offline' || lowerStatus === 'stopped') {
    return {
      icon: <CloseCircleFilled style={{ color: '#ff4d4f' }} />,
      color: 'red',
      tagColor: 'error'
    };
  } else if (lowerStatus === 'degraded' || lowerStatus === 'warning') {
    return {
      icon: <WarningFilled style={{ color: '#faad14' }} />,
      color: 'yellow',
      tagColor: 'warning'
    };
  }
  
  return {
    icon: null,
    color: 'default',
    tagColor: 'default'
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
      <Card style={{ width: '100%' }}>
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
          加载中...
        </div>
      </Card>
    );
  }

  if (!data) {
    return (
      <Card style={{ width: '100%' }}>
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
          未找到数据
        </div>
      </Card>
    );
  }

  const domains = parseDomains(data.domains);
  const poolMembers = parsePoolMembers(data.pool_members);
  const statusInfo = getStatusInfo(data.status);
  const poolStatusInfo = getStatusInfo(data.pool_status);

  // 字段项组件
  const FieldItem = ({ label, value, className = "" }: { label: string; value: React.ReactNode; className?: string }) => {
    return (
      <div className={`border rounded-md bg-white p-4 ${className}`} style={{ border: '1px solid #f0f0f0', borderRadius: '4px', padding: '16px' }}>
        <div style={{ color: '#606060', fontSize: '14px', marginBottom: '4px' }}>{label}</div>
        <div style={{ fontWeight: 500 }}>{value}</div>
      </div>
    );
  };

  return (
    <div style={{ marginBottom: '24px' }}>
      <Card 
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <CloudServerOutlined style={{ fontSize: '20px', color: '#1890ff', marginRight: '12px' }} />
            <span style={{ fontSize: '18px', fontWeight: 500 }}>F5 详情</span>
            <Tag color="success" style={{ marginLeft: '12px' }}>
              {data.name}
            </Tag>
          </div>
        }
        extra={
          <div style={{ display: 'flex', gap: '8px' }}>
            <Button 
              icon={<RollbackOutlined />} 
              onClick={() => navigate('/f5')}
            >
              返回列表
            </Button>
            <Button 
              onClick={() => handleToggleIgnore(!data.ignored)}
            >
              {data.ignored ? "取消忽略" : "忽略"}
            </Button>
            <Button 
              type="primary" 
              danger 
              icon={<DeleteOutlined />} 
              onClick={handleDelete}
            >
              删除
            </Button>
          </div>
        }
        style={{ width: '100%' }}
      >
        <div style={{ marginBottom: '24px' }}>
          <h3 style={{ display: 'flex', alignItems: 'center', marginBottom: '16px', fontSize: '16px', fontWeight: 500 }}>
            <InfoCircleOutlined style={{ marginRight: '8px', color: '#1890ff' }} />
            基本信息
          </h3>
          
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '16px' }}>
            <FieldItem label="名称" value={data.name} />
            <FieldItem label="VIP" value={data.vip} />
            <FieldItem label="端口" value={data.port} />
            <FieldItem label="appid" value={data.appid} />
            <FieldItem label="实例组" value={data.instance_group} />
            <FieldItem 
              label="状态" 
              value={
                <Tag color={statusInfo.tagColor}>
                  <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                    {statusInfo.icon}
                    <span>{data.status}</span>
                  </span>
                </Tag>
              } 
            />
            <FieldItem 
              label="忽略状态" 
              value={
                <span style={{ background: '#f5f5f5', padding: '2px 8px', borderRadius: '2px', fontSize: '14px' }}>
                  {data.ignored ? '是' : '否'}
                </span>
              } 
            />
            <FieldItem label="创建时间" value={formatDateTime(data.created_at)} />
            <FieldItem label="更新时间" value={formatDateTime(data.updated_at)} />
          </div>
          
          {domains.length > 0 && (
            <div style={{ marginTop: '16px' }}>
              <h4 style={{ fontSize: '15px', fontWeight: 500, marginBottom: '8px' }}>域名</h4>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                {domains.map((domain, index) => (
                  <div key={index} style={{ display: 'flex', alignItems: 'center', gap: '8px', border: '1px solid #f0f0f0', padding: '8px', borderRadius: '4px', background: 'white' }}>
                    <GlobalOutlined style={{ color: '#8c8c8c' }} />
                    <span>{domain}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
        
        <div style={{ marginBottom: '24px' }}>
          <h3 style={{ display: 'flex', alignItems: 'center', marginBottom: '16px', fontSize: '16px', fontWeight: 500 }}>
            <ApiOutlined style={{ marginRight: '8px', color: '#1890ff' }} />
            Pool 信息
          </h3>
          
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '16px', marginBottom: '16px' }}>
            <FieldItem label="Pool名称" value={data.pool_name} />
            <FieldItem 
              label="Pool状态" 
              value={
                <Tag color={poolStatusInfo.tagColor}>
                  <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                    {poolStatusInfo.icon}
                    <span>{data.pool_status}</span>
                  </span>
                </Tag>
              } 
            />
          </div>
          
          {poolMembers.length > 0 && (
            <div>
              <h4 style={{ fontSize: '15px', fontWeight: 500, marginBottom: '8px' }}>Pool成员</h4>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                {poolMembers.map((member, index) => (
                  <div key={index} style={{ display: 'flex', alignItems: 'center', gap: '8px', border: '1px solid #f0f0f0', padding: '8px', borderRadius: '4px', background: 'white' }}>
                    {member.status === 'online' ? 
                      <CheckCircleFilled style={{ color: '#52c41a' }} /> : 
                      <CloseCircleFilled style={{ color: '#ff4d4f' }} />
                    }
                    <span>{`${member.ip} (${member.status})`}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
        
        <div>
          <h3 style={{ display: 'flex', alignItems: 'center', marginBottom: '16px', fontSize: '16px', fontWeight: 500 }}>
            <CloudServerOutlined style={{ marginRight: '8px', color: '#1890ff' }} />
            关联信息
          </h3>
          
          <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: '16px' }}>
            <FieldItem label="K8s集群名称" value={data.k8s_cluster_name || '未知集群'} />
          </div>
        </div>
      </Card>
    </div>
  );
};

export default F5InfoDetail; 