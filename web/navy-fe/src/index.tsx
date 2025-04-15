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