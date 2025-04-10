import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, Button, message, Tag, Spin } from 'antd';
import {
  CloudServerOutlined,
  RollbackOutlined,
  DesktopOutlined,
  GlobalOutlined,
  ClusterOutlined,
  DownloadOutlined
} from '@ant-design/icons';
import { getDeviceDetail } from '../../services/deviceService';
import type { Device } from '../../types/device';
import '../../styles/device-management.css';

// 字段项组件
interface FieldItemProps {
  label: string;
  value: React.ReactNode;
}

const FieldItem: React.FC<FieldItemProps> = ({ label, value }) => (
  <div className="field-item">
    <div className="field-label">{label}:</div>
    <div className="field-value">{value || '-'}</div>
  </div>
);

const DeviceDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Device | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      if (!id) return;
      setLoading(true);
      try {
        const response = await getDeviceDetail(id);
        setData(response);
      } catch (error) {
        console.error('获取设备详情失败:', error);
        message.error('获取设备详情失败');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [id]);

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '300px' }}>
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  if (!data) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <h3>未找到设备信息</h3>
        <Button type="primary" onClick={() => navigate('/device')}>返回列表</Button>
      </div>
    );
  }

  return (
    <div style={{ marginBottom: '24px' }}>
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <CloudServerOutlined style={{ fontSize: '20px', color: '#1890ff', marginRight: '12px' }} />
            <span style={{ fontSize: '18px', fontWeight: 500 }}>设备详情</span>
            <Tag color="success" style={{ marginLeft: '12px' }}>
              {data.deviceId}
            </Tag>
          </div>
        }
        extra={
          <div style={{ display: 'flex', gap: '8px' }}>
            <Button
              icon={<RollbackOutlined />}
              onClick={() => navigate(-1)}
            >
              返回
            </Button>
          </div>
        }
        loading={loading}
        className="device-detail-card"
      >
        {/* 基本信息 */}
        <div style={{ marginBottom: '24px' }}>
          <h3 style={{ display: 'flex', alignItems: 'center', marginBottom: '16px', fontSize: '16px', fontWeight: 500 }}>
            <DesktopOutlined style={{ marginRight: '8px', color: '#1890ff' }} />
            基本信息
          </h3>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '16px' }}>
            <FieldItem label="设备ID" value={data.deviceId} />
            <FieldItem label="IP地址" value={data.ip} />
            <FieldItem label="机器类型" value={data.machineType} />
            <FieldItem label="架构" value={data.arch} />
            <FieldItem label="创建时间" value={data.createdAt} />
            <FieldItem label="更新时间" value={data.updatedAt} />
          </div>
        </div>

        {/* 集群信息 */}
        <div style={{ marginBottom: '24px' }}>
          <h3 style={{ display: 'flex', alignItems: 'center', marginBottom: '16px', fontSize: '16px', fontWeight: 500 }}>
            <ClusterOutlined style={{ marginRight: '8px', color: '#1890ff' }} />
            集群信息
          </h3>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '16px' }}>
            <FieldItem label="所属集群" value={data.cluster} />
            <FieldItem label="集群角色" value={data.role} />
            <FieldItem label="APPID" value={data.appId} />
            <FieldItem label="资源池/产品" value={data.resourcePool} />
          </div>
        </div>

        {/* 位置信息 */}
        <div style={{ marginBottom: '24px' }}>
          <h3 style={{ display: 'flex', alignItems: 'center', marginBottom: '16px', fontSize: '16px', fontWeight: 500 }}>
            <GlobalOutlined style={{ marginRight: '8px', color: '#1890ff' }} />
            位置信息
          </h3>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '16px' }}>
            <FieldItem label="IDC" value={data.idc} />
            <FieldItem label="Room" value={data.room} />
            <FieldItem label="机房" value={data.datacenter} />
            <FieldItem label="机柜号" value={data.cabinet} />
            <FieldItem label="网络区域" value={data.network} />
          </div>
        </div>
      </Card>
    </div>
  );
};

export default DeviceDetail;
