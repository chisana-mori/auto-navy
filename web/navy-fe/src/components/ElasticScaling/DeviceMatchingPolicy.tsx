import React, { useState, useEffect } from 'react';
import {
  Card, Button, Table, Space, Modal, Form, Input, Select, message, Tag, Badge, Tooltip,
  Row, Col, Radio, Alert, Checkbox
} from 'antd';
import {
  PlusOutlined, EditOutlined, DeleteOutlined, ExclamationCircleOutlined,
  CloudUploadOutlined, CloudDownloadOutlined, CheckCircleOutlined, CloseCircleOutlined,
  ToolOutlined, DatabaseOutlined, SearchOutlined, EyeOutlined,
  InfoCircleOutlined, SettingOutlined, FilterOutlined, ClusterOutlined, LinkOutlined
} from '@ant-design/icons';
import './DeviceMatchingPolicy.css';
import { v4 as uuidv4 } from 'uuid';
import type { ColumnsType } from 'antd/es/table';
import {
  FilterType,
  ConditionType,
  LogicalOperator,
  FilterBlock,
  FilterGroup
} from '../../types/deviceQuery';
import {
  getFilterOptions,
  getLabelValues,
  getTaintValues
} from '../../services/deviceQueryService';
import {
  getResourcePoolDeviceMatchingPolicies,
  getResourcePoolDeviceMatchingPolicy,
  createResourcePoolDeviceMatchingPolicy,
  updateResourcePoolDeviceMatchingPolicy,
  updateResourcePoolDeviceMatchingPolicyStatus,
  deleteResourcePoolDeviceMatchingPolicy
} from '../../services/resourcePoolDeviceMatchingPolicyService';
import {
  getQueryTemplates
} from '../../services/queryTemplateService';
import { statsApi } from '../../services/elasticScalingService';

// 扩展Window接口，添加openCreateOrderModal方法
declare global {
  interface Window {
    openCreateOrderModal?: () => void;
  }
}

const { Option } = Select;
const { confirm } = Modal;

// 查询模板类型
interface QueryTemplate {
  id: number;
  name: string;
  description: string;
  groups: FilterGroup[];
}

// 资源池设备匹配策略类型
interface ResourcePoolDeviceMatchingPolicy {
  id?: number;
  name: string;
  description: string;
  resourcePoolType: string;
  actionType: 'pool_entry' | 'pool_exit';
  queryTemplateId: number;
  queryGroups?: FilterGroup[];  // 从查询模板获取，非直接存储字段
  queryTemplate?: QueryTemplate; // 关联的查询模板
  status: 'enabled' | 'disabled';
  additionConds?: string[];     // 额外动态条件，仅入池时有效
  createdBy?: string;
  updatedBy?: string;
  createdAt?: string;
  updatedAt?: string;
}

// 分页响应类型
interface PaginatedResponse<T> {
  list: T[];
  total: number;
  page: number;
  size: number;
}

const DeviceMatchingPolicy: React.FC = () => {
  // 状态管理
  const [loading, setLoading] = useState<boolean>(false);
  const [policies, setPolicies] = useState<PaginatedResponse<ResourcePoolDeviceMatchingPolicy> | null>(null);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [createModalVisible, setCreateModalVisible] = useState<boolean>(false);
  const [editModalVisible, setEditModalVisible] = useState<boolean>(false);
  const [currentPolicy, setCurrentPolicy] = useState<ResourcePoolDeviceMatchingPolicy | null>(null);
  const [filterOptions, setFilterOptions] = useState<Record<string, any>>({});
  const [labelValues, setLabelValues] = useState<Record<string, any>>({});
  const [taintValues, setTaintValues] = useState<Record<string, any>>({});
  const [loadingValues, setLoadingValues] = useState<boolean>(false);
  const [queryTemplates, setQueryTemplates] = useState<QueryTemplate[]>([]);
  const [loadingTemplates, setLoadingTemplates] = useState<boolean>(false);
  const [selectedCreateTemplate, setSelectedCreateTemplate] = useState<QueryTemplate | null>(null);
  const [selectedEditTemplate, setSelectedEditTemplate] = useState<QueryTemplate | null>(null);

  // 表单实例
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  // 资源池类型选项 - 从后端获取，不使用硬编码数据
  const [resourcePoolTypeOptions, setResourcePoolTypeOptions] = useState<{ label: string; value: string }[]>([]);

  // 初始化
  useEffect(() => {
    fetchPolicies();
    fetchFilterOptions();
    fetchQueryTemplates();
    fetchResourcePoolTypes();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 获取资源池类型
  const fetchResourcePoolTypes = async () => {
    try {
      // 从后端API获取资源池类型
      const resourceTypes = await statsApi.getResourcePoolTypes();

      // 将API返回的资源池类型转换为前端需要的格式
      const poolTypes = resourceTypes.map((type: string) => {
        // 根据资源池类型生成对应的名称
        let label = '未知资源池';
        if (type === 'total') label = '全局资源';
        else if (type === 'total_intel') label = 'Intel资源池';
        else if (type === 'total_arm') label = 'ARM资源池';
        else if (type === 'total_hg') label = '高性能资源池';
        else if (type === 'total_gpu') label = 'GPU资源池';
        else if (type === 'total_taint') label = '特殊节点资源池';
        else if (type === 'total_common') label = '通用资源池';
        else if (type === 'compute') label = '计算资源池';
        else if (type === 'storage') label = '存储资源池';
        else if (type === 'network') label = '网络资源池';
        else if (type === 'gpu') label = 'GPU资源池';
        else if (type === 'memory') label = '内存资源池';
        else label = `${type}资源池`;

        return { label, value: type };
      });

      console.log('DeviceMatchingPolicy - 获取到资源池类型:', poolTypes);
      setResourcePoolTypeOptions(poolTypes);
    } catch (error) {
      console.error('获取资源池类型失败:', error);
      // 出错时保留默认值，不更新
    }
  };

  // 监听表单值变化，用于更新预览按钮状态
  useEffect(() => {
    const createFormValues = form.getFieldsValue();
    const editFormValues = editForm.getFieldsValue();

    // 强制重新渲染以更新预览按钮状态
    if (createFormValues.queryTemplateId || editFormValues.queryTemplateId) {
      setLoading(prev => !prev);
      setLoading(prev => !prev);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [createModalVisible, editModalVisible, form, editForm]);

  // 获取查询模板列表
  const fetchQueryTemplates = async () => {
    setLoadingTemplates(true);
    try {
      const response = await getQueryTemplates(1, 100); // 获取足够多的模板
      console.log('查询模板响应:', response);
      if (response && response.list) {
        setQueryTemplates(response.list);
      }
    } catch (error) {
      console.error('获取查询模板失败:', error);
      message.error('获取查询模板失败');
    } finally {
      setLoadingTemplates(false);
    }
  };

  // 获取策略列表
  const fetchPolicies = async (page = 1, size = 10) => {
    setLoading(true);
    try {
      const response = await getResourcePoolDeviceMatchingPolicies(page, size);
      setPolicies(response);
      setPagination({
        ...pagination,
        current: response.page,
        pageSize: response.size,
        total: response.total
      });
    } catch (error) {
      console.error('获取资源池设备匹配策略失败:', error);
      message.error('获取资源池设备匹配策略失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取筛选选项
  const fetchFilterOptions = async () => {
    try {
      const options = await getFilterOptions();
      setFilterOptions(options);
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
        setLabelValues(prev => ({ ...prev, [key]: response }));
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
        setTaintValues(prev => ({ ...prev, [key]: response }));
      }
    } catch (error) {
      console.error('获取污点值失败:', error);
      message.error('获取污点值失败');
    } finally {
      setLoadingValues(false);
    }
  };

  // 表格列定义
  const columns: ColumnsType<ResourcePoolDeviceMatchingPolicy> = [
    {
      title: '策略名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      render: (text: string, record: ResourcePoolDeviceMatchingPolicy) => (
        <a href="#!" onClick={(e) => { e.preventDefault(); handleEdit(record.id!); }}>{text}</a>
      ),
    },
    {
      title: '资源池类型',
      dataIndex: 'resourcePoolType',
      key: 'resourcePoolType',
      width: 120,
      render: (text: string) => {
        const option = resourcePoolTypeOptions.find(opt => opt.value === text);
        // 如果是compute类型，显示为空
        const displayText = text === 'compute' ? '' : (option ? option.label : text);
        return (
          <Tag color="blue" style={{ borderStyle: 'dashed' }}>
            #{displayText}
          </Tag>
        );
      },
    },
    {
      title: '查询模板',
      key: 'queryTemplate',
      width: 240,
      render: (_, record) => {
        // 显示关联的查询模板信息
        const templateName = record.queryTemplate?.name || '未知模板';

        // 计算匹配条件的数量（如果有查询组信息）
        const groupCount = record.queryGroups?.length || 0;
        const blockCount = record.queryGroups?.reduce((total, group) => total + (group.blocks?.length || 0), 0) || 0;

        return (
          <Space direction="vertical" size={4} style={{ width: '100%' }}>
            <div style={{ fontSize: 13, display: 'flex', alignItems: 'center' }}>
              <DatabaseOutlined style={{ marginRight: 8, color: '#1890ff' }} />
              <span>{templateName}</span>
            </div>
            {groupCount > 0 && (
              <div style={{ fontSize: 13, display: 'flex', alignItems: 'center' }}>
                <FilterOutlined style={{ marginRight: 8, color: '#722ed1' }} />
                <span>{groupCount} 个筛选组, {blockCount} 个条件</span>
              </div>
            )}
            {record.queryGroups && record.queryGroups.length > 1 && (
              <div style={{ fontSize: 13, display: 'flex', alignItems: 'center' }}>
                <LinkOutlined style={{ marginRight: 8, color: '#52c41a' }} />
                <span>组间关系: {record.queryGroups[0].operator === LogicalOperator.And ?
                  <Tag color="blue" style={{ margin: 0 }}>AND</Tag> :
                  <Tag color="orange" style={{ margin: 0 }}>OR</Tag>}
                </span>
              </div>
            )}
          </Space>
        );
      },
    },
    {
      title: '动作类型',
      dataIndex: 'actionType',
      key: 'actionType',
      width: 90,
      align: 'center' as const,
      render: (text: string) => (
        <Tag color={text === 'pool_entry' ? 'blue' : 'orange'}>
          {text === 'pool_entry' ? <CloudUploadOutlined style={{ marginRight: 4 }} /> : <CloudDownloadOutlined style={{ marginRight: 4 }} />}
          {text === 'pool_entry' ? '入池' : '退池'}
        </Tag>
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
          text={status === 'enabled' ? '启用' : '禁用'}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      align: 'center' as const,
      render: (_, record) => (
        <Space size="middle" className="action-buttons">
          <Tooltip title="编辑" placement="top">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record.id!)}
              className="edit-button"
            />
          </Tooltip>
          <Tooltip title={record.status === 'enabled' ? '禁用' : '启用'} placement="top">
            <Button
              type="text"
              icon={record.status === 'enabled' ? <CloseCircleOutlined /> : <CheckCircleOutlined />}
              danger={record.status === 'enabled'}
              onClick={() => handleToggleStatus(record.id!, record.status)}
              className={record.status === 'enabled' ? "disable-button" : "enable-button"}
            />
          </Tooltip>
          <Tooltip title="删除" placement="top">
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete(record.id!)}
              className="delete-button"
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  // 处理表格分页变化
  const handleTableChange = (pagination: any) => {
    fetchPolicies(pagination.current, pagination.pageSize);
  };

  // 处理创建策略
  const handleCreate = () => {
    form.resetFields();
    setSelectedCreateTemplate(null);
    setCreateModalVisible(true);
  };

  // 处理编辑策略
  const handleEdit = async (id: number) => {
    setLoading(true);
    try {
      const policy = await getResourcePoolDeviceMatchingPolicy(id);
      setCurrentPolicy(policy);

      // 设置表单值
      editForm.setFieldsValue({
        name: policy.name,
        description: policy.description,
        resourcePoolType: policy.resourcePoolType,
        actionType: policy.actionType,
        queryTemplateId: policy.queryTemplateId,
        status: policy.status,
        additionConds: policy.additionConds || [],
      });

      // 查找并设置选中的模板
      if (policy.queryTemplateId) {
        const template = queryTemplates.find(t => t.id === policy.queryTemplateId);
        setSelectedEditTemplate(template || null);
      }

      setEditModalVisible(true);
    } catch (error) {
      console.error('获取策略详情失败:', error);
      message.error('获取策略详情失败');
    } finally {
      setLoading(false);
    }
  };

  // 处理预览模板
  const handlePreviewTemplate = (templateId: number) => {
    // 打开新标签页跳转到设备中心的高级查询页面，并携带模板ID参数
    // 移除时间戳参数，避免导致页面重新加载产生无限请求
    window.open(`/device?tab=advanced&templateId=${templateId}`, '_blank');
  };

  // 处理删除策略
  const handleDelete = (id: number) => {
    confirm({
      title: '确认删除',
      icon: <ExclamationCircleOutlined />,
      content: '确定要删除此策略吗？此操作无法撤销。',
      onOk: async () => {
        setLoading(true);
        try {
          await deleteResourcePoolDeviceMatchingPolicy(id);
          message.success('删除成功');
          fetchPolicies(pagination.current, pagination.pageSize);
        } catch (error) {
          console.error('删除策略失败:', error);
          message.error('删除策略失败');
        } finally {
          setLoading(false);
        }
      },
    });
  };

  // 处理切换状态
  const handleToggleStatus = (id: number, currentStatus: string) => {
    const newStatus = currentStatus === 'enabled' ? 'disabled' : 'enabled';
    confirm({
      title: `确认${newStatus === 'enabled' ? '启用' : '禁用'}策略`,
      icon: <ExclamationCircleOutlined />,
      content: `确定要${newStatus === 'enabled' ? '启用' : '禁用'}此策略吗？`,
      onOk: async () => {
        setLoading(true);
        try {
          await updateResourcePoolDeviceMatchingPolicyStatus(id, newStatus);
          message.success(`${newStatus === 'enabled' ? '启用' : '禁用'}成功`);
          fetchPolicies(pagination.current, pagination.pageSize);
        } catch (error) {
          console.error('更新策略状态失败:', error);
          message.error('更新策略状态失败');
        } finally {
          setLoading(false);
        }
      },
    });
  };

  // 提交创建表单
  const handleCreateSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      const policy: ResourcePoolDeviceMatchingPolicy = {
        name: values.name,
        description: values.description || '',
        resourcePoolType: values.resourcePoolType,
        actionType: values.actionType,
        queryTemplateId: values.queryTemplateId,
        status: values.status,
      };

      // 如果是入池操作，添加额外动态条件
      if (values.actionType === 'pool_entry' && values.additionConds) {
        policy.additionConds = values.additionConds;
      }

      await createResourcePoolDeviceMatchingPolicy(policy);
      message.success('创建成功');
      setCreateModalVisible(false);
      fetchPolicies(pagination.current, pagination.pageSize);
    } catch (error) {
      console.error('创建策略失败:', error);
      message.error('创建策略失败');
    } finally {
      setLoading(false);
    }
  };

  // 提交编辑表单
  const handleEditSubmit = async () => {
    try {
      const values = await editForm.validateFields();
      setLoading(true);

      const policy: ResourcePoolDeviceMatchingPolicy = {
        id: currentPolicy!.id,
        name: values.name,
        description: values.description || '',
        resourcePoolType: values.resourcePoolType,
        actionType: values.actionType,
        queryTemplateId: values.queryTemplateId,
        status: values.status,
      };

      // 如果是入池操作，添加额外动态条件
      if (values.actionType === 'pool_entry' && values.additionConds) {
        policy.additionConds = values.additionConds;
      }

      await updateResourcePoolDeviceMatchingPolicy(policy);
      message.success('更新成功');
      setEditModalVisible(false);
      fetchPolicies(pagination.current, pagination.pageSize);
    } catch (error) {
      console.error('更新策略失败:', error);
      message.error('更新策略失败');
    } finally {
      setLoading(false);
    }
  };

  // 以下函数在使用查询模板后不再需要，但为了保持代码完整性，保留它们的声明
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const addFilterGroup = (formInstance: any) => {
    const queryGroups = formInstance.getFieldValue('queryGroups') || [];
    const newGroup: FilterGroup = {
      id: uuidv4(),
      blocks: [],
      operator: LogicalOperator.And,
    };
    formInstance.setFieldsValue({
      queryGroups: [...queryGroups, newGroup],
    });
  };

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const removeFilterGroup = (formInstance: any, groupId: string) => {
    const queryGroups = formInstance.getFieldValue('queryGroups') || [];
    formInstance.setFieldsValue({
      queryGroups: queryGroups.filter((group: FilterGroup) => group.id !== groupId),
    });
  };

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const addFilterBlock = (formInstance: any, groupId: string, type: FilterType) => {
    const queryGroups = formInstance.getFieldValue('queryGroups') || [];

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

    const updatedGroups = queryGroups.map((group: FilterGroup) => {
      if (group.id === groupId) {
        return {
          ...group,
          blocks: [...(group.blocks || []), newBlock],
        };
      }
      return group;
    });

    formInstance.setFieldsValue({
      queryGroups: updatedGroups,
    });
  };

  // 更新筛选块
  const updateFilterBlock = (formInstance: any, groupId: string, blockId: string, updatedBlock: Partial<FilterBlock>) => {
    const queryGroups = formInstance.getFieldValue('queryGroups') || [];

    const updatedGroups = queryGroups.map((group: FilterGroup) => {
      if (group.id === groupId) {
        return {
          ...group,
          blocks: (group.blocks || []).map(block =>
            block.id === blockId ? { ...block, ...updatedBlock } : block
          ),
        };
      }
      return group;
    });

    formInstance.setFieldsValue({
      queryGroups: updatedGroups,
    });
  };

  // 删除筛选块
  const removeFilterBlock = (formInstance: any, groupId: string, blockId: string) => {
    const queryGroups = formInstance.getFieldValue('queryGroups') || [];

    const updatedGroups = queryGroups.map((group: FilterGroup) => {
      if (group.id === groupId) {
        return {
          ...group,
          blocks: (group.blocks || []).filter(block => block.id !== blockId),
        };
      }
      return group;
    });

    formInstance.setFieldsValue({
      queryGroups: updatedGroups,
    });
  };

  // 渲染筛选块 - 不再使用，但保留代码以供参考
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const renderFilterBlock = (formInstance: any, block: FilterBlock, groupId: string) => {
    const getBlockTitle = () => {
      switch (block.type) {
        case FilterType.NodeLabel:
          return '节点标签筛选';
        case FilterType.Taint:
          return '节点污点筛选';
        case FilterType.Device:
          return '设备属性筛选';
        default:
          return '筛选条件';
      }
    };

    const getBlockIcon = () => {
      switch (block.type) {
        case FilterType.NodeLabel:
          return <DatabaseOutlined style={{ marginRight: 8, color: '#52c41a' }} />;
        case FilterType.Taint:
          return <ToolOutlined style={{ marginRight: 8, color: '#faad14' }} />;
        case FilterType.Device:
          return <SearchOutlined style={{ marginRight: 8, color: '#1890ff' }} />;
        default:
          return null;
      }
    };

    return (
      <div key={block.id} className="filter-block">
        <div className="filter-block-header">
          <div className="filter-block-type">
            {getBlockIcon()}
            {getBlockTitle()}
          </div>
          <Button
            type="text"
            icon={<DeleteOutlined />}
            onClick={() => removeFilterBlock(formInstance, groupId, block.id)}
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
                updateFilterBlock(formInstance, groupId, block.id, { key: value, value: undefined });

                // 获取对应的值选项
                if (block.type === FilterType.NodeLabel) {
                  fetchLabelValues(value as string);
                } else if (block.type === FilterType.Taint) {
                  fetchTaintValues(value as string);
                }
              }}
              style={{ width: 200, marginRight: 8 }}
              loading={loadingValues}
              allowClear
              showSearch
              optionFilterProp="children"
              filterOption={(input, option) =>
                (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
              }
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
                updateFilterBlock(formInstance, groupId, block.id, { key: value, value: undefined });
              }}
              style={{ width: 200, marginRight: 8 }}
              loading={loadingValues}
              allowClear
              showSearch
              optionFilterProp="children"
              filterOption={(input, option) =>
                (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
              }
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
                updateFilterBlock(formInstance, groupId, block.id, { conditionType: value, value: block.value[0] });
              } else {
                updateFilterBlock(formInstance, groupId, block.id, { conditionType: value });
              }
            }}
            style={{ width: 120, marginRight: 8 }}
            showSearch
            optionFilterProp="children"
            filterOption={(input, option) =>
              (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
            }
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
                    updateFilterBlock(formInstance, groupId, block.id, { value, conditionType: ConditionType.In });
                    return;
                  }
                }
                updateFilterBlock(formInstance, groupId, block.id, { value });
              }}
              style={{ width: 200 }}
              mode={(block.type !== FilterType.Device || block.conditionType === ConditionType.In || block.conditionType === ConditionType.NotIn) ? 'multiple' : undefined}
              loading={loadingValues}
              allowClear
              showSearch
              optionFilterProp="children"
              filterOption={(input, option) =>
                (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
              }
              maxTagCount={3}
              maxTagTextLength={10}
            >
              {block.type === FilterType.NodeLabel && block.key && labelValues[block.key]?.map((option: any) => (
                <Option key={option.value} value={option.value}>{option.label}</Option>
              ))}
              {block.type === FilterType.Taint && block.key && taintValues[block.key]?.map((option: any) => (
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
            onChange={(value) => updateFilterBlock(formInstance, groupId, block.id, { operator: value })}
            style={{ width: 80 }}
            showSearch
            optionFilterProp="children"
            filterOption={(input, option) =>
              (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
            }
          >
            <Option value={LogicalOperator.And}>
              <Tag color="blue" style={{ margin: 0 }}>AND</Tag>
            </Option>
            <Option value={LogicalOperator.Or}>
              <Tag color="orange" style={{ margin: 0 }}>OR</Tag>
            </Option>
          </Select>
        </div>
      </div>
    );
  };

  // 这些注释已不再需要，因为我们已经添加了eslint-disable注释

  return (
    <div className="device-matching-policy-container">
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <ToolOutlined style={{ marginRight: 8, color: '#1890ff' }} />
            <span>设备匹配策略</span>
          </div>
        }
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={handleCreate}
          >
            创建策略
          </Button>
        }
        bordered={false}
        className="policy-card"
      >
        <Table
          columns={columns}
          dataSource={policies?.list || []}
          rowKey="id"
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条记录`,
          }}
          onChange={handleTableChange}
          className="policy-table"
          size="middle"
          bordered
        />
      </Card>

      {/* 创建策略模态框 */}
      <Modal
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <PlusOutlined style={{ marginRight: 8, color: '#1890ff' }} />
            <span>创建设备匹配策略</span>
          </div>
        }
        open={createModalVisible}
        onOk={handleCreateSubmit}
        onCancel={() => setCreateModalVisible(false)}
        width={800}
        okText="确定"
        cancelText="取消"
        confirmLoading={loading}
        destroyOnClose
        className="policy-modal"
      >
        <Alert
          message="设备匹配策略用于定义资源池的设备入池和退池条件"
          description="您可以通过设置不同的筛选条件来匹配符合要求的设备，系统将根据策略自动执行入池或退池操作。"
          type="info"
          showIcon
          style={{ marginBottom: 24 }}
        />

        <Form
          form={form}
          layout="vertical"
          initialValues={{
            status: 'disabled',
            actionType: 'pool_entry',
          }}
          className="policy-form"
        >
          <Card
            title={
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <InfoCircleOutlined style={{ marginRight: 8, color: '#1890ff' }} />
                <span style={{ fontSize: '14px', fontWeight: 500 }}>基本信息</span>
              </div>
            }
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Form.Item
              name="name"
              label="策略名称"
              rules={[{ required: true, message: '请输入策略名称' }]}
            >
              <Input placeholder="请输入策略名称" />
            </Form.Item>

            <Form.Item
              name="description"
              label="策略描述"
            >
              <Input.TextArea placeholder="请输入策略描述" rows={2} />
            </Form.Item>
          </Card>

          <Card
            title={
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <SettingOutlined style={{ marginRight: 8, color: '#52c41a' }} />
                <span style={{ fontSize: '14px', fontWeight: 500 }}>策略配置</span>
              </div>
            }
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="resourcePoolType"
                  label={<span style={{ fontWeight: 500 }}>资源池类型</span>}
                  rules={[{ required: true, message: '请选择资源池类型' }]}
                >
                  <Select
                    placeholder="请选择资源池类型"
                    showSearch
                    optionFilterProp="children"
                    filterOption={(input, option) =>
                      (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
                    }
                  >
                    {resourcePoolTypeOptions.map(option => (
                      <Option key={option.value} value={option.value}>
                        <ClusterOutlined style={{ marginRight: 4, color: '#1890ff' }} />
                        {option.value === 'compute' ? '' : option.value}
                      </Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="actionType"
                  label={<span style={{ fontWeight: 500 }}>动作类型</span>}
                  rules={[{ required: true, message: '请选择动作类型' }]}
                >
                  <Select
                    placeholder="请选择动作类型"
                    showSearch
                    optionFilterProp="children"
                    filterOption={(input, option) =>
                      (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
                    }
                  >
                    <Option value="pool_entry">
                      <CloudUploadOutlined style={{ color: '#1890ff', marginRight: 4 }} /> 入池
                    </Option>
                    <Option value="pool_exit">
                      <CloudDownloadOutlined style={{ color: '#ff7a45', marginRight: 4 }} /> 退池
                    </Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              name="status"
              label={<span style={{ fontWeight: 500 }}>策略状态</span>}
              rules={[{ required: true, message: '请选择状态' }]}
            >
              <Radio.Group buttonStyle="solid">
                <Radio.Button value="enabled">
                  <CheckCircleOutlined style={{ marginRight: 4 }} /> 启用
                </Radio.Button>
                <Radio.Button value="disabled">
                  <CloseCircleOutlined style={{ marginRight: 4 }} /> 禁用
                </Radio.Button>
              </Radio.Group>
            </Form.Item>
          </Card>

          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) => prevValues.actionType !== currentValues.actionType}
          >
            {({ getFieldValue }) => {
              const actionType = getFieldValue('actionType');
              return actionType === 'pool_entry' ? (
                <Card
                  title={
                    <div style={{ display: 'flex', alignItems: 'center' }}>
                      <FilterOutlined style={{ marginRight: 8, color: '#1890ff' }} />
                      <span style={{ fontSize: '14px', fontWeight: 500 }}>额外动态条件</span>
                    </div>
                  }
                  size="small"
                  style={{ marginBottom: '24px' }}
                  headStyle={{ backgroundColor: '#f5f7fa' }}
                  bodyStyle={{ padding: '16px 24px' }}
                >
                  <Alert
                    message="这些条件将在入池时自动添加到查询条件中"
                    description="选中的条件将确保设备与目标集群的位置信息匹配"
                    type="info"
                    showIcon
                    style={{ marginBottom: 16 }}
                  />
                  <Form.Item name="additionConds" initialValue={['idc', 'zone', 'room']}>
                    <Checkbox.Group style={{ width: '100%' }}>
                      <Row>
                        <Col span={8}>
                          <Checkbox value="idc">目标集群同IDC</Checkbox>
                        </Col>
                        <Col span={8}>
                          <Checkbox value="zone">目标集群同安全域</Checkbox>
                        </Col>
                        <Col span={8}>
                          <Checkbox value="room">目标集群同Room</Checkbox>
                        </Col>
                      </Row>
                    </Checkbox.Group>
                  </Form.Item>
                </Card>
              ) : null;
            }}
          </Form.Item>

          <Card
            title={
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <FilterOutlined style={{ marginRight: 8, color: '#722ed1' }} />
                <span style={{ fontSize: '14px', fontWeight: 500 }}>设备匹配条件</span>
              </div>
            }
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Alert
              message="提示：请选择一个查询模板作为设备匹配条件"
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <Form.Item
              name="queryTemplateId"
              label="查询模板"
              rules={[{ required: true, message: '请选择查询模板' }]}
            >
              <Select
                placeholder="请选择查询模板"
                loading={loadingTemplates}
                style={{ width: '100%' }}
                optionFilterProp="children"
                showSearch
                onChange={(value) => {
                  // 当选择模板时，更新表单值
                  form.setFieldsValue({ queryTemplateId: value });
                  // 查找并设置选中的模板
                  const template = queryTemplates.find(t => t.id === value);
                  setSelectedCreateTemplate(template || null);
                }}
              >
                {queryTemplates.map(template => (
                  <Option key={template.id} value={template.id}>
                    <Space>
                      <DatabaseOutlined style={{ color: '#1890ff' }} />
                      <span>{template.name}</span>
                    </Space>
                  </Option>
                ))}
              </Select>
            </Form.Item>

            {selectedCreateTemplate && (
              <div className="template-preview-card">
                <div className="template-info">
                  <CheckCircleOutlined className="success-icon" />
                  <div className="template-details">
                    <div className="template-name">当前选择的模板: {selectedCreateTemplate.name}</div>
                    <div className="template-description">{selectedCreateTemplate.description || '无描述'}</div>
                  </div>
                </div>
                <Button
                  type="primary"
                  icon={<EyeOutlined />}
                  className="preview-button"
                  onClick={() => handlePreviewTemplate(selectedCreateTemplate.id)}
                >
                  预览
                </Button>
              </div>
            )}
          </Card>
        </Form>
      </Modal>

      {/* 编辑策略模态框 */}
      <Modal
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <EditOutlined style={{ marginRight: 8, color: '#1890ff' }} />
            <span>编辑设备匹配策略</span>
          </div>
        }
        open={editModalVisible}
        onOk={handleEditSubmit}
        onCancel={() => setEditModalVisible(false)}
        width={800}
        okText="确定"
        cancelText="取消"
        confirmLoading={loading}
        destroyOnClose
        className="policy-modal"
      >
        <Form
          form={editForm}
          layout="vertical"
          className="policy-form"
        >
          <Card
            title={
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <InfoCircleOutlined style={{ marginRight: 8, color: '#1890ff' }} />
                <span style={{ fontSize: '14px', fontWeight: 500 }}>基本信息</span>
              </div>
            }
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Form.Item
              name="name"
              label="策略名称"
              rules={[{ required: true, message: '请输入策略名称' }]}
            >
              <Input placeholder="请输入策略名称" />
            </Form.Item>

            <Form.Item
              name="description"
              label="策略描述"
            >
              <Input.TextArea placeholder="请输入策略描述" rows={2} />
            </Form.Item>
          </Card>

          <Card
            title={
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <SettingOutlined style={{ marginRight: 8, color: '#52c41a' }} />
                <span style={{ fontSize: '14px', fontWeight: 500 }}>策略配置</span>
              </div>
            }
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item
                  name="resourcePoolType"
                  label={<span style={{ fontWeight: 500 }}>资源池类型</span>}
                  rules={[{ required: true, message: '请选择资源池类型' }]}
                >
                  <Select
                    placeholder="请选择资源池类型"
                    showSearch
                    optionFilterProp="children"
                    filterOption={(input, option) =>
                      (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
                    }
                  >
                    {resourcePoolTypeOptions.map(option => (
                      <Option key={option.value} value={option.value}>
                        <ClusterOutlined style={{ marginRight: 4, color: '#1890ff' }} />
                        {option.value === 'compute' ? '' : option.value}
                      </Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="actionType"
                  label={<span style={{ fontWeight: 500 }}>动作类型</span>}
                  rules={[{ required: true, message: '请选择动作类型' }]}
                >
                  <Select
                    placeholder="请选择动作类型"
                    showSearch
                    optionFilterProp="children"
                    filterOption={(input, option) =>
                      (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
                    }
                  >
                    <Option value="pool_entry">
                      <CloudUploadOutlined style={{ color: '#1890ff', marginRight: 4 }} /> 入池
                    </Option>
                    <Option value="pool_exit">
                      <CloudDownloadOutlined style={{ color: '#ff7a45', marginRight: 4 }} /> 退池
                    </Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              name="status"
              label={<span style={{ fontWeight: 500 }}>策略状态</span>}
              rules={[{ required: true, message: '请选择状态' }]}
            >
              <Radio.Group buttonStyle="solid">
                <Radio.Button value="enabled">
                  <CheckCircleOutlined style={{ marginRight: 4 }} /> 启用
                </Radio.Button>
                <Radio.Button value="disabled">
                  <CloseCircleOutlined style={{ marginRight: 4 }} /> 禁用
                </Radio.Button>
              </Radio.Group>
            </Form.Item>
          </Card>

          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) => prevValues.actionType !== currentValues.actionType}
          >
            {({ getFieldValue }) => {
              const actionType = getFieldValue('actionType');
              return actionType === 'pool_entry' ? (
                <Card
                  title={
                    <div style={{ display: 'flex', alignItems: 'center' }}>
                      <FilterOutlined style={{ marginRight: 8, color: '#1890ff' }} />
                      <span style={{ fontSize: '14px', fontWeight: 500 }}>额外动态条件</span>
                    </div>
                  }
                  size="small"
                  style={{ marginBottom: '24px' }}
                  headStyle={{ backgroundColor: '#f5f7fa' }}
                  bodyStyle={{ padding: '16px 24px' }}
                >
                  <Alert
                    message="这些条件将在入池时自动添加到查询条件中"
                    description="选中的条件将确保设备与目标集群的位置信息匹配"
                    type="info"
                    showIcon
                    style={{ marginBottom: 16 }}
                  />
                  <Form.Item name="additionConds" initialValue={['idc', 'zone', 'room']}>
                    <Checkbox.Group style={{ width: '100%' }}>
                      <Row>
                        <Col span={8}>
                          <Checkbox value="idc">目标集群同IDC</Checkbox>
                        </Col>
                        <Col span={8}>
                          <Checkbox value="zone">目标集群同安全域</Checkbox>
                        </Col>
                        <Col span={8}>
                          <Checkbox value="room">目标集群同Room</Checkbox>
                        </Col>
                      </Row>
                    </Checkbox.Group>
                  </Form.Item>
                </Card>
              ) : null;
            }}
          </Form.Item>

          <Card
            title={
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <FilterOutlined style={{ marginRight: 8, color: '#722ed1' }} />
                <span style={{ fontSize: '14px', fontWeight: 500 }}>设备匹配条件</span>
              </div>
            }
            size="small"
            style={{ marginBottom: '24px' }}
            headStyle={{ backgroundColor: '#f5f7fa' }}
            bodyStyle={{ padding: '16px 24px' }}
          >
            <Alert
              message="提示：请选择一个查询模板作为设备匹配条件"
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <Form.Item
              name="queryTemplateId"
              label="查询模板"
              rules={[{ required: true, message: '请选择查询模板' }]}
            >
              <Select
                placeholder="请选择查询模板"
                loading={loadingTemplates}
                style={{ width: '100%' }}
                optionFilterProp="children"
                showSearch
                onChange={(value) => {
                  // 当选择模板时，更新表单值
                  editForm.setFieldsValue({ queryTemplateId: value });
                  // 查找并设置选中的模板
                  const template = queryTemplates.find(t => t.id === value);
                  setSelectedEditTemplate(template || null);
                }}
              >
                {queryTemplates.map(template => (
                  <Option key={template.id} value={template.id}>
                    <Space>
                      <DatabaseOutlined style={{ color: '#1890ff' }} />
                      <span>{template.name}</span>
                    </Space>
                  </Option>
                ))}
              </Select>
            </Form.Item>

            {selectedEditTemplate && (
              <div className="template-preview-card">
                <div className="template-info">
                  <CheckCircleOutlined className="success-icon" />
                  <div className="template-details">
                    <div className="template-name">当前选择的模板: {selectedEditTemplate.name}</div>
                    <div className="template-description">{selectedEditTemplate.description || '无描述'}</div>
                  </div>
                </div>
                <Button
                  type="primary"
                  icon={<EyeOutlined />}
                  className="preview-button"
                  onClick={() => handlePreviewTemplate(selectedEditTemplate.id)}
                >
                  预览
                </Button>
              </div>
            )}
          </Card>
        </Form>
      </Modal>
    </div>
  );
};

export default DeviceMatchingPolicy;