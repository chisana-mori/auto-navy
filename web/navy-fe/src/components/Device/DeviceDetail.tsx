import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, Button, message, Tag, Spin } from 'antd';
import {
  CloudServerOutlined,
  RollbackOutlined,
  DesktopOutlined,
  GlobalOutlined,
  ClusterOutlined,
  // DownloadOutlined // Removed as it's unused
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
              {data.ciCode}
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
        {/* 设备基本信息 */}
        <div className="detail-section">
          <div className="section-header">
            <DesktopOutlined className="section-icon" />
            <span className="section-title">设备基本信息</span>
          </div>

          <div className="section-content">
            <FieldItem label="设备编码" value={data.ciCode} />
            <FieldItem label="IP地址" value={data.ip} />
            <FieldItem label="机器用途" value={data.group} />
            <FieldItem label="型号" value={data.model} />
            <FieldItem label="状态" value={data.status} />
            <FieldItem label="厂商" value={data.company} />
            <FieldItem label="是否国产化" value={data.isLocalization ? '是' : '否'} />
            <FieldItem label="创建时间" value={data.createdAt} />
            <FieldItem label="更新时间" value={data.updatedAt} />
          </div>
        </div>

        {/* 硬件信息 */}
        <div className="detail-section">
          <div className="section-header">
            <ClusterOutlined className="section-icon" />
            <span className="section-title">硬件信息</span>
          </div>

          <div className="section-content">
            <FieldItem label="CPU架构" value={data.archType} />
            <FieldItem label="CPU数量" value={data.cpu} />
            <FieldItem label="内存大小" value={data.memory} />
            <FieldItem label="KVM IP" value={data.kvmIp} />
          </div>
        </div>

        {/* 集群信息 */}
        <div className="detail-section">
          <div className="section-header">
            <ClusterOutlined className="section-icon" />
            <span className="section-title">集群信息</span>
          </div>

          <div className="section-content">
            <FieldItem label="所属集群" value={data.cluster} />
            <FieldItem label="集群ID" value={data.clusterId} />
            <FieldItem label="角色" value={data.role} />
            <FieldItem label="APPID" value={data.appId} />
          </div>
        </div>

        {/* 操作系统信息 */}
        <div className="detail-section">
          <div className="section-header">
            <GlobalOutlined className="section-icon" />
            <span className="section-title">操作系统信息</span>
          </div>

          <div className="section-content">
            <FieldItem label="操作系统" value={data.os} />
            <FieldItem label="操作系统名称" value={data.osName} />
            <FieldItem label="操作系统版本" value={data.osIssue} />
            <FieldItem label="操作系统内核" value={data.osKernel} />
            <FieldItem label="操作系统创建时间" value={data.osCreateTime} />
          </div>
        </div>

        {/* 位置信息 */}
        <div className="detail-section">
          <div className="section-header">
            <GlobalOutlined className="section-icon" />
            <span className="section-title">位置信息</span>
          </div>

          <div className="section-content">
            <FieldItem label="IDC" value={data.idc} />
            <FieldItem label="机房" value={data.room} />
            <FieldItem label="所属机柜" value={data.cabinet} />
            <FieldItem label="机柜编号" value={data.cabinetNO} />
            <FieldItem label="网络类型" value={data.infraType} />
            <FieldItem label="网络区域" value={data.netZone} />
          </div>
        </div>
      </Card>
    </div>
  );
};

export default DeviceDetail;
