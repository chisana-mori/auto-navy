import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Card,
  Button,
  Table,
  Space,
  message,
  Modal,
  Form,
  Input,
  Progress,
  Tag,
  Drawer,
  Typography,
  Spin
} from 'antd';
import {
  PlayCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
  FileTextOutlined,
  CheckCircleFilled,
  CloseCircleFilled,
  LoadingOutlined,
  ToolOutlined
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table/interface';
import request from '../../utils/request';
import type { OpsJob, OpsJobListResponse, OpsJobQuery, OpsJobStatusUpdate } from '../../types/ops';

const { TextArea } = Input;
const { Title, Paragraph, Text } = Typography;

const OpsManagement: React.FC = () => {
  const [loading, setLoading] = useState<boolean>(false);
  const [data, setData] = useState<OpsJob[]>([]);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });
  const [createModalVisible, setCreateModalVisible] = useState<boolean>(false);
  const [createForm] = Form.useForm();
  const [logDrawerVisible, setLogDrawerVisible] = useState<boolean>(false);
  const [currentJob, setCurrentJob] = useState<OpsJob | null>(null);
  const [wsConnected, setWsConnected] = useState<boolean>(false);
  const [jobRunning, setJobRunning] = useState<boolean>(false);
  const [logContent, setLogContent] = useState<string>('');

  const wsRef = useRef<WebSocket | null>(null);
  const logEndRef = useRef<HTMLDivElement>(null);

  // 获取任务列表
  const fetchData = useCallback(async (page = 1, size = 10, query: Partial<OpsJobQuery> = {}) => {
    setLoading(true);
    try {
      const params: OpsJobQuery = {
        page,
        size,
        ...query
      };

      const response = await request.get<any, OpsJobListResponse>('/ops/job', {
        params
      });

      console.log('运维任务原始数据:', JSON.stringify(response));

      if (response && response.list) {
        // 确保ID字段正确映射
        const processedData = response.list.map(item => ({
          ...item,
          id: item.id || 0 // 确保有id字段，如果没有则默认为0
        }));

        console.log('处理后的数据:', processedData);
        setData(processedData);
        setPagination({
          current: page,
          pageSize: size,
          total: response.total || 0,
        });
      } else {
        console.error('API返回数据格式不正确:', response);
        setData([]);
      }
    } catch (error) {
      console.error('获取数据失败:', error);
      message.error('获取运维任务列表失败');
      setData([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // 初始加载数据
  useEffect(() => {
    fetchData(1, 10);
  }, [fetchData]);

  // 创建新任务
  const handleCreateJob = async (values: any) => {
    try {
      // 只发送必要的字段，确保不包含id字段
      const { name, description } = values;
      const jobData = { name, description };

      console.log('发送创建任务数据:', jobData);
      const response = await request.post('/ops/job', jobData);
      console.log('创建任务响应:', JSON.stringify(response));
      message.success('创建运维任务成功');
      setCreateModalVisible(false);
      createForm.resetFields();
      // 等待一下再刷新数据，确保数据库写入完成
      setTimeout(() => {
        fetchData(1, 10);
      }, 500);
      return response.data;
    } catch (error) {
      console.error('创建任务失败:', error);
      message.error('创建运维任务失败');
    }
  };

  // 查看任务详情
  const handleViewJob = async (id: number) => {
    try {
      const response = await request.get<any, OpsJob>(`/ops/job/${id}`);
      setCurrentJob(response);
      setLogContent(response.log_content || '');
      setLogDrawerVisible(true);

      // 连接WebSocket
      connectWebSocket(id);
    } catch (error) {
      console.error('获取任务详情失败:', error);
      message.error('获取任务详情失败');
    }
  };

  // 启动任务
  const handleStartJob = () => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      message.error('WebSocket连接未建立，无法启动任务');
      return;
    }

    wsRef.current.send('start');
    setJobRunning(true);
    message.info('任务启动中...');
  };

  // 连接WebSocket
  const connectWebSocket = (jobId: number) => {
    // 关闭之前的连接
    if (wsRef.current) {
      wsRef.current.close();
    }

    // 创建新的WebSocket连接
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.hostname}:8081/fe-v1/ops/job/${jobId}/ws`;

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log('WebSocket连接已建立');
      setWsConnected(true);
    };

    ws.onmessage = (event) => {
      try {
        const data: OpsJobStatusUpdate = JSON.parse(event.data);
        console.log('收到WebSocket消息:', data);

        // 更新任务状态
        if (currentJob && currentJob.id === data.id) {
          setCurrentJob(prev => {
            if (!prev) return prev;
            return {
              ...prev,
              status: data.status,
              progress: data.progress
            };
          });
        }

        // 更新日志内容
        if (data.log_line) {
          setLogContent(prev => prev + data.log_line + '\n');
          // 滚动到日志底部
          if (logEndRef.current) {
            logEndRef.current.scrollIntoView({ behavior: 'smooth' });
          }
        }

        // 任务完成
        if (data.status === 'completed' || data.status === 'failed') {
          setJobRunning(false);
          // 刷新任务列表
          fetchData(pagination.current, pagination.pageSize);

          if (data.status === 'completed') {
            message.success('任务执行完成');
          } else {
            message.error('任务执行失败');
          }
        }
      } catch (error) {
        console.error('解析WebSocket消息失败:', error);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket错误:', error);
      setWsConnected(false);
      message.error('WebSocket连接错误');
    };

    ws.onclose = () => {
      console.log('WebSocket连接已关闭');
      setWsConnected(false);
    };
  };

  // 关闭WebSocket连接
  useEffect(() => {
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  // 自动滚动到日志底部
  useEffect(() => {
    if (logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logContent]);

  // 关闭日志抽屉
  const handleCloseLogDrawer = () => {
    setLogDrawerVisible(false);
    if (wsRef.current) {
      wsRef.current.close();
    }
    setWsConnected(false);
    setJobRunning(false);
  };

  // 获取状态标签
  const getStatusTag = (status: string) => {
    switch (status) {
      case 'pending':
        return <Tag color="blue">等待中</Tag>;
      case 'running':
        return <Tag color="processing" icon={<LoadingOutlined />}>执行中</Tag>;
      case 'completed':
        return <Tag color="success" icon={<CheckCircleFilled />}>已完成</Tag>;
      case 'failed':
        return <Tag color="error" icon={<CloseCircleFilled />}>失败</Tag>;
      default:
        return <Tag color="default">{status}</Tag>;
    }
  };

  // 表格列定义
  const columns: ColumnsType<OpsJob> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <span
          className="task-name-link"
          onClick={() => handleViewJob(record.id)}
          style={{
            color: '#1890ff',
            cursor: 'pointer',
            textDecoration: 'none'
          }}
        >
          {text}
        </span>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status) => getStatusTag(status),
    },
    {
      title: '进度',
      dataIndex: 'progress',
      key: 'progress',
      width: 150,
      render: (progress) => <Progress percent={progress} size="small" />,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (date) => new Date(date).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_, record) => (
        <Space size={8}>
          <Button
            type="link"
            size="small"
            icon={<FileTextOutlined />}
            onClick={() => handleViewJob(record.id)}
          >
            查看
          </Button>
          {record.status === 'pending' && (
            <Button
              type="link"
              size="small"
              icon={<PlayCircleOutlined />}
              onClick={() => {
                handleViewJob(record.id).then(() => {
                  // Wait a bit for WebSocket connection to establish
                  setTimeout(() => {
                    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
                      handleStartJob();
                    }
                  }, 500);
                });
              }}
            >
              执行
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <ToolOutlined style={{ fontSize: '20px', color: '#1890ff', marginRight: '12px' }} />
            <span style={{ fontSize: '18px', fontWeight: 500 }}>运维管理</span>
          </div>
        }
      >
        <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'flex-end' }}>
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={() => fetchData(pagination.current, pagination.pageSize)}
            >
              刷新
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateModalVisible(true)}
            >
              提交运维任务
            </Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          pagination={{
            ...pagination,
            showTotal: (total) => `共 ${total} 条记录`,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50'],
            onChange: (page, pageSize) => {
              fetchData(page, pageSize);
            },
          }}
        />
      </Card>

      {/* 创建任务弹窗 */}
      <Modal
        title="提交运维任务"
        open={createModalVisible}
        onCancel={() => setCreateModalVisible(false)}
        footer={null}
      >
        <Form
          form={createForm}
          layout="vertical"
          onFinish={handleCreateJob}
        >
          <Form.Item
            name="name"
            label="任务名称"
            rules={[{ required: true, message: '请输入任务名称' }]}
          >
            <Input placeholder="请输入任务名称" />
          </Form.Item>
          <Form.Item
            name="description"
            label="任务描述"
            rules={[{ required: true, message: '请输入任务描述' }]}
          >
            <TextArea rows={4} placeholder="请输入任务描述" />
          </Form.Item>
          <Form.Item>
            <Space style={{ display: 'flex', justifyContent: 'flex-end' }}>
              <Button onClick={() => setCreateModalVisible(false)}>取消</Button>
              <Button type="primary" htmlType="submit">提交</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 任务日志抽屉 */}
      <Drawer
        title={
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>任务详情 {currentJob && `#${currentJob.id} - ${currentJob.name}`}</span>
            <Space>
              {wsConnected ? (
                <Tag color="success">WebSocket已连接</Tag>
              ) : (
                <Tag color="error">WebSocket未连接</Tag>
              )}
              {currentJob && getStatusTag(currentJob.status)}
            </Space>
          </div>
        }
        placement="right"
        width={700}
        onClose={handleCloseLogDrawer}
        open={logDrawerVisible}
        extra={
          <Space>
            <Button
              type="primary"
              icon={<PlayCircleOutlined />}
              onClick={handleStartJob}
              disabled={!wsConnected || jobRunning || (currentJob?.status === 'completed' || currentJob?.status === 'failed')}
            >
              启动任务
            </Button>
          </Space>
        }
      >
        {currentJob ? (
          <div>
            <div style={{ marginBottom: '20px' }}>
              <Title level={5}>任务信息</Title>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                <Paragraph>
                  <Text strong>ID:</Text> {currentJob.id}
                </Paragraph>
                <Paragraph>
                  <Text strong>名称:</Text> {currentJob.name}
                </Paragraph>
                <Paragraph>
                  <Text strong>状态:</Text> {getStatusTag(currentJob.status)}
                </Paragraph>
                <Paragraph>
                  <Text strong>进度:</Text> {currentJob.progress}%
                </Paragraph>
                <Paragraph>
                  <Text strong>创建时间:</Text> {new Date(currentJob.created_at).toLocaleString()}
                </Paragraph>
                <Paragraph>
                  <Text strong>开始时间:</Text> {new Date(currentJob.start_time).toLocaleString()}
                </Paragraph>
              </div>
              <Paragraph>
                <Text strong>描述:</Text> {currentJob.description}
              </Paragraph>
            </div>

            <div style={{ marginBottom: '20px' }}>
              <Title level={5}>任务进度</Title>
              <Progress
                percent={currentJob.progress}
                status={
                  currentJob.status === 'failed'
                    ? 'exception'
                    : currentJob.status === 'completed'
                      ? 'success'
                      : 'active'
                }
              />
            </div>

            <div>
              <Title level={5}>执行日志</Title>
              <div
                style={{
                  backgroundColor: '#f5f5f5',
                  padding: '12px',
                  borderRadius: '4px',
                  height: '300px',
                  overflowY: 'auto',
                  fontFamily: 'monospace',
                  whiteSpace: 'pre-wrap',
                  fontSize: '12px',
                  lineHeight: '1.5',
                  color: '#333'
                }}
              >
                {logContent || '暂无日志'}
                <div ref={logEndRef} />
              </div>
            </div>
          </div>
        ) : (
          <Spin tip="加载中...">
            <div style={{ height: '400px', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
              正在加载任务信息...
            </div>
          </Spin>
        )}
      </Drawer>
    </div>
  );
};

export default OpsManagement;
