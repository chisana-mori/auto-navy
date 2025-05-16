const { createProxyMiddleware } = require('http-proxy-middleware');

module.exports = function(app) {
  // 代理/api路径到8080端口
  app.use(
    '/api',
    createProxyMiddleware({
      target: 'http://localhost:8080',
      changeOrigin: true,
      pathRewrite: {
        '^/api': '', // 去掉请求路径中的 /api 前缀
      },
      // 打印代理日志，便于调试
      logLevel: 'debug',
      onProxyReq: (proxyReq, req) => {
        console.log('代理请求 (/api):', req.method, req.path);
      },
      onProxyRes: (proxyRes, req) => {
        console.log('代理响应 (/api):', proxyRes.statusCode, req.path);
      },
    })
  );

  // 代理/fe-v1路径到8081端口
  app.use(
    '/fe-v1',
    createProxyMiddleware({
      target: 'http://localhost:8081',
      changeOrigin: true,
      // 无需pathRewrite，保留原路径
      logLevel: 'debug',
      onProxyReq: (proxyReq, req) => {
        console.log('代理请求 (/fe-v1):', req.method, req.path);
      },
      onProxyRes: (proxyRes, req) => {
        console.log('代理响应 (/fe-v1):', proxyRes.statusCode, req.path);
      },
    })
  );
}; 