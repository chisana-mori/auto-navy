import React, { useState, useEffect, useCallback, useRef } from 'react';
import '../../styles/advanced-query.css';
import {
  Button,
  Card,
  Select,
  Space,
  Typography,
  message,
  Tooltip,
  Modal,
  Form,
  Input,
  Badge,
  Tag
} from 'antd';
import {
  PlusOutlined,
  SaveOutlined,
  DeleteOutlined,
  FilterOutlined,
  SearchOutlined,
  CloseCircleOutlined,
  DesktopOutlined,
  TagsOutlined,
  ExclamationCircleOutlined
} from '@ant-design/icons';
// 使用CSS动画替代react-transition-group
import { v4 as uuidv4 } from 'uuid';
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
  saveQueryTemplate,
  getLabelValues,
  getTaintValues,
  getDeviceFieldValues
} from '../../services/deviceQueryService';

const { Text } = Typography;
const { Option } = Select;

// 注释掉未使用的函数
// 转义特殊字符，防止SQL注入
// const escapeValue = (value: string): string => {
//   if (!value) return '';
//   // 转义 % 和 _ 等特殊字符
//   let escapedValue = value.replace(/%/g, '\\%');
//   escapedValue = escapedValue.replace(/_/g, '\\_');
//   return escapedValue;
// };

interface AdvancedQueryPanelProps {
  filterGroups: FilterGroup[];
  onFilterGroupsChange: (groups: FilterGroup[]) => void;
  onQuery: () => void;
  loading: boolean;
  sourceTemplateId?: number; // 模板来源ID，如果是从模板加载的查询条件
  sourceTemplateName?: string; // 模板来源名称
  onTemplateSaved?: () => void; // 保存模板后的回调函数，用于刷新模板列表
  onSwitchToTemplateTab?: () => void; // 切换到模板标签页的回调函数
}

const AdvancedQueryPanel: React.FC<AdvancedQueryPanelProps> = ({
  filterGroups,
  onFilterGroupsChange,
  onQuery,
  loading,
  sourceTemplateId,
  sourceTemplateName,
  onTemplateSaved,
  onSwitchToTemplateTab
}) => {
  // 状态
  const [filterOptions, setFilterOptions] = useState<Record<string, any>>({});
  const [templateModalVisible, setTemplateModalVisible] = useState(false);
  const [saveMode, setSaveMode] = useState<'save' | 'saveAs'>('save'); // 保存模式：保存或另存为
  // 保留状态，因为它在代码中被使用
  const [isConditionModified, setIsConditionModified] = useState(false); // 条件是否已经被修改
  const [templateForm] = Form.useForm();

  // 标签、污点和设备字段值选项
  const [labelValues, setLabelValues] = useState<Record<string, FilterOption[]>>({});
  const [taintValues, setTaintValues] = useState<Record<string, FilterOption[]>>({});
  const [deviceFieldValues, setDeviceFieldValues] = useState<Record<string, FilterOption[]>>({});
  const [loadingValues, setLoadingValues] = useState(false);

  // --- Data Fetching Callbacks ---

  // 获取标签值
  const fetchLabelValues = useCallback(async (key: string) => {
    if (!key) return;
    try {
      setLoadingValues(true);
      const response = await getLabelValues(key);
      if (Array.isArray(response)) {
        const options = response as unknown as FilterOption[];
        setLabelValues(prev => ({ ...prev, [key]: options }));
      }
    } catch (error) {
      console.error('获取标签值失败:', error);
      message.error('获取标签值失败');
    } finally {
      setLoadingValues(false);
    }
  }, [setLoadingValues, setLabelValues]);

  // 获取污点值
  const fetchTaintValues = useCallback(async (key: string) => {
    if (!key) return;
    try {
      setLoadingValues(true);
      const response = await getTaintValues(key);
      if (Array.isArray(response)) {
        const options = response as unknown as FilterOption[];
        setTaintValues(prev => ({ ...prev, [key]: options }));
      }
    } catch (error) {
      console.error('获取污点值失败:', error);
      message.error('获取污点值失败');
    } finally {
      setLoadingValues(false);
    }
  }, [setLoadingValues, setTaintValues]);

  // 获取设备字段值
  const fetchDeviceFieldValues = useCallback(async (field: string) => {
    if (!field) return;
    try {
      setLoadingValues(true);
      const response = await getDeviceFieldValues(field);
      if (Array.isArray(response)) {
        const options = response.map(value => ({
          id: value,
          label: value,
          value: value
        }));
        setDeviceFieldValues(prev => ({ ...prev, [field]: options }));
      }
    } catch (error) {
      console.error('获取设备字段值失败:', error);
      message.error('获取设备字段值失败');
    } finally {
      setLoadingValues(false);
    }
  }, [setLoadingValues, setDeviceFieldValues]);

  // 获取筛选选项
  const fetchFilterOptions = useCallback(async () => {
    try {
      const options = await getFilterOptions();
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
  }, [setFilterOptions, fetchLabelValues, fetchTaintValues]);

  // 初始化
  useEffect(() => {
    fetchFilterOptions();
  }, [fetchFilterOptions]);

  // 当模板 ID 变化时，重置 isConditionModified 状态
  useEffect(() => {
    setIsConditionModified(false);
  }, [sourceTemplateId]);

  // 包装 onFilterGroupsChange 函数，在条件变化时设置 isConditionModified 为 true
  const handleFilterGroupsChange = (groups: FilterGroup[]) => {
    // 如果有模板 ID，设置条件已修改
    if (sourceTemplateId) {
      setIsConditionModified(true);
    }
    onFilterGroupsChange(groups);
  };

  // 添加筛选组
  const addFilterGroup = () => {
    const newGroup: FilterGroup = {
      id: uuidv4(),
      blocks: [],
      operator: LogicalOperator.And,
    };

    // 打印日志，用于调试
    console.log('Adding new filter group with operator:', newGroup.operator);
    console.log('LogicalOperator enum values:', LogicalOperator);

    // 确保filterGroups不为null
    const currentGroups = filterGroups || [];
    handleFilterGroupsChange([...currentGroups, newGroup]);
  };

  // 删除筛选组
  const removeFilterGroup = (groupId: string) => {
    // 确保filterGroups不为null
    const currentGroups = filterGroups || [];
    handleFilterGroupsChange(currentGroups.filter(group => group.id !== groupId));
  };

  // 更新筛选组
  const updateFilterGroup = (groupId: string, updatedGroup: Partial<FilterGroup>) => {
    // 确保filterGroups不为null
    const currentGroups = filterGroups || [];

    // 打印日志，用于调试
    console.log(`Updating filter group ${groupId} with:`, updatedGroup);

    // 检查是否更新了操作符
    const isOperatorUpdated = updatedGroup.operator !== undefined;

    handleFilterGroupsChange(
      currentGroups.map(group => {
        if (group.id === groupId) {
          // 如果更新了操作符，同步更新该组内所有块的操作符
          let updatedBlocks = group.blocks;
          if (isOperatorUpdated && updatedGroup.operator !== undefined) {
            console.log(`Updating all blocks in group ${groupId} to use operator: ${updatedGroup.operator}`);
            updatedBlocks = group.blocks.map(block => ({
              ...block,
              operator: updatedGroup.operator as LogicalOperator
            }));
          }

          const updatedGroupData = {
            ...group,
            ...updatedGroup,
            blocks: updatedBlocks
          };
          console.log(`Group ${groupId} after update:`, updatedGroupData);
          return updatedGroupData;
        }
        return group;
      })
    );
  };

  // 添加筛选块
  const addFilterBlock = (groupId: string, type: FilterType) => {
    // 对所有类型的筛选块，默认使用Equal条件
    const defaultConditionType = ConditionType.Equal;

    // 获取默认字段
    let defaultField = '';
    if (type === FilterType.NodeLabel && filterOptions.nodeLabelKeys?.length > 0) {
      defaultField = filterOptions.nodeLabelKeys[0].value;
    } else if (type === FilterType.Taint && filterOptions.nodeTaintKeys?.length > 0) {
      defaultField = filterOptions.nodeTaintKeys[0].value;
    } else if (type === FilterType.Device && filterOptions.deviceFields?.length > 0) {
      defaultField = filterOptions.deviceFields[0].value;
    }

    // 获取当前组的操作符
    const currentGroups = filterGroups || [];
    const currentGroup = currentGroups.find(g => g.id === groupId);
    const groupOperator = currentGroup?.operator || LogicalOperator.And;

    // 新块的操作符与组的操作符保持一致
    const newBlock: FilterBlock = {
      id: uuidv4(),
      type,
      conditionType: defaultConditionType,
      field: defaultField,
      key: defaultField,  // 确保key和field保持一致
      operator: groupOperator, // 使用组的操作符
    };

    console.log(`Adding new block with operator: ${groupOperator} (from group)`);


    // 如果有默认字段，预加载对应的值
    if (defaultField) {
      if (type === FilterType.NodeLabel) {
        fetchLabelValues(defaultField);
      } else if (type === FilterType.Taint) {
        fetchTaintValues(defaultField);
      } else if (type === FilterType.Device) {
        fetchDeviceFieldValues(defaultField);
      }
    }

    // 使用上面已经声明的currentGroups变量
    handleFilterGroupsChange(
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

    // 如果更新了field字段但没有更新key字段，或者更新了key字段但没有更新field字段
    // 则同步更新另一个字段
    let finalUpdatedBlock = { ...updatedBlock };
    if (updatedBlock.field !== undefined && updatedBlock.key === undefined) {
      finalUpdatedBlock.key = updatedBlock.field;
    } else if (updatedBlock.key !== undefined && updatedBlock.field === undefined) {
      finalUpdatedBlock.field = updatedBlock.key;
    }

    handleFilterGroupsChange(
      currentGroups.map(group => {
        if (group.id === groupId) {
          return {
            ...group,
            blocks: (group.blocks || []).map(block =>
              block.id === blockId ? { ...block, ...finalUpdatedBlock } : block
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
    handleFilterGroupsChange(
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

  // 重置查询
  const handleReset = () => {
    handleFilterGroupsChange([]);
  };

  // 保存模板
  const handleSaveTemplate = () => {
    // 确保filterGroups不为null
    const groups = filterGroups || [];
    if (groups.length === 0) {
      message.warning('请添加至少一个筛选条件');
      return;
    }

    // 如果是从模板加载的，或者条件已经被修改，则提示用户选择保存模式
    if (sourceTemplateId) {
      Modal.confirm({
        title: '保存模板',
        content: `当前查询条件来自模板「${sourceTemplateName || ''}」，请选择保存方式：`,
        okText: '更新原模板',
        cancelText: '另存为新模板',
        onOk: () => {
          // 更新原模板
          setSaveMode('save');
          templateForm.setFieldsValue({
            name: sourceTemplateName || '',
            description: ''
          });
          setTemplateModalVisible(true);
        },
        onCancel: () => {
          // 另存为新模板
          setSaveMode('saveAs');
          templateForm.resetFields();
          setTemplateModalVisible(true);
        }
      });
    } else {
      // 如果不是从模板加载的，直接打开保存对话框
      setSaveMode('saveAs');
      templateForm.resetFields();
      setTemplateModalVisible(true);
    }
  };

  // 提交保存模板
  const handleSubmitTemplate = async () => {
    try {
      const values = await templateForm.validateFields();

      // 确保filterGroups不为null
      const submitGroups = filterGroups || [];

      // 处理数组类型的value，将其转换为逗号分隔的字符串
      // 同时确保每个block都有key字段
      const processedGroups = submitGroups.map(group => ({
        ...group,
        blocks: group.blocks.map(block => {
          let processedBlock = { ...block };

          // 确保key和field字段存在并保持一致
          if (!processedBlock.key && processedBlock.field) {
            processedBlock.key = processedBlock.field;
          } else if (processedBlock.key && !processedBlock.field) {
            processedBlock.field = processedBlock.key;
          }

          // 处理数组类型的value
          if (Array.isArray(processedBlock.value)) {
            processedBlock.value = processedBlock.value.join(',');
          }

          return processedBlock;
        })
      }));

      const template: QueryTemplate = {
        name: values.name,
        description: values.description || '',
        groups: processedGroups,
      };

      // 如果是更新原模板，需要添加模板ID
      if (saveMode === 'save' && sourceTemplateId) {
        template.id = sourceTemplateId;
      }

      await saveQueryTemplate(template);

      // 根据保存模式显示不同的成功提示
      if (saveMode === 'save' && sourceTemplateId) {
        message.success(`模板「${values.name}」更新成功`);
      } else {
        message.success(`模板「${values.name}」保存成功`);
      }

      setTemplateModalVisible(false);
      templateForm.resetFields();

      // 调用回调函数，刷新模板列表
      if (onTemplateSaved) {
        onTemplateSaved();
      }

      // 切换到模板标签页
      if (onSwitchToTemplateTab) {
        setTimeout(() => {
          onSwitchToTemplateTab();
        }, 300); // 等待一小段时间，确保模板列表已经刷新
      }
    } catch (error) {
      console.error('保存模板失败:', error);
      message.error('保存模板失败');
    }
  };

  // 渲染筛选块
  const renderFilterBlock = (block: FilterBlock, groupId: string) => {
    const getBlockTitle = () => {
      switch (block.type) {
        case FilterType.Device:
          return '设备筛选';
        case FilterType.NodeLabel:
          return '节点筛选';
        case FilterType.Taint:
          return '污点筛选';
        default:
          return '未知类型';
      }
    };

    // 获取字段选项
    const getFieldOptions = () => {
      switch (block.type) {
        case FilterType.Device:
          return filterOptions.deviceFields || [];
        case FilterType.NodeLabel:
          return filterOptions.nodeLabelKeys || [];
        case FilterType.Taint:
          return filterOptions.nodeTaintKeys || [];
        default:
          return [];
      }
    };

    // 获取条件类型选项
    const getConditionOptions = () => {
      switch (block.type) {
        case FilterType.Device:
          return [
            { label: '等于', value: ConditionType.Equal },
            { label: '不等于', value: ConditionType.NotEqual },
            { label: '包含', value: ConditionType.Contains },
            { label: '不包含', value: ConditionType.NotContains },
            { label: '在列表中', value: ConditionType.In },
            { label: '不在列表中', value: ConditionType.NotIn },
            { label: '大于', value: ConditionType.GreaterThan },
            { label: '小于', value: ConditionType.LessThan },
            { label: '为空', value: ConditionType.IsEmpty },
            { label: '不为空', value: ConditionType.IsNotEmpty },
          ];
        case FilterType.NodeLabel:
        case FilterType.Taint:
          return [
            { label: '等于', value: ConditionType.Equal },
            { label: '不等于', value: ConditionType.NotEqual },
            { label: '存在', value: ConditionType.Exists },
            { label: '不存在', value: ConditionType.NotExists },
            { label: '在列表中', value: ConditionType.In },
            { label: '不在列表中', value: ConditionType.NotIn },
          ];
        default:
          return [];
      }
    };

    // 获取值选项
    const getValueOptions = () => {
      if (!block.field) return [];

      switch (block.type) {
        case FilterType.NodeLabel:
          return labelValues[block.field] || [];
        case FilterType.Taint:
          return taintValues[block.field] || [];
        case FilterType.Device:
          return deviceFieldValues[block.field] || [];
        default:
          return [];
      }
    };

    // 是否需要显示值输入
    const shouldShowValueInput = () => {
      if (!block.conditionType) return false;
      // 以下条件不需要值输入
      return ![ConditionType.Exists, ConditionType.NotExists, ConditionType.IsEmpty, ConditionType.IsNotEmpty].includes(block.conditionType);
    };

    // 是否是多选条件
    const isMultipleValueCondition = () => {
      // 只有在列表中和不在列表中条件是多选的
      return block.conditionType === ConditionType.In || block.conditionType === ConditionType.NotIn;
    };

    // 获取块类型的颜色
    const getBlockTypeColor = () => {
      switch (block.type) {
        case FilterType.Device:
          return 'blue';
        case FilterType.NodeLabel:
          return 'green';
        case FilterType.Taint:
          return 'orange';
        default:
          return 'default';
      }
    };

    return (
      <div className="filter-block">
        <div className="filter-block-header">
          <Space>
            <Tag color={getBlockTypeColor()}>{getBlockTitle()}</Tag>
            {block.field && (
              <Text type="secondary" style={{ fontSize: '12px' }}>
                {getFieldOptions().find((opt: FilterOption) => opt.value === block.field)?.label || block.field}
              </Text>
            )}
          </Space>

          {/* 删除按钮移动到右上角 */}
          <div
            className="delete-block-button"
            onClick={() => removeFilterBlock(groupId, block.id)}
          >
            <Tooltip title="删除此条件">
              <DeleteOutlined style={{ fontSize: '16px' }} />
            </Tooltip>
          </div>
        </div>
        <div className="filter-block-content">
          {/* 字段选择 */}
          <div className="filter-block-item">
            <Text>字段</Text>
            <Select
              style={{ width: '100%' }}
              placeholder="选择字段"
              value={block.field}
              showSearch
              optionFilterProp="children"
              onChange={(value) => {
                // 如果字段发生了变化，则加载对应的值选项
                if (value !== block.field) {
                  if (block.type === FilterType.NodeLabel) {
                    fetchLabelValues(value);
                  } else if (block.type === FilterType.Taint) {
                    fetchTaintValues(value);
                  } else if (block.type === FilterType.Device) {
                    fetchDeviceFieldValues(value);
                  }
                }
                // 同时更新 field 和 key 字段，确保它们保持同步
                updateFilterBlock(groupId, block.id, { field: value, key: value });
              }}
            >
              {getFieldOptions().map((option: FilterOption) => (
                <Option key={option.value} value={option.value}>
                  {option.label}
                </Option>
              ))}
            </Select>
          </div>

          {/* 条件类型选择 */}
          <div className="filter-block-item">
            <Text>条件</Text>
            <Select
              style={{ width: '100%' }}
              placeholder="选择条件"
              value={block.conditionType}
              onChange={(value) => updateFilterBlock(groupId, block.id, { conditionType: value })}
              optionLabelProp="label"
            >
              {getConditionOptions().map((option: { label: string, value: ConditionType }) => (
                <Option key={option.value} value={option.value} label={option.label}>
                  <Space>
                    {option.value === ConditionType.Equal && <Tag color="blue">=</Tag>}
                    {option.value === ConditionType.NotEqual && <Tag color="red">≠</Tag>}
                    {option.value === ConditionType.Contains && <Tag color="green">⊃</Tag>}
                    {option.value === ConditionType.NotContains && <Tag color="orange">⊅</Tag>}
                    {option.value === ConditionType.In && <Tag color="purple">∈</Tag>}
                    {option.value === ConditionType.NotIn && <Tag color="magenta">∉</Tag>}
                    {option.value === ConditionType.Exists && <Tag color="cyan">∃</Tag>}
                    {option.value === ConditionType.NotExists && <Tag color="volcano">∄</Tag>}
                    {option.value === ConditionType.GreaterThan && <Tag color="geekblue">&gt;</Tag>}
                    {option.value === ConditionType.LessThan && <Tag color="lime">&lt;</Tag>}
                    {option.value === ConditionType.IsEmpty && <Tag color="gold">∅</Tag>}
                    {option.value === ConditionType.IsNotEmpty && <Tag color="purple">∅̸</Tag>}
                    <span>{option.label}</span>
                  </Space>
                </Option>
              ))}
            </Select>
          </div>

          {/* 值输入 */}
          {shouldShowValueInput() && (
            <div className="filter-block-item">
              <Text>值</Text>
              <Select
                style={{ width: '100%' }}
                placeholder="输入或选择值"
                mode={isMultipleValueCondition() ? 'multiple' : undefined}
                value={block.value}
                onChange={(value) => updateFilterBlock(groupId, block.id, { value })}
                loading={loadingValues}
                showSearch
                allowClear
                optionFilterProp="children"
                popupMatchSelectWidth={false}
                // showArrow 属性已被废弃，现在是默认行为
                tagRender={props => (
                  <Tag
                    closable
                    onClose={props.onClose}
                    style={{ marginRight: 3 }}
                    color={block.conditionType === ConditionType.Equal ? 'blue' :
                          block.conditionType === ConditionType.NotEqual ? 'red' : undefined}
                  >
                    {props.label}
                  </Tag>
                )}
              >
                {getValueOptions().map((option: FilterOption) => (
                  <Option key={option.value} value={option.value}>
                    {option.label}
                  </Option>
                ))}
              </Select>
            </div>
          )}
        </div>


      </div>
    );
  };

  // 添加状态管理
  const [activeDropdownGroupId, setActiveDropdownGroupId] = useState<string | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // 切换下拉菜单状态
  const toggleDropdown = (groupId: string) => {
    if (activeDropdownGroupId === groupId) {
      setActiveDropdownGroupId(null);
    } else {
      setActiveDropdownGroupId(groupId);
    }
  };

  // 点击外部关闭下拉菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setActiveDropdownGroupId(null);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, []);

  // 渲染添加条件下拉菜单
  const renderAddConditionDropdown = (groupId: string) => {
    if (activeDropdownGroupId !== groupId) return null;

    return (
      <div className="add-condition-dropdown" ref={dropdownRef}>
        <div
          className="add-condition-dropdown-item"
          onClick={() => {
            addFilterBlock(groupId, FilterType.Device);
            setActiveDropdownGroupId(null);
          }}
        >
          <DesktopOutlined style={{ color: '#1890ff' }} />
          <span>添加设备字段条件</span>
        </div>
        <div
          className="add-condition-dropdown-item"
          onClick={() => {
            addFilterBlock(groupId, FilterType.NodeLabel);
            setActiveDropdownGroupId(null);
          }}
        >
          <TagsOutlined style={{ color: '#52c41a' }} />
          <span>添加节点标签条件</span>
        </div>
        <div
          className="add-condition-dropdown-item"
          onClick={() => {
            addFilterBlock(groupId, FilterType.Taint);
            setActiveDropdownGroupId(null);
          }}
        >
          <ExclamationCircleOutlined style={{ color: '#fa8c16' }} />
          <span>添加节点污点条件</span>
        </div>

      </div>
    );
  };

  // 渲染筛选组
  const renderFilterGroup = (group: FilterGroup) => {
    return (
      <Card key={group.id} className="filter-group">
        <div className="filter-group-header">
          <Space>
            <Badge color="#1890ff" />
            <Text strong>条件组</Text>
            <Select
              value={group.operator}
              onChange={(value) => {
                console.log('Changing operator to:', value);
                updateFilterGroup(group.id, { operator: value });
              }}
              style={{ width: 240 }}
              popupMatchSelectWidth={false}
            >
              <Option value={LogicalOperator.And}>
                <Space>
                  <Tag color="blue">AND</Tag>
                  <span>所有条件都满足</span>
                </Space>
              </Option>
              <Option value={LogicalOperator.Or}>
                <Space>
                  <Tag color="orange">OR</Tag>
                  <span>满足任一条件</span>
                </Space>
              </Option>
            </Select>
          </Space>
        </div>

        {/* 筛选块列表 */}
        {group.blocks && group.blocks.length > 0 ? (
          <div className="filter-blocks">
            {group.blocks.map((block) => (
              <div key={block.id} className="filter-block-wrapper">
                {renderFilterBlock(block, group.id)}
              </div>
            ))}

            <div className="filter-group-bottom-actions">
              {/* 浮动添加按钮 */}
              <div
                className="add-condition-button"
                onClick={() => toggleDropdown(group.id)}
              >
                <Tooltip title="添加条件">
                  <PlusOutlined style={{ fontSize: '18px', color: '#1890ff' }} />
                </Tooltip>
              </div>

              {/* 浮动删除按钮 */}
              <div
                className="delete-condition-button"
                onClick={() => removeFilterGroup(group.id)}
              >
                <Tooltip title="删除条件组">
                  <DeleteOutlined style={{ fontSize: '18px' }} />
                </Tooltip>
              </div>

              {/* 添加条件下拉菜单 */}
              {renderAddConditionDropdown(group.id)}
            </div>
          </div>
        ) : (
          <div className="filter-blocks">
            <div className="empty-blocks">
              <Space direction="vertical" align="center">
                <FilterOutlined style={{ fontSize: 24, color: '#bfbfbf' }} />
                <Text type="secondary">请添加筛选条件</Text>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => toggleDropdown(group.id)}
                >
                  添加条件
                </Button>
              </Space>
            </div>

            <div className="filter-group-bottom-actions">
              {/* 添加条件下拉菜单 */}
              {renderAddConditionDropdown(group.id)}

              {/* 浮动删除按钮 */}
              <div
                className="delete-condition-button"
                onClick={() => removeFilterGroup(group.id)}
              >
                <Tooltip title="删除条件组">
                  <DeleteOutlined style={{ fontSize: '18px' }} />
                </Tooltip>
              </div>
            </div>
          </div>
        )}
      </Card>
    );
  };

  return (
    <div className="advanced-query-panel">
      {/* 筛选组列表 */}
      {filterGroups && filterGroups.length > 0 ? (
        <div className="filter-groups">
          {filterGroups.map(group => (
            <div key={group.id} className="filter-group-wrapper">
              {renderFilterGroup(group)}
            </div>
          ))}
        </div>
      ) : (
        <div className="empty-groups">
          <Space direction="vertical" align="center">
            <FilterOutlined style={{ fontSize: 32, color: '#bfbfbf' }} />
            <Text type="secondary">请添加条件组开始高级查询</Text>
          </Space>
        </div>
      )}

      {/* 添加筛选组按钮 */}
      <div className="filter-group-actions">
        <Button
          type="dashed"
          icon={<PlusOutlined />}
          onClick={addFilterGroup}
          size="large"
          style={{ width: '100%', height: '60px', borderRadius: '8px', borderStyle: 'dashed', borderWidth: '2px' }}
        >
          <span style={{ fontSize: '16px' }}>添加条件组</span>
        </Button>
      </div>

      {/* 查询操作按钮 */}
      <div className="query-actions">
        <Space size="middle" style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space>
            <Button
              onClick={handleReset}
              icon={<CloseCircleOutlined />}
            >
              重置条件
            </Button>
            <Button
              type="primary"
              ghost
              icon={<SaveOutlined />}
              onClick={handleSaveTemplate}
            >
              保存模板
            </Button>
          </Space>

          <Button
            type="primary"
            icon={<SearchOutlined />}
            onClick={onQuery}
            loading={loading}
            size="large"
            style={{ minWidth: '150px' }}
          >
            执行查询
          </Button>
        </Space>
      </div>

      {/* 保存模板对话框 */}
      <Modal
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <SaveOutlined style={{ color: '#1890ff', marginRight: 8 }} />
            <span>{saveMode === 'save' ? '更新查询模板' : '保存查询模板'}</span>
          </div>
        }
        open={templateModalVisible}
        onOk={handleSubmitTemplate}
        onCancel={() => setTemplateModalVisible(false)}
        destroyOnClose
        okText="保存"
        cancelText="取消"
        centered
      >
        <Form form={templateForm} layout="vertical">
          <Form.Item
            name="name"
            label="模板名称"
            rules={[{ required: true, message: '请输入模板名称' }]}
          >
            <Input
              placeholder="输入模板名称"
              prefix={<FilterOutlined style={{ color: '#bfbfbf' }} />}
            />
          </Form.Item>
          <Form.Item name="description" label="模板描述">
            <Input.TextArea
              placeholder="输入模板描述（可选）"
              rows={4}
              showCount
              maxLength={200}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default AdvancedQueryPanel;
