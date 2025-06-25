/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useEffect, useState, useRef, useCallback } from 'react';
import {
  Card, Row, Col, Progress, Button, Table, Tag, Space,
  Select, Tabs, Empty, Divider, Alert, Tooltip, Badge,
  Modal, Form, Input, InputNumber, Radio, Drawer, Descriptions, message,
  Spin,
  Result, Checkbox
} from 'antd';
import {
  CloudServerOutlined, RocketOutlined, AppstoreOutlined, SettingOutlined,
  PlusOutlined, SearchOutlined, ReloadOutlined, EditOutlined,
  DeleteOutlined, CloudUploadOutlined, CloudDownloadOutlined, CheckCircleOutlined,
  ArrowUpOutlined, ArrowDownOutlined, ClusterOutlined, BarChartOutlined,
  ClockCircleOutlined, PauseCircleOutlined, WarningOutlined,
  LinkOutlined, DisconnectOutlined, AreaChartOutlined,
  DesktopOutlined, ExclamationCircleOutlined, CloseCircleOutlined, EyeOutlined,
  StopOutlined, DatabaseOutlined, CopyOutlined, CheckOutlined
} from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';

import type { OrderStatus } from '../../types/order';
import {
  ResourceAllocationTrend,
  ResourceTypeData,
  Strategy,
  StrategyDetail,
  OrderDetail,
  OrderListItem,
  Device as ElasticScalingDevice,
  DashboardStats,
  PaginatedResponse,
  OrderStats
} from '../../types/elastic-scaling';

import { statsApi, strategyApi, orderApi } from '../../services/elasticScalingService';
import clusterService, { getResourcePoolAllocationRate, ResourcePoolAllocationRate } from '../../services/clusterService';
import { getDeviceFeatureDetails } from '../../services/deviceQueryService';

import OrderStatusFlow from './OrderStatusFlow';
import DeviceMatchingPolicy from './DeviceMatchingPolicy';
import EmptyOrderState from './EmptyOrderState';
import CreateOrderModal from './CreateOrderModal';
import ProseKitViewer from './ProseKitViewer';

import './Dashboard.css';
import './DeviceMatchingPolicy.less';

// 扩展DOM元素类型定义以支持自定义属性
declare global {
  interface HTMLElement {
    tooltip?: HTMLElement | null;
    handleMouseMove?: ((event: MouseEvent) => void) | null;
  }
}

// 扩展弹性伸缩的Device类型以包含设备中心的属性
interface Device extends ElasticScalingDevice {
  group?: string;        // 机器用途
  appName?: string;      // 应用名称
}

const { confirm } = Modal;
const { Option } = Select;
const { TabPane } = Tabs;

// Define type aliases for complex types to make code more readable
type StrategiesState = PaginatedResponse<Strategy> | null;
type OrdersState = PaginatedResponse<OrderListItem> | null;


const Dashboard: React.FC = () => {
  // 状态管理
  const [isLoading, setIsLoading] = useState<boolean>(false);

  // 订单实际分配率数据缓存
  const allocationDataCache = useRef(new Map<number, { data: OrderListItem, timestamp: number }>()).current;
  const ALLOCATION_CACHE_DURATION = 5 * 60 * 1000; // 5分钟

  const createOrderModalRef = useRef<any>(null);
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [strategies, setStrategies] = useState<StrategiesState>(null);
  const [orders, setOrders] = useState<OrdersState>(null);
  const [pendingOrders, setPendingOrders] = useState<OrderListItem[]>([]);
  const pendingOrdersRef = useRef(pendingOrders);
  useEffect(() => {
    pendingOrdersRef.current = pendingOrders;
  }, [pendingOrders]);
  const [processingOrders, setProcessingOrders] = useState<OrderListItem[]>([]);
  const [completedOrders, setCompletedOrders] = useState<OrderListItem[]>([]);
  const [allOrders, setAllOrders] = useState<PaginatedResponse<OrderListItem> | null>(null);
  const [bottomAllOrders, setBottomAllOrders] = useState<PaginatedResponse<OrderListItem> | null>(null);
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
  const [statusFilter, setStatusFilter] = useState<string | null>(null);
  const [nameFilter, setNameFilter] = useState<string>('');
  const searchTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const [orderStatusFilter, setOrderStatusFilter] = useState<string>('completed'); // 默认显示已完成订单
  const [customTabVisible, setCustomTabVisible] = useState<boolean>(false);
  const [customTabStatus, setCustomTabStatus] = useState<string>('');
  const [createStrategyModalVisible, setCreateStrategyModalVisible] = useState(false);
  const [editStrategyModalVisible, setEditStrategyModalVisible] = useState(false);
  const [currentEditStrategyId, setCurrentEditStrategyId] = useState<number | null>(null);
  const [createOrderModalVisible, setCreateOrderModalVisible] = useState(false);
  const [clonedOrderInfo, setClonedOrderInfo] = useState<any>(null);
  const [resourcePools, setResourcePools] = useState<any[]>([]);

  // 策略执行历史相关状态
  const [executionHistoryDrawerVisible, setExecutionHistoryDrawerVisible] = useState(false);
  const [selectedStrategyForHistory, setSelectedStrategyForHistory] = useState<Strategy | null>(null);
  const [executionHistory, setExecutionHistory] = useState<any[]>([]);
  const [executionHistoryLoading, setExecutionHistoryLoading] = useState(false);
  const [executionHistoryTotal, setExecutionHistoryTotal] = useState(0);
  const [executionHistoryPagination, setExecutionHistoryPagination] = useState({ page: 1, size: 10 });
  const [historyClusterFilter, setHistoryClusterFilter] = useState<string>('');

  const handleCloneOrder = async (order: OrderListItem) => {
    try {
      // 先获取订单详情以获取设备列表
      const orderDetail = await orderApi.getOrder(order.id);

      const cloneData = {
        name: order.name || order.orderNumber,
        resourcePoolType: order.resourcePoolType,
        deviceCount: order.deviceCount,
        devices: orderDetail.devices || []
      };

      setCreateOrderModalVisible(true);
      if (createOrderModalRef.current) {
        createOrderModalRef.current.open(cloneData);
      }
    } catch (error: any) {
      console.error('获取订单详情失败:', error);
      const errorMsg = error?.response?.data?.msg || error?.message || '获取订单详情失败';
      message.error(errorMsg);
      // 如果获取详情失败，仍然打开模态框但不包含设备信息
      const fallbackData = {
        name: order.name || order.orderNumber,
        resourcePoolType: order.resourcePoolType,
        deviceCount: order.deviceCount,
        devices: []
      };
      console.log('Dashboard克隆订单 - 失败时传递给Modal的数据:', fallbackData);

      setCreateOrderModalVisible(true);
      if (createOrderModalRef.current) {
        createOrderModalRef.current.open(fallbackData);
      }
    }
  };

  // 查看策略执行历史
  const handleViewExecutionHistory = async (strategy: Strategy, clusterNameFilter?: string, resetPagination = true) => {
    setSelectedStrategyForHistory(strategy);
    setExecutionHistoryDrawerVisible(true);
    setExecutionHistoryLoading(true);
    if (clusterNameFilter === undefined) {
      setHistoryClusterFilter(''); // 重置集群过滤器
    }
    
    // 重置分页到第一页
    if (resetPagination) {
      setExecutionHistoryPagination({ page: 1, size: 10 });
    }

    try {
      const response = await strategyApi.getStrategyExecutionHistory(strategy.id, {
        page: resetPagination ? 1 : executionHistoryPagination.page,
        size: resetPagination ? 10 : executionHistoryPagination.size,
        clusterName: clusterNameFilter || historyClusterFilter || undefined
      });
      // 从分页响应中提取历史数据
      const history = response.data || [];
      const total = response.total || 0;
      // 后端已经返回了集群名称，不需要前端再次处理
      setExecutionHistory(history || []);
      setExecutionHistoryTotal(total);
    } catch (error: any) {
      console.error('获取策略执行历史失败:', error);
      const errorMsg = error?.response?.data?.msg || error?.message || '获取策略执行历史失败';
      message.error(errorMsg);
      setExecutionHistory([]);
      setExecutionHistoryTotal(0);
    } finally {
      setExecutionHistoryLoading(false);
    }
  };

  // 处理分页变化
  const handleExecutionHistoryPaginationChange = async (page: number, size: number) => {
    setExecutionHistoryPagination({ page, size });
    if (selectedStrategyForHistory) {
      setExecutionHistoryLoading(true);
      try {
        const response = await strategyApi.getStrategyExecutionHistory(selectedStrategyForHistory.id, {
          page,
          size,
          clusterName: historyClusterFilter || undefined
        });
        const history = response.data || [];
        const total = response.total || 0;
        setExecutionHistory(history || []);
        setExecutionHistoryTotal(total);
      } catch (error: any) {
        console.error('获取策略执行历史失败:', error);
        const errorMsg = error?.response?.data?.msg || error?.message || '获取策略执行历史失败';
        message.error(errorMsg);
      } finally {
        setExecutionHistoryLoading(false);
      }
    }
  };

  // 处理集群过滤器变化
  const handleClusterFilterChange = (value: string) => {
    setHistoryClusterFilter(value);
    // 当过滤条件改变时，重新获取数据并重置分页
    if (selectedStrategyForHistory) {
      handleViewExecutionHistory(selectedStrategyForHistory, value, true);
    }
    // 延迟搜索，避免频繁请求
    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }
    searchTimeoutRef.current = setTimeout(() => {
      if (selectedStrategyForHistory) {
        handleViewExecutionHistory(selectedStrategyForHistory, value);
      }
    }, 500);
  };

  // 资源池类型列表
  const [resourceTypeOptions, setResourceTypeOptions] = useState<string[]>([]);
  // 集群列表
  const [clusters, setClusters] = useState<any[]>([]);

  // 已取消订单
  const [cancelledOrders, setCancelledOrders] = useState<OrderListItem[]>([]);
  const [cancelledOrdersLoading, setCancelledOrdersLoading] = useState<boolean>(false);
  const [cancelledOrdersError, setCancelledOrdersError] = useState<string | null>(null);
  const [orderStats, setOrderStats] = useState<OrderStats | null>(null);
  const [ordersLoading, setOrdersLoading] = useState<boolean>(false);
  const [ordersError, setOrdersError] = useState<string | null>(null);

  // 策略表单
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  // 获取订单的实际分配率数据（带缓存）
  const fetchOrderAllocationData = useCallback(async (order: OrderListItem, availableClusters?: any[]): Promise<OrderListItem> => {
    console.log(`[fetchOrderAllocationData] 开始获取订单 ${order.id} 的分配率数据`);
    console.log(`[fetchOrderAllocationData] 订单信息:`, { 
      id: order.id, 
      clusterId: order.clusterId, 
      resourcePoolType: order.resourcePoolType,
      clusterName: order.clusterName 
    });
    
    const cachedData = allocationDataCache.get(order.id);
    if (cachedData && Date.now() - cachedData.timestamp < ALLOCATION_CACHE_DURATION) {
      console.log(`[fetchOrderAllocationData] 使用缓存数据，订单 ${order.id}`);
      return cachedData.data;
    }

    try {
      // 使用传入的集群数据或当前状态的集群数据
      const clustersToUse = availableClusters || clusters;
      console.log(`[fetchOrderAllocationData] 可用集群数量:`, clustersToUse?.length || 0);
      console.log(`[fetchOrderAllocationData] 可用集群列表:`, clustersToUse?.map(c => ({ id: c.id, name: c.name })) || []);
      
      // 修复：传递 clusterName 和 resourcePoolType 字符串，而不是 clusterId 和数组
      const cluster = clustersToUse?.find(c => c.id === order.clusterId);
      if (!cluster) {
        console.error(`[fetchOrderAllocationData] 未找到 ID 为 ${order.clusterId} 的集群`);
        console.error(`[fetchOrderAllocationData] 当前可用集群:`, clustersToUse?.map(c => ({ id: c.id, name: c.name })) || []);
        throw new Error(`未找到 ID 为 ${order.clusterId} 的集群`);
      }

      console.log(`[fetchOrderAllocationData] 找到集群:`, { id: cluster.id, name: cluster.name });
      console.log(`[fetchOrderAllocationData] 准备调用API: clusterName=${cluster.name}, resourcePool=${order.resourcePoolType || 'total'}`);

      const allocationData = await getResourcePoolAllocationRate(cluster.name, order.resourcePoolType || 'total');
      
      console.log(`[fetchOrderAllocationData] API响应:`, allocationData);

      // 修复：正确处理可能为 null 的 allocationData 并访问对象属性
      const enhancedOrder = {
        ...order,
        actualCpuAllocation: allocationData?.cpu_rate,
        actualMemoryAllocation: allocationData?.memory_rate,
        hasAllocationData: allocationData !== null,
      };
      
      console.log(`[fetchOrderAllocationData] 增强后的订单数据:`, {
        id: enhancedOrder.id,
        actualCpuAllocation: enhancedOrder.actualCpuAllocation,
        actualMemoryAllocation: enhancedOrder.actualMemoryAllocation,
        hasAllocationData: enhancedOrder.hasAllocationData
      });
      
      allocationDataCache.set(order.id, { data: enhancedOrder, timestamp: Date.now() });
      return enhancedOrder;
    } catch (error) {
      console.error(`[fetchOrderAllocationData] 获取订单 ${order.id} 的分配率数据失败:`, error);
      return { ...order, hasAllocationData: false };
    }
  }, [allocationDataCache, clusters, ALLOCATION_CACHE_DURATION]);

  // 清理过期的分配率数据缓存
  const clearAllocationDataCache = useCallback(() => {
    const now = Date.now();
    // 修复：使用 forEach 替代 for...of 来避免 TS2802 错误
    allocationDataCache.forEach((value, key) => {
      if (now - value.timestamp > ALLOCATION_CACHE_DURATION) {
        allocationDataCache.delete(key);
      }
    });
  }, [allocationDataCache, ALLOCATION_CACHE_DURATION]);

  // 策略表格列定义
  const strategyColumns = [
    {
      title: '策略名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      render: (text: string, record: Strategy) => (
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <SettingOutlined style={{ marginRight: 8, color: '#1890ff', fontSize: '14px' }} />
          <a 
            href="#!" 
            onClick={(e) => { e.preventDefault(); editStrategy(record.id); }}
            style={{ fontWeight: 500, color: '#1890ff' }}
          >
            {text}
          </a>
        </div>
      ),
    },
    {
      title: '阈值配置',
      key: 'thresholdConfig',
      width: 240,
      render: (text: string, record: Strategy) => {
        const hasCpuConfig = record.cpuThresholdValue && record.cpuTargetValue;
        const hasMemoryConfig = record.memoryThresholdValue && record.memoryTargetValue;

        if (!hasCpuConfig && !hasMemoryConfig) {
          return <span style={{ color: 'rgba(0, 0, 0, 0.45)' }}>--</span>;
        }

        return (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {/* CPU 阈值配置 */}
            {hasCpuConfig && (
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <div style={{ display: 'flex', alignItems: 'center', minWidth: '40px' }}>
                  <span style={{ fontSize: '12px', color: '#1890ff', fontWeight: 500 }}>CPU</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '4px', flex: 1 }}>
                  {/* 触发阈值 */}
                  <div
                    style={{
                      background: getPercentageColor(record.cpuThresholdValue),
                      color: 'white',
                      padding: '2px 6px',
                      borderRadius: '4px',
                      fontSize: '11px',
                      fontWeight: 500,
                      minWidth: '35px',
                      textAlign: 'center'
                    }}
                    title={`CPU阈值: ${record.cpuThresholdValue}%, 颜色: ${getPercentageColor(record.cpuThresholdValue)}`}
                  >
                    {record.cpuThresholdValue ? `${record.cpuThresholdValue}%` : '--'}
                  </div>

                  {/* 波动箭头 */}
                  <div style={{ display: 'flex', alignItems: 'center', margin: '0 2px' }}>
                    {record.thresholdTriggerAction === 'pool_entry' ? (
                      <ArrowDownOutlined style={{ color: '#52c41a', fontSize: '12px' }} />
                    ) : (
                      <ArrowUpOutlined style={{ color: '#ff7a45', fontSize: '12px' }} />
                    )}
                  </div>

                  {/* 目标值 */}
                  <div style={{
                    background: getPercentageColor(record.cpuTargetValue),
                    color: 'white',
                    padding: '2px 6px',
                    borderRadius: '4px',
                    fontSize: '11px',
                    fontWeight: 500,
                    minWidth: '35px',
                    textAlign: 'center'
                  }}>
                    {record.cpuTargetValue ? `${record.cpuTargetValue}%` : '--'}
                  </div>
                </div>
              </div>
            )}

            {/* 内存阈值配置 */}
            {hasMemoryConfig && (
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <div style={{ display: 'flex', alignItems: 'center', minWidth: '40px' }}>
                  <span style={{ fontSize: '12px', color: '#722ed1', fontWeight: 500 }}>MEM</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '4px', flex: 1 }}>
                  {/* 触发阈值 */}
                  <div style={{
                    background: getPercentageColor(record.memoryThresholdValue),
                    color: 'white',
                    padding: '2px 6px',
                    borderRadius: '4px',
                    fontSize: '11px',
                    fontWeight: 500,
                    minWidth: '35px',
                    textAlign: 'center'
                  }}>
                    {record.memoryThresholdValue ? `${record.memoryThresholdValue}%` : '--'}
                  </div>

                  {/* 波动箭头 */}
                  <div style={{ display: 'flex', alignItems: 'center', margin: '0 2px' }}>
                    {record.thresholdTriggerAction === 'pool_entry' ? (
                      <ArrowDownOutlined style={{ color: '#52c41a', fontSize: '12px' }} />
                    ) : (
                      <ArrowUpOutlined style={{ color: '#ff7a45', fontSize: '12px' }} />
                    )}
                  </div>

                  {/* 目标值 */}
                  <div style={{
                    background: getPercentageColor(record.memoryTargetValue),
                    color: 'white',
                    padding: '2px 6px',
                    borderRadius: '4px',
                    fontSize: '11px',
                    fontWeight: 500,
                    minWidth: '35px',
                    textAlign: 'center'
                  }}>
                    {record.memoryTargetValue ? `${record.memoryTargetValue}%` : '--'}
                  </div>
                </div>
              </div>
            )}
          </div>
        );
      },
    },
    {
      title: '条件',
      key: 'condition',
      width: 90,
      align: 'center' as const,
      render: (text: string, record: Strategy) => (
        record.cpuThresholdValue && record.memoryThresholdValue ? (
          <Tag 
            color={record.conditionLogic === 'AND' ? 'blue' : 'orange'}
            style={{ borderRadius: '6px', fontWeight: 500 }}
          >
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
      width: 90,
      align: 'center' as const,
      render: (text: string) => (
        <Tag 
          color={text === 'pool_entry' ? 'blue' : 'orange'}
          style={{ borderRadius: '6px', fontWeight: 500, minWidth: '60px', textAlign: 'center' }}
        >
          {text === 'pool_entry' ? <CloudUploadOutlined /> : <CloudDownloadOutlined />}
          <span style={{ marginLeft: 4 }}>{text === 'pool_entry' ? '入池' : '退池'}</span>
        </Tag>
      ),
    },
    {
      title: '时间配置',
      key: 'timeConfig',
      width: 140,
      render: (text: string, record: Strategy) => (
        <div style={{ padding: '4px 0' }}>
          <div style={{ 
            display: 'flex', 
            alignItems: 'center', 
            marginBottom: 6,
            padding: '3px 8px',
            backgroundColor: '#f0f9ff',
            borderRadius: '4px',
            fontSize: '12px',
            border: '1px solid #e6f7ff'
          }}>
            <ClockCircleOutlined style={{ marginRight: 4, color: '#1890ff', fontSize: '12px' }} />
            <span style={{ color: 'rgba(0, 0, 0, 0.65)' }}>持续</span>
            <span style={{ fontWeight: 500, marginLeft: 'auto', color: '#1890ff' }}>
              {record.durationMinutes !== undefined ? Math.floor(record.durationMinutes / (24 * 60)) : '--'} 天
            </span>
          </div>
          <div style={{ 
            display: 'flex', 
            alignItems: 'center',
            padding: '3px 8px',
            backgroundColor: '#fffbf0',
            borderRadius: '4px',
            fontSize: '12px',
            border: '1px solid #fff7e6'
          }}>
            <PauseCircleOutlined style={{ marginRight: 4, color: '#faad14', fontSize: '12px' }} />
            <span style={{ color: 'rgba(0, 0, 0, 0.65)' }}>冷却</span>
            <span style={{ fontWeight: 500, marginLeft: 'auto', color: '#faad14' }}>
              {record.cooldownMinutes !== undefined ? Math.floor(record.cooldownMinutes / (24 * 60)) : '--'} 天
            </span>
          </div>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      align: 'center' as const,
      render: (status: string) => (
        <Badge
          status={status === 'enabled' ? 'success' : 'default'}
          text={
            <span style={{ 
              fontWeight: 500,
              color: status === 'enabled' ? '#52c41a' : 'rgba(0, 0, 0, 0.45)'
            }}>
              {status === 'enabled' ? '启用' : '禁用'}
            </span>
          }
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      align: 'center' as const,
      render: (text: string, record: Strategy) => (
        <Space size="small" className="action-buttons">
          <Tooltip title="执行历史" placement="top">
            <Button
              type="text"
              size="small"
              icon={<ClockCircleOutlined />}
              onClick={() => handleViewExecutionHistory(record)}
              className="history-button"
              style={{
                width: 28,
                height: 28,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: '6px',
                color: '#722ed1'
              }}
            />
          </Tooltip>
          <Tooltip title="编辑" placement="top">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => editStrategy(record.id)}
              className="edit-button"
              style={{
                width: 28,
                height: 28,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: '6px'
              }}
            />
          </Tooltip>
          <Tooltip title={record.status === 'enabled' ? '禁用' : '启用'} placement="top">
            <Button
              type="text"
              size="small"
              icon={record.status === 'enabled' ? <CloseCircleOutlined /> : <CheckCircleOutlined />}
              danger={record.status === 'enabled'}
              onClick={() => toggleStrategyStatus(record.id, record.status)}
              className={record.status === 'enabled' ? "disable-button" : "enable-button"}
              style={{
                width: 28,
                height: 28,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: '6px'
              }}
            />
          </Tooltip>
          <Tooltip title="删除" placement="top">
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => deleteStrategy(record.id)}
              className="delete-button"
              style={{
                width: 28,
                height: 28,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: '6px'
              }}
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
        console.log('开始初始化数据加载');
        
        // 并行获取资源类型和集群列表
        const [resourceTypesData, clustersData] = await Promise.all([
          fetchResourceTypes(),
          fetchClusters()
        ]);

        // 获取工作台数据（包括待处理订单），确保 pendingOrders 状态已更新
        await fetchData(); 

        let clusterSelected = false;
        // 优先从待处理订单中选择集群
        // 确保 pendingOrders 是最新的状态，而不是 fetchInitialData 闭包中的旧值
        if (pendingOrdersRef.current && pendingOrdersRef.current.length > 0) {
          const randomIndex = Math.floor(Math.random() * pendingOrdersRef.current.length);
          const randomOrder = pendingOrdersRef.current[randomIndex];

          if (randomOrder && randomOrder.clusterId) {
            console.log('从待处理订单中选择集群:', randomOrder.clusterId, '订单ID:', randomOrder.id);
            setSelectedClusterId(randomOrder.clusterId);
            const defaultResourceTypes = ['total'];
            setSelectedResourceTypes(defaultResourceTypes);
            await fetchResourceTrend(randomOrder.clusterId, selectedTimeRange, defaultResourceTypes);
            clusterSelected = true;
          }
        }

        // 如果没有从待处理订单中选择集群，并且集群列表不为空，则选择第一个集群
        if (!clusterSelected && clustersData && clustersData.length > 0) {
          console.log('自动选择第一个集群:', clustersData[0]);
          setSelectedClusterId(clustersData[0].id);
          const defaultResourceTypes = ['total'];
          setSelectedResourceTypes(defaultResourceTypes);
          await fetchResourceTrend(clustersData[0].id, selectedTimeRange, defaultResourceTypes);
        }
        
        console.log('初始化数据加载完成');
      } catch (error: any) {
        console.error('初始化数据加载失败:', error);
        const errorMsg = error?.response?.data?.msg || error?.message || '初始化数据加载失败';
        message.error(errorMsg);
      }
    };

    fetchInitialData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 数据缓存
  const [dataCache, setDataCache] = useState({
    resourceTypes: null as any,
    clusters: null as any,
    lastFetchTime: {
      resourceTypes: 0,
      clusters: 0,
      dashboard: 0
    }
  });

  // 缓存有效期（5分钟）
  const CACHE_DURATION = 5 * 60 * 1000;

  // 检查缓存是否有效
  const isCacheValid = useCallback((cacheKey: string) => {
    const lastFetch = dataCache.lastFetchTime[cacheKey as keyof typeof dataCache.lastFetchTime];
    return Date.now() - lastFetch < CACHE_DURATION;
  }, [dataCache, CACHE_DURATION]);

  // 获取资源池类型（带缓存）
  const fetchResourceTypes = useCallback(async (forceRefresh = false) => {
    // 如果缓存有效且不强制刷新，直接使用缓存
    if (!forceRefresh && dataCache.resourceTypes && isCacheValid('resourceTypes')) {
      console.log('使用缓存的资源池类型数据');
      // 从缓存中获取数据时，同时更新相关状态
      setResourceTypeOptions(dataCache.resourceTypes.resourceTypes);
      setResourcePools(dataCache.resourceTypes.poolTypes);
      return dataCache.resourceTypes;
    }

    try {
      console.log('从API获取资源池类型数据');
      // 从后端API获取资源池类型
      const resourceTypes = await statsApi.getResourcePoolTypes();

      // 设置资源类型选项（用于图表筛选）
      setResourceTypeOptions(resourceTypes);

      // 将API返回的资源池类型转换为前端需要的格式
      const poolTypes = resourceTypes.map((type: string) => {
        // 直接使用原始值作为名称，不进行中文翻译
        return { type, name: type };
      });

      console.log('获取到资源池类型:', poolTypes);
      setResourcePools(poolTypes);

      // 更新缓存
      setDataCache(prev => ({
        ...prev,
        resourceTypes: { resourceTypes, poolTypes },
        lastFetchTime: {
          ...prev.lastFetchTime,
          resourceTypes: Date.now()
        }
      }));

      return { resourceTypes, poolTypes };
    } catch (error: any) {
      console.error('获取资源池类型失败:', error);
      const errorMsg = error?.response?.data?.msg || error?.message || '获取资源池类型失败';
      message.error(errorMsg);
      // 出错时不使用默认值，保持空状态
      setResourceTypeOptions([]);
      setResourcePools([]);

      return { resourceTypes: [], poolTypes: [] };
    }
  }, [dataCache, isCacheValid]);

  // 获取集群列表（带缓存）
  const fetchClusters = useCallback(async (forceRefresh = false) => {
    // 如果缓存有效且不强制刷新，直接使用缓存
    if (!forceRefresh && dataCache.clusters && isCacheValid('clusters')) {
      console.log('使用缓存的集群列表数据');
      return dataCache.clusters;
    }

    try {
      console.log('从API获取集群列表数据');
      // 通过API获取集群列表
      const response = await clusterService.getClusters();
      const clustersData = response.list.map(cluster => {
        // 从集群名中解析room信息
        let roomFromName = '';
        if (cluster.clusterName) {
          // 集群名格式为 ${idc}-${zone}${room}-calico|flannel-xxx
          // 例如: bj-zone1room2-calico-xxx 或 sh-zone3room5-flannel-xxx
          const matches = cluster.clusterName.match(/^[^-]+-[^-]+?(\d+)-(calico|flannel)-/);
          if (matches && matches.length >= 2) {
            // matches[1]是room数字
            roomFromName = matches[1];
          } 
          // 如果没有匹配到或者集群名不符合预期格式，roomFromName 保持初始的空字符串状态，这样在UI上会显示 room = ''

        }

        return {
          id: cluster.id,
          name: cluster.clusterName || cluster.clusterNameCn || cluster.alias || `集群-${cluster.id}`,
          // 直接使用API返回的idc和zone字段
          idc: cluster.idc || '',
          zone: cluster.zone || '',
          // room仍然从集群名中解析
          room: roomFromName || '',
          status: cluster.status
        };
      });
      setClusters(clustersData);

      // 更新缓存
      setDataCache(prev => ({
        ...prev,
        clusters: clustersData,
        lastFetchTime: {
          ...prev.lastFetchTime,
          clusters: Date.now()
        }
      }));

      return clustersData;
    } catch (error: any) {
      console.error('获取集群列表失败:', error);
      const errorMsg = error?.response?.data?.msg || error?.message || '获取集群列表失败';
      message.error(errorMsg);
      // 出错时使用空数组，避免页面崩溃
      setClusters([]);
      return [];
    }
  }, [dataCache, isCacheValid]);

  // 防抖状态
  const [isDataFetching, setIsDataFetching] = useState(false);
  const fetchDataTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  // 数据加载函数（带防抖和缓存）
  const fetchData = useCallback(async (forceRefresh = false) => {
    // 防抖：如果正在获取数据，则忽略新的请求
    if (isDataFetching && !forceRefresh) {
      console.log('数据正在加载中，忽略重复请求');
      return;
    }

    // 检查缓存（仅在非强制刷新时）
    if (!forceRefresh && isCacheValid('dashboard')) {
      console.log('使用缓存的仪表板数据');
      return;
    }

    // 清除之前的定时器
    if (fetchDataTimeoutRef.current) {
      clearTimeout(fetchDataTimeoutRef.current);
    }

    // 设置防抖定时器
    fetchDataTimeoutRef.current = setTimeout(async () => {
      setIsDataFetching(true);
      setIsLoading(true);
      
      // 强制刷新时清空分配数据缓存
      if (forceRefresh) {
        clearAllocationDataCache();
      }
      
      try {
        console.log('[fetchData] 开始获取仪表板数据');
        
        // 步骤1: 先获取集群和资源类型数据（这些是后续处理的依赖）
        console.log('[fetchData] 步骤1: 获取集群和资源类型数据');
        await fetchResourceTypes(forceRefresh);
        const clustersData = await fetchClusters(forceRefresh);
        
        console.log('[fetchData] 集群数据获取完成:', clustersData?.length || 0, '个集群');
        console.log('[fetchData] 集群列表:', clustersData?.map((c: any) => ({ id: c.id, name: c.name })) || []);
        
        // 步骤2: 并行获取其他基础数据
        console.log('[fetchData] 步骤2: 获取其他基础数据');
        const [statsResult, strategiesResult, ordersResult, bottomOrdersResult, orderStatsResult] = await Promise.allSettled([
          // 获取工作台统计数据
          statsApi.getDashboardStats(),
          // 获取策略列表
          strategyApi.getStrategies({ page: 1, pageSize: 5 }),
          // 获取所有订单状态的数据（顶部卡片不受搜索过滤影响）
          Promise.all([
            orderApi.getOrders({ status: 'pending', page: 1, pageSize: 10 }),
            orderApi.getOrders({ status: 'processing', page: 1, pageSize: 10 }),
            orderApi.getOrders({ status: 'returning', page: 1, pageSize: 10 }),
            // 获取已完成订单：包括completed、return_completed、no_return状态
            Promise.all([
              orderApi.getOrders({ status: 'completed', page: 1, pageSize: 100 }),
              orderApi.getOrders({ status: 'return_completed', page: 1, pageSize: 100 }),
              orderApi.getOrders({ status: 'no_return', page: 1, pageSize: 100 })
            ]).then(([completedRes, returnCompletedRes, noReturnRes]) => ({
              list: [...completedRes.list, ...returnCompletedRes.list, ...noReturnRes.list],
              total: completedRes.total + returnCompletedRes.total + noReturnRes.total,
              page: 1,
              pageSize: 100
            })),
            orderApi.getOrders({ status: 'cancelled', page: 1, pageSize: 10 }),
            orderApi.getOrders({ page: 1, pageSize: 10 })
          ]),
          // 获取底部全部订单数据（受搜索过滤影响）
          orderApi.getOrders({ 
            ...(nameFilter.trim() ? { name: nameFilter.trim() } : {}), 
            page: 1, 
            pageSize: 10 
          }),
          // 获取订单统计数据
          statsApi.getOrderStats('week')
        ]);

        // 处理结果
        if (statsResult.status === 'fulfilled') {
          setStats(statsResult.value);
        }

        if (strategiesResult.status === 'fulfilled') {
          setStrategies(strategiesResult.value);
        }

        if (ordersResult.status === 'fulfilled') {
          console.log('[fetchData] 步骤3: 处理订单数据并增强分配率信息');
          const [pendingOrdersData, processingOrdersData, returningOrdersData, completedOrdersData, cancelledOrdersData, allOrdersData] = ordersResult.value;
          
          console.log('[fetchData] 原始订单数据:', {
            pending: pendingOrdersData.list.length,
            processing: processingOrdersData.list.length,
            returning: returningOrdersData.list.length,
            completed: completedOrdersData.list.length,
            cancelled: cancelledOrdersData.list.length,
            all: allOrdersData.list.length
          });
          
          // 为订单获取实际分配率数据（传入集群数据）
          const enhanceOrdersWithAllocationData = async (orders: OrderListItem[], orderType: string) => {
            console.log(`[fetchData] 开始增强${orderType}订单分配率数据，订单数量: ${orders.length}`);
            const enhancedOrders = await Promise.all(
              orders.map(async (order) => {
                try {
                  return await fetchOrderAllocationData(order, clustersData);
                } catch (error) {
                  console.error(`[fetchData] 获取${orderType}订单 ${order.id} 分配率数据失败:`, error);
                  return { ...order, hasAllocationData: false };
                }
              })
            );
            console.log(`[fetchData] ${orderType}订单分配率数据增强完成`);
            return enhancedOrders;
          };

          // 并行获取所有订单的分配率数据
          console.log('[fetchData] 开始并行获取所有订单的分配率数据');
          const [enhancedPendingOrders, enhancedProcessingOrders, enhancedReturningOrders, enhancedCompletedOrders, enhancedCancelledOrders] = await Promise.all([
            enhanceOrdersWithAllocationData(pendingOrdersData.list, '待处理'),
            enhanceOrdersWithAllocationData(processingOrdersData.list, '处理中'),
            enhanceOrdersWithAllocationData(returningOrdersData.list, '归还中'),
            enhanceOrdersWithAllocationData(completedOrdersData.list, '已完成'),
            enhanceOrdersWithAllocationData(cancelledOrdersData.list, '已取消')
          ]);

          // 将待处理、处理中和归还中的订单合并显示在待处理订单页面
          setPendingOrders([...enhancedPendingOrders, ...enhancedProcessingOrders, ...enhancedReturningOrders]);
          // 处理中订单页面只显示processing状态的订单
          setProcessingOrders(enhancedProcessingOrders);
          setCompletedOrders(enhancedCompletedOrders);
          setCancelledOrders(enhancedCancelledOrders);
          
          // 为allOrders也更新分配率数据（不受搜索过滤影响）
          const enhancedAllOrdersList = await enhanceOrdersWithAllocationData(allOrdersData.list, '全部');
          setAllOrders({
            ...allOrdersData,
            list: enhancedAllOrdersList
          });
        }

        // 步骤4: 处理底部全部订单数据（受搜索过滤影响）
        if (bottomOrdersResult.status === 'fulfilled') {
          console.log('[fetchData] 步骤4: 处理底部全部订单数据');
          const bottomOrdersData = bottomOrdersResult.value;
          
          // 为底部订单获取实际分配率数据
          const enhanceBottomOrdersWithAllocationData = async (orders: OrderListItem[]) => {
            const enhancedOrders = await Promise.all(
              orders.map(async (order) => {
                try {
                  return await fetchOrderAllocationData(order, clustersData);
                } catch (error) {
                  console.error(`[fetchData] 获取底部订单 ${order.id} 分配率数据失败:`, error);
                  return { ...order, hasAllocationData: false };
                }
              })
            );
            return enhancedOrders;
          };

          const enhancedBottomOrdersList = await enhanceBottomOrdersWithAllocationData(bottomOrdersData.list);
          setBottomAllOrders({
            ...bottomOrdersData,
            list: enhancedBottomOrdersList
          });
        }

        if (orderStatsResult.status === 'fulfilled') {
          setOrderStats(orderStatsResult.value);
        }

        console.log('[fetchData] 所有数据获取完成');
        setIsLoading(false);
        setIsDataFetching(false);
      } catch (error: any) {
        console.error('[fetchData] 获取仪表板数据失败:', error);
        const errorMsg = error?.response?.data?.msg || error?.message || '获取数据失败';
        setOrdersError(errorMsg);
        message.error(errorMsg);
        setOrdersLoading(false);
        setIsLoading(false);
        setIsDataFetching(false);
      }
    }, 200);
  }, [isDataFetching, nameFilter, fetchOrderAllocationData, clearAllocationDataCache, isCacheValid, fetchClusters, fetchResourceTypes]);

  // 使用 ref 来持有最新的 fetchData 函数，避免闭包问题
  const fetchDataRef = useRef(fetchData);
  useEffect(() => {
    fetchDataRef.current = fetchData;
  }, [fetchData]);

  // 处理订单名字过滤器变化
  const handleOrderNameFilterChange = (value: string) => {
    setNameFilter(value);
    // 延迟搜索，避免频繁请求
    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }
    searchTimeoutRef.current = setTimeout(() => {
      // 调用 ref 中最新的 fetchData 函数
      fetchDataRef.current(true);
    }, 500);
  };

  // 防抖搜索函数（保留兼容性）
  const debouncedSearch = useCallback((searchValue: string) => {
    handleOrderNameFilterChange(searchValue);
  }, []);

  // 清理定时器
  useEffect(() => {
    return () => {
      if (fetchDataTimeoutRef.current) {
        clearTimeout(fetchDataTimeoutRef.current);
      }
      if (searchTimeoutRef.current) {
        clearTimeout(searchTimeoutRef.current);
      }
    };
  }, []);

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
            // 根据时间范围选择合适的格式
            if (range === '24h') {
              return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
            } else {
              return date.toLocaleDateString([], { month: '2-digit', day: '2-digit' }) + ' ' + 
                     date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
            }
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
            // 根据时间范围选择合适的格式
            if (range === '24h') {
              return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
            } else {
              return date.toLocaleDateString([], { month: '2-digit', day: '2-digit' }) + ' ' + 
                     date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
            }
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
      const errorMsg = (error as any)?.response?.data?.msg || (error as any)?.message || '获取资源趋势数据失败';
      message.error(errorMsg);
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
    
    // 如果勾选了"同集群Room"，筛选出相同Room的集群
    if (filterBySameRoom) {
      const selectedCluster = clusters.find(c => c.id === clusterId);
      if (selectedCluster && selectedCluster.room) {
        // 筛选出相同Room的集群，并保存到状态中以便UI显示
        const sameRoomClusters = clusters.filter(
          c => c.room === selectedCluster.room && c.id !== clusterId
        );
        
        // 可以根据需要显示提示信息
        if (sameRoomClusters.length > 0) {
          message.info(`找到${sameRoomClusters.length}个相同Room(${selectedCluster.room})的其他集群`);
        }
      }
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
        // 转换分钟到天，支持0值
        durationDays: strategyDetail.durationMinutes !== undefined ?
          Math.floor(strategyDetail.durationMinutes / (24 * 60)) : 0,
        cooldownDays: strategyDetail.cooldownMinutes !== undefined ?
          Math.floor(strategyDetail.cooldownMinutes / (24 * 60)) : 0,
        status: strategyDetail.status,
      });

      console.log('策略详情:', strategyDetail);

      // 设置当前编辑的策略ID
      setCurrentEditStrategyId(id);
    } catch (error: any) {
      console.error('获取策略详情失败:', error);
      const errorMsg = error?.response?.data?.msg || error?.message || '获取策略详情失败';
      message.error(errorMsg);
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
        } catch (error: any) {
          console.error('更新策略状态失败:', error);
          const errorMsg = error?.response?.data?.msg || error?.message || '更新策略状态失败';
          message.error(errorMsg);
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
        } catch (error: any) {
          console.error('Error deleting strategy:', error);
          const errorMsg = error?.response?.data?.msg || error?.message || '删除策略失败';
          message.error(errorMsg);
        }
      }
    });
  };

  // 渲染统计卡片
  const renderStatCards = () => {
    if (!stats) return null;

    return (
            <Row gutter={24} className="stats-cards">
        <Col xs={24} sm={6} md={6}>
          <Card className="stat-card success">
            <div className="stat-value">{`${stats.triggeredTodayCount}/${stats.enabledStrategyCount}`}</div>
            <div className="stat-label">今日已巡检/总策略</div>
            <Progress percent={parseFloat((stats.enabledStrategyCount > 0 ? (stats.triggeredTodayCount / stats.enabledStrategyCount) * 100 : 0).toFixed(1))} size="small" />
          </Card>
        </Col>
        <Col xs={24} sm={6} md={6}>
          <Card className="stat-card info">
            <div className="stat-value">{`${stats.inspectedResourcePoolCount || 0}/${stats.targetResourcePoolCount || 0}`}</div>
            <div className="stat-label">今日已巡检/目标资源池数</div>
            <Progress percent={parseFloat((stats.targetResourcePoolCount && stats.targetResourcePoolCount > 0 ? ((stats.inspectedResourcePoolCount || 0) / stats.targetResourcePoolCount!) * 100 : 0).toFixed(1))} size="small" />
          </Card>
        </Col>
        <Col xs={24} sm={6} md={6}>
          <Card className={`stat-card ${stats.abnormalClusterCount > 0 ? 'warning' : ''}`}>
            <div className="stat-value">{`${stats.clusterCount - stats.abnormalClusterCount}/${stats.clusterCount}`}</div>
            <div className="stat-label">正常集群/总集群数</div>
            {stats.abnormalClusterCount > 0 && (
              <div className="stat-trend" style={{ color: "#faad14" }}>{stats.abnormalClusterCount}个集群需要处理</div>
            )}
          </Card>
        </Col>
        <Col xs={24} sm={6} md={6}>
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
  const renderOrderCard = (order: OrderListItem, showResourceInfo: boolean = false, isPendingSection: boolean = false, isSimpleCard: boolean = false) => {
    const actionTypeText = order.actionType === 'pool_entry' ? '入池' : '退池';
    const orderStatusText =
      order.status === 'pending' ? '待处理' :
      order.status === 'processing' ? '处理中' :
      order.status === 'returning' ? '归还中' :
      order.status === 'return_completed' ? '归还完成' :
      order.status === 'no_return' ? '无需归还' :
      order.status === 'cancelled' ? '已取消' :
      order.status === 'failed' ? '失败' :
      '已完成';

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
            {order.strategyName ? `${order.strategyName}` : (order.name || order.orderNumber)} - {order.actionType === 'pool_entry' ? '入池' : '退池'}
          </div>
          <Tag color={
            order.status === 'pending' ? 'error' :
            order.status === 'processing' ? 'processing' :
            order.status === 'returning' ? 'purple' :
            order.status === 'return_completed' ? 'cyan' :
            order.status === 'no_return' ? 'success' :
            order.status === 'failed' ? 'error' :
            order.status === 'cancelled' ? 'default' :
            'success'
          }>
            {orderStatusText}
          </Tag>
        </div>
        <div className="order-card-body">
          <div className="order-meta">
            <div className="order-meta-item order-detail-meta-item" style={{ flex: 1 }}>
              <div className="order-meta-label order-detail-meta-label">集群</div>
              <div className="order-meta-value order-detail-meta-value">
                <ClusterOutlined style={{ color: '#52c41a', marginRight: 4 }} />
                {order.clusterName}
              </div>
            </div>
            <div className="order-meta-item order-detail-meta-item" style={{ flex: 1 }}>
              <div className="order-meta-label order-detail-meta-label">设备数量</div>
              <div className="order-meta-value order-detail-meta-value">
                <DesktopOutlined />
                {order.deviceCount} 台
              </div>
            </div>
          </div>

          {/* 资源利用率信息，仅在需要时显示 */}
          {showResourceInfo && (
            <div className="resource-info-card">
              <div className="resource-header">
                <DatabaseOutlined style={{ marginRight: 8, color: '#1890ff' }} />
                资源池: {(order.resourcePoolType === 'compute' ? '' : order.resourcePoolType) || 'total'}
              </div>
              <div className="resource-grid">
                <div>
                  <div className="resource-item-header">
                    <span>CPU分配率</span>
                    <span style={{ 
                      color: getCpuColor(order),
                      fontWeight: 'bold' 
                    }}>
                      {getCpuDisplayText(order)}
                      {order.hasAllocationData !== false && (order.actionType === 'pool_entry' ? <ArrowUpOutlined style={{ fontSize: 12 }} /> : <ArrowDownOutlined style={{ fontSize: 12 }} />)}
                      {order.hasAllocationData === true ? 
                        <Tooltip title="实时数据"><Badge status="success" style={{ marginLeft: 4 }} /></Tooltip> : 
                        order.hasAllocationData === false ?
                        <Tooltip title="暂无数据"><Badge status="default" style={{ marginLeft: 4 }} /></Tooltip> :
                        <Tooltip title="模拟数据"><Badge status="default" style={{ marginLeft: 4 }} /></Tooltip>
                      }
                    </span>
                  </div>
                  <Progress
                    percent={parseFloat(getCpuValue(order).toFixed(1))}
                    size="small"
                    status={order.hasAllocationData === false ? "normal" : (getCpuValue(order) >= 80 ? "exception" : "normal")}
                    strokeColor={getCpuColor(order)}
                    strokeWidth={8}
                  />
                </div>
                <div>
                  <div className="resource-item-header">
                    <span>内存分配率</span>
                    <span style={{ 
                      color: getMemColor(order),
                      fontWeight: 'bold' 
                    }}>
                      {getMemDisplayText(order)}
                      {order.hasAllocationData !== false && (order.actionType === 'pool_entry' ? <ArrowUpOutlined style={{ fontSize: 12 }} /> : <ArrowDownOutlined style={{ fontSize: 12 }} />)}
                      {order.hasAllocationData === true ? 
                        <Tooltip title="实时数据"><Badge status="success" style={{ marginLeft: 4 }} /></Tooltip> : 
                        order.hasAllocationData === false ?
                        <Tooltip title="暂无数据"><Badge status="default" style={{ marginLeft: 4 }} /></Tooltip> :
                        <Tooltip title="模拟数据"><Badge status="default" style={{ marginLeft: 4 }} /></Tooltip>
                      }
                    </span>
                  </div>
                  <Progress
                    percent={parseFloat(getMemValue(order).toFixed(1))}
                    size="small"
                    status={order.hasAllocationData === false ? "normal" : (getMemValue(order) >= 80 ? "exception" : "normal")}
                    strokeColor={getMemColor(order)}
                    strokeWidth={8}
                  />
                </div>
              </div>
              <Alert
                message={
                  order.hasAllocationData === false || getCpuValue(order) === 0 ?
                    '暂无分配率数据，无法评估集群状态' :
                    (() => {
                      const cpuValue = getCpuValue(order);
                      if (cpuValue >= 75) {
                        return 'CPU分配率已超过阈值(80%)，需添加节点提升集群容量';
                      } else if (cpuValue <= 55) {
                        return 'CPU分配率低于阈值(55%)，可回收闲置节点';
                      } else {
                        return 'CPU分配率正常，集群资源充足';
                      }
                    })()
                }
                type={
                  order.hasAllocationData === false || getCpuValue(order) === 0 ?
                    "info" :
                    (() => {
                      const cpuValue = getCpuValue(order);
                      if (cpuValue >= 80) {
                        return "error";
                      } else if (cpuValue <= 55) {
                        return "warning";
                      } else {
                        return "success";
                      }
                    })()
                }
                showIcon
                style={{
                  marginTop: 12,
                  fontSize: '12px', // 减小字体大小
                  padding: '4px 8px', // 减小内边距
                }}
                banner
              />
            </div>
          )}
        </div>
        <div className="order-card-footer">
          <Space>
            <Button type="link" icon={<EyeOutlined />} onClick={(e) => {
              e.stopPropagation();
              handleViewOrderDetails(order.id);
            }}>
              详情
            </Button>
            <Button type="link" icon={<CopyOutlined />} onClick={(e) => {
              e.stopPropagation();
              handleCloneOrder(order);
            }}>
              克隆
            </Button>
          </Space>
          {/* 只有在待处理区域且不是简略卡片时才显示操作按钮 */}
          {!isSimpleCard && isPendingSection && order.status === 'pending' && (
            <div className="order-action-buttons">
              <Button
                danger
                icon={<StopOutlined />}
                onClick={(e) => {
                  e.stopPropagation();
                  confirm({
                    title: '确定取消该订单吗？',
                    icon: <ExclamationCircleOutlined />,
                    content: '取消后订单将无法恢复。',
                    okText: '确定',
                    okType: 'danger',
                    cancelText: '取消',
                    onOk() {
                      ignoreOrder(order.id);
                    },
                  });
                }}
              >
                取消
              </Button>
              <Button
                type="primary"
                icon={order.actionType === 'pool_entry' ? <CloudUploadOutlined /> : <CloudDownloadOutlined />}
                onClick={(e) => {
                  e.stopPropagation();
                  confirm({
                    title: `确定执行${actionTypeText}操作吗？`,
                    icon: <ExclamationCircleOutlined />,
                    content: '执行后将开始处理该订单。',
                    okText: '确定',
                    cancelText: '取消',
                    onOk() {
                      executeOrder(order.id, order.actionType);
                    },
                  });
                }}
              >
                执行{actionTypeText}
              </Button>
            </div>
          )}
          {/* 只有在不是简略卡片时才显示状态操作按钮 */}
          {!isSimpleCard && order.status === 'processing' && order.actionType === 'pool_entry' && (
            <div className="order-action-buttons">
              <Button
                danger
                icon={<StopOutlined />}
                onClick={(e) => {
                  e.stopPropagation();
                  confirm({
                    title: '确定取消该订单吗？',
                    icon: <ExclamationCircleOutlined />,
                    content: '取消后订单将无法恢复。',
                    okText: '确定',
                    okType: 'danger',
                    cancelText: '取消',
                    onOk() {
                      updateOrderStatus(order.id, 'cancelled');
                    },
                  });
                }}
              >
                取消
              </Button>
              <Button
                type="primary"
                icon={<CheckCircleOutlined />}
                onClick={(e) => {
                  e.stopPropagation();
                  confirm({
                    title: '确定标记为入池完成吗？',
                    icon: <ExclamationCircleOutlined />,
                    content: '标记后订单状态将更新。',
                    okText: '确定',
                    cancelText: '取消',
                    onOk() {
                      updateOrderStatus(order.id, 'completed');
                    },
                  });
                }}
              >
                入池完成
              </Button>
            </div>
          )}
          {/* 只有在不是简略卡片时才显示状态操作按钮 */}
          {!isSimpleCard && order.status === 'processing' && order.actionType === 'pool_exit' && (
            <div className="order-action-buttons">
              <Button
                danger
                icon={<StopOutlined />}
                onClick={(e) => {
                  e.stopPropagation();
                  confirm({
                    title: '确定取消该订单吗？',
                    icon: <ExclamationCircleOutlined />,
                    content: '取消后订单将无法恢复。',
                    okText: '确定',
                    okType: 'danger',
                    cancelText: '取消',
                    onOk() {
                      updateOrderStatus(order.id, 'cancelled');
                    },
                  });
                }}
              >
                取消
              </Button>
              <Button
                type="primary"
                icon={<CheckCircleOutlined />}
                onClick={(e) => {
                  e.stopPropagation();
                  confirm({
                    title: '确定标记为退池完成吗？',
                    icon: <ExclamationCircleOutlined />,
                    content: '标记后订单状态将更新。',
                    okText: '确定',
                    cancelText: '取消',
                    onOk() {
                      updateOrderStatus(order.id, 'returning');
                    },
                  });
                }}
              >
                退池完成
              </Button>
            </div>
          )}
          {/* 只有在不是简略卡片时才显示状态操作按钮 */}
          {!isSimpleCard && order.status === 'returning' && (
            <div className="order-action-buttons">
              <Button
                icon={<CheckOutlined />}
                onClick={(e) => {
                  e.stopPropagation();
                  confirm({
                    title: '确定标记为无须归还吗？',
                    icon: <ExclamationCircleOutlined />,
                    content: '标记后订单状态将更新。',
                    okText: '确定',
                    cancelText: '取消',
                    onOk() {
                      updateOrderStatus(order.id, 'no_return');
                    },
                  });
                }}
              >
                无须归还
              </Button>
              <Button
                type="primary"
                icon={<CheckCircleOutlined />}
                onClick={(e) => {
                  e.stopPropagation();
                  confirm({
                    title: '确定标记为归还完成吗？',
                    icon: <ExclamationCircleOutlined />,
                    content: '标记后订单状态将更新。',
                    okText: '确定',
                    cancelText: '取消',
                    onOk() {
                      updateOrderStatus(order.id, 'return_completed');
                    },
                  });
                }}
              >
                归还完成
              </Button>
            </div>
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

    // 待处理订单现在包含pending、processing、returning三种状态
    const pendingCount = pendingOrders.length;
    // 处理中订单现在只包含processing状态
    const processingCount = processingOrders.length;
    const completedCount = completedOrders.length;
    // 使用已取消订单列表的长度
    const cancelledCount = cancelledOrders.length;
    const totalCount = allOrders.total;

    return (
      <div>
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
            <div className="order-status-value order-status-cancelled">{cancelledCount}</div>
            <div className="order-status-label">已取消</div>
          </div>
          <div className="order-status-item">
            <div className="order-status-value">{totalCount}</div>
            <div className="order-status-label">总订单</div>
          </div>
        </div>
      </div>
    );
  };

  // 渲染设备项
  const renderDeviceItem = (device: Device) => {
    return (
      <div 
        key={device.id} 
        className="device-item"
        onMouseEnter={async (e) => {
          // 如果是特殊设备或有应用名称，显示提示框
          if (device.isSpecial || (device.appName && device.appName.trim() !== '')) {
            try {
              // 创建提示框
              const tooltip = document.createElement('div');
              tooltip.className = 'device-feature-tooltip';

              // 初始内容 - 美化样式
              tooltip.innerHTML = `
                <div style="font-weight: 600; font-size: 14px; color: #1677ff; margin-bottom: 8px; border-bottom: 1px solid rgba(0,0,0,0.06); padding-bottom: 6px;">
                  <span style="margin-right: 5px;">&#x1F4CB;</span>设备特性
                </div>
                <div style="color: #666; font-size: 13px;">
                  <div style="display: flex; align-items: center;">
                    <span style="margin-right: 5px;">&#x231B;</span>正在加载特性详情...
                  </div>
                </div>
              `;

              // 计算位置 - 跟随鼠标
              tooltip.style.position = 'fixed';
              tooltip.style.left = `${e.clientX + 15}px`; // 鼠标右侧偏移15px
              tooltip.style.top = `${e.clientY - 20}px`;  // 鼠标上方偏移20px

              // 美化样式 - 半透明卡片
              tooltip.style.backgroundColor = 'rgba(255, 255, 255, 0.9)';
              tooltip.style.backdropFilter = 'blur(5px)'; // 模糊效果
              tooltip.style.border = '1px solid rgba(217, 217, 217, 0.6)';
              tooltip.style.borderRadius = '8px';
              tooltip.style.padding = '12px';
              tooltip.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.08)';
              tooltip.style.zIndex = '9999';
              tooltip.style.maxWidth = '350px';
              tooltip.style.transition = 'all 0.2s ease-in-out';

              document.body.appendChild(tooltip);

              // 存储提示框引用，便于移除
              e.currentTarget.tooltip = tooltip;

              // 创建一个引用副本，避免使用 e.currentTarget
              const tooltipElement = tooltip;
              const deviceElement = e.currentTarget;

              // 添加鼠标移动事件，使提示框跟随鼠标
              const handleMouseMove = (moveEvent: MouseEvent) => {
                // 检查元素和提示框是否仍然存在
                if (deviceElement && deviceElement.tooltip && tooltipElement && document.body.contains(tooltipElement)) {
                  // 更新提示框位置
                  tooltipElement.style.left = `${moveEvent.clientX + 15}px`;
                  tooltipElement.style.top = `${moveEvent.clientY - 20}px`;

                  // 检查是否超出右边界
                  const tooltipRect = tooltipElement.getBoundingClientRect();
                  const windowWidth = window.innerWidth;
                  if (tooltipRect.right > windowWidth) {
                    // 如果超出右边界，则将提示框放在鼠标左侧
                    tooltipElement.style.left = `${moveEvent.clientX - tooltipRect.width - 15}px`;
                  }
                } else {
                  // 如果元素或提示框不存在，移除事件监听器
                  document.removeEventListener('mousemove', handleMouseMove);
                }
              };

              // 将鼠标移动事件处理函数存储在元素上，便于移除
              e.currentTarget.handleMouseMove = handleMouseMove;
              document.addEventListener('mousemove', handleMouseMove);

              // 构建特性详情数组
              const featureDetails: string[] = [];

              // 添加机器用途
              if (device.group && device.group.trim() !== '') {
                featureDetails.push(`机器用途: ${device.group}`);
              }

              // 添加应用名称
              if (device.appName && device.appName.trim() !== '') {
                featureDetails.push(`应用名称: ${device.appName}`);
              }

              // 如果有标签特性或污点特性，获取详情
              if (device.featureCount && device.featureCount > (device.group ? 1 : 0)) {
                // 获取设备特性详情
                const details = await getDeviceFeatureDetails(device.ciCode);

                // 添加标签详情
                if (details.labels && details.labels.length > 0) {
                  featureDetails.push(`存在标签:`);
                  details.labels.forEach(label => {
                    featureDetails.push(`  ${label.key}=${label.value}`);
                  });
                }

                // 添加污点详情
                if (details.taints && details.taints.length > 0) {
                  featureDetails.push(`存在污点:`);
                  details.taints.forEach(taint => {
                    featureDetails.push(`  ${taint.key}=${taint.value}:${taint.effect}`);
                  });
                }
              }

              // 更新提示框内容 - 美化样式
              let tooltipContent = `
                <div style="font-weight: 600; font-size: 14px; color: #1677ff; margin-bottom: 8px; border-bottom: 1px solid rgba(0,0,0,0.06); padding-bottom: 6px;">
                  <span style="margin-right: 5px;">&#x1F4CB;</span>设备特性
                </div>
                <div style="color: #666; font-size: 13px;">
              `;

              // 添加机器用途信息
              const groupInfo = featureDetails.find(detail => detail.startsWith('机器用途:'));
              if (groupInfo) {
                const groupValue = groupInfo.split(':')[1].trim();
                tooltipContent += `
                  <div style="margin-bottom: 12px;">
                    <div style="display: flex; align-items: center;">
                      <span style="margin-right: 5px; color: #1677ff;">&#x1F4BB;</span>
                      <span style="font-weight: 500;">机器用途:</span>
                    </div>
                    <div>
                      <span style="background-color: rgba(22, 119, 255, 0.1); padding: 2px 8px; border-radius: 4px; color: #1677ff; display: inline-block; text-align: left;">${groupValue.trim()}</span>
                    </div>
                  </div>
                `;
              }

              // 添加应用名称信息
              const appNameInfo = featureDetails.find(detail => detail.startsWith('应用名称:'));
              if (appNameInfo) {
                const appNameValue = appNameInfo.split(':')[1].trim();
                tooltipContent += `
                  <div style="margin-bottom: 12px;">
                    <div style="display: flex; align-items: center;">
                      <span style="margin-right: 5px; color: #722ed1;">&#x1F4F1;</span>
                      <span style="font-weight: 500;">应用名称:</span>
                    </div>
                    <div>
                      <span style="background-color: rgba(114, 46, 209, 0.1); padding: 2px 8px; border-radius: 4px; color: #722ed1; display: inline-block; text-align: left;">${appNameValue.trim()}</span>
                    </div>
                  </div>
                `;
              }

              // 添加标签信息
              const labelIndex = featureDetails.findIndex(detail => detail === '存在标签:');
              if (labelIndex !== -1) {
                tooltipContent += `
                  <div style="margin-bottom: 8px;">
                    <div style="display: flex; align-items: center;">
                      <span style="margin-right: 5px; color: #52c41a;">&#x1F3F7;</span>
                      <span style="font-weight: 500;">标签信息:</span>
                    </div>
                  </div>
                  <div style="margin-bottom: 12px;">
                `;

                // 收集标签详情
                let i = labelIndex + 1;
                while (i < featureDetails.length && featureDetails[i].startsWith('  ')) {
                  const labelParts = featureDetails[i].trim().split('=');
                  if (labelParts.length === 2) {
                    tooltipContent += `
                      <div style="margin-bottom: 8px;">
                        <div style="color: #666; font-weight: 500;">${labelParts[0]}:</div>
                        <div>
                          <span style="background-color: rgba(82, 196, 26, 0.1); padding: 2px 8px; border-radius: 4px; color: #52c41a; display: inline-block; text-align: left;">${labelParts[1].trim()}</span>
                        </div>
                      </div>
                    `;
                  }
                  i++;
                }

                tooltipContent += `</div>`;
              }

              // 添加污点信息
              const taintIndex = featureDetails.findIndex(detail => detail === '存在污点:');
              if (taintIndex !== -1) {
                tooltipContent += `
                  <div style="margin-bottom: 8px;">
                    <div style="display: flex; align-items: center;">
                      <span style="margin-right: 5px; color: #f5222d;">&#x26A0;</span>
                      <span style="font-weight: 500;">污点信息:</span>
                    </div>
                  </div>
                  <div style="margin-bottom: 12px;">
                `;

                // 收集污点详情
                let i = taintIndex + 1;
                while (i < featureDetails.length && featureDetails[i].startsWith('  ')) {
                  const taintInfo = featureDetails[i].trim();
                  const taintParts = taintInfo.split('=');
                  if (taintParts.length === 2) {
                    const keyPart = taintParts[0];
                    const valueParts = taintParts[1].split(':');
                    if (valueParts.length === 2) {
                      const valuePart = valueParts[0];
                      const effectPart = valueParts[1];

                      // 根据 effect 类型选择颜色
                      let effectColor = '#f5222d'; // 默认红色
                      if (effectPart === 'PreferNoSchedule') {
                        effectColor = '#faad14'; // 黄色
                      } else if (effectPart === 'NoExecute') {
                        effectColor = '#f5222d'; // 红色
                      }

                      tooltipContent += `
                        <div style="margin-bottom: 12px;">
                          <div style="color: #666; font-weight: 500;">${keyPart}:</div>
                          <div>
                            <div>
                              <span style="background-color: rgba(245, 34, 45, 0.1); padding: 2px 8px; border-radius: 4px; color: #f5222d; display: inline-block; text-align: left;">${valuePart.trim()}</span>
                            </div>
                            <div>
                              <span style="background-color: rgba(0, 0, 0, 0.04); padding: 2px 8px; border-radius: 4px; color: ${effectColor}; font-size: 12px; display: inline-block; text-align: left;">${effectPart.trim()}</span>
                            </div>
                          </div>
                        </div>
                      `;
                    }
                  }
                  i++;
                }

                tooltipContent += `</div>`;
              }

              tooltipContent += `</div>`;
              tooltip.innerHTML = tooltipContent;
              console.log('提示框内容已设置完成');
            } catch (error) {
              console.error('获取设备特性详情失败:', error);
            }
          } else {
            console.log('不满足显示条件，跳过提示框显示');
          }
        }}
        onMouseLeave={(e) => {
          try {
            // 安全地移除提示框
            const tooltip = e.currentTarget?.tooltip;
            if (tooltip && document.body.contains(tooltip)) {
              document.body.removeChild(tooltip);
            }

            // 清除引用
            if (e.currentTarget) {
              e.currentTarget.tooltip = null;
            }

            // 安全地移除鼠标移动事件监听器
            const handleMouseMove = e.currentTarget?.handleMouseMove;
            if (handleMouseMove) {
              document.removeEventListener('mousemove', handleMouseMove);

              // 清除引用
              if (e.currentTarget) {
                e.currentTarget.handleMouseMove = null;
              }
            }
          } catch (error) {
            console.error('清除提示框资源失败:', error);
          }
        }}
      >
        <div className="device-info">
          <div className="device-name">
            <DesktopOutlined style={{ color: '#1890ff' }} />
            {device.ciCode}
          </div>
          <div className="device-meta">
            <div className="device-meta-item">
              <span className="device-meta-label">IP:</span>
              <span>{device.ip}</span>
            </div>
            <div className="device-meta-item">
              <span className="device-meta-label">集群:</span>
              <span>{device.cluster || '未分配'}</span>
            </div>
            {device.cpu !== undefined && (
              <div className="device-meta-item">
                <span className="device-meta-label">CPU:</span>
                <span>{device.cpu} 核</span>
              </div>
            )}
            {device.memory !== undefined && (
              <div className="device-meta-item">
                <span className="device-meta-label">内存:</span>
                <span>{device.memory} GB</span>
              </div>
            )}
          </div>
        </div>
        <span className={`device-status ${device.isSpecial ? 'status-special' : (device.cluster ? 'status-in-cluster' : 'status-available')}`}>
          {device.isSpecial ? 
            <><ExclamationCircleOutlined style={{ marginRight: '4px' }} /> 特殊设备</> : 
            (device.cluster ? 
              <><CheckCircleOutlined style={{ marginRight: '4px' }} /> 已入池</> : 
              <><ClockCircleOutlined style={{ marginRight: '4px' }} /> 可入池</>)
          }
        </span>
      </div>
    );
  };

  // 初始化订单状态图表
  useEffect(() => {
    // 检查是否有订单数据和订单统计数据
    if (pendingOrders && processingOrders && completedOrders && orderStats) {
      const pendingCount = pendingOrders.length;
      const processingCount = processingOrders.length;
      const completedCount = completedOrders.length;

      // 使用API返回的已取消订单数量
      const cancelledCount = orderStats.cancelledCount || 0;

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
                itemStyle: { color: '#3b82f6' }
              },
              {
                value: processingCount,
                name: '处理中',
                itemStyle: { color: '#06b6d4' }
              },
              {
                value: completedCount,
                name: '已完成',
                itemStyle: { color: '#10b981' }
              },
              {
                value: cancelledCount,
                name: '已取消',
                itemStyle: { color: '#64748b' }
              }
            ]
          }
        ]
      });
    }
  }, [pendingOrders, processingOrders, completedOrders, orderStats]);

  // 处理查看订单详情
  const handleViewOrderDetails = async (orderId: number) => {
    try {
      const orderDetail = await orderApi.getOrder(orderId);
      setSelectedOrder(orderDetail);
      setSelectedStrategy(null);
      setDrawerVisible(true);
    } catch (error: any) {
      console.error('Error fetching order details:', error);
      const errorMsg = error?.response?.data?.msg || error?.message || '获取订单详情失败';
      message.error(errorMsg);
    }
  };

  // 执行订单
  const executeOrder = async (orderId: number, actionType: string) => {
    try {
      // 更新订单状态为处理中
      await orderApi.updateOrderStatus(orderId, { status: 'processing' as OrderStatus });

      // 提示用户
      Modal.success({
        title: '操作成功',
        content: `已开始执行${actionType === 'pool_entry' ? '入池' : '退池'}操作`,
      });

      // 刷新订单列表
      fetchData(true);
    } catch (error: any) {
      console.error('Error executing order:', error);
      const errorMsg = error?.response?.data?.msg || error?.message || '执行订单失败';
      message.error(errorMsg);
    }
  };

  // 忽略订单
  const ignoreOrder = async (orderId: number) => {
    try {
      // 更新订单状态为已取消
      await orderApi.updateOrderStatus(orderId, {
        status: 'cancelled' as OrderStatus,
        type: 'elastic_scaling' // 确保订单类型为弹性伸缩
      });

      // 提示用户
      message.success('订单已取消');

      // 刷新订单列表
      fetchData(true);
    } catch (error: any) {
      console.error('Error ignoring order:', error);
      const errorMsg = error?.response?.data?.msg || error?.message || '取消订单失败';
      message.error(errorMsg);
    }
  };

  // 更新订单状态
  const updateOrderStatus = async (orderId: number, status: OrderStatus) => {
    try {
      await orderApi.updateOrderStatus(orderId, {
        status: status,
        type: 'elastic_scaling'
      });

      const statusText = 
        status === 'no_return' ? '无须归还' :
        status === 'return_completed' ? '归还完成' :
        status;

      message.success(`订单状态已更新为：${statusText}`);
      fetchData(true);
    } catch (error: any) {
      console.error('Error updating order status:', error);
      const errorMsg = error?.response?.data?.msg || error?.message || '更新订单状态失败';
      message.error(errorMsg);
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
    // 确保每次打开都重置表单状态，清除缓存数据
    if (createOrderModalRef.current) {
      createOrderModalRef.current.open(); // 不传参数，表示新建模式
    }
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
        deviceCount: (values.devices || []).length, // 添加设备数量字段
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
      fetchData(true);
    } catch (error: any) {
      console.error('创建订单失败:', error);
      const errorMsg = error?.response?.data?.msg || error?.message || '创建订单失败';
      message.error(errorMsg);
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
    // 使用从后端获取的资源池类型数据，而不是硬编码的选项
    const options = resourcePools.map(pool => ({
      label: pool.type,
      value: pool.type
    }));


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
        suffixIcon={<DatabaseOutlined style={{ color: '#1890ff' }} />}
      >
        {options.map(option => (
          <Option key={option.value} value={option.value}>
            <DatabaseOutlined style={{ color: '#1890ff', marginRight: 8 }} />
            {option.value === 'compute' ? '' : option.label}
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
              <Option key={type} value={type}>{type === 'compute' ? '' : type}</Option>
            ))}
          </Select>
        </Form.Item>

        {/* ... other form fields ... */}
      </Form>
    );
  };


  // 获取CPU分配率值
  const getCpuValue = (order: OrderListItem): number => {
    // 只使用实际数据，不使用mock数据
    if (order.hasAllocationData && order.actualCpuAllocation !== undefined) {
      return order.actualCpuAllocation;
    }
    // 没有实际数据时返回0
    return 0;
  };

  // 获取CPU分配率显示文本
  const getCpuDisplayText = (order: OrderListItem): string => {
    if (order.hasAllocationData === false) {
      return '暂无数据';
    }
    return `${getCpuValue(order).toFixed(1)}%`;
  };

  // 获取CPU分配率颜色
  const getCpuColor = (order: OrderListItem): string => {
    // 没有数据时显示灰色
    if (order.hasAllocationData === false) {
      return '#d9d9d9';
    }
    const value = getCpuValue(order);
    if (value >= 80) return '#f5222d'; // 红色
    if (value >= 70) return '#faad14'; // 黄色
    if (value >= 55) return '#52c41a'; // 绿色
    return '#1890ff'; // 蓝色
  };

  // 获取内存分配率值
  const getMemValue = (order: OrderListItem): number => {
    // 只使用实际数据，不使用mock数据
    if (order.hasAllocationData && order.actualMemoryAllocation !== undefined) {
      return order.actualMemoryAllocation;
    }
    // 没有实际数据时返回0
    return 0;
  };

  // 获取内存分配率显示文本
  const getMemDisplayText = (order: OrderListItem): string => {
    if (order.hasAllocationData === false) {
      return '暂无数据';
    }
    return `${getMemValue(order).toFixed(1)}%`;
  };

  // 获取内存分配率颜色
  const getMemColor = (order: OrderListItem): string => {
    // 没有数据时显示灰色
    if (order.hasAllocationData === false) {
      return '#d9d9d9';
    }
    const value = getMemValue(order);
    if (value >= 80) return '#f5222d'; // 红色
    if (value >= 70) return '#faad14'; // 黄色
    if (value >= 55) return '#52c41a'; // 绿色
    return '#1890ff'; // 蓝色
  };

  // 获取百分比颜色（与订单分配率颜色策略一致）
  const getPercentageColor = (value: number | string | undefined): string => {
    if (!value) return '#d9d9d9'; // 灰色 - 未设置
    // 确保值是数字类型（后端返回整数如90，直接表示90%）
    const numValue = typeof value === 'string' ? parseFloat(value) : value;
    if (isNaN(numValue)) return '#d9d9d9'; // 无效值返回灰色
    // 后端返回的是整数百分比值，直接使用
    if (numValue >= 80) return '#f5222d'; // 红色 - 高风险 (>=80%)
    if (numValue >= 70) return '#faad14'; // 黄色 - 中风险 (>=70%)
    if (numValue >= 55) return '#52c41a'; // 绿色 - 正常 (>=55%)
    return '#1890ff'; // 蓝色 - 低使用率 (<55%)
  };

  // 添加是否筛选同集群Room的状态
  const [filterBySameRoom, setFilterBySameRoom] = useState<boolean>(false);
  // 添加是否筛选同集群IDC的状态
  const [filterBySameIDC, setFilterBySameIDC] = useState<boolean>(false);

  // 筛选集群的函数
  const getFilteredClusters = useCallback(() => {
    if ((!filterBySameRoom && !filterBySameIDC) || !selectedClusterId) {
      // 如果没有选择集群或没有勾选任何筛选项，返回全部集群
      return clusters;
    }
    
    const selectedCluster = clusters.find(c => c.id === selectedClusterId);
    if (!selectedCluster) {
      return clusters;
    }
    
    // 根据选中的筛选条件进行筛选
    return clusters.filter(c => {
      // 如果勾选了同集群Room，需要匹配room
      if (filterBySameRoom && (!selectedCluster.room || c.room !== selectedCluster.room)) {
        return false;
      }
      
      // 如果勾选了同集群IDC，需要匹配idc
      if (filterBySameIDC && (!selectedCluster.idc || c.idc !== selectedCluster.idc)) {
        return false;
      }
      
      return true;
    });
  }, [filterBySameRoom, filterBySameIDC, selectedClusterId, clusters]);

  // 当筛选条件变化时处理筛选
  useEffect(() => {
    // 如果当前没有选中任何集群，则不需要处理
    if (!selectedClusterId) return;
    
    const selectedCluster = clusters.find(c => c.id === selectedClusterId);
    if (!selectedCluster) return;
    
    // 如果开启了Room筛选但当前选中的集群没有room信息，显示提示
    if (filterBySameRoom && !selectedCluster.room) {
      message.warning('当前选中的集群没有Room信息，无法筛选同Room集群');
    }
    
    // 如果开启了IDC筛选但当前选中的集群没有idc信息，显示提示
    if (filterBySameIDC && !selectedCluster.idc) {
      message.warning('当前选中的集群没有IDC信息，无法筛选同IDC集群');
    }
    
    // 如果有筛选条件被启用，更新筛选后的集群列表
    if (filterBySameRoom || filterBySameIDC) {
      const filteredClusters = getFilteredClusters();
      if (filteredClusters.length <= 1) {
        message.info('根据当前筛选条件，没有找到其他匹配的集群');
      } else {
        const filterConditions = [];
        if (filterBySameRoom && selectedCluster.room) filterConditions.push(`Room(${selectedCluster.room})`);
        if (filterBySameIDC && selectedCluster.idc) filterConditions.push(`IDC(${selectedCluster.idc})`);
        
        if (filterConditions.length > 0) {
          message.info(`已筛选出${filteredClusters.length}个满足${filterConditions.join('和')}条件的集群`);
        }
      }
    }
  }, [filterBySameRoom, filterBySameIDC, selectedClusterId, clusters, getFilteredClusters]);

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
        title="待处理订单"
        extra={
          <Space>
            <Button
              type="primary"
              size="small"
              icon={<PlusOutlined />}
              onClick={handleOpenCreateOrderModal}
              style={{ borderRadius: '6px', width: '32px', height: '32px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
            />
            <Select
              placeholder="订单状态"
              style={{ width: 120 }}
              allowClear
              onChange={(value) => setStatusFilter(value)}
            >
              <Option value="pending">待处理</Option>
              <Option value="processing">处理中</Option>
              <Option value="returning">归还中</Option>
            </Select>
            <Select
               placeholder="订单类型"
               style={{ width: 120 }}
               allowClear
               onChange={(value) => setOrderFilter(value)}
             >
              <Option value="pool_entry">入池</Option>
              <Option value="pool_exit">退池</Option>
            </Select>

          </Space>
        }
      >
        {(() => {
          const filteredOrders = pendingOrders.filter(order => {
            const statusMatch = !statusFilter || order.status === statusFilter;
            const typeMatch = !orderFilter || order.actionType === orderFilter;
            return statusMatch && typeMatch;
          });

          if (filteredOrders.length > 0) {
            return (
              <Row gutter={16}>
                {filteredOrders.map(order => (
                  <Col xs={24} sm={12} md={6} key={order.id}>
                    {renderOrderCard(order, true, true)}
                  </Col>
                ))}
              </Row>
            );
          } else if (statusFilter || orderFilter) {
            // 有筛选条件但无结果时的显示
            let filterText = '';
            if (statusFilter && orderFilter) {
              const statusText = statusFilter === 'pending' ? '待处理' : statusFilter === 'processing' ? '处理中' : '归还中';
              const actionTypeText = orderFilter === 'pool_entry' ? '入池' : '退池';
              filterText = `${statusText}的${actionTypeText}`;
            } else if (statusFilter) {
              filterText = statusFilter === 'pending' ? '待处理' : statusFilter === 'processing' ? '处理中' : '归还中';
            } else {
              filterText = orderFilter === 'pool_entry' ? '入池' : '退池';
            }
            return (
              <div style={{
                textAlign: 'center',
                padding: '60px 20px',
                background: '#fafafa',
                borderRadius: '8px',
                border: '1px dashed #d9d9d9'
              }}>
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={
                    <div>
                      <h3 style={{
                        fontSize: '16px',
                        fontWeight: 500,
                        margin: '16px 0 8px',
                        color: 'rgba(0, 0, 0, 0.85)'
                      }}>
                        暂无{filterText}订单
                      </h3>
                      <p style={{
                        color: 'rgba(0, 0, 0, 0.45)',
                        margin: '0 0 24px',
                        fontSize: '14px'
                      }}>
                        当前没有{filterText}类型的订单，您可以创建一个新的订单
                      </p>
                    </div>
                  }
                >
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={handleOpenCreateOrderModal}
                    size="large"
                    style={{
                      borderRadius: '6px',
                      boxShadow: '0 2px 0 rgba(5, 145, 255, 0.1)'
                    }}
                  >
                    创建订单
                  </Button>
                </Empty>
              </div>
            );
          } else {
            // 无筛选条件且无订单时的显示
            return <EmptyOrderState onCreateOrder={handleOpenCreateOrderModal} />;
          }
        })()}
      </Card>

      {/* 资源用量趋势 */}
      <Card
        className="content-card"
        title="资源用量趋势"
        extra={
          <Space>

            <Select
              value={selectedClusterId}
              style={{ width: 225 }}
              onChange={(value) => handleTrendParamsChange(value, selectedTimeRange, selectedResourceTypes)}
              placeholder="搜索或选择集群"
              optionLabelProp="label"
              suffixIcon={<ClusterOutlined style={{ color: '#52c41a' }} />}
              showSearch
              filterOption={(input, option) => {
                const label = option?.label?.toString().toLowerCase() || '';
                const children = option?.children?.toString().toLowerCase() || '';
                return label.includes(input.toLowerCase()) || children.includes(input.toLowerCase());
              }}
            >
              {getFilteredClusters().map(cluster => (
                <Option 
                  key={cluster.id} 
                  value={cluster.id}
                  label={cluster.name || `集群-${cluster.id}`}
                >
                  <ClusterOutlined style={{ color: '#52c41a', marginRight: 8 }} />
                  <span>{cluster.name || cluster.clusterName || cluster.clusterNameCn || cluster.alias || `集群-${cluster.id}`}</span>
                </Option>
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
              size="middle"
              scroll={{ x: 'max-content' }}
              pagination={{
                total: strategies.total,
                current: strategies.page,
                pageSize: strategies.size,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
                pageSizeOptions: ['10', '20', '50', '100']
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
                const standardTabs = ['processing', 'completed', 'cancelled', 'all'];
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
              <Option value="cancelled">已取消</Option>
              <Option value="all">全部</Option>
              <Option value="return_completed">已归还</Option>
              <Option value="no_return">无须归还</Option>
            </Select>
            <Select defaultValue="7d" style={{ width: 100 }}>
              <Option value="7d">最近7天</Option>
              <Option value="30d">最近30天</Option>
              <Option value="90d">最近90天</Option>
            </Select>
            <Input
              placeholder="搜索订单名称"
              style={{ width: 200 }}
              value={nameFilter}
              onChange={(e) => handleOrderNameFilterChange(e.target.value)}
              allowClear
              prefix={isDataFetching ? <Spin size="small" /> : <SearchOutlined />}
            />
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
                    setCustomTabStatus('return_completed'); // 默认使用'return_completed'作为自定义状态
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
                    completedOrders.map(order => renderOrderCard(order, false, false, true))
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
                    已取消订单
                    <span className={`order-count-badge ${cancelledOrders && cancelledOrders.length > 0 ? 'cancelled' : 'empty'}`}>
                      {cancelledOrders ? cancelledOrders.length : 0}
                    </span>
                  </span>
                }
                key="cancelled"
              >
                <div className="order-cards-grid">
                  {cancelledOrdersLoading ? (
                    <div style={{ textAlign: 'center', padding: '50px 0' }}>
                      <Spin size="large" tip="正在加载已取消订单..." />
                    </div>
                  ) : cancelledOrdersError ? (
                    <Result
                      status="error"
                      title="加载已取消订单失败"
                      subTitle={cancelledOrdersError}
                      extra={<Button type="primary" onClick={() => fetchData(true)}>重试</Button>}
                    />
                  ) : cancelledOrders && cancelledOrders.length > 0 ? (
                    cancelledOrders.map(order => renderOrderCard(order, false, false, true))
                  ) : (
                    <div style={{ padding: '20px 0', textAlign: 'center' }}>
                      <Empty
                        description={
                          <span style={{ color: 'rgba(0,0,0,0.45)' }}>暂无已取消订单</span>
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
                    <Badge status="default" />
                    全部订单
                    <span className="order-count-badge all">
                      {bottomAllOrders ? bottomAllOrders.total : 0}
                    </span>
                  </span>
                }
                key="all"
              >
                <div className="order-cards-grid">
                  {bottomAllOrders && bottomAllOrders.list.length > 0 ? (
                    bottomAllOrders.list.map((order: OrderListItem) => renderOrderCard(order, false, false, true))
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
                      {customTabStatus === 'return_completed' ? '已归还订单' :
                       customTabStatus === 'cancelled' ? '已取消订单' :
                       customTabStatus === 'no_return' ? '无须归还订单' :
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
                      getCustomStatusOrders().map((order: OrderListItem) => renderOrderCard(order, false, false, true))
                    ) : (
                      <div style={{ padding: '20px 0', textAlign: 'center' }}>
                        <Empty
                          description={
                            <span style={{ color: 'rgba(0,0,0,0.45)' }}>暂无{customTabStatus === 'return_completed' ? '已归还' :
                              customTabStatus === 'cancelled' ? '已取消' :
                              customTabStatus === 'no_return' ? '无须归还' :
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
        title={selectedOrder ? `${selectedOrder.strategyName ? selectedOrder.strategyName : (selectedOrder.name || '手动创建')} - ${selectedOrder.actionType === 'pool_entry' ? '入池' : '退池'}` : (selectedStrategy ? `策略详情: ${selectedStrategy.name}` : '')}
        placement="right"
        width={1000}
        onClose={handleCloseDrawer}
        visible={drawerVisible}
        className="detail-drawer"
        zIndex={1100}
      >
        {selectedOrder && (
          <div className="detail-drawer-content">
            <div className="detail-section">
              <Descriptions bordered size="small" column={2} labelStyle={{ width: '120px' }}>
                {selectedOrder.name && (
                  <Descriptions.Item label="订单名称" span={2}>
                    <span style={{ fontWeight: 500, fontSize: '14px' }}>{selectedOrder.name}</span>
                  </Descriptions.Item>
                )}
                <Descriptions.Item label="订单类型" span={2}>
                  <Tag color={selectedOrder.actionType === 'pool_entry' ? 'blue' : 'orange'} style={{ padding: '4px 8px', fontSize: '14px' }}>
                    {selectedOrder.actionType === 'pool_entry' ? <CloudUploadOutlined style={{ marginRight: '4px' }} /> : <CloudDownloadOutlined style={{ marginRight: '4px' }} />}
                    {selectedOrder.actionType === 'pool_entry' ? '入池' : '退池'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="订单状态">
                  <Badge
                    status={
                      selectedOrder.status === 'pending' ? 'error' :
                      selectedOrder.status === 'processing' ? 'processing' :
                      selectedOrder.status === 'returning' ? 'processing' :
                      selectedOrder.status === 'return_completed' ? 'success' :
                      selectedOrder.status === 'no_return' ? 'success' :
                      selectedOrder.status === 'cancelled' ? 'default' :
                      selectedOrder.status === 'failed' ? 'error' :
                      'success'
                    }
                    text={
                      <span style={{ fontWeight: 500 }}>
                        {selectedOrder.status === 'pending' ? '待处理' :
                        selectedOrder.status === 'processing' ? '处理中' :
                        selectedOrder.status === 'returning' ? '归还中' :
                        selectedOrder.status === 'return_completed' ? '归还完成' :
                        selectedOrder.status === 'no_return' ? '无需归还' :
                        selectedOrder.status === 'cancelled' ? '已取消' :
                        selectedOrder.status === 'failed' ? '失败' :
                        '已完成'}
                      </span>
                    }
                  />
                </Descriptions.Item>
                <Descriptions.Item label="触发时间">
                  <span style={{ fontWeight: 500 }}>{new Date(selectedOrder.createdAt).toLocaleString()}</span>
                </Descriptions.Item>
                <Descriptions.Item label="关联策略">
                  {selectedOrder.strategyName ?
                    <Tag color="purple" style={{ padding: '2px 6px' }}>{selectedOrder.strategyName}</Tag> :
                    <Tag color="default" style={{ padding: '2px 6px' }}>手动创建</Tag>
                  }
                </Descriptions.Item>
                <Descriptions.Item label="集群">
                  <Tag color="geekblue" style={{ padding: '2px 6px' }}>{selectedOrder.clusterName}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="创建人">
                  <Tag color="cyan" style={{ padding: '2px 6px' }}>{selectedOrder.createdBy || '系统'}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="执行人">
                  <Tag color="orange" style={{ padding: '2px 6px' }}>{selectedOrder.executor || '未指定'}</Tag>
                </Descriptions.Item>
              </Descriptions>

              {/* 订单描述美化显示 */}
              {selectedOrder.description && (
                <div style={{ 
                  marginTop: '20px', 
                  padding: '20px', 
                  background: 'linear-gradient(135deg, #f8fbff 0%, #f0f7ff 100%)',
                  borderRadius: '12px', 
                  border: '1px solid #e1f0ff',
                  boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)',
                  position: 'relative',
                  overflow: 'hidden'
                }}>
                  <div style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    width: '4px',
                    height: '100%',
                    background: 'linear-gradient(to bottom, #1890ff, #40a9ff)',
                    borderRadius: '0 2px 2px 0'
                  }} />
                  <ProseKitViewer content={selectedOrder.description} />
                </div>
              )}

              {/* 订单状态流转图 */}
              <div style={{ marginTop: '16px', padding: '12px', backgroundColor: '#fafafa', borderRadius: '6px', border: '1px solid #f0f0f0' }}>
                <OrderStatusFlow 
                  actionType={selectedOrder.actionType} 
                  currentStatus={selectedOrder.status} 
                />
              </div>
            </div>

            <div className="detail-section">
              <div className="detail-section-title">
                {selectedOrder.actionType === 'pool_entry' ? 
                  <><CloudUploadOutlined style={{ marginRight: '8px', color: '#1890ff' }} /> 匹配设备列表</> : 
                  <><CloudDownloadOutlined style={{ marginRight: '8px', color: '#fa8c16' }} /> 关联设备列表</>
                }
              </div>
              <div className="device-list" style={{ padding: '0 16px 16px' }}>
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
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <PlusOutlined style={{ marginRight: 8, color: '#1890ff' }} />
            <span>新建监控策略</span>
          </div>
        }
        open={createStrategyModalVisible}
        width={1000}
        okText="确定"
        cancelText="取消"
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
                durationMinutes: (values.durationDays || 0) * 24 * 60,
                cooldownMinutes: (values.cooldownDays || 0) * 24 * 60,

                // CPU相关参数
                cpuThresholdValue: values.cpuThresholdValue,
                cpuThresholdType: 'usage' as 'usage' | 'allocated',
                cpuTargetValue: values.cpuTargetValue,

                // 内存相关参数
                memoryThresholdValue: values.memoryThresholdValue,
                memoryThresholdType: 'usage' as 'usage' | 'allocated',
                memoryTargetValue: values.memoryTargetValue,

                // 条件逻辑和状态
                conditionLogic: values.conditionLogic as 'AND' | 'OR' || 'AND',
                status: values.status as 'enabled' | 'disabled' || 'disabled',

                // 其他必要字段
                createdBy: values.createdBy || 'system',
                clusters: []  // 这个字段会由后端填充
              };

              console.log('发送创建策略请求:', strategyData);

              // 调用创建策略API
              const result = await strategyApi.createStrategy(strategyData);
              console.log('策略创建成功:', result);

              // 刷新列表
              fetchData(true);

              // 关闭弹窗并重置表单
              setCreateStrategyModalVisible(false);
              form.resetFields();
            } catch (error: any) {
              console.error('创建策略失败:', error);
              const errorMsg = error?.response?.data?.msg || error?.message || '创建策略失败';
              message.error(errorMsg);
            }
          }).catch(errorInfo => {
            console.log('表单验证失败:', errorInfo);
          });
        }}
        onCancel={() => {
          setCreateStrategyModalVisible(false);
          form.resetFields();
        }}
        className="create-strategy-modal"
        bodyStyle={{ padding: '24px', background: '#f9fbfd' }}
      >
        <div style={{ marginBottom: '24px' }}>
          <Alert
            message="策略用于监控集群资源使用情况，当达到阈值条件时自动触发弹性伸缩"
            type="info"
            showIcon
            style={{ marginBottom: 0, borderRadius: '4px' }}
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
            bordered={true}
          >
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="thresholdTriggerAction"
                  label={<span style={{ fontWeight: 500 }}>触发动作</span>}
                  rules={[{ required: true, message: '请选择触发动作!' }]}
                >
                  <Select placeholder="请选择动作" showArrow style={{ width: '100%' }}>
                    <Option value="pool_entry">
                      <CloudUploadOutlined style={{ color: '#1890ff', marginRight: 4 }} /> 入池
                    </Option>
                    <Option value="pool_exit">
                      <CloudDownloadOutlined style={{ color: '#ff7a45', marginRight: 4 }} /> 退池
                    </Option>
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="resourceTypes"
                  label={<span style={{ fontWeight: 500 }}>资源池类型</span>}
                  rules={[{ required: true, message: '请选择资源池类型!' }]}
                >
                  <Select
                    mode="multiple"
                    placeholder="请选择资源池类型"
                    showSearch
                    optionFilterProp="children"
                    filterOption={(input, option) =>
                      (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
                    }
                    showArrow
                    maxTagCount="responsive"
                    tagRender={(props) => {
                      const { label, value, closable, onClose } = props;
                      const onPreventMouseDown = (event: React.MouseEvent<HTMLSpanElement>) => {
                        event.preventDefault();
                        event.stopPropagation();
                      };
                      return (
                        <Tag
                          color="blue"
                          onMouseDown={onPreventMouseDown}
                          closable={closable}
                          onClose={onClose}
                          style={{
                            marginRight: 3,
                            marginBottom: 3,
                            borderRadius: 6,
                            fontSize: '12px',
                            padding: '2px 8px',
                            lineHeight: '20px',
                            display: 'inline-flex',
                            alignItems: 'center'
                          }}
                        >
                          <DatabaseOutlined style={{ marginRight: 4, fontSize: '12px' }} />
                          {value === 'compute' ? '' : label}
                        </Tag>
                      );
                    }}
                  >
                    {resourcePools.map(pool => (
                      <Option key={pool.type} value={pool.type}>
                        <DatabaseOutlined style={{ marginRight: 4, color: '#1890ff' }} />
                        {pool.type === 'compute' ? '' : pool.type}
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
                showSearch
                optionFilterProp="children"
                filterOption={(input, option) =>
                  (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
                }
                showArrow
                maxTagCount="responsive"
                style={{ width: '100%' }}
                tagRender={(props) => {
                  const { label, value, closable, onClose } = props;
                  const onPreventMouseDown = (event: React.MouseEvent<HTMLSpanElement>) => {
                    event.preventDefault();
                    event.stopPropagation();
                  };
                  return (
                    <Tag
                      color="green"
                      onMouseDown={onPreventMouseDown}
                      closable={closable}
                      onClose={onClose}
                      style={{
                        marginRight: 3,
                        marginBottom: 3,
                        borderRadius: 6,
                        fontSize: '12px',
                        padding: '2px 8px',
                        lineHeight: '20px',
                        display: 'inline-flex',
                        alignItems: 'center'
                      }}
                    >
                      <ClusterOutlined style={{ marginRight: 4, fontSize: '12px' }} />
                      {label}
                    </Tag>
                  );
                }}
              >
                {clusters.map(cluster => (
                  <Option key={cluster.id} value={cluster.id}>
                    <ClusterOutlined style={{ marginRight: 4, color: '#52c41a' }} />
                    {cluster.name || cluster.clusterName || cluster.clusterNameCn || cluster.alias || `集群-${cluster.id}`}
                  </Option>
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
            bordered={true}
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
            bordered={true}
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
                  <InputNumber min={0} style={{ width: '100%' }} />
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
                  <InputNumber min={0} style={{ width: '100%' }} />
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
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <EditOutlined style={{ marginRight: 8, color: '#1890ff' }} />
            <span>编辑监控策略</span>
          </div>
        }
        open={editStrategyModalVisible}
        maskClosable={false}
        destroyOnClose={false}
        width={1000}
        okText="确定"
        cancelText="取消"
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
                durationMinutes: (values.durationDays || 0) * 24 * 60,
                cooldownMinutes: (values.cooldownDays || 0) * 24 * 60,

                // CPU相关参数
                cpuThresholdValue: values.cpuThresholdValue,
                cpuThresholdType: 'usage' as 'usage' | 'allocated',
                cpuTargetValue: values.cpuTargetValue,

                // 内存相关参数
                memoryThresholdValue: values.memoryThresholdValue,
                memoryThresholdType: 'usage' as 'usage' | 'allocated',
                memoryTargetValue: values.memoryTargetValue,

                // 设备数量和条件逻辑
                deviceCount: values.deviceCount || 1,
                conditionLogic: values.conditionLogic as 'AND' | 'OR',
                status: values.status as 'enabled' | 'disabled',

                // 其他必要字段
                nodeSelector: values.nodeSelector || '',
                createdBy: values.createdBy || 'system',
                clusters: []  // 这个字段会由后端填充
              };

              console.log('发送更新策略请求:', strategyData);

              // 调用更新策略API
              await strategyApi.updateStrategy(currentEditStrategyId, strategyData);

              // 刷新列表
              fetchData(true);

              // 关闭弹窗并重置状态
              setEditStrategyModalVisible(false);
              setCurrentEditStrategyId(null);
              editForm.resetFields();

              message.success('策略更新成功');
            } catch (error: any) {
              console.error('更新策略失败:', error);
              const errorMsg = error?.response?.data?.msg || error?.message || '更新策略失败，请重试';
              message.error(errorMsg);
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
            bordered={true}
          >
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="thresholdTriggerAction"
                  label={<span style={{ fontWeight: 500 }}>触发动作</span>}
                  rules={[{ required: true, message: '请选择触发动作!' }]}
                >
                  <Select placeholder="请选择动作" showArrow style={{ width: '100%' }}>
                    <Option value="pool_entry">
                      <CloudUploadOutlined style={{ color: '#1890ff', marginRight: 4 }} /> 入池
                    </Option>
                    <Option value="pool_exit">
                      <CloudDownloadOutlined style={{ color: '#ff7a45', marginRight: 4 }} /> 退池
                    </Option>
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="resourceTypes"
                  label={<span style={{ fontWeight: 500 }}>资源池类型</span>}
                  rules={[{ required: true, message: '请选择资源池类型!' }]}
                >
                  <Select
                    mode="multiple"
                    placeholder="请选择资源池类型"
                    showSearch
                    optionFilterProp="children"
                    filterOption={(input, option) =>
                      (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
                    }
                    showArrow
                    maxTagCount="responsive"
                    tagRender={(props) => {
                      const { label, value, closable, onClose } = props;
                      const onPreventMouseDown = (event: React.MouseEvent<HTMLSpanElement>) => {
                        event.preventDefault();
                        event.stopPropagation();
                      };
                      return (
                        <Tag
                          color="blue"
                          onMouseDown={onPreventMouseDown}
                          closable={closable}
                          onClose={onClose}
                          style={{
                            marginRight: 3,
                            marginBottom: 3,
                            borderRadius: 6,
                            fontSize: '12px',
                            padding: '2px 8px',
                            lineHeight: '20px',
                            display: 'inline-flex',
                            alignItems: 'center'
                          }}
                        >
                          <DatabaseOutlined style={{ marginRight: 4, fontSize: '12px' }} />
                          {value === 'compute' ? '' : label}
                        </Tag>
                      );
                    }}
                  >
                    {resourcePools.map(pool => (
                      <Option key={pool.type} value={pool.type}>
                        <DatabaseOutlined style={{ marginRight: 4, color: '#1890ff' }} />
                        {pool.type === 'compute' ? '' : pool.type}
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
                showSearch
                optionFilterProp="children"
                filterOption={(input, option) =>
                  (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
                }
                showArrow
                maxTagCount="responsive"
                style={{ width: '100%' }}
                tagRender={(props) => {
                  const { label, value, closable, onClose } = props;
                  const onPreventMouseDown = (event: React.MouseEvent<HTMLSpanElement>) => {
                    event.preventDefault();
                    event.stopPropagation();
                  };
                  return (
                    <Tag
                      color="green"
                      onMouseDown={onPreventMouseDown}
                      closable={closable}
                      onClose={onClose}
                      style={{
                        marginRight: 3,
                        marginBottom: 3,
                        borderRadius: 6,
                        fontSize: '12px',
                        padding: '2px 8px',
                        lineHeight: '20px',
                        display: 'inline-flex',
                        alignItems: 'center'
                      }}
                    >
                      <ClusterOutlined style={{ marginRight: 4, fontSize: '12px' }} />
                      {label}
                    </Tag>
                  );
                }}
              >
                {clusters.map(cluster => (
                  <Option key={cluster.id} value={cluster.id}>
                    <ClusterOutlined style={{ marginRight: 4, color: '#52c41a' }} />
                    {cluster.name || cluster.clusterName || cluster.clusterNameCn || cluster.alias || `集群-${cluster.id}`}
                  </Option>
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
            bordered={true}
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
            bordered={true}
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
                  <InputNumber min={0} style={{ width: '100%' }} />
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
                  <InputNumber min={0} style={{ width: '100%' }} />
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

      {/* 策略执行历史抽屉 */}
      <Drawer
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <ClockCircleOutlined style={{ marginRight: 8, color: '#722ed1' }} />
            <span>策略执行历史</span>
            {selectedStrategyForHistory && (
              <Tag color="blue" style={{ marginLeft: 8 }}>
                {selectedStrategyForHistory.name}
              </Tag>
            )}
          </div>
        }
        width={1000}
        open={executionHistoryDrawerVisible}
        onClose={() => setExecutionHistoryDrawerVisible(false)}
        className="execution-history-drawer"
        zIndex={1000}
        extra={
          <Space>
            <Input
              placeholder="搜索集群名称"
              value={historyClusterFilter}
              onChange={(e) => handleClusterFilterChange(e.target.value)}
              style={{ width: 200 }}
              prefix={<SearchOutlined />}
              allowClear
            />
            <Button
              icon={<ReloadOutlined />}
              onClick={() => selectedStrategyForHistory && handleViewExecutionHistory(selectedStrategyForHistory)}
            >
              刷新
            </Button>
          </Space>
        }
      >
        {executionHistoryLoading ? (
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <Spin size="large" tip="正在加载执行历史..." />
          </div>
        ) : (
          <div>
            {/* 执行历史统计 */}
            <Card size="small" style={{ marginBottom: 16 }}>
              <Row gutter={16}>
                <Col span={6}>
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: 20, fontWeight: 'bold', color: '#52c41a' }}>
                      {executionHistory.filter(h => h.result === 'order_created').length}
                    </div>
                    <div style={{ fontSize: 12, color: '#666' }}>成功生成订单</div>
                  </div>
                </Col>
                <Col span={6}>
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: 20, fontWeight: 'bold', color: '#ff7a45' }}>
                      {executionHistory.filter(h => h.result === 'order_created_no_devices').length}
                    </div>
                    <div style={{ fontSize: 12, color: '#666' }}>设备不足提醒</div>
                  </div>
                </Col>
                <Col span={6}>
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: 20, fontWeight: 'bold', color: '#faad14' }}>
                      {executionHistory.filter(h => h.result === 'order_created_partial').length}
                    </div>
                    <div style={{ fontSize: 12, color: '#666' }}>部分匹配订单</div>
                  </div>
                </Col>
                <Col span={6}>
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: 20, fontWeight: 'bold', color: '#ff4d4f' }}>
                      {executionHistory.filter(h => h.result.includes('failure')).length}
                    </div>
                    <div style={{ fontSize: 12, color: '#666' }}>执行失败</div>
                  </div>
                </Col>
              </Row>
            </Card>

            {/* 执行历史列表 */}
            <Table
              dataSource={executionHistory}
              rowKey="id"
              loading={executionHistoryLoading}
              pagination={{
                current: executionHistoryPagination.page,
                pageSize: executionHistoryPagination.size,
                total: executionHistoryTotal,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条记录`,
                onChange: (page, size) => handleExecutionHistoryPaginationChange(page, size || 10),
                onShowSizeChange: (current, size) => handleExecutionHistoryPaginationChange(1, size)
              }}
              size="small"
              columns={[
                {
                  title: '执行时间',
                  dataIndex: 'executionTime',
                  key: 'executionTime',
                  width: 140,
                  render: (time: string) => (
                    <div style={{ fontSize: 12 }}>
                      {new Date(time).toLocaleString('zh-CN', {
                        month: '2-digit',
                        day: '2-digit',
                        hour: '2-digit',
                        minute: '2-digit'
                      })}
                    </div>
                  )
                },
                {
                  title: '集群',
                  dataIndex: 'clusterName',
                  key: 'clusterName',
                  width: 120,
                  render: (clusterName: string) => (
                    <div style={{ display: 'flex', alignItems: 'center' }}>
                      <ClusterOutlined style={{ marginRight: 4, color: '#52c41a' }} />
                      <span style={{ fontSize: 12 }}>{clusterName || '未知集群'}</span>
                    </div>
                  )
                },
                {
                  title: '资源池',
                  dataIndex: 'resourcePool',
                  key: 'resourcePool',
                  width: 80,
                  render: (resourcePool: string) => (
                    <Tag color="blue">
                      {resourcePool || 'total'}
                    </Tag>
                  )
                },
                // {
                //   title: '触发值/阈值',
                //   key: 'values',
                //   width: 100,
                //   render: (record: any) => (
                //     <div style={{ fontSize: 11 }}>
                //       <div>触发: {record.triggeredValue}</div>
                //       <div style={{ color: '#666' }}>阈值: {record.thresholdValue}</div>
                //     </div>
                //   )
                // },
                {
                  title: '执行结果',
                  dataIndex: 'result',
                  key: 'result',
                  width: 120,
                  render: (result: string) => {
                    const getResultConfig = (result: string) => {
                      switch (result) {
                        case 'order_created':
                          return { color: 'success', text: '生成订单', icon: <CheckCircleOutlined /> };
                        case 'order_created_no_devices':
                          return { color: 'warning', text: '设备不足提醒', icon: <WarningOutlined /> };
                        case 'order_created_partial':
                          return { color: 'processing', text: '部分匹配', icon: <ExclamationCircleOutlined /> };
                        case 'order_created_partial_devices':
                          return { color: 'processing', text: '部分设备匹配', icon: <ExclamationCircleOutlined /> };
                        case 'failure_threshold_not_met':
                          return { color: 'default', text: '阈值未满足', icon: <CloseCircleOutlined /> };
                        default:
                          if (result.includes('failure')) {
                            return { color: 'error', text: '执行失败', icon: <CloseCircleOutlined /> };
                          }
                          return { color: 'default', text: result, icon: <ClockCircleOutlined /> };
                      }
                    };

                    const config = getResultConfig(result);
                    return (
                      <Tag color={config.color} style={{ fontSize: 11 }}>
                        {config.icon}
                        <span style={{ marginLeft: 4 }}>{config.text}</span>
                      </Tag>
                    );
                  }
                },
                {
                  title: '订单',
                  dataIndex: 'orderId',
                  key: 'orderId',
                  width: 80,
                  render: (orderId: number) => (
                    orderId ? (
                      <Button
                        type="link"
                        size="small"
                        onClick={() => handleViewOrderDetails(orderId)}
                        style={{ padding: 0, fontSize: 11 }}
                      >
                        #{orderId}
                      </Button>
                    ) : (
                      <span style={{ color: '#ccc', fontSize: 11 }}>--</span>
                    )
                  )
                },
                {
                  title: '详情',
                  dataIndex: 'reason',
                  key: 'reason',
                  ellipsis: true,
                  render: (reason: string) => (
                    <Tooltip title={reason} placement="topLeft">
                      <span style={{ fontSize: 11, color: '#666' }}>
                        {reason.length > 30 ? `${reason.substring(0, 30)}...` : reason}
                      </span>
                    </Tooltip>
                  )
                }
              ]}
            />
          </div>
        )}
      </Drawer>

      {/* 创建订单模态框 */}
      <CreateOrderModal
        ref={createOrderModalRef}
        visible={createOrderModalVisible}
        onCancel={handleCloseCreateOrderModal}
        onSubmit={handleCreateOrderSubmit}
        clusters={clusters}
        resourcePools={resourcePools}
        initialValues={clonedOrderInfo} // 传递克隆的订单信息
      />
    </div>
  );
};

export default Dashboard;