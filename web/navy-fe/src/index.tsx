import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { CssBaseline, KubedConfigProvider } from '@kubed/components';
import { StagewiseToolbar } from '@stagewise/toolbar-react';
import App from './App';
import './styles/f5-info.css'; // 导入F5信息样式
import './styles/ops-management.css'; // 导入运维管理样式
import './styles/device-management.css'; // 导入设备管理样式
import './styles/device-query.css'; // 导入设备查询样式
import './styles/device-center.css'; // 导入设备中心样式

const stagewiseConfig = {
  plugins: []
};

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

// 创建工具栏的根节点
const toolbarRoot = document.createElement('div');
toolbarRoot.id = 'stagewise-toolbar-root';
document.body.appendChild(toolbarRoot);

const toolbarContainer = ReactDOM.createRoot(toolbarRoot);

root.render(
  <React.StrictMode>
    <BrowserRouter>
      <KubedConfigProvider>
        <CssBaseline />
        <App />
      </KubedConfigProvider>
    </BrowserRouter>
  </React.StrictMode>
);

// 只在开发模式下渲染工具栏
if (process.env.NODE_ENV === 'development') {
  toolbarContainer.render(
    <React.StrictMode>
      <StagewiseToolbar config={stagewiseConfig} />
    </React.StrictMode>
  );
}