import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { CssBaseline, KubedConfigProvider } from '@kubed/components';
import App from './App';
import './styles/f5-info.css'; // 导入F5信息样式

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