import React, { useState } from 'react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Typography, Avatar } from 'antd';
import { DashboardOutlined, CloudServerOutlined, SettingOutlined, UserOutlined, NodeIndexOutlined, ToolOutlined, DesktopOutlined, DatabaseOutlined } from '@ant-design/icons';
import F5InfoList from './components/F5Info/F5InfoList';
import F5InfoDetail from './components/F5Info/F5InfoDetail';
import CalicoNetworkTopology from './components/Calico/CalicoNetworkTopology';
import OpsManagement from './components/Ops/OpsManagement';
import DeviceManagement from './components/Device/DeviceManagement';
import DeviceDetail from './components/Device/DeviceDetail';
import DeviceQuerySimple from './components/Device/DeviceQuerySimple';
import DeviceCenter from './components/Device/DeviceCenter';

const { Header, Content, Sider } = Layout;
const { Title } = Typography;

const App: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [selectedKey, setSelectedKey] = useState<string>(() => {
    const path = location.pathname;
    if (path.startsWith('/f5')) return '1';
    if (path.startsWith('/calico')) return '4';
    if (path.startsWith('/k8s')) return '2';
    if (path.startsWith('/ops')) return '5';
    if (path === '/device-query') return '7';
    if (path.startsWith('/device')) return '6';
    if (path.startsWith('/settings')) return '3';
    return '1';
  });

  const handleMenuClick = (key: string) => {
    setSelectedKey(key);
    switch (key) {
      case '1':
        navigate('/f5');
        break;
      case '2':
        navigate('/k8s');
        break;
      case '3':
        navigate('/settings');
        break;
      case '4':
        navigate('/calico');
        break;
      case '5':
        navigate('/ops');
        break;
      case '6':
        navigate('/device');
        break;
      case '7':
        navigate('/device-query');
        break;
    }
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{
        background: 'white',
        height: '64px',
        padding: '0 24px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        boxShadow: '0 1px 4px rgba(0,21,41,.08)'
      }}>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <CloudServerOutlined style={{ fontSize: '24px', color: '#1677ff', marginRight: '12px' }} />
          <Title level={4} style={{ margin: 0 }}>Navy-NG 管理平台</Title>
        </div>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <Avatar style={{ backgroundColor: '#1677ff', marginLeft: '12px' }} icon={<UserOutlined />} />
        </div>
      </Header>
      <Layout>
        <Sider width={200} style={{ background: 'white' }}>
          <Menu
            mode="inline"
            selectedKeys={[selectedKey]}
            style={{ height: '100%', borderRight: 0 }}
            items={[
              {
                key: '1',
                icon: <DashboardOutlined />,
                label: 'F5 信息管理',
                onClick: () => handleMenuClick('1')
              },
              {
                key: '4',
                icon: <NodeIndexOutlined />,
                label: 'Calico网络拓扑',
                onClick: () => handleMenuClick('4')
              },
              {
                key: '2',
                icon: <CloudServerOutlined />,
                label: 'K8s 集群',
                onClick: () => handleMenuClick('2')
              },
              {
                key: '5',
                icon: <ToolOutlined />,
                label: '运维管理',
                onClick: () => handleMenuClick('5')
              },
              {
                key: '6',
                icon: <DesktopOutlined />,
                label: '设备中心',
                onClick: () => handleMenuClick('6')
              },
              {
                key: '3',
                icon: <SettingOutlined />,
                label: '系统设置',
                onClick: () => handleMenuClick('3')
              },
            ]}
          />
        </Sider>
        <Layout style={{ padding: '24px', background: '#f0f2f5' }}>
          <Content style={{
            margin: 0,
            minHeight: 280,
            borderRadius: '8px',
          }}>
            <Routes>
              <Route path="/" element={<F5InfoList />} />
              <Route path="/f5" element={<F5InfoList />} />
              <Route path="/f5/:id" element={<F5InfoDetail />} />
              <Route path="/calico" element={<CalicoNetworkTopology />} />
              <Route path="/k8s" element={<div>K8s 集群管理（待开发）</div>} />
              <Route path="/ops" element={<OpsManagement />} />
              <Route path="/device" element={<DeviceCenter />} />
              <Route path="/device-management" element={<DeviceManagement />} />
              <Route path="/device/:id/detail" element={<DeviceDetail />} />
              <Route path="/device-query" element={<DeviceQuerySimple />} />
              <Route path="/settings" element={<div>系统设置（待开发）</div>} />
            </Routes>
          </Content>
        </Layout>
      </Layout>
    </Layout>
  );
};

export default App;