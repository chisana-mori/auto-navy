// src/main.tsx
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import { StagewiseToolbar } from '@stagewise/toolbar-react';
import './index.css';

// Render the main app
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

// Initialize toolbar separately only in development mode
if (process.env.NODE_ENV === 'development') {
  // Stagewise toolbar configuration
  const toolbarConfig = {
    plugins: []
  };
  
  document.addEventListener('DOMContentLoaded', () => {
    const toolbarRoot = document.createElement('div');
    toolbarRoot.id = 'stagewise-toolbar-root';
    document.body.appendChild(toolbarRoot);
  
    createRoot(toolbarRoot).render(
      <StrictMode>
        <StagewiseToolbar config={toolbarConfig} />
      </StrictMode>
    );
  });
}