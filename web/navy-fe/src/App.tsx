import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { Layout, Menu, Typography, Avatar } from 'antd';
import { DashboardOutlined, CloudServerOutlined, SettingOutlined, UserOutlined } from '@ant-design/icons';
import F5InfoList from './components/F5Info/F5InfoList';
import F5InfoDetail from './components/F5Info/F5InfoDetail';

const { Header, Content, Sider } = Layout;
const { Title } = Typography;

const App: React.FC = () => {
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
            defaultSelectedKeys={['1']}
            style={{ height: '100%', borderRight: 0 }}
            items={[
              {
                key: '1',
                icon: <DashboardOutlined />,
                label: 'F5 信息管理',
              },
              {
                key: '2',
                icon: <CloudServerOutlined />,
                label: 'K8s 集群',
              },
              {
                key: '3',
                icon: <SettingOutlined />,
                label: '系统设置',
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
            </Routes>
          </Content>
        </Layout>
      </Layout>
    </Layout>
  );
};

export default App; 