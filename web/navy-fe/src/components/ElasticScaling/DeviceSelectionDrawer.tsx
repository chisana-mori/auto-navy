import React, { useState, useEffect, useCallback } from 'react';
import {
  Drawer,
  Button,
  Table,
  Space,
  Spin,
  message,
  Badge,
  Typography,
  Tag,
  Input,
} from 'antd';
import {
  SelectOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { Device } from '../../types/device';
import { queryDevices } from '../../services/deviceQueryService';
import { FilterGroup, FilterType, ConditionType, LogicalOperator } from '../../types/deviceQuery';

const { Text } = Typography;

interface DeviceSelectionDrawerProps {
  visible: boolean;
  onClose: () => void;
  onSelectDevices: (devices: Device[]) => void;
  filterGroups?: FilterGroup[];
  appliedFilters?: any[];
  selectedDevices: Device[];
  loading?: boolean;
  simpleMode?: boolean; // 是否使用简单模式（只显示关键字搜索）
  initialSelectionActionType?: string; // 新增属性，用于判断初始动作类型
}

const DeviceSelectionDrawer: React.FC<DeviceSelectionDrawerProps> = ({
  visible,
  onClose,
  onSelectDevices,
  filterGroups = [],
  appliedFilters = [], // 添加 appliedFilters 的解构和默认值
  selectedDevices,
  loading: externalLoading,
  simpleMode = false,
  initialSelectionActionType
}) => {
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(false);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [localSelectedDevices, setLocalSelectedDevices] = useState<Device[]>([]);
  const [keyword, setKeyword] = useState<string>('');
  const [multilineKeywords, setMultilineKeywords] = useState<string>('');
  const [clientSearchText, setClientSearchText] = useState<string>('');
  const [displayedDevices, setDisplayedDevices] = useState<Device[]>([]);

  // 查询设备
  const fetchDevices = useCallback(async (page = 1, pageSize = 500, keywordList?: string[]) => {
    try {
      setLoading(true);

      let response;

      if (simpleMode) {
        // 简单模式：使用关键字搜索
        // 构建基本查询条件
        const queryGroups: FilterGroup[] = [];

        // 处理多行关键字
        if (keywordList && keywordList.length > 0) {
          // 创建一个OR组合的查询组
          const group: FilterGroup = {
            id: '1',
            operator: LogicalOperator.Or,
            blocks: keywordList.map((kw, index) => ({
              id: `${index + 1}`,
              type: FilterType.Device,
              field: 'ip', // 默认搜索IP字段
              key: 'ip',
              operator: LogicalOperator.And,
              conditionType: ConditionType.Contains,
              value: [kw.trim()]
            }))
          };
          queryGroups.push(group);
        } else if (keyword.trim()) {
          // 如果有单行关键字，创建一个基本的查询组
          const group: FilterGroup = {
            id: '1',
            operator: LogicalOperator.And,
            blocks: [{
              id: '1',
              type: FilterType.Device,
              field: 'ip', // 默认搜索IP字段
              key: 'ip',
              operator: LogicalOperator.And,
              conditionType: ConditionType.Contains,
              value: [keyword.trim()]
            }]
          };
          queryGroups.push(group);
        }

        // 执行查询
        response = await queryDevices({
          groups: queryGroups.length > 0 ? queryGroups : [{ id: '0', operator: LogicalOperator.And, blocks: [] }],
          page,
          size: pageSize,
        });
        
        // 调试：检查返回的数据
        console.log('queryDevices API 返回的数据:', response);
      } else {
        // 高级模式：使用传入的筛选条件
        if (!filterGroups || filterGroups.length === 0) {
          message.warning('请先添加筛选条件');
          setLoading(false);
          return;
        }

        // 深拷贝筛选条件，避免修改原始数据
        const clonedGroups = JSON.parse(JSON.stringify(filterGroups));

        // 处理筛选条件中的数组值和undefined值
        const processedGroups = clonedGroups.map((group: FilterGroup) => {
          return {
            ...group,
            blocks: group.blocks.map((block) => {
              const processedBlock = { ...block };

              // 处理undefined值
              if (processedBlock.value === undefined) {
                processedBlock.value = '';
              }

              // 处理数组值
              if (Array.isArray(processedBlock.value)) {
                processedBlock.value = processedBlock.value.join(',');
              }

              return processedBlock;
            })
          };
        });

        // 执行查询，支持分页
        response = await queryDevices({
          groups: processedGroups,
          page,
          size: pageSize,
        });
      }

      // 更新设备列表和分页信息
      setDevices(response.list || []);
      setPagination({
        current: page,
        pageSize,
        total: response.total || 0,
      });
    } catch (error) {
      console.error('查询设备失败:', error);
      message.error('查询设备失败');
    } finally {
      setLoading(false);
    }
  }, [filterGroups, keyword, simpleMode]);

  // 初始化本地选中设备
  useEffect(() => {
    if (visible) {
      // 如果是退池操作，则不默认选中任何设备
      if (initialSelectionActionType === 'pool_exit') {
        setLocalSelectedDevices([]);
        setSelectedRowKeys([]);
      } else if (selectedDevices.length > 0) {
        // 其他情况，如果已有选中设备，则保留
        setLocalSelectedDevices(selectedDevices);
        setSelectedRowKeys(selectedDevices.map(device => device.id));
      } else {
        // 其他情况且无选中设备，则清空
        setLocalSelectedDevices([]);
        setSelectedRowKeys([]);
      }

      // 重置分页信息，确保每次打开抽屉时都从第一页开始
      setPagination({
        current: 1,
        pageSize: 500, // 默认分页大小调整为500
        total: 0
      });

      // 当抽屉打开时，自动执行查询
      fetchDevices(1, 500); // 默认分页大小调整为500
    }
  }, [visible, selectedDevices, initialSelectionActionType, fetchDevices]);

  // Effect to apply client-side search whenever clientSearchText or the base 'devices' list changes
  useEffect(() => {
    if (!clientSearchText.trim()) {
      setDisplayedDevices(devices);
      return;
    }

    const searchTerms = clientSearchText.toLowerCase().split('\n').filter(term => term.trim() !== '');
    if (searchTerms.length === 0) {
      setDisplayedDevices(devices);
      return;
    }

    const filtered = devices.filter(device => {
      return searchTerms.some(term => {
        const searchableProperties = [
          String(device.id),
          device.ciCode,
          device.ip,
          device.archType,
          device.idc,
          device.room,
          device.cabinet,
          device.cabinetNO,
          device.infraType,
          device.netZone,
          device.group,
          device.appId,
          device.appName,
          device.model,
          device.kvmIp,
          device.os,
          device.company,
          device.osName,
          device.osIssue,
          device.osKernel,
          device.status,
          device.role,
          device.cluster,
          device.isLocalization ? '是' : '否',
        ];
        return searchableProperties.some(prop =>
          prop && typeof prop === 'string' && prop.toLowerCase().includes(term)
        );
      });
    });
    setDisplayedDevices(filtered);
  }, [clientSearchText, devices]);

  // 处理多行关键字搜索
  const handleMultilineSearch = () => {
    if (!multilineKeywords.trim()) {
      message.warning('请输入搜索关键字');
      return;
    }

    // 按行分割关键字
    const keywords = multilineKeywords.split('\n').filter(k => k.trim());
    if (keywords.length === 0) {
      message.warning('请输入有效的搜索关键字');
      return;
    }

    // 执行查询
    fetchDevices(1, pagination.pageSize, keywords);
  };

  // 处理表格分页变化
  const handleTableChange = (newPagination: any) => {
    // 更新分页信息
    setPagination({
      current: newPagination.current,
      pageSize: newPagination.pageSize,
      total: pagination.total // 保留总记录数
    });
    
    // 使用新的分页参数重新获取数据
    fetchDevices(newPagination.current, newPagination.pageSize);
  };

  const handleClientSearchChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setClientSearchText(e.target.value);
  };

  // 处理行选择变化
  const handleRowSelectionChange = (selectedRowKeys: React.Key[], selectedRows: Device[]) => {
    setSelectedRowKeys(selectedRowKeys);
    setLocalSelectedDevices(selectedRows);
  };

  // 确认选择设备
  const handleConfirmSelection = () => {
    onSelectDevices(localSelectedDevices);
    onClose();
  };

  // 表格列定义
  const columns: ColumnsType<Device> = [
    {
      title: '设备编码',
      dataIndex: 'ciCode',
      key: 'ciCode',
      width: 140,
      ellipsis: true,
      render: (text) => (
        <Text style={{ fontSize: '13px', fontFamily: 'Monaco, monospace' }}>
          {text || '-'}
        </Text>
      ),
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
      width: 130,
      render: (text) => (
        <Text style={{ fontSize: '13px', fontFamily: 'Monaco, monospace', color: '#1890ff' }}>
          {text || '-'}
        </Text>
      ),
    },
    {
      title: '机器用途',
      dataIndex: 'group',
      key: 'group',
      width: 120,
      ellipsis: true,
      render: (text) => (
        <Text style={{ fontSize: '13px' }}>
          {text || '-'}
        </Text>
      ),
    },
    {
      title: '所属集群',
      dataIndex: 'cluster',
      key: 'cluster',
      width: 120,
      ellipsis: true,
      render: (text) => (
        <Text style={{ fontSize: '13px' }}>
          {text || '-'}
        </Text>
      ),
    },
    {
      title: '集群角色',
      dataIndex: 'role',
      key: 'role',
      width: 100,
      ellipsis: true,
      render: (text) => (
        <Text style={{ fontSize: '13px' }}>
          {text || '-'}
        </Text>
      ),
    },
    {
      title: '国产化',
      dataIndex: 'isLocalization',
      key: 'isLocalization',
      width: 80,
      align: 'center',
      render: (isLocalization) => (
        <Tag 
          color={isLocalization ? 'success' : 'default'} 
          style={{ 
            fontSize: '12px', 
            borderRadius: '4px',
            margin: 0,
            minWidth: '32px',
            textAlign: 'center'
          }}
        >
          {isLocalization ? '是' : '否'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      align: 'center',
      render: (status) => {
        let color = 'default';
        let text = '未知';
        
        if (status === 'active' || status === 'normal') {
          color = 'success';
          text = '正常';
        } else if (status === 'maintenance') {
          color = 'warning';
          text = '维护';
        } else if (status === 'offline') {
          color = 'error';
          text = '离线';
        } else if (status) {
          text = status;
        }
        
        return (
          <Tag 
            color={color} 
            style={{ 
              fontSize: '12px', 
              borderRadius: '4px',
              margin: 0,
              minWidth: '40px',
              textAlign: 'center'
            }}
          >
            {text}
          </Tag>
        );
      },
    },
  ];

  return (
    <Drawer
      title={
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <SelectOutlined style={{ marginRight: 8, color: '#1890ff' }} />
          <span style={{ fontSize: '16px', fontWeight: 500 }}>选择设备</span>
          {localSelectedDevices.length > 0 && (
            <Badge
              count={localSelectedDevices.length}
              style={{ marginLeft: 12, backgroundColor: '#52c41a' }}
            />
          )}
        </div>
      }
      width={1000}
      placement="right"
      onClose={onClose}
      open={visible}
      mask={true}
      maskClosable={true}
      maskStyle={{ backgroundColor: 'rgba(0, 0, 0, 0.45)' }}
      style={{ 
        boxShadow: '0 4px 24px rgba(0, 0, 0, 0.15)'
      }}
      bodyStyle={{ 
        padding: 0,
        backgroundColor: '#fafafa',
        height: 'calc(100vh - 108px)',
        display: 'flex',
        flexDirection: 'column'
      }}
      headerStyle={{ 
        backgroundColor: '#fff',
        borderBottom: '1px solid #e8e8e8',
        padding: '16px 24px',
        boxShadow: '0 1px 4px rgba(0, 0, 0, 0.04)'
      }}
      footerStyle={{
        backgroundColor: '#fff',
        borderTop: '1px solid #e8e8e8',
        padding: '16px 24px',
        boxShadow: '0 -1px 4px rgba(0, 0, 0, 0.04)'
      }}
      zIndex={1001}
      extra={
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => fetchDevices(pagination.current, pagination.pageSize)}
            disabled={loading || externalLoading}
          >
            刷新
          </Button>
        </Space>
      }
      footer={
        <div style={{ textAlign: 'right' }}>
          <Space>
            <Button onClick={onClose}>
              取消
            </Button>
            <Button
              type="primary"
              icon={<CheckCircleOutlined />}
              onClick={handleConfirmSelection}
              disabled={localSelectedDevices.length === 0}
            >
              确认选择 ({localSelectedDevices.length})
            </Button>
          </Space>
        </div>
      }
    >
      <div style={{ 
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden'
      }}>
        <Spin spinning={loading || externalLoading} style={{ height: '100%' }}>
          {/* 搜索区域 */}
          <div style={{ 
            padding: '20px 24px 16px',
            backgroundColor: '#fff',
            borderBottom: '1px solid #f0f0f0',
            flexShrink: 0
          }}>
            <Input.TextArea
              rows={2}
              placeholder="输入关键字进行前端模糊搜索（多行输入，每行一个关键字）..."
              value={clientSearchText}
              onChange={handleClientSearchChange}
              allowClear
              style={{ 
                backgroundColor: '#fafafa', 
                border: '1px solid #e8e8e8', 
                borderRadius: '8px',
                fontSize: '14px'
              }}
            />
          </div>
          {/* 简单模式搜索 */}
          {simpleMode && (
            <div style={{ 
              padding: '16px 24px',
              backgroundColor: '#fff',
              borderBottom: '1px solid #f0f0f0',
              flexShrink: 0
            }}>
              <div style={{ marginBottom: 16 }}>
                <Text strong style={{ fontSize: '14px', color: '#262626' }}>设备搜索</Text>
                <div style={{ marginTop: 8 }}>
                  <Input.Search
                    placeholder="输入关键字搜索设备（如IP、设备编码等）"
                    value={keyword}
                    onChange={(e) => setKeyword(e.target.value)}
                    onSearch={() => fetchDevices(1, pagination.pageSize)}
                    enterButton="搜索"
                    style={{ width: '100%' }}
                    allowClear
                  />
                </div>
              </div>

              <div>
                <div style={{ marginBottom: 8 }}>
                  <Text style={{ fontSize: '14px', color: '#595959' }}>批量查询（每行一个关键字）：</Text>
                </div>
                <Input.TextArea
                  placeholder="请输入多行关键字，每行一个，如IP地址列表"
                  value={multilineKeywords}
                  onChange={(e) => setMultilineKeywords(e.target.value)}
                  rows={3}
                  style={{ 
                    marginBottom: 12, 
                    backgroundColor: '#fafafa', 
                    border: '1px solid #e8e8e8',
                    borderRadius: '6px'
                  }}
                />
                <Button
                  type="primary"
                  icon={<SearchOutlined />}
                  onClick={handleMultilineSearch}
                  style={{ width: '100%', borderRadius: '6px' }}
                >
                  批量查询
                </Button>
              </div>
            </div>
          )}

          {/* 筛选条件区域 */}
          <div style={{
            padding: '16px 24px',
            backgroundColor: '#fff',
            borderBottom: '1px solid #f0f0f0',
            flexShrink: 0
          }}>
            {appliedFilters && appliedFilters.length > 0 && (
              <div style={{ marginBottom: '12px' }}>
                <Text strong style={{ marginRight: '8px', fontSize: '14px', color: '#262626' }}>当前筛选条件:</Text>
                <div style={{ marginTop: '8px', display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                  {appliedFilters.map((filterGroup: any, groupIndex: number) => (
                    <React.Fragment key={`group-${groupIndex}`}>
                      {filterGroup.blocks?.map((filter: any, filterIndex: number) => {
                        return (
                          <Tag key={`filter-${groupIndex}-${filterIndex}`} color="blue" style={{ margin: 0, borderRadius: '4px' }}>
                            {filter.label}
                          </Tag>
                        );
                      })}
                    </React.Fragment>
                  ))}
                </div>
              </div>
            )}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Text strong style={{ fontSize: '14px', color: '#262626' }}>请选择需要添加到订单的设备：</Text>
              <Text type="secondary" style={{ fontSize: '13px' }}>共 {pagination.total} 条记录</Text>
            </div>
          </div>

          {/* 表格区域 - 使用flex-grow让表格区域填充剩余空间 */}
          <div style={{ 
            flex: 1,
            overflow: 'auto',
            padding: '0 24px 16px',
            backgroundColor: '#fff'
          }}>
            <Table
              rowKey="id"
              columns={columns}
              dataSource={displayedDevices}
              pagination={{
                ...pagination,
                showTotal: (total) => `共 ${total} 条记录`,
                showSizeChanger: true,
                pageSizeOptions: ['10', '20', '50', '100'],
                showQuickJumper: true,
                size: 'small'
              }}
              loading={loading}
              rowSelection={{
                type: 'checkbox',
                selectedRowKeys,
                onChange: handleRowSelectionChange,
                selections: [
                  Table.SELECTION_ALL,
                  Table.SELECTION_INVERT,
                  Table.SELECTION_NONE,
                ],
                columnWidth: 40
              }}
              onChange={handleTableChange}
              scroll={{ y: 'calc(100vh - 400px)' }} // 动态计算表格高度
              size="small"
              bordered={false}
              style={{ marginTop: '12px' }}
              onRow={(record) => {
                // 根据条件决定背景色
                let bgColor = '';
                
                // 调试信息
                if (record.ciCode === 'DEV001') {
                  console.log('DEV001 设备信息:', {
                    isSpecial: record.isSpecial,
                    group: record.group,
                    featureCount: record.featureCount,
                    cluster: record.cluster,
                    appName: record.appName,
                    full_record: record
                  });
                }
                
                // 特殊设备（有机器用途或其他特殊标记）使用浅黄色背景
                if (record.isSpecial ||
                    (record.group && record.group.trim() !== '') ||
                    (record.featureCount && record.featureCount > 0) ||
                    (record.appName && record.appName.trim() !== '')) {
                  // 浅黄色背景 - 特殊设备
                  bgColor = '#fffbe6';
                }
                // 有集群但不是特殊设备的使用浅绿色背景
                else if (record.cluster && record.cluster.trim() !== '') {
                  // 浅绿色背景 - 集群不为空且非特殊设备
                  bgColor = '#f6ffed';
                }
                return {
                  style: { backgroundColor: bgColor },
                  onClick: () => {
                    // 点击行切换选中状态
                    const isSelected = selectedRowKeys.includes(record.id);
                    const newSelectedKeys = isSelected
                      ? selectedRowKeys.filter(key => key !== record.id)
                      : [...selectedRowKeys, record.id];

                    const newSelectedDevices = isSelected
                      ? localSelectedDevices.filter(device => device.id !== record.id)
                      : [...localSelectedDevices, record];

                    setSelectedRowKeys(newSelectedKeys);
                    setLocalSelectedDevices(newSelectedDevices);
                  },
                };
              }}
            />
          </div>
        </Spin>
      </div>
    </Drawer>
  );
};

export default DeviceSelectionDrawer;
