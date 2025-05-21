import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { CssBaseline, KubedConfigProvider } from '@kubed/components';
import App from './App';
import './styles/f5-info.css'; // 导入F5信息样式
import './styles/ops-management.css'; // 导入运维管理样式
import './styles/device-management.css'; // 导入设备管理样式
import './styles/device-query.css'; // 导入设备查询样式
import './styles/device-center.css'; // 导入设备中心样式
import { StagewiseToolbar } from '@stagewise/toolbar-react';

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

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

// Initialize stagewise toolbar in development mode only
if (process.env.NODE_ENV === 'development') {
  const stagewiseConfig = {
    plugins: []
  };
  
  // Create a separate DOM element for the toolbar
  const toolbarElement = document.createElement('div');
  toolbarElement.id = 'stagewise-toolbar-root';
  document.body.appendChild(toolbarElement);
  
  // Create a separate React root for the toolbar
  const toolbarRoot = ReactDOM.createRoot(toolbarElement);
  toolbarRoot.render(
    <StagewiseToolbar config={stagewiseConfig} />
  );
}