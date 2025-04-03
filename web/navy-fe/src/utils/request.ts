import axios from 'axios';
import { message } from 'antd';

console.log('API 基础路径:', process.env.REACT_APP_API_BASE_URL);

const request = axios.create({
  baseURL: 'http://localhost:8080/fe-v1', // 直接使用完整的API地址
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    console.log('发送请求:', config.method, config.url, config.params || config.data);
    return config;
  },
  (error) => {
    console.error('请求错误:', error);
    return Promise.reject(error);
  }
);

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    console.log('收到响应:', response.status, response.config.url, response.data);
    return response.data;
  },
  (error) => {
    console.error('响应错误:', error);
    
    if (error.response) {
      // 服务器返回了错误状态码
      console.error('服务器错误:', error.response.status, error.response.data);
      message.error(error.response.data?.error || `请求失败 (${error.response.status})`);
    } else if (error.request) {
      // 请求发送了但没有收到响应
      console.error('网络错误: 没有收到响应');
      message.error('网络错误: 无法连接到服务器');
    } else {
      // 请求配置出错
      console.error('请求配置错误:', error.message);
      message.error(`请求错误: ${error.message}`);
    }
    
    return Promise.reject(error);
  }
);

export default request; 