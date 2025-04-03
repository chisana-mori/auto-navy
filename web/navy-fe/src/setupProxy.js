const { createProxyMiddleware } = require('http-proxy-middleware');

module.exports = function(app) {
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
        console.log('代理请求:', req.method, req.path);
      },
      onProxyRes: (proxyRes, req) => {
        console.log('代理响应:', proxyRes.statusCode, req.path);
      },
    })
  );
}; 