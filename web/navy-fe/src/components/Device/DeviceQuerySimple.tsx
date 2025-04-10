import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Card,
  Button,
  Table,
  message,
  Tabs,
  Modal,
  Form,
  Input,
  Select,
  Space,
  Divider,
  Typography
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  DatabaseOutlined,
  PlusOutlined,
  SaveOutlined,
  SearchOutlined,
  DeleteOutlined,
  EditOutlined,
  PlayCircleOutlined,
  ToolOutlined
} from '@ant-design/icons';
import { v4 as uuidv4 } from 'uuid';
import { Device } from '../../types/device';
import {
  FilterType,
  ConditionType,
  LogicalOperator,
  FilterBlock,
  FilterGroup,
  QueryTemplate,
  FilterOption
} from '../../types/deviceQuery';
import {
  getFilterOptions,
  queryDevices,
  saveQueryTemplate,
  getQueryTemplates,
  getQueryTemplate,
  deleteQueryTemplate,
  getLabelValues,
  getTaintValues
} from '../../services/deviceQueryService';
import '../../styles/device-query.css';

const { Text } = Typography;
const { TabPane } = Tabs;
const { Option } = Select;

const DeviceQuerySimple: React.FC = () => {
  const navigate = useNavigate();
  // 状态
  const [queryLoading, setQueryLoading] = useState(false);
  const [filterOptions, setFilterOptions] = useState<Record<string, any>>({});
  const [filterGroups, setFilterGroups] = useState<FilterGroup[]>([]);
  const [queryResults, setQueryResults] = useState<Device[]>([]);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [activeTab, setActiveTab] = useState('query');
  const [templates, setTemplates] = useState<QueryTemplate[]>([]);
  const [templateModalVisible, setTemplateModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<QueryTemplate | null>(null);
  const [templateForm] = Form.useForm();
  const [searchKeyword, setSearchKeyword] = useState('');

  // 标签和污点值选项
  const [labelValues, setLabelValues] = useState<Record<string, FilterOption[]>>({});
  const [taintValues, setTaintValues] = useState<Record<string, FilterOption[]>>({});
  const [loadingValues, setLoadingValues] = useState(false);

  // 初始化
  useEffect(() => {
    fetchFilterOptions();
    fetchTemplates();
  }, []);

  // 获取筛选选项
  const fetchFilterOptions = async () => {
    try {
      const options = await getFilterOptions();
      console.log('获取到的筛选选项:', options);
      if (options) {
        setFilterOptions(options);
        // 预加载所有标签和污点的值
        if (options.nodeLabelKeys) {
          for (const key of options.nodeLabelKeys) {
            await fetchLabelValues(key.value);
          }
        }
        if (options.nodeTaintKeys) {
          for (const key of options.nodeTaintKeys) {
            await fetchTaintValues(key.value);
          }
        }
      }
    } catch (error) {
      console.error('获取筛选选项失败:', error);
      message.error('获取筛选选项失败');
    }
  };

  // 获取标签值
  const fetchLabelValues = async (key: string) => {
    if (!key) return;

    try {
      setLoadingValues(true);
      const response = await getLabelValues(key);
      if (Array.isArray(response)) {
        const options = response as unknown as FilterOption[];
        setLabelValues(prev => ({
          ...prev,
          [key]: options
        }));
      }
    } catch (error) {
      console.error('获取标签值失败:', error);
      message.error('获取标签值失败');
    } finally {
      setLoadingValues(false);
    }
  };

  // 获取污点值
  const fetchTaintValues = async (key: string) => {
    if (!key) return;

    try {
      setLoadingValues(true);
      const response = await getTaintValues(key);
      if (Array.isArray(response)) {
        const options = response as unknown as FilterOption[];
        setTaintValues(prev => ({
          ...prev,
          [key]: options
        }));
      }
    } catch (error) {
      console.error('获取污点值失败:', error);
      message.error('获取污点值失败');
    } finally {
      setLoadingValues(false);
    }
  };

  // 获取模板列表
  const fetchTemplates = async () => {
    try {
      const templates = await getQueryTemplates();
      // 确保每个模板都有有效的ID
      const validTemplates = templates.map(template => ({
        ...template,
        id: template.id || 0 // 如果id不存在，设置为0
      }));
      setTemplates(validTemplates);
    } catch (error) {
      console.error('获取查询模板失败:', error);
      message.error('获取查询模板失败');
    }
  };

  // 添加筛选组
  const addFilterGroup = () => {
    const newGroup: FilterGroup = {
      id: uuidv4(),
      blocks: [],
      operator: LogicalOperator.And,
    };
    // 确保filterGroups不为null
    const currentGroups = filterGroups || [];
    setFilterGroups([...currentGroups, newGroup]);
  };

  // 删除筛选组
  const removeFilterGroup = (groupId: string) => {
    // 确保filterGroups不为null
    const currentGroups = filterGroups || [];
    setFilterGroups(currentGroups.filter(group => group.id !== groupId));
  };

  // 更新筛选组
  const updateFilterGroup = (groupId: string, updatedGroup: Partial<FilterGroup>) => {
    // 确保filterGroups不为null
    const currentGroups = filterGroups || [];
    setFilterGroups(
      currentGroups.map(group =>
        group.id === groupId ? { ...group, ...updatedGroup } : group
      )
    );
  };

  // 添加筛选块
  const addFilterBlock = (groupId: string, type: FilterType) => {
    // 对于标签和污点，默认使用In条件，对于设备字段使用Equal条件
    const defaultConditionType = type !== FilterType.Device
      ? ConditionType.In
      : ConditionType.Equal;

    const newBlock: FilterBlock = {
      id: uuidv4(),
      type,
      conditionType: defaultConditionType,
      operator: LogicalOperator.And,
    };

    // 确保filterGroups不为null
    const currentGroups = filterGroups || [];
    setFilterGroups(
      currentGroups.map(group => {
        if (group.id === groupId) {
          return {
            ...group,
            blocks: [...(group.blocks || []), newBlock],
          };
        }
        return group;
      })
    );
  };

  // 更新筛选块
  const updateFilterBlock = (groupId: string, blockId: string, updatedBlock: Partial<FilterBlock>) => {
    // 确保filterGroups不为null
    const currentGroups = filterGroups || [];
    setFilterGroups(
      currentGroups.map(group => {
        if (group.id === groupId) {
          return {
            ...group,
            blocks: (group.blocks || []).map(block =>
              block.id === blockId ? { ...block, ...updatedBlock } : block
            ),
          };
        }
        return group;
      })
    );
  };

  // 删除筛选块
  const removeFilterBlock = (groupId: string, blockId: string) => {
    // 确保filterGroups不为null
    const currentGroups = filterGroups || [];
    setFilterGroups(
      currentGroups.map(group => {
        if (group.id === groupId) {
          return {
            ...group,
            blocks: (group.blocks || []).filter(block => block.id !== blockId),
          };
        }
        return group;
      })
    );
  };

  // 渲染设备字段筛选按钮
  const renderDeviceFieldButton = (group: FilterGroup) => (
    <Button
      type="text"
      icon={<DatabaseOutlined />}
      onClick={() => addFilterBlock(group.id, FilterType.Device)}
    >
      设备字段
    </Button>
  );

  // 执行查询
  const handleQuery = async () => {
    // 确保filterGroups不为null
    const currentGroups = filterGroups || [];
    if (currentGroups.length === 0) {
      message.warning('请添加至少一个筛选条件');
      return;
    }

    // 处理多选值，根据条件类型进行不同的处理
    const processedGroups = currentGroups.map(group => ({
      ...group,
      blocks: group.blocks.map(block => {
        // 如果值是数组
        if (Array.isArray(block.value)) {
          // 如果是In或NotIn条件，则使用逗号分隔的字符串
          if (block.conditionType === ConditionType.In || block.conditionType === ConditionType.NotIn) {
            return {
              ...block,
              value: block.value.join(',')
            };
          } else if (block.value.length > 0) {
            // 如果是其他条件，但值是数组，则取第一个值
            return {
              ...block,
              value: block.value[0]
            };
          }
        }
        return block;
      })
    }));

    try {
      setQueryLoading(true);
      const response = await queryDevices({
        groups: processedGroups,
        page: pagination.current,
        size: pagination.pageSize,
      });

      setQueryResults(response.list);
      setPagination({
        ...pagination,
        total: response.total,
      });

      message.success('查询成功');
    } catch (error) {
      console.error('查询失败:', error);
      message.error('查询失败');
    } finally {
      setQueryLoading(false);
    }
  };

  // 重置查询
  const handleReset = () => {
    setFilterGroups([]);
    setQueryResults([]);
    setPagination({ current: 1, pageSize: 10, total: 0 });
  };

  // 保存模板
  const handleSaveTemplate = () => {
    // 确保filterGroups不为null
    const currentGroups = filterGroups || [];
    if (currentGroups.length === 0) {
      message.warning('请添加至少一个筛选条件');
      return;
    }

    // 重置编辑状态
    setEditingTemplate(null);
    templateForm.resetFields();
    setTemplateModalVisible(true);
  };

  // 编辑模板
  const handleEditTemplate = (template: QueryTemplate) => {
    setEditingTemplate(template);
    templateForm.setFieldsValue({
      name: template.name,
      description: template.description || ''
    });
    // 确保groups不为null
    setFilterGroups(template.groups || []);
    setActiveTab('query');
  };

  // 提交保存模板
  const handleSubmitTemplate = async () => {
    try {
      const values = await templateForm.validateFields();

      // 确保filterGroups不为null
      const currentGroups = filterGroups || [];

      // 处理数组类型的value，将其转换为逗号分隔的字符串
      const processedGroups = currentGroups.map(group => ({
        ...group,
        blocks: group.blocks.map(block => {
          if (Array.isArray(block.value)) {
            return {
              ...block,
              value: block.value.join(',')
            };
          }
          return block;
        })
      }));

      const template: QueryTemplate = {
        id: editingTemplate ? editingTemplate.id : undefined,
        name: values.name,
        description: values.description || '',
        groups: processedGroups,
      };

      await saveQueryTemplate(template);
      message.success(editingTemplate ? '模板更新成功' : '模板保存成功');
      setTemplateModalVisible(false);
      setEditingTemplate(null);
      templateForm.resetFields();
      fetchTemplates();
    } catch (error) {
      console.error('保存模板失败:', error);
      message.error('保存模板失败');
    }
  };

  // 加载模板
  const handleLoadTemplate = async (templateId: number | undefined) => {
    if (templateId === undefined || templateId === 0) {
      message.error('模板ID无效');
      return;
    }
    try {
      const template = await getQueryTemplate(templateId);
      if (template && template.groups) {
        setFilterGroups(template.groups);
        // 切换到查询构建器页面
        setActiveTab('query');
        message.success(`已加载模板「${template.name}」，可以进行编辑或执行查询`);
      } else {
        setFilterGroups([]);
        setActiveTab('query');
        message.warning('模板数据不完整，已初始化为空');
      }
    } catch (error) {
      console.error('加载模板失败:', error);
      message.error('加载模板失败');
    }
  };

  // 执行模板查询
  const handleExecuteTemplate = async (templateId: number | undefined) => {
    if (!templateId) {
      message.error('模板ID无效');
      return;
    }
    try {
      // 加载模板
      const template = await getQueryTemplate(templateId);
      if (!template || !template.groups || template.groups.length === 0) {
        message.warning('模板数据不完整或没有查询条件');
        return;
      }

      // 设置查询条件
      setFilterGroups(template.groups);

      // 切换到查询构建器页面
      setActiveTab('query');

      // 处理多选值，根据条件类型进行不同的处理
      const processedGroups = template.groups.map(group => ({
        ...group,
        blocks: group.blocks.map(block => {
          // 如果值是数组
          if (Array.isArray(block.value)) {
            // 如果是In或NotIn条件，则使用逗号分隔的字符串
            if (block.conditionType === ConditionType.In || block.conditionType === ConditionType.NotIn) {
              return {
                ...block,
                value: block.value.join(',')
              };
            } else if (block.value.length > 0) {
              // 如果是其他条件，但值是数组，则取第一个值
              return {
                ...block,
                value: block.value[0]
              };
            }
          }
          return block;
        })
      }));

      // 执行查询
      setQueryLoading(true);
      const response = await queryDevices({
        groups: processedGroups,
        page: 1, // 重置到第一页
        size: pagination.pageSize,
      });

      // 更新查询结果
      setQueryResults(response.list);
      setPagination({
        ...pagination,
        current: 1, // 重置到第一页
        total: response.total,
      });

      // 显示成功消息
      message.success(`已成功执行模板「${template.name}」的查询，共找到 ${response.total} 条结果`);

      // 如果有结果，自动滚动到结果区域
      if (response.list.length > 0) {
        setTimeout(() => {
          const resultsElement = document.querySelector('.query-results');
          if (resultsElement) {
            resultsElement.scrollIntoView({ behavior: 'smooth', block: 'start' });
          }
        }, 300);
      }
    } catch (error) {
      console.error('执行模板查询失败:', error);
      message.error('执行模板查询失败');
    } finally {
      setQueryLoading(false);
    }
  };

  // 删除模板
  const handleDeleteTemplate = async (templateId: number | undefined) => {
    if (!templateId) {
      message.error('模板ID无效');
      return;
    }
    try {
      await deleteQueryTemplate(templateId);
      message.success('模板删除成功');
      fetchTemplates();
    } catch (error) {
      console.error('删除模板失败:', error);
      message.error('删除模板失败');
    }
  };

  // 渲染筛选块
  const renderFilterBlock = (block: FilterBlock, groupId: string) => {
    const getBlockTitle = () => {
      switch (block.type) {
        case FilterType.NodeLabel:
          return '节点标签筛选';
        case FilterType.Taint:
          return '污点筛选';
        case FilterType.Device:
          return '设备字段筛选';
        default:
          return '筛选';
      }
    };

    console.log('渲染筛选块时的 filterOptions:', filterOptions);
    console.log('当前块类型:', block.type);
    console.log('标签键选项:', filterOptions["labelKeys"]);
    console.log('污点键选项:', filterOptions["taintKeys"]);

    return (
      <div key={block.id} className="filter-block">
        <div className="filter-block-header">
          <div className="filter-block-type">{getBlockTitle()}</div>
          <Button
            type="text"
            icon={<DeleteOutlined />}
            onClick={() => removeFilterBlock(groupId, block.id)}
            danger
            size="small"
          />
        </div>

        <div className="filter-block-content">
          {block.type !== FilterType.Device && (
            <Select
              placeholder="选择键"
              value={block.key}
              onChange={(value) => {
                // 当选择新的key时，清除之前的value
                updateFilterBlock(groupId, block.id, { key: value, value: undefined });

                // 获取对应的值选项
                if (block.type === FilterType.NodeLabel) {
                  fetchLabelValues(value as string);
                } else if (block.type === FilterType.Taint) {
                  fetchTaintValues(value as string);
                }
              }}
              style={{ width: 200, marginRight: 8 }}
            >
              {block.type === FilterType.NodeLabel && filterOptions.nodeLabelKeys?.map((option: any) => (
                <Option key={option.value} value={option.value}>{option.label}</Option>
              ))}
              {block.type === FilterType.Taint && filterOptions.nodeTaintKeys?.map((option: any) => (
                <Option key={option.value} value={option.value}>{option.label}</Option>
              ))}
            </Select>
          )}

          {block.type === FilterType.Device && (
            <Select
              placeholder="选择字段"
              value={block.key}
              onChange={(value) => {
                updateFilterBlock(groupId, block.id, { key: value, value: undefined });
              }}
              style={{ width: 200, marginRight: 8 }}
            >
              {filterOptions.deviceFieldValues?.map((field: any) => (
                <Option key={field.field} value={field.field}>{field.field}</Option>
              ))}
            </Select>
          )}

          <Select
            placeholder="选择条件"
            value={block.conditionType}
            onChange={(value) => {
              // 如果切换到非In/NotIn条件，且当前值是数组，则取第一个值
              if (value !== ConditionType.In && value !== ConditionType.NotIn && Array.isArray(block.value) && block.value.length > 0) {
                updateFilterBlock(groupId, block.id, { conditionType: value, value: block.value[0] });
              } else {
                updateFilterBlock(groupId, block.id, { conditionType: value });
              }
            }}
            style={{ width: 120, marginRight: 8 }}
          >
            <Option value={ConditionType.Equal}>等于</Option>
            <Option value={ConditionType.NotEqual}>不等于</Option>
            <Option value={ConditionType.In}>在列表中</Option>
            <Option value={ConditionType.NotIn}>不在列表中</Option>
            <Option value={ConditionType.Contains}>包含</Option>
            <Option value={ConditionType.NotContains}>不包含</Option>
            {block.type !== FilterType.Device && (
              <>
                <Option value={ConditionType.Exists}>存在</Option>
                <Option value={ConditionType.NotExists}>不存在</Option>
              </>
            )}
          </Select>

          {(block.conditionType !== ConditionType.Exists &&
            block.conditionType !== ConditionType.NotExists) && (
            <Select
              placeholder="选择值"
              value={block.value}
              onChange={(value) => {
                if (Array.isArray(value) && value.length > 1) {
                  if (block.conditionType !== ConditionType.In && block.conditionType !== ConditionType.NotIn) {
                    updateFilterBlock(groupId, block.id, { value, conditionType: ConditionType.In });
                    return;
                  }
                }
                updateFilterBlock(groupId, block.id, { value });
              }}
              style={{ width: 200 }}
              mode={block.type !== FilterType.Device ? 'multiple' : undefined}
              loading={loadingValues}
              showSearch
              optionFilterProp="children"
            >

              {block.type === FilterType.NodeLabel && block.key && labelValues[block.key]?.map((option) => (
                <Option key={option.value} value={option.value}>{option.label}</Option>
              ))}
              {block.type === FilterType.Taint && block.key && taintValues[block.key]?.map((option) => (
                <Option key={option.value} value={option.value}>{option.label}</Option>
              ))}
              {block.type === FilterType.Device && block.key &&
                filterOptions.deviceFieldValues?.find((field: any) => field.field === block.key)?.values.map((option: any) => (
                  <Option key={option.value} value={option.value}>{option.label}</Option>
                ))
              }
            </Select>
          )}
        </div>

        {/* 逻辑运算符 */}
        <div className="filter-block-footer">
          <Select
            value={block.operator}
            onChange={(value) => updateFilterBlock(groupId, block.id, { operator: value })}
            style={{ width: 80 }}
          >
            <Option value={LogicalOperator.And}>AND</Option>
            <Option value={LogicalOperator.Or}>OR</Option>
          </Select>
        </div>
      </div>
    );
  };

  // 表格列定义
  const columns: ColumnsType<Device> = [
    {
      title: '设备ID',
      dataIndex: 'deviceId',
      key: 'deviceId',
      width: 180,
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
      width: 150,
    },
    {
      title: '机器类型',
      dataIndex: 'machineType',
      key: 'machineType',
      width: 150,
    },
    {
      title: '所属集群',
      dataIndex: 'cluster',
      key: 'cluster',
      width: 150,
    },
    {
      title: '集群角色',
      dataIndex: 'role',
      key: 'role',
      width: 120,
    },
    {
      title: '架构',
      dataIndex: 'arch',
      key: 'arch',
      width: 100,
    },
    {
      title: 'IDC',
      dataIndex: 'idc',
      key: 'idc',
      width: 100,
    },
    {
      title: 'Room',
      dataIndex: 'room',
      key: 'room',
      width: 120,
    },
    {
      title: '机房',
      dataIndex: 'datacenter',
      key: 'datacenter',
      width: 120,
    },
    {
      title: '机柜号',
      dataIndex: 'cabinet',
      key: 'cabinet',
      width: 120,
    },
    {
      title: '网络区域',
      dataIndex: 'network',
      key: 'network',
      width: 120,
    },
    {
      title: 'APPID',
      dataIndex: 'appId',
      key: 'appId',
      width: 120,
    },
    {
      title: '资源池',
      dataIndex: 'resourcePool',
      key: 'resourcePool',
      width: 120,
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right' as const,
      width: 120,
      render: (_: unknown, record: Device) => (
        <Space size={8}>
          <Button
            type="link"
            size="small"
            onClick={() => navigate(`/device/${record.id}`)}
          >
            详情
          </Button>
          <Button
            type="link"
            size="small"
            onClick={() => navigate(`/device/${record.id}`)}
          >
            标记
          </Button>
        </Space>
      ),
    },
  ];

  // 处理表格分页变化
  const handleTableChange = (pagination: any) => {
    setPagination({
      ...pagination,
      current: pagination.current,
      pageSize: pagination.pageSize,
    });
  };

  // 添加搜索过滤函数
  const filterTemplates = (templates: QueryTemplate[], keyword: string) => {
    if (!keyword) return templates;
    return templates.filter(template => 
      template.name.toLowerCase().includes(keyword.toLowerCase())
    );
  };

  return (
    <div className="device-query-container">
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <DatabaseOutlined style={{ marginRight: '12px', color: '#1677ff', fontSize: '18px' }} />
            <span>设备查询器</span>
          </div>
        }
      >
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          <TabPane tab="查询构建器" key="query">
            <div className="query-builder">
              <div className="filter-groups">
                {(filterGroups || []).length === 0 ? (
                  <div className="filter-area-empty">
                    <Text type="secondary">点击下方按钮添加筛选组</Text>
                  </div>
                ) : (
                  (filterGroups || []).map((group, groupIndex) => (
                    <div key={group.id} className="filter-group">
                      <div className="filter-group-header">
                        <div className="filter-group-title">筛选组 {groupIndex + 1}</div>
                        <Button
                          type="text"
                          icon={<DeleteOutlined />}
                          onClick={() => removeFilterGroup(group.id)}
                          danger
                        />
                      </div>

                      <div className="filter-blocks">
                        {(group.blocks || []).map(block => renderFilterBlock(block, group.id))}
                      </div>

                      <div className="filter-group-footer">
                        <Space>
                          <Button
                            icon={<PlusOutlined />}
                            onClick={() => addFilterBlock(group.id, FilterType.NodeLabel)}
                          >
                            添加标签筛选
                          </Button>
                          <Button
                            icon={<PlusOutlined />}
                            onClick={() => addFilterBlock(group.id, FilterType.Taint)}
                          >
                            添加污点筛选
                          </Button>
                          <Button
                            icon={<PlusOutlined />}
                            onClick={() => addFilterBlock(group.id, FilterType.Device)}
                          >
                            添加设备字段筛选
                          </Button>
                        </Space>
                      </div>

                      {groupIndex < filterGroups.length - 1 && (
                        <div className="logic-operator">
                          <div className="logic-operator-content">
                            <Select
                              value={group.operator}
                              onChange={(value) => updateFilterGroup(group.id, { operator: value })}
                              style={{ width: 80 }}
                            >
                              <Option value={LogicalOperator.And}>AND</Option>
                              <Option value={LogicalOperator.Or}>OR</Option>
                            </Select>
                          </div>
                        </div>
                      )}
                    </div>
                  ))
                )}
              </div>

              <div className="query-actions">
                <Button
                  icon={<PlusOutlined />}
                  onClick={addFilterGroup}
                >
                  添加筛选组
                </Button>
                <Button onClick={handleReset}>重置</Button>
                <Button onClick={handleSaveTemplate} icon={<SaveOutlined />}>
                  保存为模板
                </Button>
                <Button
                  type="primary"
                  onClick={handleQuery}
                  loading={queryLoading}
                  icon={<SearchOutlined />}
                >
                  执行查询
                </Button>
              </div>
            </div>

            {queryResults.length > 0 && (
              <div className="query-results">
                <Divider orientation="left">查询结果</Divider>
                <Table
                  columns={columns}
                  dataSource={queryResults}
                  rowKey="id"
                  loading={queryLoading}
                  pagination={{
                    ...pagination,
                    showTotal: (total) => `共 ${total} 条记录`,
                    showSizeChanger: true,
                    pageSizeOptions: ['10', '20', '50', '100'],
                    size: 'default',
                    showQuickJumper: true,
                  }}
                  onChange={handleTableChange}
                  scroll={{ x: 1500 }}
                  size="middle"
                />
              </div>
            )}
          </TabPane>

          <TabPane tab="模板管理" key="templates">
            <div className="template-list">
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
                <Input.Search
                  placeholder="搜索模板名称"
                  allowClear
                  value={searchKeyword}
                  onChange={(e) => setSearchKeyword(e.target.value)}
                  style={{ width: 300 }}
                />
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={handleSaveTemplate}
                >
                  创建新模板
                </Button>
              </div>

              {(filterTemplates(templates || [], searchKeyword) || []).length === 0 ? (
                <div style={{ textAlign: 'center', padding: '20px' }}>
                  <Text type="secondary">
                    {searchKeyword ? '没有找到匹配的模板' : '暂无保存的模板'}
                  </Text>
                </div>
              ) : (
                filterTemplates(templates || [], searchKeyword).map(template => (
                  <div key={template.id || template.name} className="template-item">
                    <div className="template-item-info">
                      <div className="template-item-name">{template.name}</div>
                      <div className="template-item-desc">{template.description}</div>
                    </div>
                    <div className="template-item-actions">
                      <Button
                        size="small"
                        type="primary"
                        onClick={() => handleExecuteTemplate(template.id)}
                        icon={<PlayCircleOutlined />}
                      >
                        获取结果
                      </Button>
                      <Button
                        size="small"
                        type="primary"
                        onClick={() => handleLoadTemplate(template.id)}
                        icon={<ToolOutlined />}
                      >
                        加载编辑
                      </Button>
                      <Button
                        size="small"
                        danger
                        onClick={() => handleDeleteTemplate(template.id)}
                        icon={<DeleteOutlined />}
                      >
                        删除
                      </Button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </TabPane>
        </Tabs>
      </Card>

      {/* 保存模板对话框 */}
      <Modal
        title={editingTemplate ? '编辑查询模板' : '保存查询模板'}
        open={templateModalVisible}
        onOk={handleSubmitTemplate}
        onCancel={() => {
          setTemplateModalVisible(false);
          setEditingTemplate(null);
          templateForm.resetFields();
        }}
        okText={editingTemplate ? '更新' : '保存'}
        cancelText="取消"
      >
        <Form form={templateForm} layout="vertical">
          <Form.Item
            name="name"
            label="模板名称"
            rules={[{ required: true, message: '请输入模板名称' }]}
          >
            <Input placeholder="请输入模板名称" />
          </Form.Item>
          <Form.Item
            name="description"
            label="模板描述"
          >
            <Input.TextArea placeholder="请输入模板描述" rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DeviceQuerySimple;
