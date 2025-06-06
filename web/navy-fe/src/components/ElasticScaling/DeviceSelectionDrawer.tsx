import React, { useState, useEffect, useMemo } from 'react';
import {
  Drawer,
  Button,
  Table,
  Space,
  Spin,
  Empty,
  message,
  Divider,
  Badge,
  Typography,
  Tag,
  Input,
  Tooltip,
  Card,
  Popover,
  Checkbox,
  Tabs
} from 'antd';
import {
  SelectOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  SearchOutlined,
  FilterOutlined,
  DownloadOutlined
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
}

const DeviceSelectionDrawer: React.FC<DeviceSelectionDrawerProps> = ({
  visible,
  onClose,
  onSelectDevices,
  filterGroups = [],
  appliedFilters = [], // 添加 appliedFilters 的解构和默认值
  selectedDevices,
  loading: externalLoading,
  simpleMode = false
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

  // 初始化本地选中设备
  useEffect(() => {
    if (visible) {
      // 抽屉打开时，执行查询但不自动选中任何设备
      // 除非用户已经明确选择了设备，才保留选中状态
      if (selectedDevices.length > 0) {
        setLocalSelectedDevices(selectedDevices);
        setSelectedRowKeys(selectedDevices.map(device => device.id));
      } else {
        // 重置选中状态
        setLocalSelectedDevices([]);
        setSelectedRowKeys([]);
      }

      // 重置分页信息，确保每次打开抽屉时都从第一页开始
      setPagination({
        current: 1,
        pageSize: 20,
        total: 0
      });

      // 当抽屉打开时，自动执行查询
      fetchDevices(1, 20);
    }
  }, [visible, selectedDevices]);

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

  // 查询设备
  const fetchDevices = async (page = 1, pageSize = 20, keywordList?: string[]) => {
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
      width: 150,
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
      width: 120,
    },
    {
      title: '机器用途',
      dataIndex: 'group',
      key: 'group',
      width: 120,
      render: (text) => {
        if (!text) return '-';
        // 如果文本超过10个字符，显示渐变效果
        if (text.length > 10) {
          return (
            <div className="truncated-text" style={{
              maxWidth: '100px',
              overflow: 'hidden',
              position: 'relative',
              whiteSpace: 'nowrap'
            }}>
              <span>{text}</span>
              <div style={{
                position: 'absolute',
                top: 0,
                right: 0,
                width: '30px',
                height: '100%',
                background: 'linear-gradient(to right, transparent, #fff)'
              }}></div>
            </div>
          );
        }
        return text;
      },
    },
    {
      title: '所属集群',
      dataIndex: 'cluster',
      key: 'cluster',
      width: 120,
      render: (text) => text || '-',
    },
    {
      title: '集群角色',
      dataIndex: 'role',
      key: 'role',
      width: 100,
      render: (text) => text || '-',
    },
    {
      title: '是否国产化',
      dataIndex: 'isLocalization',
      key: 'isLocalization',
      width: 100,
      render: (isLocalization) => (
        isLocalization ?
          <Tag color="green">是</Tag> :
          <Tag color="orange">否</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status) => {
        if (status === 'active' || status === 'normal') {
          return <Tag color="success">正常</Tag>;
        } else if (status === 'maintenance') {
          return <Tag color="warning">维护中</Tag>;
        } else if (status === 'offline') {
          return <Tag color="error">离线</Tag>;
        }
        return <Tag>{status || '未知'}</Tag>;
      },
    },
  ];

  return (
    <Drawer
      title={
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <SelectOutlined style={{ marginRight: 8, color: '#1890ff' }} />
          <span>选择设备</span>
          {localSelectedDevices.length > 0 && (
            <Badge
              count={localSelectedDevices.length}
              style={{ marginLeft: 8, backgroundColor: '#52c41a' }}
            />
          )}
        </div>
      }
      width={900}
      placement="right"
      onClose={onClose}
      open={visible}
      mask={true}
      maskClosable={true}
      maskStyle={{ backgroundColor: 'rgba(0, 0, 0, 0.65)' }}
      style={{ boxShadow: '0 0 20px rgba(0, 0, 0, 0.2)' }}
      bodyStyle={{ 
        padding: '20px', 
        backgroundColor: '#f5f7fa' 
      }}
      headerStyle={{ 
        backgroundColor: '#fff',
        borderBottom: '1px solid #f0f0f0',
        padding: '16px 24px'
      }}
      footerStyle={{
        backgroundColor: '#fff',
        borderTop: '1px solid #f0f0f0',
        padding: '12px 24px'
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
      <div style={{ padding: '0' }}>
        <Spin spinning={loading || externalLoading}>
          <div style={{ marginBottom: 16 }}>
            <Input.TextArea
              rows={3}
              placeholder="输入关键字进行前端模糊搜索（多行输入，每行一个关键字）..."
              value={clientSearchText}
              onChange={handleClientSearchChange}
              allowClear
              style={{ backgroundColor: '#fff', border: '1px solid #d9d9d9', borderRadius: '6px' }}
            />
          </div>
          {simpleMode && (
            <Card 
              size="small" 
              style={{ 
                marginBottom: 16, 
                borderRadius: '8px',
                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
              }}
              bodyStyle={{ padding: '16px' }}
            >
              <Text strong>简单设备搜索</Text>
              <div style={{ marginBottom: 16 }}>
                <Input.Search
                  placeholder="输入关键字搜索设备（如IP、设备编码等）"
                  value={keyword}
                  onChange={(e) => setKeyword(e.target.value)}
                  onSearch={() => fetchDevices(1, pagination.pageSize)}
                  enterButton
                  style={{ width: '100%' }}
                  allowClear
                />
              </div>

              <div>
                <div style={{ marginBottom: 8 }}>
                  <Text>多行批量查询（每行一个关键字）：</Text>
                </div>
                <Input.TextArea
                  placeholder="请输入多行关键字，每行一个，如IP地址列表"
                  value={multilineKeywords}
                  onChange={(e) => setMultilineKeywords(e.target.value)}
                  rows={4}
                  style={{ marginBottom: 8, backgroundColor: '#fff', border: '1px solid #d9d9d9' }}
                />
                <Button
                  type="primary"
                  icon={<SearchOutlined />}
                  onClick={handleMultilineSearch}
                  style={{ width: '100%' }}
                >
                  批量查询
                </Button>
              </div>
            </Card>
          )}

          <div style={{
            marginBottom: 16,
            backgroundColor: '#fff',
            padding: '12px 16px',
            borderRadius: '6px',
            border: '1px solid #f0f0f0'
          }}>
            {appliedFilters && appliedFilters.length > 0 && (
              <div style={{ marginBottom: '12px' }}>
                <Text strong style={{ marginRight: '8px' }}>当前筛选条件:</Text>
                {appliedFilters.map((filterGroup: any, groupIndex: number) => (
                  <React.Fragment key={`group-${groupIndex}`}>
                    {filterGroup.blocks?.map((filter: any, filterIndex: number) => {
                      return (
                         <Tag key={`filter-${groupIndex}-${filterIndex}`} color="blue" style={{ margin: '2px' }}>
                           {filter.label}
                         </Tag>
                       );
                    })}
                  </React.Fragment>
                ))}
              </div>
            )}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Text strong>根据筛选条件找到的设备列表，请选择需要添加到订单的设备：</Text>
              <Text type="secondary">共 {pagination.total} 条记录</Text>
            </div>
          </div>

          <div style={{ 
            backgroundColor: '#fff', 
            borderRadius: '8px',
            boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)',
            padding: '1px'
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
                showQuickJumper: true
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
              }}
              onChange={handleTableChange}
              scroll={{ y: 500 }}
              size="middle"
              onRow={(record) => {
                // 根据条件决定背景色
                let bgColor = '';
                // 特殊设备（有机器用途或其他特殊标记）使用浅黄色背景
                if (record.isSpecial ||
                    (record.group && record.group.trim() !== '') ||
                    (record.featureCount && record.featureCount > 0)) {
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
