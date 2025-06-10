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
import { queryDevices, getDeviceFeatureDetails } from '../../services/deviceQueryService';
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
                  onMouseEnter: async (e) => {
                    // 如果是特殊设备或有应用名称，显示提示框
                    if (record.isSpecial || (record.appName && record.appName.trim() !== '')) {
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
                        (e.currentTarget as any).tooltip = tooltip;

                        // 创建一个引用副本，避免使用 e.currentTarget
                        const tooltipElement = tooltip;
                        const rowElement = e.currentTarget;

                        // 添加鼠标移动事件，使提示框跟随鼠标
                        const handleMouseMove = (moveEvent: MouseEvent) => {
                          // 检查元素和提示框是否仍然存在
                          if (rowElement && (rowElement as any).tooltip && tooltipElement && document.body.contains(tooltipElement)) {
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
                        (e.currentTarget as any).handleMouseMove = handleMouseMove;
                        document.addEventListener('mousemove', handleMouseMove);

                        // 构建特性详情数组
                        const featureDetails: string[] = [];

                        // 添加机器用途
                        if (record.group && record.group.trim() !== '') {
                          featureDetails.push(`机器用途: ${record.group}`);
                        }

                        // 添加应用名称
                        if (record.appName && record.appName.trim() !== '') {
                          featureDetails.push(`应用名称: ${record.appName}`);
                        }

                        // 如果有标签特性或污点特性，获取详情
                        if (record.featureCount && record.featureCount > (record.group ? 1 : 0)) {
                          // 获取设备特性详情
                          const details = await getDeviceFeatureDetails(record.ciCode);

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
                      } catch (error) {
                        console.error('获取设备特性详情失败:', error);
                      }
                    }
                  },
                  onMouseLeave: (e) => {
                    try {
                      // 安全地移除提示框
                      const tooltip = (e.currentTarget as any)?.tooltip;
                      if (tooltip && document.body.contains(tooltip)) {
                        document.body.removeChild(tooltip);
                      }

                      // 清除引用
                      if (e.currentTarget) {
                        (e.currentTarget as any).tooltip = null;
                      }

                      // 安全地移除鼠标移动事件监听器
                      const handleMouseMove = (e.currentTarget as any)?.handleMouseMove;
                      if (handleMouseMove) {
                        document.removeEventListener('mousemove', handleMouseMove);

                        // 清除引用
                        if (e.currentTarget) {
                          (e.currentTarget as any).handleMouseMove = null;
                        }
                      }
                    } catch (error) {
                      console.error('清除提示框资源失败:', error);
                    }
                  }
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
