import React, { useState, useEffect } from 'react';
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
  Input
} from 'antd';
import {
  SelectOutlined,
  ReloadOutlined,
  CheckCircleOutlined
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
  selectedDevices: Device[];
  loading?: boolean;
  simpleMode?: boolean; // 是否使用简单模式（只显示关键字搜索）
}

const DeviceSelectionDrawer: React.FC<DeviceSelectionDrawerProps> = ({
  visible,
  onClose,
  onSelectDevices,
  filterGroups = [],
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

  // 初始化本地选中设备
  useEffect(() => {
    if (visible) {
      setLocalSelectedDevices(selectedDevices);
      setSelectedRowKeys(selectedDevices.map(device => device.id));
    }
  }, [visible, selectedDevices]);

  // 查询设备
  const fetchDevices = async (page = 1, pageSize = 10) => {
    try {
      setLoading(true);

      let response;

      if (simpleMode) {
        // 简单模式：使用关键字搜索
        // 构建基本查询条件
        const queryGroups: FilterGroup[] = [];

        if (keyword.trim()) {
          // 如果有关键字，创建一个基本的查询组
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

        // 处理筛选条件中的数组值
        const processedGroups = clonedGroups.map((group: FilterGroup) => {
          return {
            ...group,
            blocks: group.blocks.map((block) => {
              const processedBlock = { ...block };
              if (Array.isArray(processedBlock.value)) {
                processedBlock.value = processedBlock.value.join(',');
              }
              return processedBlock;
            })
          };
        });

        // 执行查询
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
  const handleTableChange = (pagination: any) => {
    fetchDevices(pagination.current, pagination.pageSize);
  };

  // 处理行选择变化
  const handleRowSelectionChange = (selectedRowKeys: React.Key[], selectedRows: Device[]) => {
    setSelectedRowKeys(selectedRowKeys);
    setLocalSelectedDevices(selectedRows);
  };

  // 确认选择设备
  const handleConfirmSelection = () => {
    onSelectDevices(localSelectedDevices);
    message.success(`已选择 ${localSelectedDevices.length} 台设备`);
    onClose();
  };

  // 表格列定义
  const columns: ColumnsType<Device> = [
    {
      title: '设备编码',
      dataIndex: 'ciCode',
      key: 'ciCode',
      width: 180,
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
      width: 150,
    },
    {
      title: '机器用途',
      dataIndex: 'group',
      key: 'group',
      width: 150,
      render: (text) => text || '-',
    },
    {
      title: '所属集群',
      dataIndex: 'cluster',
      key: 'cluster',
      width: 150,
      render: (text) => text || '-',
    },
    {
      title: '集群角色',
      dataIndex: 'role',
      key: 'role',
      width: 120,
      render: (text) => text || '-',
    },
    {
      title: 'CPU/内存',
      key: 'resources',
      width: 150,
      render: (_, record) => (
        <span>
          {record.cpu || '-'} CPU / {record.memory || '-'} GB
        </span>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
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
      {simpleMode ? (
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
      ) : (
        <div style={{ marginBottom: 16 }}>
          <Text>根据筛选条件找到的设备列表，请选择需要添加到订单的设备：</Text>
        </div>
      )}

      <Divider style={{ margin: '12px 0' }} />

      {loading || externalLoading ? (
        <div style={{ textAlign: 'center', padding: '40px 0' }}>
          <Spin size="large" tip="正在加载设备数据..." />
        </div>
      ) : devices.length > 0 ? (
        <Table
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
          columns={columns}
          dataSource={devices}
          rowKey="id"
          pagination={{
            ...pagination,
            showTotal: (total) => `共 ${total} 条记录`,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
          }}
          onChange={handleTableChange}
          scroll={{ y: 500 }}
          size="middle"
          onRow={(record) => {
            // 根据条件决定背景色
            let bgColor = '';
            if (record.isSpecial) {
              // 浅黄色背景 - 特殊设备
              bgColor = '#fffbe6';
            } else if (record.cluster && record.cluster.trim() !== '') {
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
      ) : (
        <Empty
          description="暂无设备数据"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      )}
    </Drawer>
  );
};

export default DeviceSelectionDrawer;
