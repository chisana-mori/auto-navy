/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useEffect, useState } from 'react';
import './Dashboard.css';
import './DeviceMatchingPolicy.less';
import {
  Card, Row, Col, Progress, Button, Table, Tag, Space,
  Select, Tabs, Empty, Divider, Alert, Tooltip, Badge,
  Modal, Form, Input, InputNumber, Radio, Drawer, Descriptions, message,
  Spin
} from 'antd';
import {
  CloudUploadOutlined, CloudDownloadOutlined, ReloadOutlined,
  PlusOutlined, SearchOutlined, EyeOutlined, EditOutlined,
  CloseCircleOutlined, CheckCircleOutlined, DeleteOutlined,
  ArrowUpOutlined, ArrowDownOutlined, ClusterOutlined, BarChartOutlined,
  ClockCircleOutlined, PauseCircleOutlined, WarningOutlined,
  LinkOutlined, DisconnectOutlined, AreaChartOutlined
} from '@ant-design/icons';
import { statsApi, strategyApi, orderApi } from '../../services/elasticScalingService';
import {
  ResourceAllocationTrend,
  ResourceTypeData,
  Strategy,
  StrategyDetail,
  OrderDetail,
  OrderListItem,
  Device,
  DashboardStats,
  PaginatedResponse
} from '../../types/elastic-scaling';
import ReactECharts from 'echarts-for-react';
import DeviceMatchingPolicy from './DeviceMatchingPolicy';
import EmptyOrderState from './EmptyOrderState';
import CreateOrderModal from './CreateOrderModal';

const { Option } = Select;
const { TabPane } = Tabs;

// Define type aliases for complex types to make code more readable
type StrategiesState = PaginatedResponse<Strategy> | null;
type OrdersState = PaginatedResponse<OrderListItem> | null;

const Dashboard: React.FC = () => {
  // 状态管理
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [strategies, setStrategies] = useState<StrategiesState>(null);
  const [orders, setOrders] = useState<OrdersState>(null);
  const [pendingOrders, setPendingOrders] = useState<OrderListItem[]>([]);
  const [processingOrders, setProcessingOrders] = useState<OrderListItem[]>([]);
  const [completedOrders, setCompletedOrders] = useState<OrderListItem[]>([]);
  const [allOrders, setAllOrders] = useState<OrdersState>(null);
  const [selectedClusterId, setSelectedClusterId] = useState<number | null>(null);
  const [selectedTimeRange, setSelectedTimeRange] = useState('7d');
  const [selectedResourceTypes, setSelectedResourceTypes] = useState<string[]>([]);
  const [cpuData, setCpuData] = useState<any>(null);
  const [memoryData, setMemoryData] = useState<any>(null);
  const [orderStatusData, setOrderStatusData] = useState<any>(null);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [selectedOrder, setSelectedOrder] = useState<OrderDetail | null>(null);
  const [selectedStrategy, setSelectedStrategy] = useState<Strategy | null>(null);
  const [orderFilter, setOrderFilter] = useState<string | null>(null);
  const [orderStatusFilter, setOrderStatusFilter] = useState<string>('processing'); // 默认显示处理中订单
  const [customTabVisible, setCustomTabVisible] = useState<boolean>(false);
  const [customTabStatus, setCustomTabStatus] = useState<string>('');
  const [createStrategyModalVisible, setCreateStrategyModalVisible] = useState(false);
  const [editStrategyModalVisible, setEditStrategyModalVisible] = useState(false);
  const [currentEditStrategyId, setCurrentEditStrategyId] = useState<number | null>(null);
  const [createOrderModalVisible, setCreateOrderModalVisible] = useState(false);
  const [resourcePools, setResourcePools] = useState<any[]>([]);

  // 资源池类型列表
  const [resourceTypeOptions, setResourceTypeOptions] = useState<string[]>([]);
  // 集群列表
  const [clusters, setClusters] = useState<any[]>([]);

  // 策略表单
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  // 策略表格列定义
  const strategyColumns = [
    {
      title: '策略名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: Strategy) => (
        <a href="#!" onClick={(e) => { e.preventDefault(); editStrategy(record.id); }}>{text}</a>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: 'CPU阈值',
      key: 'cpuThreshold',
      render: (text: string, record: Strategy) => (
        record.cpuThresholdValue ? (
          <Progress
            percent={record.cpuThresholdValue}
            size="small"
            strokeColor="#1890ff"
            style={{ width: 100 }}
          />
        ) : (
          <span style={{ color: 'rgba(0, 0, 0, 0.45)' }}>--</span>
        )
      ),
    },
    {
      title: '内存阈值',
      key: 'memoryThreshold',
      render: (text: string, record: Strategy) => (
        record.memoryThresholdValue ? (
          <Progress
            percent={record.memoryThresholdValue}
            size="small"
            strokeColor="#52c41a"
            style={{ width: 100 }}
          />
        ) : (
          <span style={{ color: 'rgba(0, 0, 0, 0.45)' }}>--</span>
        )
      ),
    },
    {
      title: '条件',
      key: 'condition',
      width: 80,
      render: (text: string, record: Strategy) => (
        record.cpuThresholdValue && record.memoryThresholdValue ? (
          <Tag color={record.conditionLogic === 'AND' ? 'blue' : 'orange'}>
            {record.conditionLogic === 'AND' ? (
              <span><LinkOutlined style={{ marginRight: 4 }} />同时</span>
            ) : (
              <span><DisconnectOutlined style={{ marginRight: 4 }} />任一</span>
            )}
          </Tag>
        ) : (
          <span style={{ color: 'rgba(0, 0, 0, 0.45)' }}>--</span>
        )
      ),
    },
    {
      title: '动作',
      dataIndex: 'thresholdTriggerAction',
      key: 'thresholdTriggerAction',
      render: (text: string) => (
        <Tag color={text === 'pool_entry' ? 'blue' : 'orange'}>
          {text === 'pool_entry' ? <CloudUploadOutlined /> : <CloudDownloadOutlined />}
          {text === 'pool_entry' ? '入池' : '退池'}
        </Tag>
      ),
    },
    {
      title: '配置',
      key: 'config',
      render: (text: string, record: Strategy) => (
        <Space direction="vertical" size={4} style={{ width: '100%' }}>
          <div style={{ fontSize: 13, display: 'flex', alignItems: 'center' }}>
            <BarChartOutlined style={{ marginRight: 8, color: '#1890ff' }} />
            <span style={{ whiteSpace: 'nowrap' }}>目标: CPU {record.cpuTargetValue ? `${record.cpuTargetValue}%` : '--'}, 内存 {record.memoryTargetValue ? `${record.memoryTargetValue}%` : '--'}</span>
          </div>
          <div style={{ fontSize: 13, display: 'flex', alignItems: 'center' }}>
            <ClockCircleOutlined style={{ marginRight: 8, color: '#52c41a' }} />
            <span>持续: {record.durationMinutes ? Math.floor(record.durationMinutes / (24 * 60)) : '--'} 天</span>
          </div>
          <div style={{ fontSize: 13, display: 'flex', alignItems: 'center' }}>
            <PauseCircleOutlined style={{ marginRight: 8, color: '#faad14' }} />
            <span>冷却: {record.cooldownMinutes ? Math.floor(record.cooldownMinutes / (24 * 60)) : '--'} 天</span>
          </div>
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Badge
          status={status === 'enabled' ? 'success' : 'default'}
          text={status === 'enabled' ? '启用' : '禁用'}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (text: string, record: Strategy) => (
        <Space size="small">
          <Tooltip title="编辑">
            <Button type="text" icon={<EditOutlined />} onClick={() => editStrategy(record.id)} />
          </Tooltip>
          <Tooltip title={record.status === 'enabled' ? '禁用' : '启用'}>
            <Button
              type="text"
              icon={record.status === 'enabled' ? <CloseCircleOutlined /> : <CheckCircleOutlined />}
              danger={record.status === 'enabled'}
              onClick={() => toggleStrategyStatus(record.id, record.status)}
            />
          </Tooltip>
          <Tooltip title="删除">
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              onClick={() => deleteStrategy(record.id)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  // 加载数据
  useEffect(() => {
    const fetchInitialData = async () => {
      try {
        // 先获取资源类型和集群列表
        await fetchResourceTypes();
        const clustersData = await fetchClusters();

        // 获取工作台数据（包括待处理订单）
        await fetchData();

        // 优先从待处理订单中选择集群
        if (pendingOrders && pendingOrders.length > 0) {
          // 从待处理订单中随机选择一个
          const randomIndex = Math.floor(Math.random() * pendingOrders.length);
          const randomOrder = pendingOrders[randomIndex];

          if (randomOrder && randomOrder.clusterId) {
            console.log('从待处理订单中选择集群:', randomOrder.clusterId, '订单ID:', randomOrder.id);
            setSelectedClusterId(randomOrder.clusterId);
            // 设置默认资源类型为"所有资源"
            const defaultResourceTypes = ['total'];
            setSelectedResourceTypes(defaultResourceTypes);
            await fetchResourceTrend(randomOrder.clusterId, '24h', defaultResourceTypes);
            return; // 已选择集群，不需要继续执行
          }
        }

        // 如果没有待处理订单或订单中没有有效的集群ID，则选择第一个集群
        if (clustersData && clustersData.length > 0) {
          console.log('自动选择第一个集群:', clustersData[0]);
          setSelectedClusterId(clustersData[0].id);
          // 设置默认资源类型为"所有资源"
          const defaultResourceTypes = ['total'];
          setSelectedResourceTypes(defaultResourceTypes);
          await fetchResourceTrend(clustersData[0].id, '24h', defaultResourceTypes);
        }
      } catch (error) {
        console.error('初始化数据加载失败:', error);
      }
    };

    fetchInitialData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 获取资源池类型
  const fetchResourceTypes = async () => {
    try {
      // 实际应用中应调用API获取资源类型列表
      // const response = await api.getResourceTypes();
      // setResourceTypeOptions(response.data.data);

      // 模拟数据
      setResourceTypeOptions([
        'total',
        'compute',
        'memory',
        'storage',
        'gpu',
        'network'
      ]);

      // 模拟资源池数据
      setResourcePools([
        { type: 'compute', name: '计算型资源池' },
        { type: 'memory', name: '内存优化型资源池' },
        { type: 'storage', name: '存储优化型资源池' },
        { type: 'gpu', name: 'GPU加速型资源池' },
        { type: 'network', name: '网络优化型资源池' },
        { type: 'total', name: '全局资源' }
      ]);
    } catch (error) {
      console.error('Error fetching resource types:', error);
    }
  };

  // 获取集群列表
  const fetchClusters = async () => {
    try {
      // 实际应该通过API获取集群列表
      // const response = await axios.get('/fe-v1/clusters');
      // const clustersData = response.data.data;
      // setClusters(clustersData);
      // return clustersData;

      // 模拟数据
      const clustersData = [
        { id: 1, name: '集群-01' },
        { id: 2, name: '集群-02' },
        { id: 3, name: '集群-03' },
        { id: 4, name: '集群-04' },
        { id: 5, name: '生产集群-A' },
        { id: 6, name: '生产集群-B' },
      ];
      setClusters(clustersData);
      return clustersData;
    } catch (error) {
      console.error('Error fetching clusters:', error);
      return [];
    }
  };

  // 数据加载函数
  const fetchData = async () => {
    setIsLoading(true);
    try {
      // 获取工作台统计数据
      try {
        const statsData = await statsApi.getDashboardStats();
        setStats(statsData);
      } catch (error) {
        console.error('获取工作台统计数据失败:', error);
        // 保持使用默认的统计数据，不影响UI显示
        // 如果是开发环境，可以打印额外信息
        if (process.env.NODE_ENV === 'development') {
          console.info('使用默认统计数据作为后备');
        }
      }

      // 获取策略列表
      try {
        const strategiesData = await strategyApi.getStrategies({ page: 1, pageSize: 5 });
        setStrategies(strategiesData);
      } catch (error) {
        console.error('获取策略列表失败:', error);
      }

      // 获取不同状态的订单
      try {
        // 获取待处理订单
        const pendingOrdersData = await orderApi.getOrders({ status: 'pending', page: 1, pageSize: 10 });
        const pendingOrdersList = pendingOrdersData.list;
        setPendingOrders(pendingOrdersList);

        // 如果有待处理订单，并且当前没有选择集群，则从待处理订单中随机选择一个集群
        if (pendingOrdersList && pendingOrdersList.length > 0 && !selectedClusterId) {
          const randomIndex = Math.floor(Math.random() * pendingOrdersList.length);
          const randomOrder = pendingOrdersList[randomIndex];

          if (randomOrder && randomOrder.clusterId) {
            console.log('从待处理订单中选择集群:', randomOrder.clusterId, '订单ID:', randomOrder.id);
            setSelectedClusterId(randomOrder.clusterId);
            // 设置默认资源类型为"所有资源"
            const defaultResourceTypes = ['total'];
            setSelectedResourceTypes(defaultResourceTypes);
            await fetchResourceTrend(randomOrder.clusterId, selectedTimeRange, defaultResourceTypes);
          }
        }

        // 获取其他状态的订单
        const processingOrdersData = await orderApi.getOrders({ status: 'processing', page: 1, pageSize: 10 });
        setProcessingOrders(processingOrdersData.list);

        const completedOrdersData = await orderApi.getOrders({ status: 'completed', page: 1, pageSize: 10 });
        setCompletedOrders(completedOrdersData.list);

        // 获取所有订单
        const allOrdersData = await orderApi.getOrders({ page: 1, pageSize: 10 });
        setAllOrders(allOrdersData);
      } catch (error) {
        console.error('获取订单数据失败:', error);
      }

      // 注意：资源类型和集群列表已在外层函数中加载，这里不需要重复加载

      // 如果有集群统计数据，记录下来（实际环境中可能需要使用）
      if (stats && stats.clusterCount > 0) {
        console.log(`集群统计: 共有 ${stats.clusterCount} 个集群，其中 ${stats.abnormalClusterCount} 个异常`);
      }
    } catch (error) {
      console.error('加载工作台数据失败:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // 获取资源趋势数据
  const fetchResourceTrend = async (clusterId: number, range: string, resourceTypes: string[] = []) => {
    // 设置加载状态
    setIsLoading(true);

    try {
      // 检查资源类型是否为空数组
      if (!resourceTypes || resourceTypes.length === 0) {
        console.log('未选择任何资源类型，不加载数据');
        // 清空图表数据，显示空状态
        setCpuData(null);
        setMemoryData(null);
        setIsLoading(false);
        return;
      }

      console.log(`加载集群 ${clusterId} 的资源趋势数据，时间范围: ${range}，资源类型: ${resourceTypes.join(',')}`);
      const resourceTrend = await statsApi.getResourceAllocationTrend(clusterId, range, resourceTypes);

      // 检查是否有数据
      if (!resourceTrend.timestamps || resourceTrend.timestamps.length === 0) {
        console.log('资源趋势数据为空');
        setCpuData(null);
        setMemoryData(null);
        return;
      }

      // 准备CPU图表数据
      setCpuData({
        title: {
          text: 'CPU使用率趋势',
          left: 'center'
        },
        tooltip: {
          trigger: 'axis'
        },
        legend: {
          data: resourceTrend.resourceTypes.map(type => `${type} - CPU使用率`).concat(
                resourceTrend.resourceTypes.map(type => `${type} - CPU分配率`)),
          top: 30
        },
        xAxis: {
          type: 'category',
          data: resourceTrend.timestamps.map(time => {
            const date = new Date(time);
            return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
          })
        },
        yAxis: {
          type: 'value',
          axisLabel: {
            formatter: '{value} %'
          },
          max: 100
        },
        series: [
          // 主系列 - 总体使用率
          {
            name: 'CPU使用率',
            type: 'line',
            smooth: true,
            data: resourceTrend.cpuUsageRatio,
            areaStyle: {
              color: {
                type: 'linear',
                x: 0,
                y: 0,
                x2: 0,
                y2: 1,
                colorStops: [{
                  offset: 0, color: 'rgba(24, 144, 255, 0.3)'
                }, {
                  offset: 1, color: 'rgba(24, 144, 255, 0.1)'
                }]
              }
            },
            itemStyle: {
              color: '#1890ff'
            }
          },
          {
            name: 'CPU分配率',
            type: 'line',
            smooth: true,
            data: resourceTrend.cpuAllocationRatio,
            lineStyle: {
              type: 'dashed'
            },
            itemStyle: {
              color: '#1890ff'
            }
          },
          // 动态添加每种资源类型的系列
          ...resourceTrend.resourceTypes.filter(type => type !== 'total').flatMap((type, index) => {
            const typeData = resourceTrend.resourceTypeData[type];
            if (!typeData) return [];

            // 为每种资源类型生成不同的颜色
            const baseColors = ['#ff7a45', '#52c41a', '#722ed1', '#faad14', '#13c2c2'];
            const color = baseColors[index % baseColors.length];

            return [
              {
                name: `${type} - CPU使用率`,
                type: 'line',
                smooth: true,
                data: typeData.cpuUsageRatio,
                itemStyle: { color },
                lineStyle: { width: 1 }
              },
              {
                name: `${type} - CPU分配率`,
                type: 'line',
                smooth: true,
                data: typeData.cpuAllocationRatio,
                itemStyle: { color },
                lineStyle: { width: 1, type: 'dashed' }
              }
            ];
          })
        ]
      });

      // 准备内存图表数据
      setMemoryData({
        title: {
          text: '内存使用率趋势',
          left: 'center'
        },
        tooltip: {
          trigger: 'axis'
        },
        legend: {
          data: resourceTrend.resourceTypes.map(type => `${type} - 内存使用率`).concat(
                resourceTrend.resourceTypes.map(type => `${type} - 内存分配率`)),
          top: 30
        },
        xAxis: {
          type: 'category',
          data: resourceTrend.timestamps.map(time => {
            const date = new Date(time);
            return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
          })
        },
        yAxis: {
          type: 'value',
          axisLabel: {
            formatter: '{value} %'
          },
          max: 100
        },
        series: [
          // 主系列 - 总体使用率
          {
            name: '内存使用率',
            type: 'line',
            smooth: true,
            data: resourceTrend.memUsageRatio,
            areaStyle: {
              color: {
                type: 'linear',
                x: 0,
                y: 0,
                x2: 0,
                y2: 1,
                colorStops: [{
                  offset: 0, color: 'rgba(82, 196, 26, 0.3)'
                }, {
                  offset: 1, color: 'rgba(82, 196, 26, 0.1)'
                }]
              }
            },
            itemStyle: {
              color: '#52c41a'
            }
          },
          {
            name: '内存分配率',
            type: 'line',
            smooth: true,
            data: resourceTrend.memAllocationRatio,
            lineStyle: {
              type: 'dashed'
            },
            itemStyle: {
              color: '#52c41a'
            }
          },
          // 动态添加每种资源类型的系列
          ...resourceTrend.resourceTypes.filter(type => type !== 'total').flatMap((type, index) => {
            const typeData = resourceTrend.resourceTypeData[type];
            if (!typeData) return [];

            // 为每种资源类型生成不同的颜色
            const baseColors = ['#ff7a45', '#52c41a', '#722ed1', '#faad14', '#13c2c2'];
            const color = baseColors[index % baseColors.length];

            return [
              {
                name: `${type} - 内存使用率`,
                type: 'line',
                smooth: true,
                data: typeData.memUsageRatio,
                itemStyle: { color },
                lineStyle: { width: 1 }
              },
              {
                name: `${type} - 内存分配率`,
                type: 'line',
                smooth: true,
                data: typeData.memAllocationRatio,
                itemStyle: { color },
                lineStyle: { width: 1, type: 'dashed' }
              }
            ];
          })
        ]
      });
    } catch (error) {
      console.error('Error fetching resource trend data:', error);
      // 清空图表数据，显示空状态
      setCpuData(null);
      setMemoryData(null);
      message.error('获取资源趋势数据失败');
    } finally {
      // 无论成功还是失败，都重置加载状态
      setIsLoading(false);
    }
  };

  // 处理资源趋势参数变更
  const handleTrendParamsChange = (clusterId: number, range: string, resourceTypes: string[] = []) => {
    setSelectedClusterId(clusterId);
    setSelectedTimeRange(range);

    // 如果没有选择资源类型，使用默认值"所有资源"
    const typesToUse = resourceTypes.length > 0 ? resourceTypes : ['total'];
    setSelectedResourceTypes(typesToUse);

    // 只有当有选择资源类型时才加载数据
    if (typesToUse.length > 0) {
      fetchResourceTrend(clusterId, range, typesToUse);
    }
  };

  // 使用局部加载状态，避免整个页面重新渲染
  const [strategyLoading, setStrategyLoading] = useState<{[key: number]: boolean}>({});

  const editStrategy = async (id: number) => {
    console.log('Edit strategy with ID:', id);

    // 先显示编辑模态框，避免页面闪动
    setEditStrategyModalVisible(true);

    // 设置当前策略的加载状态，而不是整个页面的加载状态
    setStrategyLoading(prev => ({ ...prev, [id]: true }));

    try {
      // 获取策略详情
      const strategyDetail = await strategyApi.getStrategy(id);

      // 填充表单数据
      editForm.setFieldsValue({
        name: strategyDetail.name,
        description: strategyDetail.description,
        thresholdTriggerAction: strategyDetail.thresholdTriggerAction,
        // Remove resourcePool field and use resourceTypes only
        resourceTypes: typeof strategyDetail.resourceTypes === 'string' && strategyDetail.resourceTypes
          ? strategyDetail.resourceTypes.split(',').map((t: string) => t.trim())
          : (Array.isArray(strategyDetail.resourceTypes) ? strategyDetail.resourceTypes : ['compute']),
        clusterIds: strategyDetail.clusterIds || strategyDetail.clusters.map(c => parseInt(c)),
        cpuThresholdValue: strategyDetail.cpuThresholdValue || 0,
        cpuTargetValue: strategyDetail.cpuTargetValue || 0,
        memoryThresholdValue: strategyDetail.memoryThresholdValue || 0,
        memoryTargetValue: strategyDetail.memoryTargetValue || 0,
        conditionLogic: strategyDetail.conditionLogic || 'AND',
        // 转换分钟到天，如果后端没有返回时间字段，则使用默认值
        durationDays: strategyDetail.durationMinutes ?
          Math.floor(strategyDetail.durationMinutes / (24 * 60)) : 1,
        cooldownDays: strategyDetail.cooldownMinutes ?
          Math.floor(strategyDetail.cooldownMinutes / (24 * 60)) : 1,
        status: strategyDetail.status,
      });

      console.log('策略详情:', strategyDetail);

      // 设置当前编辑的策略ID
      setCurrentEditStrategyId(id);
    } catch (error) {
      console.error('获取策略详情失败:', error);
      // 如果获取失败，关闭模态框
      setEditStrategyModalVisible(false);

      Modal.error({
        title: '获取策略详情失败',
        content: '无法加载策略数据，请稍后重试'
      });
    } finally {
      // 清除当前策略的加载状态
      setStrategyLoading(prev => {
        const newState = { ...prev };
        delete newState[id];
        return newState;
      });
    }
  };

  const toggleStrategyStatus = async (id: number, currentStatus: string) => {
    const newStatus = currentStatus === 'enabled' ? 'disabled' : 'enabled';

    Modal.confirm({
      title: `确认${newStatus === 'enabled' ? '启用' : '禁用'}策略`,
      content: `确定要${newStatus === 'enabled' ? '启用' : '禁用'}该策略吗？`,
      onOk: async () => {
        try {
          console.log(`正在更新策略 ${id} 的状态为 ${newStatus}...`);
          await strategyApi.updateStrategyStatus(id, newStatus as 'enabled' | 'disabled');

          // 更新本地策略列表
          if (strategies) {
            const updatedStrategies = {
              ...strategies,
              list: strategies.list.map((strategy: Strategy) =>
                strategy.id === id ? { ...strategy, status: newStatus as 'enabled' | 'disabled' } : strategy
              )
            };
            // @ts-ignore - 类型兼容性问题，实际运行时没问题
            setStrategies(updatedStrategies);
          }
          console.log(`策略 ${id} 状态已更新为 ${newStatus}`);
        } catch (error) {
          console.error('更新策略状态失败:', error);
        }
      },
    });
  };

  const deleteStrategy = (id: number) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除此策略吗？此操作无法撤销。',
      onOk: async () => {
        try {
          await strategyApi.deleteStrategy(id);
          // 从本地列表中移除
          if (strategies) {
            const updatedStrategies = {
              ...strategies,
              list: strategies.list.filter((strategy: Strategy) => strategy.id !== id)
            };
            setStrategies(updatedStrategies);
          }
        } catch (error) {
          console.error('Error deleting strategy:', error);
        }
      }
    });
  };

  // 渲染统计卡片
  const renderStatCards = () => {
    if (!stats) return null;

    return (
      <Row gutter={24} className="stats-cards">
        <Col xs={24} sm={12} md={6}>
          <Card className="stat-card success">
            <div className="stat-value">{`${stats.enabledStrategyCount}/${stats.strategyCount}`}</div>
            <div className="stat-label">今日已巡检/总策略</div>
            <Progress percent={(stats.enabledStrategyCount / stats.strategyCount) * 100} size="small" />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card className="stat-card">
            <div className="stat-value">{`${stats.triggeredTodayCount}/${stats.enabledStrategyCount}`}</div>
            <div className="stat-label">巡检成功/已巡检策略</div>
            <div className="stat-trend">
              较昨日 <ArrowUpOutlined style={{ color: "#52c41a" }} /> <span style={{ color: "#52c41a" }}>2</span> 个成功策略
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card className={`stat-card ${stats.abnormalClusterCount > 0 ? 'warning' : ''}`}>
            <div className="stat-value">{`${stats.clusterCount - stats.abnormalClusterCount}/${stats.clusterCount}`}</div>
            <div className="stat-label">正常集群/总集群数</div>
            {stats.abnormalClusterCount > 0 && (
              <div className="stat-trend" style={{ color: "#faad14" }}>{stats.abnormalClusterCount}个集群需要处理</div>
            )}
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card className={`stat-card ${stats.pendingOrderCount > 0 ? 'error' : ''}`}>
            <div className="stat-value">{stats.pendingOrderCount}</div>
            <div className="stat-label">待处理资源伸缩任务</div>
            {stats.pendingOrderCount > 0 && (
              <div className="stat-trend">需立即处理的集群变更任务</div>
            )}
          </Card>
        </Col>
      </Row>
    );
  };

  // 渲染订单卡片
  const renderOrderCard = (order: OrderListItem, showResourceInfo: boolean = false) => {
    const actionTypeText = order.actionType === 'pool_entry' ? '入池' : '退池';
    const orderStatusText =
      order.status === 'pending' ? '待处理' :
      (order.status === 'processing' ? '处理中' : '已完成');

    return (
      <div
        key={order.id}
        className={`order-card ${order.actionType === 'pool_entry' ? 'pool-in' : 'pool-out'}`}
        onClick={() => handleViewOrderDetails(order.id)}
      >
        <div className="order-card-header">
          <div className="order-card-title">
            {order.actionType === 'pool_entry' ? (
              <CloudUploadOutlined style={{ color: '#1890ff' }} />
            ) : (
              <CloudDownloadOutlined style={{ color: '#faad14' }} />
            )}
            订单 #{order.orderNumber} - {order.strategyName || '未知策略'}
          </div>
          <Tag color={
            order.status === 'pending' ? 'error' :
            (order.status === 'processing' ? 'processing' : 'success')
          }>
            {orderStatusText}
          </Tag>
        </div>
        <div className="order-card-body">
          <div className="order-meta">
            <div className="order-meta-item">
              <div className="order-meta-label">类型</div>
              <div className="order-meta-value">{actionTypeText}</div>
            </div>
            <div className="order-meta-item">
              <div className="order-meta-label">触发时间</div>
              <div className="order-meta-value">{new Date(order.createdAt).toLocaleString()}</div>
            </div>
            <div className="order-meta-item">
              <div className="order-meta-label">集群</div>
              <div className="order-meta-value">{order.clusterName}</div>
            </div>
            <div className="order-meta-item">
              <div className="order-meta-label">设备数量</div>
              <div className="order-meta-value">{order.deviceCount} 台</div>
            </div>
          </div>

          {/* 资源利用率信息，仅在需要时显示 */}
          {showResourceInfo && (
            <div className="resource-info-card">
              <div className="resource-header">
                <ClusterOutlined style={{ marginRight: 8, color: '#1890ff' }} />
                集群: {order.clusterName}
              </div>
              <div className="resource-grid">
                <div>
                  <div className="resource-item-header">
                    <span>CPU使用率</span>
                    <span style={{ color: order.actionType === 'pool_entry' ? '#f5222d' : '#1890ff', fontWeight: 'bold' }}>
                      {order.actionType === 'pool_entry' ? 85 : 30}%
                      {order.actionType === 'pool_entry' ? <ArrowUpOutlined style={{ fontSize: 12 }} /> : <ArrowDownOutlined style={{ fontSize: 12 }} />}
                    </span>
                  </div>
                  <Progress
                    percent={order.actionType === 'pool_entry' ? 85 : 30}
                    size="small"
                    status={order.actionType === 'pool_entry' ? "exception" : "normal"}
                    strokeWidth={8}
                  />
                </div>
                <div>
                  <div className="resource-item-header">
                    <span>内存使用率</span>
                    <span style={{ color: order.actionType === 'pool_entry' ? '#f5222d' : '#52c41a', fontWeight: 'bold' }}>
                      {order.actionType === 'pool_entry' ? 75 : 35}%
                      {order.actionType === 'pool_entry' ? <ArrowUpOutlined style={{ fontSize: 12 }} /> : <ArrowDownOutlined style={{ fontSize: 12 }} />}
                    </span>
                  </div>
                  <Progress
                    percent={order.actionType === 'pool_entry' ? 75 : 35}
                    size="small"
                    status={order.actionType === 'pool_entry' ? "exception" : "normal"}
                    strokeColor={order.actionType === 'pool_entry' ? "#f5222d" : "#52c41a"}
                    strokeWidth={8}
                  />
                </div>
              </div>
              <Alert
                message={
                  order.actionType === 'pool_entry' ?
                    `CPU使用率已超过阈值(80%)，需添加节点提升集群容量` :
                    `CPU使用率低于阈值(35%)，可回收闲置节点`
                }
                type={order.actionType === 'pool_entry' ? "error" : "warning"}
                showIcon
                style={{ marginTop: 12 }}
                banner
              />
            </div>
          )}
        </div>
        <div className="order-card-footer">
          <Button type="link" icon={<EyeOutlined />} onClick={(e) => {
            e.stopPropagation();
            handleViewOrderDetails(order.id);
          }}>
            查看详情
          </Button>
          {order.status === 'pending' && (
            <Button
              type="primary"
              icon={order.actionType === 'pool_entry' ? <CloudUploadOutlined /> : <CloudDownloadOutlined />}
              onClick={(e) => {
                e.stopPropagation();
                executeOrder(order.id, order.actionType);
              }}
            >
              执行{actionTypeText}
            </Button>
          )}
        </div>
      </div>
    );
  };

  // 获取自定义状态的订单
  const getCustomStatusOrders = () => {
    if (!customTabStatus || !allOrders || !allOrders.list) return [];

    // Filter all orders to find those with the custom status
    return allOrders.list.filter((order: OrderListItem) => order.status === customTabStatus);
  };

  // 渲染订单统计卡片
  const renderOrderStats = () => {
    if (!allOrders) return null;

    const pendingCount = pendingOrders.length;
    const processingCount = processingOrders.length;
    const completedCount = completedOrders.length;
    // 假设忽略的订单数量，实际应从API获取
    const ignoredCount = 2;
    const totalCount = allOrders.total;

    return (
      <div className="order-status-summary">
        <div className="order-status-item">
          <div className="order-status-value order-status-pending">{pendingCount}</div>
          <div className="order-status-label">待处理</div>
        </div>
        <div className="order-status-item">
          <div className="order-status-value order-status-processing">{processingCount}</div>
          <div className="order-status-label">处理中</div>
        </div>
        <div className="order-status-item">
          <div className="order-status-value order-status-done">{completedCount}</div>
          <div className="order-status-label">已完成</div>
        </div>
        <div className="order-status-item">
          <div className="order-status-value order-status-ignored">{ignoredCount}</div>
          <div className="order-status-label">已忽略</div>
        </div>
        <div className="order-status-item">
          <div className="order-status-value">{totalCount}</div>
          <div className="order-status-label">总订单</div>
        </div>
      </div>
    );
  };

  // 渲染设备项
  const renderDeviceItem = (device: Device) => {
    return (
      <div key={device.id} className="device-item">
        <div className="device-info">
          <div className="device-name">{device.ciCode}</div>
          <div className="device-meta">
            <span>IP: {device.ip}</span>
            <span>集群: {device.cluster || '未分配'}</span>
          </div>
        </div>
        <span className={`device-status ${device.isSpecial ? 'status-special' : (device.cluster ? 'status-in-cluster' : 'status-available')}`}>
          {device.isSpecial ? '特殊设备' : (device.cluster ? '已入池' : '可入池')}
        </span>
      </div>
    );
  };

  // 初始化订单状态图表
  useEffect(() => {
    // 检查是否有订单数据
    if (pendingOrders && processingOrders && completedOrders) {
      const pendingCount = pendingOrders.length;
      const processingCount = processingOrders.length;
      const completedCount = completedOrders.length;

      // 假设忽略的订单数量，实际应从API获取
      const ignoredCount = 2;

      // 设置饼图数据
      setOrderStatusData({
        tooltip: {
          trigger: 'item',
          formatter: '{a} <br/>{b}: {c} ({d}%)'
        },
        legend: {
          top: '5%',
          left: 'center'
        },
        series: [
          {
            name: '订单状态',
            type: 'pie',
            radius: ['50%', '70%'],
            avoidLabelOverlap: false,
            itemStyle: {
              borderRadius: 10,
              borderWidth: 2
            },
            label: {
              show: false,
              position: 'center'
            },
            emphasis: {
              label: {
                show: true,
                fontSize: '16',
                fontWeight: 'bold'
              }
            },
            labelLine: {
              show: false
            },
            data: [
              {
                value: pendingCount,
                name: '待处理',
                itemStyle: { color: '#f5222d' }
              },
              {
                value: processingCount,
                name: '处理中',
                itemStyle: { color: '#faad14' }
              },
              {
                value: completedCount,
                name: '已完成',
                itemStyle: { color: '#52c41a' }
              },
              {
                value: ignoredCount,
                name: '已忽略',
                itemStyle: { color: '#8c8c8c' }
              }
            ]
          }
        ]
      });
    }
  }, [pendingOrders, processingOrders, completedOrders]);

  // 处理查看订单详情
  const handleViewOrderDetails = async (orderId: number) => {
    try {
      const orderDetail = await orderApi.getOrder(orderId);
      setSelectedOrder(orderDetail);
      setSelectedStrategy(null);
      setDrawerVisible(true);
    } catch (error) {
      console.error('Error fetching order details:', error);
    }
  };

  // 执行订单
  const executeOrder = async (orderId: number, actionType: string) => {
    try {
      // 更新订单状态为处理中
      await orderApi.updateOrderStatus(orderId, 'processing');

      // 提示用户
      Modal.success({
        title: '操作成功',
        content: `已开始执行${actionType === 'pool_entry' ? '入池' : '退池'}操作`,
      });

      // 刷新订单列表
      fetchData();
    } catch (error) {
      console.error('Error executing order:', error);
    }
  };

  // 关闭抽屉
  const handleCloseDrawer = () => {
    setDrawerVisible(false);
    setSelectedOrder(null);
    setSelectedStrategy(null);
  };

  // 打开创建订单模态框
  const handleOpenCreateOrderModal = () => {
    setCreateOrderModalVisible(true);
  };

  // 关闭创建订单模态框
  const handleCloseCreateOrderModal = () => {
    setCreateOrderModalVisible(false);
  };

  // 提交创建订单
  const handleCreateOrderSubmit = async (values: any) => {
    try {
      console.log('创建订单:', values);

      // 构建订单数据
      const orderData = {
        name: values.name,
        description: values.description || '',
        clusterId: values.clusterId,
        resourcePoolType: values.resourcePoolType,
        actionType: values.actionType,
        devices: values.devices || [],
        status: 'pending',
        createdBy: 'admin'
      };

      // 调用创建订单API
      await orderApi.createOrder(orderData);

      // 提示用户
      message.success('订单创建成功');

      // 关闭模态框
      setCreateOrderModalVisible(false);

      // 刷新订单列表
      fetchData();
    } catch (error) {
      console.error('创建订单失败:', error);
      message.error('创建订单失败');
    }
  };

  // 设置全局函数，供DeviceMatchingPolicy组件调用
  useEffect(() => {
    window.openCreateOrderModal = handleOpenCreateOrderModal;

    return () => {
      delete window.openCreateOrderModal;
    };
  }, []);

  // 添加资源类型多选
  const renderResourceTypeSelector = () => {
    const resourceTypeOptions = [
      { label: '所有资源', value: 'total' },
      { label: '计算资源', value: 'compute' },
      { label: '内存资源', value: 'memory' },
      { label: '存储资源', value: 'storage' },
      { label: '网络资源', value: 'network' },
    ];

    return (
      <Select
        mode="multiple"
        value={selectedResourceTypes}
        style={{ width: 200 }}
        onChange={(values: string[]) => {
          // 更新选中的资源类型
          setSelectedResourceTypes(values);

          // 只有当选择了集群且有选择资源类型时才加载数据
          if (selectedClusterId && values.length > 0) {
            fetchResourceTrend(selectedClusterId, selectedTimeRange, values);
          } else if (selectedClusterId && values.length === 0) {
            // 如果清空了所有选项，清空图表数据
            setCpuData(null);
            setMemoryData(null);
          }
        }}
        placeholder="选择资源类型"
        maxTagCount={2}
        allowClear
      >
        {resourceTypeOptions.map(option => (
          <Option key={option.value} value={option.value}>
            {option.label}
          </Option>
        ))}
      </Select>
    );
  };

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const renderCreateStrategyForm = () => {
    return (
      <Form
        form={form}
        layout="vertical"
      >
        {/* ... other form fields ... */}

        <Form.Item
          name="resourceTypes"
          label="资源类型"
          rules={[{ required: true, message: '请选择资源类型' }]}
        >
          <Select mode="multiple" placeholder="请选择资源类型">
            {resourceTypeOptions.map(type => (
              <Option key={type} value={type}>{type}</Option>
            ))}
          </Select>
        </Form.Item>

        {/* ... other form fields ... */}
      </Form>
    );
  };

  return (
    <div className="dashboard">
      {/* 页面标题 */}
      <div className="page-header">
        <div className="header-title">
          <BarChartOutlined className="header-icon" />
          <span>弹性伸缩管理</span>
        </div>
      </div>

      {/* 统计卡片 */}
      {renderStatCards()}

      <Divider />

      {/* 待处理订单区域 */}
      <Card
        className="content-card"
        title="待处理弹性伸缩订单"
        extra={
          <Space>
            <Select
              placeholder="订单类型"
              style={{ width: 120 }}
              allowClear
              onChange={(value) => setOrderFilter(value)}
            >
              <Option value="pool_entry">入池</Option>
              <Option value="pool_exit">退池</Option>
            </Select>
            <Button icon={<SearchOutlined />} onClick={() => fetchData()}>搜索</Button>
          </Space>
        }
      >
        {pendingOrders.length > 0 ? (
          <Row gutter={16}>
            {pendingOrders
              .filter(order => !orderFilter || order.actionType === orderFilter)
              .map(order => (
                <Col xs={24} sm={12} md={8} key={order.id}>
                  {renderOrderCard(order, true)}
                </Col>
              ))
            }
          </Row>
        ) : (
          <EmptyOrderState onCreateOrder={handleOpenCreateOrderModal} />
        )}
      </Card>

      {/* 资源用量趋势 */}
      <Card
        className="content-card"
        title="资源用量趋势"
        extra={
          <Space>
            <Select
              value={selectedClusterId}
              style={{ width: 150 }}
              onChange={(value) => handleTrendParamsChange(value, selectedTimeRange, selectedResourceTypes)}
              placeholder="选择集群"
            >
              {clusters.map(cluster => (
                <Option key={cluster.id} value={cluster.id}>{cluster.name}</Option>
              ))}
            </Select>
            {renderResourceTypeSelector()}
            <Select
              value={selectedTimeRange}
              style={{ width: 120 }}
              onChange={(value) => handleTrendParamsChange(selectedClusterId!, value, selectedResourceTypes)}
            >
              <Option value="24h">24小时</Option>
              <Option value="7d">7天</Option>
              <Option value="30d">30天</Option>
            </Select>
            <Button icon={<ReloadOutlined />} onClick={() => fetchResourceTrend(selectedClusterId!, selectedTimeRange, selectedResourceTypes)} />
          </Space>
        }
      >
        {isLoading ? (
          <div style={{ padding: '40px 0', textAlign: 'center' }}>
            <Spin size="large" tip="正在加载资源趋势数据..." />
          </div>
        ) : cpuData && memoryData ? (
          <Row gutter={24}>
            <Col xs={24} md={12}>
              <ReactECharts option={cpuData} className="chart-container" />
            </Col>
            <Col xs={24} md={12}>
              <ReactECharts option={memoryData} className="chart-container" />
            </Col>
          </Row>
        ) : (
          <Row gutter={24}>
            <Col xs={24} md={12}>
              <div className="empty-chart-container">
                <div className="empty-chart-content">
                  <BarChartOutlined className="empty-chart-icon" />
                  <div className="empty-chart-title">CPU使用率趋势</div>
                  <div className="empty-chart-subtitle">
                    {selectedClusterId ? '当前集群暂无CPU使用数据' : '请选择集群查看CPU使用趋势'}
                  </div>
                </div>
              </div>
            </Col>
            <Col xs={24} md={12}>
              <div className="empty-chart-container">
                <div className="empty-chart-content">
                  <AreaChartOutlined className="empty-chart-icon" />
                  <div className="empty-chart-title">内存使用率趋势</div>
                  <div className="empty-chart-subtitle">
                    {selectedClusterId ? '当前集群暂无内存使用数据' : '请选择集群查看内存使用趋势'}
                  </div>
                </div>
              </div>
            </Col>
            <Col xs={24} span={24} style={{ marginTop: '20px', textAlign: 'center' }}>
              <Button type="primary" icon={<ReloadOutlined />} onClick={() => {
                if (selectedClusterId) {
                  fetchResourceTrend(selectedClusterId, selectedTimeRange, selectedResourceTypes);
                } else if (pendingOrders && pendingOrders.length > 0) {
                  // 如果有待处理订单，优先从中随机选择一个集群
                  const randomIndex = Math.floor(Math.random() * pendingOrders.length);
                  const randomOrder = pendingOrders[randomIndex];

                  if (randomOrder && randomOrder.clusterId) {
                    console.log('从待处理订单中选择集群:', randomOrder.clusterId, '订单ID:', randomOrder.id);
                    setSelectedClusterId(randomOrder.clusterId);
                    fetchResourceTrend(randomOrder.clusterId, selectedTimeRange, selectedResourceTypes);
                  } else if (clusters.length > 0) {
                    // 如果待处理订单中没有有效的集群ID，则选择第一个集群
                    setSelectedClusterId(clusters[0].id);
                    fetchResourceTrend(clusters[0].id, selectedTimeRange, selectedResourceTypes);
                  }
                } else if (clusters.length > 0) {
                  // 如果没有待处理订单，选择第一个集群
                  setSelectedClusterId(clusters[0].id);
                  fetchResourceTrend(clusters[0].id, selectedTimeRange, selectedResourceTypes);
                } else {
                  message.warning('暂无可用集群');
                }
              }}>
                加载数据
              </Button>
            </Col>
          </Row>
        )}
      </Card>

      {/* 监控策略管理 */}
      <Card
        className="content-card"
        title="监控策略管理"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateStrategyModalVisible(true)}
          >
            新建策略
          </Button>
        }
      >
        {strategies ? (
          strategies.list && strategies.list.length > 0 ? (
            <Table
              columns={strategyColumns}
              dataSource={strategies.list}
              rowKey="id"
              className="strategy-table"
              pagination={{
                total: strategies.total,
                current: strategies.page,
                pageSize: strategies.size,
                showSizeChanger: true,
                showTotal: (total) => `共 ${total} 条`
              }}
            />
          ) : (
            <div className="empty-container">
              <ClusterOutlined className="empty-icon" />
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={
                  <div>
                    <p style={{ fontSize: '16px', marginBottom: '8px', color: 'rgba(0,0,0,0.65)' }}>暂无监控策略</p>
                    <p style={{ fontSize: '14px', color: 'rgba(0,0,0,0.45)' }}>
                      创建策略来监控集群资源，达到阈值条件时将自动触发弹性伸缩
                    </p>
                  </div>
                }
              />
              <div className="empty-action">
                <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateStrategyModalVisible(true)}>
                  创建第一个策略
                </Button>
              </div>
            </div>
          )
        ) : (
          <div className="loading-container">
            <p>加载中...</p>
          </div>
        )}
      </Card>

      {/* 设备匹配策略管理 */}
      <div style={{ marginTop: '24px', marginBottom: '24px' }}>
        <DeviceMatchingPolicy />
      </div>

      {/* 全部订单与统计 */}
      <Card
        className="content-card"
        title="全部订单与统计"
        extra={
          <Space>
            <Select
              value={orderStatusFilter === 'custom' ? customTabStatus : orderStatusFilter}
              style={{ width: 100 }}
              onChange={(value) => {
                // 检查选择的状态是否是标准标签页之一
                const standardTabs = ['processing', 'completed', 'ignored', 'all'];
                if (standardTabs.includes(value)) {
                  setOrderStatusFilter(value);
                  setCustomTabVisible(false);
                } else {
                  // 如果不是标准标签页，显示自定义标签页
                  setOrderStatusFilter('custom');
                  setCustomTabStatus(value);
                  setCustomTabVisible(true);
                }
              }}
            >
              <Option value="processing">处理中</Option>
              <Option value="completed">已完成</Option>
              <Option value="ignored">已忽略</Option>
              <Option value="all">全部</Option>
              <Option value="failed">失败</Option>
              <Option value="cancelled">已取消</Option>
              <Option value="expired">已过期</Option>
            </Select>
            <Select defaultValue="30d" style={{ width: 100 }}>
              <Option value="7d">最近7天</Option>
              <Option value="30d">最近30天</Option>
              <Option value="90d">最近90天</Option>
            </Select>
          </Space>
        }
      >
        <Row gutter={24}>
          <Col xs={24} md={8}>
            {orderStatusData && <ReactECharts option={orderStatusData} style={{ height: 300 }} />}
          </Col>
          <Col xs={24} md={16}>
            {/* 订单状态摘要 */}
            {renderOrderStats()}

            <Tabs
              activeKey={orderStatusFilter === 'custom' ? 'custom' : orderStatusFilter}
              onChange={(key) => {
                if (key === 'custom') {
                  // 保持自定义标签页激活
                  // 不需要修改orderStatusFilter，因为它已经是'custom'
                  // 但我们需要确保customTabStatus有值
                  if (!customTabStatus) {
                    setCustomTabStatus('failed'); // 默认使用'failed'作为自定义状态
                  }
                } else {
                  // 切换到标准标签页
                  setOrderStatusFilter(key);
                  setCustomTabVisible(false);
                }
              }}
              className="order-tabs"
            >
              <TabPane
                tab={
                  <span>
                    <Badge status="processing" />
                    处理中订单
                    <span className="order-count-badge processing">
                      {processingOrders.length}
                    </span>
                  </span>
                }
                key="processing"
              >
                <div className="order-cards-grid">
                  {processingOrders.length > 0 ? (
                    processingOrders.map(order => renderOrderCard(order))
                  ) : (
                    <div style={{ padding: '20px 0', textAlign: 'center' }}>
                      <Empty
                        description={
                          <span style={{ color: 'rgba(0,0,0,0.45)' }}>暂无处理中订单</span>
                        }
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                      />
                    </div>
                  )}
                </div>
              </TabPane>
              <TabPane
                tab={
                  <span>
                    <Badge status="success" />
                    已完成订单
                    <span className="order-count-badge done">
                      {completedOrders.length}
                    </span>
                  </span>
                }
                key="completed"
              >
                <div className="order-cards-grid">
                  {completedOrders.length > 0 ? (
                    completedOrders.map(order => renderOrderCard(order))
                  ) : (
                    <div style={{ padding: '20px 0', textAlign: 'center' }}>
                      <Empty
                        description={
                          <span style={{ color: 'rgba(0,0,0,0.45)' }}>暂无完成订单</span>
                        }
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                      />
                    </div>
                  )}
                </div>
              </TabPane>
              <TabPane
                tab={
                  <span>
                    <Badge status="default" color="#8c8c8c" />
                    已忽略订单
                    <span className="order-count-badge ignored">
                      2
                    </span>
                  </span>
                }
                key="ignored"
              >
                <div className="order-cards-grid">
                  <div style={{ padding: '20px 0', textAlign: 'center' }}>
                    <Empty
                      description={
                        <span style={{ color: 'rgba(0,0,0,0.45)' }}>暂无已忽略订单</span>
                      }
                      image={Empty.PRESENTED_IMAGE_SIMPLE}
                    />
                  </div>
                </div>
              </TabPane>
              <TabPane
                tab={
                  <span>
                    <Badge status="default" />
                    全部订单
                    <span className="order-count-badge all">
                      {allOrders ? allOrders.total : 0}
                    </span>
                  </span>
                }
                key="all"
              >
                <div className="order-cards-grid">
                  {allOrders && allOrders.list.length > 0 ? (
                    allOrders.list.map((order: OrderListItem) => renderOrderCard(order))
                  ) : (
                    <div style={{ padding: '20px 0', textAlign: 'center' }}>
                      <Empty
                        description={
                          <span style={{ color: 'rgba(0,0,0,0.45)' }}>暂无订单数据</span>
                        }
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                      />
                    </div>
                  )}
                </div>
              </TabPane>

              {/* 动态自定义状态标签页 */}
              {customTabVisible && (
                <TabPane
                  tab={
                    <span>
                      <Badge status="warning" color="#fa8c16" />
                      {customTabStatus === 'failed' ? '失败订单' :
                       customTabStatus === 'cancelled' ? '已取消订单' :
                       customTabStatus === 'expired' ? '已过期订单' :
                       customTabStatus === 'pending' ? '待处理订单' :
                       `${customTabStatus}订单`}
                      <span className="order-count-badge custom">
                        {getCustomStatusOrders().length}
                      </span>
                    </span>
                  }
                  key="custom"
                >
                  <div className="order-cards-grid">
                    {getCustomStatusOrders().length > 0 ? (
                      getCustomStatusOrders().map((order: OrderListItem) => renderOrderCard(order))
                    ) : (
                      <div style={{ padding: '20px 0', textAlign: 'center' }}>
                        <Empty
                          description={
                            <span style={{ color: 'rgba(0,0,0,0.45)' }}>暂无{customTabStatus === 'failed' ? '失败' :
                              customTabStatus === 'cancelled' ? '已取消' :
                              customTabStatus === 'expired' ? '已过期' :
                              customTabStatus === 'pending' ? '待处理' :
                              customTabStatus}订单</span>
                          }
                          image={Empty.PRESENTED_IMAGE_SIMPLE}
                        />
                      </div>
                    )}
                  </div>
                </TabPane>
              )}
            </Tabs>
          </Col>
        </Row>
      </Card>

      {/* 订单详情抽屉 */}
      <Drawer
        title={selectedOrder ? `订单详情 #${selectedOrder.orderNumber}` : (selectedStrategy ? `策略详情: ${selectedStrategy.name}` : '')}
        placement="right"
        width={600}
        onClose={handleCloseDrawer}
        visible={drawerVisible}
        className="detail-drawer"
      >
        {selectedOrder && (
          <div className="detail-drawer-content">
            <div className="detail-section">
              <Descriptions bordered size="small" column={2}>
                <Descriptions.Item label="订单类型" span={2}>
                  <Tag color={selectedOrder.actionType === 'pool_entry' ? 'blue' : 'orange'}>
                    {selectedOrder.actionType === 'pool_entry' ? <CloudUploadOutlined /> : <CloudDownloadOutlined />}
                    {selectedOrder.actionType === 'pool_entry' ? '入池' : '退池'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="订单状态">
                  <Badge
                    status={
                      selectedOrder.status === 'pending' ? 'error' :
                      (selectedOrder.status === 'processing' ? 'processing' : 'success')
                    }
                    text={
                      selectedOrder.status === 'pending' ? '待处理' :
                      (selectedOrder.status === 'processing' ? '处理中' : '已完成')
                    }
                  />
                </Descriptions.Item>
                <Descriptions.Item label="触发时间">
                  {new Date(selectedOrder.createdAt).toLocaleString()}
                </Descriptions.Item>
                <Descriptions.Item label="关联策略">
                  {selectedOrder.strategyName || '手动创建'}
                </Descriptions.Item>
                <Descriptions.Item label="集群">
                  {selectedOrder.clusterName}
                </Descriptions.Item>
              </Descriptions>
            </div>

            <div className="detail-section">
              <div className="detail-section-title">
                {selectedOrder.actionType === 'pool_entry' ? '匹配设备列表' : '关联设备列表'}
              </div>
              <div className="device-list">
                {selectedOrder.devices && selectedOrder.devices.length > 0 ? (
                  selectedOrder.devices.map((device: Device) => renderDeviceItem(device))
                ) : (
                  <Empty description="暂无设备数据" />
                )}
              </div>
            </div>
          </div>
        )}
      </Drawer>

      {/* 创建策略的Modal */}
      <Modal
        title="新建监控策略"
        visible={createStrategyModalVisible}
        onOk={() => {
          form.validateFields().then(async values => {
            console.log('Form values:', values);
            try {
              // Make sure resourceTypes is not an empty array
              if (Array.isArray(values.resourceTypes) && values.resourceTypes.length === 0) {
                values.resourceTypes = ['total'];
              }

              // 准备API请求数据
              const strategyData = {
                name: values.name,
                description: values.description || '',
                thresholdTriggerAction: values.thresholdTriggerAction as 'pool_entry' | 'pool_exit',
                resourceTypes: Array.isArray(values.resourceTypes) ? values.resourceTypes.join(',') : values.resourceTypes || 'total',
                clusterIds: values.clusterIds,

                // 转换时间从天到分钟
                durationMinutes: (values.durationDays || 1) * 24 * 60,
                cooldownMinutes: (values.cooldownDays || 1) * 24 * 60,

                // CPU相关参数
                cpuThresholdValue: values.cpuThresholdValue,
                cpuThresholdType: 'usage' as 'usage' | 'allocated',
                cpuTargetValue: values.cpuTargetValue,

                // 内存相关参数
                memoryThresholdValue: values.memoryThresholdValue,
                memoryThresholdType: 'usage' as 'usage' | 'allocated',
                memoryTargetValue: values.memoryTargetValue,

                // 设备数量和条件逻辑
                deviceCount: 1, // 默认值，实际应根据需求设置
                conditionLogic: values.conditionLogic as 'AND' | 'OR' || 'AND',
                status: values.status as 'enabled' | 'disabled' || 'disabled',

                // 其他必要字段
                nodeSelector: '',
                createdBy: 'admin', // 默认创建者
                clusters: []  // 这个字段会由后端填充
              };

              console.log('发送创建策略请求:', strategyData);

              // 调用创建策略API
              const result = await strategyApi.createStrategy(strategyData);
              console.log('策略创建成功:', result);

              // 刷新列表
              fetchData();

              // 关闭弹窗并重置表单
              setCreateStrategyModalVisible(false);
              form.resetFields();
            } catch (error) {
              console.error('创建策略失败:', error);
            }
          }).catch(errorInfo => {
            console.log('表单验证失败:', errorInfo);
          });
        }}
        onCancel={() => {
          setCreateStrategyModalVisible(false);
          form.resetFields();
        }}
        width={700}
        bodyStyle={{ padding: '24px', background: '#f9fbfd' }}
      >
        <div style={{ padding: '0 12px 24px', marginBottom: '24px', borderBottom: '1px solid #f0f0f0' }}>
          <Alert
            message="策略用于监控集群资源使用情况，当达到阈值条件时自动触发弹性伸缩"
            type="info"
            showIcon
            style={{ marginBottom: 0 }}
          />
        </div>

        <Form
          form={form}
          layout="vertical"
          requiredMark="optional"
          className="strategy-form"
        >
          <Row gutter={24}>
            <Col span={24}>
              <Form.Item
                name="name"
                label={<span style={{ fontWeight: 500 }}>策略名称</span>}
                rules={[{ required: true, message: '请输入策略名称!' }]}
              >
                <Input placeholder="请输入策略名称" autoComplete="off" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item
                name="description"
                label={<span style={{ fontWeight: 500 }}>策略描述</span>}
              >
                <Input.TextArea rows={3} placeholder="请输入策略描述" />
              </Form.Item>
            </Col>
          </Row>

          <Card
            title={<span style={{ fontSize: '14px', fontWeight: 500 }}>基本配置</span>}
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="thresholdTriggerAction"
                  label={<span style={{ fontWeight: 500 }}>触发动作</span>}
                  rules={[{ required: true, message: '请选择触发动作!' }]}
                >
                  <Select placeholder="请选择动作">
                    <Option value="pool_entry">入池</Option>
                    <Option value="pool_exit">退池</Option>
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="resourceTypes"
                  label={<span style={{ fontWeight: 500 }}>资源类型</span>}
                  rules={[{ required: true, message: '请选择资源类型!' }]}
                >
                  <Select
                    mode="multiple"
                    placeholder="请选择资源类型"
                    maxTagCount={3}
                  >
                    {resourceTypeOptions.map(type => (
                      <Option key={type} value={type}>
                        {type === 'compute' ? '计算型资源池' :
                         type === 'memory' ? '内存优化型资源池' :
                         type === 'storage' ? '存储优化型资源池' :
                         type === 'gpu' ? 'GPU加速型资源池' :
                         type === 'network' ? '网络优化型资源池' :
                         type === 'total' ? '全局资源' :
                         `${type}资源池`}
                      </Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              name="clusterIds"
              label={<span style={{ fontWeight: 500 }}>关联集群</span>}
              rules={[{ required: true, message: '请选择至少一个集群!' }]}
            >
              <Select
                mode="multiple"
                placeholder="请选择一个或多个集群"
                showArrow
                maxTagCount={3}
                style={{ width: '100%' }}
              >
                {clusters.map(cluster => (
                  <Option key={cluster.id} value={cluster.id}>{cluster.name}</Option>
                ))}
              </Select>
            </Form.Item>
          </Card>

          <Card
            title={<span style={{ fontSize: '14px', fontWeight: 500 }}>阈值配置</span>}
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="cpuThresholdValue"
                  label={<span style={{ fontWeight: 500 }}>
                    <WarningOutlined style={{ marginRight: 4, color: '#ff4d4f' }} />CPU阈值
                  </span>}
                  rules={[{ required: true, message: '请输入CPU阈值' }]}
                >
                  <InputNumber
                    min={1}
                    max={100}
                    addonAfter="%"
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="cpuTargetValue"
                  label={<span style={{ fontWeight: 500 }}>
                    <BarChartOutlined style={{ marginRight: 4, color: '#1890ff' }} />CPU目标值
                  </span>}
                  tooltip="触发动作后希望达到的CPU值"
                  rules={[{ required: true, message: '请输入CPU目标值' }]}
                >
                  <InputNumber
                    min={1}
                    max={100}
                    addonAfter="%"
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="memoryThresholdValue"
                  label={<span style={{ fontWeight: 500 }}>
                    <WarningOutlined style={{ marginRight: 4, color: '#ff4d4f' }} />内存阈值
                  </span>}
                  rules={[{ required: true, message: '请输入内存阈值' }]}
                >
                  <InputNumber
                    min={1}
                    max={100}
                    addonAfter="%"
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="memoryTargetValue"
                  label={<span style={{ fontWeight: 500 }}>
                    <BarChartOutlined style={{ marginRight: 4, color: '#1890ff' }} />内存目标值
                  </span>}
                  tooltip="触发动作后希望达到的内存值"
                  rules={[{ required: true, message: '请输入内存目标值' }]}
                >
                  <InputNumber
                    min={1}
                    max={100}
                    addonAfter="%"
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              name="conditionLogic"
              label={<span style={{ fontWeight: 500 }}>阈值条件</span>}
              rules={[{ required: true, message: '请选择阈值条件!' }]}
            >
              <Radio.Group buttonStyle="solid">
                <Radio.Button value="AND">同时满足</Radio.Button>
                <Radio.Button value="OR">满足其一</Radio.Button>
              </Radio.Group>
            </Form.Item>
          </Card>

          <Card
            title={<span style={{ fontSize: '14px', fontWeight: 500 }}>时间配置</span>}
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="durationDays"
                  label={<span style={{ fontWeight: 500 }}>
                    <ClockCircleOutlined style={{ marginRight: 4, color: '#52c41a' }} />持续时间(天)
                  </span>}
                  rules={[{ required: true, message: '请输入持续时间!' }]}
                  tooltip="策略触发条件需要持续满足的时间"
                >
                  <InputNumber min={1} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="cooldownDays"
                  label={<span style={{ fontWeight: 500 }}>
                    <PauseCircleOutlined style={{ marginRight: 4, color: '#faad14' }} />冷却时间(天)
                  </span>}
                  rules={[{ required: true, message: '请输入冷却时间!' }]}
                  tooltip="策略触发后的冷却周期，期间不会再次触发"
                >
                  <InputNumber min={1} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            </Row>
          </Card>

          <Form.Item
            name="status"
            label={<span style={{ fontWeight: 500 }}>策略状态</span>}
            initialValue="disabled"
          >
            <Radio.Group buttonStyle="solid">
              <Radio.Button value="enabled">启用</Radio.Button>
              <Radio.Button value="disabled">禁用</Radio.Button>
            </Radio.Group>
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑策略的Modal */}
      <Modal
        title="编辑监控策略"
        visible={editStrategyModalVisible}
        maskClosable={false}
        destroyOnClose={false}
        className="edit-strategy-modal"
        onOk={() => {
          editForm.validateFields().then(async values => {
            try {
              if (!currentEditStrategyId) return;

              // Make sure resourceTypes is not an empty array
              if (Array.isArray(values.resourceTypes) && values.resourceTypes.length === 0) {
                values.resourceTypes = ['total'];
              }

              // 准备API请求数据
              const strategyData = {
                name: values.name,
                description: values.description || '',
                thresholdTriggerAction: values.thresholdTriggerAction as 'pool_entry' | 'pool_exit',
                resourceTypes: Array.isArray(values.resourceTypes) ? values.resourceTypes.join(',') : values.resourceTypes || 'total',
                clusterIds: values.clusterIds,

                // 转换时间从天到分钟
                durationMinutes: (values.durationDays || 1) * 24 * 60,
                cooldownMinutes: (values.cooldownDays || 1) * 24 * 60,

                // CPU相关参数
                cpuThresholdValue: values.cpuThresholdValue,
                cpuThresholdType: 'usage' as 'usage' | 'allocated',
                cpuTargetValue: values.cpuTargetValue,

                // 内存相关参数
                memoryThresholdValue: values.memoryThresholdValue,
                memoryThresholdType: 'usage' as 'usage' | 'allocated',
                memoryTargetValue: values.memoryTargetValue,

                // 设备数量和条件逻辑
                deviceCount: 1, // 默认值，实际应根据需求设置
                conditionLogic: values.conditionLogic as 'AND' | 'OR',
                status: values.status as 'enabled' | 'disabled',

                // 其他必要字段
                nodeSelector: '',
                createdBy: 'admin', // 默认创建者
                clusters: []  // 这个字段会由后端填充
              };

              console.log('发送更新策略请求:', strategyData);

              // 调用更新策略API
              await strategyApi.updateStrategy(currentEditStrategyId, strategyData);

              // 刷新列表
              fetchData();

              // 关闭弹窗并重置状态
              setEditStrategyModalVisible(false);
              setCurrentEditStrategyId(null);
              editForm.resetFields();

              message.success('策略更新成功');
            } catch (error) {
              console.error('更新策略失败:', error);
              message.error('更新策略失败，请重试');
            }
          }).catch(errorInfo => {
            console.log('表单验证失败:', errorInfo);
          });
        }}
        onCancel={() => {
          setEditStrategyModalVisible(false);
          setCurrentEditStrategyId(null);
          editForm.resetFields();
        }}
        width={700}
        bodyStyle={{ padding: '24px', background: '#f9fbfd' }}
      >
        {/* 加载状态指示器 */}
        {currentEditStrategyId && strategyLoading[currentEditStrategyId] ? (
          <div style={{ textAlign: 'center', padding: '30px 0' }}>
            <Spin size="large" tip="加载策略数据..." />
          </div>
        ) : (
          <>
            <div style={{ padding: '0 12px 24px', marginBottom: '24px', borderBottom: '1px solid #f0f0f0' }}>
              <Alert
                message="编辑策略参数，保存后会立即生效"
                type="info"
                showIcon
                style={{ marginBottom: 0 }}
              />
            </div>

            <Form
          form={editForm}
          layout="vertical"
          requiredMark="optional"
          className="strategy-form"
        >
          <Row gutter={24}>
            <Col span={24}>
              <Form.Item
                name="name"
                label={<span style={{ fontWeight: 500 }}>策略名称</span>}
                rules={[{ required: true, message: '请输入策略名称!' }]}
              >
                <Input placeholder="请输入策略名称" autoComplete="off" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item
                name="description"
                label={<span style={{ fontWeight: 500 }}>策略描述</span>}
              >
                <Input.TextArea rows={3} placeholder="请输入策略描述" />
              </Form.Item>
            </Col>
          </Row>

          <Card
            title={<span style={{ fontSize: '14px', fontWeight: 500 }}>基本配置</span>}
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="thresholdTriggerAction"
                  label={<span style={{ fontWeight: 500 }}>触发动作</span>}
                  rules={[{ required: true, message: '请选择触发动作!' }]}
                >
                  <Select placeholder="请选择动作">
                    <Option value="pool_entry">入池</Option>
                    <Option value="pool_exit">退池</Option>
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="resourceTypes"
                  label={<span style={{ fontWeight: 500 }}>资源类型</span>}
                  rules={[{ required: true, message: '请选择资源类型!' }]}
                >
                  <Select
                    mode="multiple"
                    placeholder="请选择资源类型"
                    maxTagCount={3}
                  >
                    {resourceTypeOptions.map(type => (
                      <Option key={type} value={type}>
                        {type === 'compute' ? '计算型资源池' :
                         type === 'memory' ? '内存优化型资源池' :
                         type === 'storage' ? '存储优化型资源池' :
                         type === 'gpu' ? 'GPU加速型资源池' :
                         type === 'network' ? '网络优化型资源池' :
                         type === 'total' ? '全局资源' :
                         `${type}资源池`}
                      </Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              name="clusterIds"
              label={<span style={{ fontWeight: 500 }}>关联集群</span>}
              rules={[{ required: true, message: '请选择至少一个集群!' }]}
            >
              <Select
                mode="multiple"
                placeholder="请选择一个或多个集群"
                showArrow
                maxTagCount={3}
                style={{ width: '100%' }}
              >
                {clusters.map(cluster => (
                  <Option key={cluster.id} value={cluster.id}>{cluster.name}</Option>
                ))}
              </Select>
            </Form.Item>
          </Card>

          <Card
            title={<span style={{ fontSize: '14px', fontWeight: 500 }}>阈值配置</span>}
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="cpuThresholdValue"
                  label={<span style={{ fontWeight: 500 }}>
                    <WarningOutlined style={{ marginRight: 4, color: '#ff4d4f' }} />CPU阈值
                  </span>}
                  rules={[{ required: true, message: '请输入CPU阈值' }]}
                >
                  <InputNumber
                    min={1}
                    max={100}
                    addonAfter="%"
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="cpuTargetValue"
                  label={<span style={{ fontWeight: 500 }}>
                    <BarChartOutlined style={{ marginRight: 4, color: '#1890ff' }} />CPU目标值
                  </span>}
                  tooltip="触发动作后希望达到的CPU值"
                  rules={[{ required: true, message: '请输入CPU目标值' }]}
                >
                  <InputNumber
                    min={1}
                    max={100}
                    addonAfter="%"
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="memoryThresholdValue"
                  label={<span style={{ fontWeight: 500 }}>
                    <WarningOutlined style={{ marginRight: 4, color: '#ff4d4f' }} />内存阈值
                  </span>}
                  rules={[{ required: true, message: '请输入内存阈值' }]}
                >
                  <InputNumber
                    min={1}
                    max={100}
                    addonAfter="%"
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="memoryTargetValue"
                  label={<span style={{ fontWeight: 500 }}>
                    <BarChartOutlined style={{ marginRight: 4, color: '#1890ff' }} />内存目标值
                  </span>}
                  tooltip="触发动作后希望达到的内存值"
                  rules={[{ required: true, message: '请输入内存目标值' }]}
                >
                  <InputNumber
                    min={1}
                    max={100}
                    addonAfter="%"
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              name="conditionLogic"
              label={<span style={{ fontWeight: 500 }}>阈值条件</span>}
              rules={[{ required: true, message: '请选择阈值条件!' }]}
            >
              <Radio.Group buttonStyle="solid">
                <Radio.Button value="AND">同时满足</Radio.Button>
                <Radio.Button value="OR">满足其一</Radio.Button>
              </Radio.Group>
            </Form.Item>
          </Card>

          <Card
            title={<span style={{ fontSize: '14px', fontWeight: 500 }}>时间配置</span>}
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="durationDays"
                  label={<span style={{ fontWeight: 500 }}>
                    <ClockCircleOutlined style={{ marginRight: 4, color: '#52c41a' }} />持续时间(天)
                  </span>}
                  rules={[{ required: true, message: '请输入持续时间!' }]}
                  tooltip="策略触发条件需要持续满足的时间"
                >
                  <InputNumber min={1} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="cooldownDays"
                  label={<span style={{ fontWeight: 500 }}>
                    <PauseCircleOutlined style={{ marginRight: 4, color: '#faad14' }} />冷却时间(天)
                  </span>}
                  rules={[{ required: true, message: '请输入冷却时间!' }]}
                  tooltip="策略触发后的冷却周期，期间不会再次触发"
                >
                  <InputNumber min={1} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            </Row>
          </Card>

          <Form.Item
            name="status"
            label={<span style={{ fontWeight: 500 }}>策略状态</span>}
          >
            <Radio.Group buttonStyle="solid">
              <Radio.Button value="enabled">启用</Radio.Button>
              <Radio.Button value="disabled">禁用</Radio.Button>
            </Radio.Group>
          </Form.Item>
        </Form>
          </>
        )}
      </Modal>

      {/* 创建订单模态框 */}
      <CreateOrderModal
        visible={createOrderModalVisible}
        onCancel={handleCloseCreateOrderModal}
        onSubmit={handleCreateOrderSubmit}
        clusters={clusters}
        resourcePools={resourcePools}
      />
    </div>
  );
};

export default Dashboard;