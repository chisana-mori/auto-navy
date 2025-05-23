import React, { useState, useEffect } from 'react';
import {
  Modal, Form, Input, Select, InputNumber, Button, Card, Alert, Divider, Table, Empty, Spin,
  Tag, Row, Col, message
} from 'antd';
import {
  PlusOutlined, CloudUploadOutlined, CloudDownloadOutlined, SearchOutlined,
  ClusterOutlined, DatabaseOutlined, InfoCircleOutlined, SelectOutlined,
  DeleteOutlined, FilterOutlined
} from '@ant-design/icons';
import { Device } from '../../types/device';
import { FilterGroup, FilterType, ConditionType, LogicalOperator } from '../../types/deviceQuery';
import { ResourcePoolDeviceMatchingPolicy, getResourcePoolDeviceMatchingPoliciesByType } from '../../services/resourcePoolDeviceMatchingPolicyService';
import { queryDevices, getFilterOptions } from '../../services/deviceQueryService';
import DeviceSelectionDrawer from './DeviceSelectionDrawer';
import { v4 as uuidv4 } from 'uuid';
import './CreateOrderModal.less';

const { Option } = Select;

interface CreateOrderModalProps {
  visible: boolean;
  onCancel: () => void;
  onSubmit: (values: any) => Promise<void>;
  clusters: any[];
  resourcePools: any[];
}

const CreateOrderModal: React.FC<CreateOrderModalProps> = ({
  visible,
  onCancel,
  onSubmit,
  clusters,
  resourcePools
}) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [matchingPolicies, setMatchingPolicies] = useState<ResourcePoolDeviceMatchingPolicy[]>([]);
  const [selectedPolicy, setSelectedPolicy] = useState<ResourcePoolDeviceMatchingPolicy | null>(null);
  const [devices, setDevices] = useState<Device[]>([]);
  const [searchingDevices, setSearchingDevices] = useState(false);

  // 设备选择相关状态
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [filterGroups, setFilterGroups] = useState<FilterGroup[]>([]);
  const [filterOptions, setFilterOptions] = useState<Record<string, any>>({});
  const [useSimpleMode, setUseSimpleMode] = useState(false);

  // 获取筛选选项
  useEffect(() => {
    const fetchFilterOptions = async () => {
      try {
        const options = await getFilterOptions();
        setFilterOptions(options);
      } catch (error) {
        console.error('获取筛选选项失败:', error);
        message.error('获取筛选选项失败');
      }
    };

    fetchFilterOptions();
  }, []);

  // 监听资源池类型和动作类型变化，获取匹配策略
  useEffect(() => {
    const fetchMatchingPolicies = async () => {
      const resourcePoolType = form.getFieldValue('resourcePoolType');
      const actionType = form.getFieldValue('actionType');

      if (resourcePoolType && actionType) {
        try {
          setLoading(true);
          const policies = await getResourcePoolDeviceMatchingPoliciesByType(resourcePoolType, actionType);
          setMatchingPolicies(policies.filter(policy => policy.status === 'enabled'));
        } catch (error) {
          console.error('获取匹配策略失败:', error);
        } finally {
          setLoading(false);
        }
      }
    };

    if (visible) {
      fetchMatchingPolicies();
    }
  }, [form, visible]);

  // 监听表单字段变化
  const handleFormValuesChange = async (changedValues: any) => {
    if (changedValues.resourcePoolType || changedValues.actionType) {
      // 重置匹配策略和设备
      form.setFieldsValue({ matchingPolicyId: undefined });
      setSelectedPolicy(null);
      setDevices([]);

      // 获取匹配策略
      const resourcePoolType = form.getFieldValue('resourcePoolType');
      const actionType = form.getFieldValue('actionType');

      if (resourcePoolType && actionType) {
        try {
          setLoading(true);
          const policies = await getResourcePoolDeviceMatchingPoliciesByType(resourcePoolType, actionType);
          setMatchingPolicies(policies.filter(policy => policy.status === 'enabled'));

          // 不再自动构造入池筛选条件，让用户手动查询
        } catch (error) {
          console.error('获取匹配策略失败:', error);
        } finally {
          setLoading(false);
        }
      }
    }

    if (changedValues.matchingPolicyId) {
      const policyId = changedValues.matchingPolicyId;
      const policy = matchingPolicies.find(p => p.id === policyId) || null;
      setSelectedPolicy(policy);
      setDevices([]);
    }

    // 监听集群变化
    if (changedValues.clusterId) {
      // 不再自动构造查询条件，让用户手动查询
    }
  };

  // 添加筛选组
  const addFilterGroup = () => {
    const newGroup: FilterGroup = {
      id: uuidv4(),
      blocks: [],
      operator: LogicalOperator.And
    };
    setFilterGroups([...filterGroups, newGroup]);
  };

  // 这些函数在当前实现中未使用，但保留在代码中以备将来使用
  // 如果需要在界面中添加筛选块功能，可以取消注释
  /*
  // 添加筛选块
  const addFilterBlock = (groupId: string, type: FilterType) => {
    const defaultConditionType = type !== FilterType.Device
      ? ConditionType.In
      : ConditionType.Equal;

    const newBlock: FilterBlock = {
      id: uuidv4(),
      type,
      conditionType: defaultConditionType,
      operator: LogicalOperator.And,
    };

    setFilterGroups(
      filterGroups.map(group => {
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
  const updateFilterBlock = (groupId: string, blockId: string, updates: Partial<FilterBlock>) => {
    setFilterGroups(
      filterGroups.map(group => {
        if (group.id === groupId) {
          return {
            ...group,
            blocks: group.blocks.map(block => {
              if (block.id === blockId) {
                return { ...block, ...updates };
              }
              return block;
            }),
          };
        }
        return group;
      })
    );
  };
  */

  // 删除筛选块
  const removeFilterBlock = (groupId: string, blockId: string) => {
    setFilterGroups(
      filterGroups.map(group => {
        if (group.id === groupId) {
          return {
            ...group,
            blocks: group.blocks.filter(block => block.id !== blockId),
          };
        }
        return group;
      }).filter(group => group.blocks.length > 0 || group.id === groupId)
    );
  };

  // 删除筛选组
  const removeFilterGroup = (groupId: string) => {
    setFilterGroups(filterGroups.filter(group => group.id !== groupId));
  };

  // 打开设备选择抽屉
  const openDeviceDrawer = () => {
    // 检查是否有匹配的策略
    const hasMatchingPolicy = matchingPolicies.length > 0;

    // 如果没有筛选条件且有匹配策略，添加一个默认的筛选组
    if (filterGroups.length === 0 && hasMatchingPolicy) {
      addFilterGroup();
    }

    setDrawerVisible(true);

    // 如果没有匹配的策略，使用简单模式
    setUseSimpleMode(!hasMatchingPolicy);
  };

  // 处理设备选择
  const handleSelectDevices = (selectedDevices: Device[]) => {
    setDevices(selectedDevices);
    message.success(`已选择 ${selectedDevices.length} 台设备`);
  };

  // 查询设备
  const handleSearchDevices = async () => {
    const deviceCount = form.getFieldValue('deviceCount') || 10;

    if (selectedPolicy) {
      try {
        setSearchingDevices(true);

        // 使用选中的策略查询设备
        const response = await queryDevices({
          groups: selectedPolicy.queryGroups || [],
          page: 1,
          size: deviceCount
        });

        setDevices(response.list || []);
      } catch (error) {
        console.error('查询设备失败:', error);
      } finally {
        setSearchingDevices(false);
      }
    }
  };

  // 提交表单
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();

      // 添加设备列表到表单值
      const formData = {
        ...values,
        devices: devices.map(device => ({ id: device.id }))
      };

      await onSubmit(formData);
    } catch (error) {
      console.error('表单验证失败:', error);
    }
  };

  // 设备表格列
  const deviceColumns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
    },
    {
      title: '集群',
      dataIndex: 'cluster',
      key: 'cluster',
    },
    {
      title: '标签',
      dataIndex: 'labels',
      key: 'labels',
      render: (labels: any) => (
        <div style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {labels ? Object.entries(labels).map(([key, value]) => `${key}=${value}`).join(', ') : ''}
        </div>
      ),
    }
  ];

  return (
    <Modal
      title={
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <PlusOutlined style={{ marginRight: 8, color: '#1890ff' }} />
          <span>创建订单</span>
        </div>
      }
      open={visible}
      onCancel={onCancel}
      width={900}
      footer={[
        <Button key="cancel" onClick={onCancel}>
          取消
        </Button>,
        <Button
          key="submit"
          type="primary"
          loading={loading}
          onClick={handleSubmit}
          disabled={devices.length === 0}
        >
          创建订单
        </Button>
      ]}
      destroyOnClose
      className="create-order-modal"
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          actionType: 'pool_entry',
          deviceCount: 10
        }}
        onValuesChange={handleFormValuesChange}
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
            label="订单名称"
            rules={[{ required: true, message: '请输入订单名称' }]}
          >
            <Input placeholder="请输入订单名称" />
          </Form.Item>

          <Form.Item
            name="description"
            label="订单描述"
          >
            <Input.TextArea placeholder="请输入订单描述" rows={2} />
          </Form.Item>
        </Card>

        <Card
          title={
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <ClusterOutlined style={{ marginRight: 8, color: '#52c41a' }} />
              <span style={{ fontSize: '14px', fontWeight: 500 }}>资源配置</span>
            </div>
          }
          size="small"
          style={{ marginBottom: '24px' }}
          headStyle={{ backgroundColor: '#f5f7fa' }}
          bodyStyle={{ padding: '16px 24px' }}
        >
          <Form.Item
            name="clusterId"
            label="集群"
            rules={[{ required: true, message: '请选择集群' }]}
          >
            <Select placeholder="请选择集群">
              {clusters.map(cluster => (
                <Option key={cluster.id} value={cluster.id}>{cluster.name}</Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="resourcePoolType"
            label="资源池类型"
            rules={[{ required: true, message: '请选择资源池类型' }]}
          >
            <Select placeholder="请选择资源池类型">
              {resourcePools.map(pool => (
                <Option key={pool.type} value={pool.type}>{pool.name}</Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="actionType"
            label="动作类型"
            rules={[{ required: true, message: '请选择动作类型' }]}
          >
            <Select placeholder="请选择动作类型">
              <Option value="pool_entry">
                <CloudUploadOutlined style={{ color: '#1890ff', marginRight: 4 }} /> 入池
              </Option>
              <Option value="pool_exit">
                <CloudDownloadOutlined style={{ color: '#ff7a45', marginRight: 4 }} /> 退池
              </Option>
            </Select>
          </Form.Item>
        </Card>

        <Card
          title={
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <DatabaseOutlined style={{ marginRight: 8, color: '#722ed1' }} />
                <span style={{ fontSize: '14px', fontWeight: 500 }}>设备选择</span>
              </div>
              <div>
                {devices.length > 0 && (
                  <Tag color="success" style={{ marginRight: 8 }}>
                    已选择 {devices.length} 台设备
                  </Tag>
                )}
              </div>
            </div>
          }
          size="small"
          style={{ marginBottom: '24px' }}
          headStyle={{ backgroundColor: '#f5f7fa' }}
          bodyStyle={{ padding: '16px 24px' }}
        >
          {form.getFieldValue('resourcePoolType') && form.getFieldValue('actionType') ? (
            matchingPolicies.length > 0 ? (
              <>
                <Form.Item
                  name="matchingPolicyId"
                  label="设备匹配策略"
                  rules={[{ required: true, message: '请选择设备匹配策略' }]}
                >
                  <Select placeholder="请选择设备匹配策略">
                    {matchingPolicies.map(policy => (
                      <Option key={policy.id} value={policy.id}>{policy.name}</Option>
                    ))}
                  </Select>
                </Form.Item>

                {selectedPolicy && (
                  <div style={{ marginBottom: 16 }}>
                    <div style={{ marginBottom: 8, fontWeight: 500 }}>查询参数：</div>
                    <div>
                      {selectedPolicy.queryGroups && selectedPolicy.queryGroups.map((group, groupIndex) => (
                        <div key={`group-${groupIndex}`} style={{ marginBottom: 8 }}>
                          {group.blocks && group.blocks.map((block, blockIndex) => {
                            let blockContent = '';
                            let tagColor = 'default';

                            if (block.type === FilterType.Device) {
                              // eslint-disable-next-line @typescript-eslint/no-explicit-any
                              const fieldLabel = filterOptions.deviceFields?.find((f: any) => f.value === block.field)?.label || block.field;
                              const conditionLabel = block.conditionType === ConditionType.Equal ? '等于' :
                                                    block.conditionType === ConditionType.NotEqual ? '不等于' :
                                                    block.conditionType === ConditionType.Contains ? '包含' :
                                                    block.conditionType === ConditionType.IsEmpty ? '为空' :
                                                    block.conditionType === ConditionType.IsNotEmpty ? '不为空' :
                                                    block.conditionType;
                              blockContent = `${fieldLabel} ${conditionLabel} ${block.value || ''}`;
                              tagColor = 'blue';
                            } else if (block.type === FilterType.NodeLabel) {
                              blockContent = `标签 ${block.key || ''} ${block.conditionType} ${block.value || ''}`;
                              tagColor = 'green';
                            } else if (block.type === FilterType.Taint) {
                              blockContent = `污点 ${block.key || ''} ${block.conditionType} ${block.value || ''}`;
                              tagColor = 'orange';
                            }

                            return (
                              <Tag
                                key={`block-${blockIndex}`}
                                color={tagColor}
                                style={{ margin: '4px' }}
                              >
                                {blockContent}
                              </Tag>
                            );
                          })}
                        </div>
                      ))}
                    </div>

                    {/* 显示额外动态条件 (additionConds) */}
                    {form.getFieldValue('actionType') === 'pool_entry' && selectedPolicy.additionConds && selectedPolicy.additionConds.length > 0 && (
                      <div style={{ marginTop: 8 }}>
                        <div style={{ marginBottom: 8, fontWeight: 500 }}>额外动态条件：</div>
                        <div>
                          {selectedPolicy.additionConds.map((cond, index) => {
                            let condLabel = '';
                            if (cond === 'same_idc') condLabel = '与目标集群同IDC';
                            else if (cond === 'same_zone') condLabel = '与目标集群同安全域';
                            else if (cond === 'same_room') condLabel = '与目标集群同机房';
                            else condLabel = cond;

                            return (
                              <Tag key={`cond-${index}`} color="purple" style={{ margin: '4px' }}>
                                {condLabel}
                              </Tag>
                            );
                          })}
                        </div>
                      </div>
                    )}
                  </div>
                )}

                <Form.Item
                  name="deviceCount"
                  label="设备数量"
                  rules={[{ required: true, message: '请输入设备数量' }]}
                >
                  <InputNumber min={1} max={100} style={{ width: '100%' }} />
                </Form.Item>

                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item>
                      <Button
                        type="primary"
                        icon={<SearchOutlined />}
                        onClick={handleSearchDevices}
                        disabled={!selectedPolicy}
                        loading={searchingDevices}
                        block
                      >
                        使用策略查询
                      </Button>
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item>
                      <Button
                        type="default"
                        icon={<FilterOutlined />}
                        onClick={openDeviceDrawer}
                        block
                      >
                        自定义筛选
                      </Button>
                    </Form.Item>
                  </Col>
                </Row>
              </>
            ) : (
              <>
                <Alert
                  message="未找到匹配的设备策略"
                  description="当前资源池类型和动作类型没有可用的设备匹配策略，您可以使用自定义筛选功能选择设备。"
                  type="info"
                  showIcon
                  style={{ marginBottom: 16 }}
                />

                <div style={{ marginTop: 16 }}>
                  <Card
                    title="基本筛选条件"
                    size="small"
                    type="inner"
                    style={{ marginBottom: 16 }}
                  >
                    {filterGroups.length === 0 ? (
                      <Empty
                        description="暂无筛选条件"
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                      />
                    ) : (
                      filterGroups.map((group, groupIndex) => (
                        <div key={group.id} className="filter-group">
                          <div className="filter-group-header">
                            <span>筛选组 {groupIndex + 1}</span>
                            <Button
                              type="text"
                              danger
                              icon={<DeleteOutlined />}
                              onClick={() => removeFilterGroup(group.id)}
                              size="small"
                            />
                          </div>

                          {group.blocks.map((block) => {
                            // 根据筛选块类型显示不同的内容
                            let blockContent = '';
                            let tagColor = 'default';

                            // 特殊处理入池条件的显示
                            const actionType = form.getFieldValue('actionType');
                            if (actionType === 'pool_entry' && block.type === FilterType.Device) {
                              if (block.field === 'cluster' && block.conditionType === ConditionType.IsEmpty) {
                                blockContent = '未入池设备';
                                tagColor = 'blue';
                              } else if (['idc', 'zone', 'room'].includes(block.field || '') && block.conditionType === ConditionType.Equal) {
                                // 优先使用自定义标签
                                if (block.label) {
                                  blockContent = block.label;
                                } else {
                                  const locationLabels: Record<string, string> = {
                                    'idc': '与集群同IDC',
                                    'zone': '与集群同安全域',
                                    'room': '与集群同机房'
                                  };
                                  blockContent = locationLabels[block.field || ''] || `${block.field} = ${block.value}`;
                                }
                                tagColor = 'green';
                              } else {
                                // 常规设备字段处理
                                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                                const fieldLabel = filterOptions.deviceFields?.find((f: any) => f.value === block.field)?.label || block.field;
                                const conditionLabel = block.conditionType === ConditionType.Equal ? '等于' :
                                                      block.conditionType === ConditionType.NotEqual ? '不等于' :
                                                      block.conditionType === ConditionType.Contains ? '包含' :
                                                      block.conditionType === ConditionType.IsEmpty ? '为空' :
                                                      block.conditionType === ConditionType.IsNotEmpty ? '不为空' :
                                                      block.conditionType;
                                blockContent = `${fieldLabel} ${conditionLabel} ${block.value || ''}`;
                              }
                            } else if (block.type === FilterType.Device) {
                              // 常规设备字段处理
                              // eslint-disable-next-line @typescript-eslint/no-explicit-any
                              const fieldLabel = filterOptions.deviceFields?.find((f: any) => f.value === block.field)?.label || block.field;
                              const conditionLabel = block.conditionType === ConditionType.Equal ? '等于' :
                                                    block.conditionType === ConditionType.NotEqual ? '不等于' :
                                                    block.conditionType === ConditionType.Contains ? '包含' :
                                                    block.conditionType === ConditionType.IsEmpty ? '为空' :
                                                    block.conditionType === ConditionType.IsNotEmpty ? '不为空' :
                                                    block.conditionType;
                              blockContent = `${fieldLabel} ${conditionLabel} ${block.value || ''}`;
                            } else if (block.type === FilterType.NodeLabel) {
                              blockContent = `标签 ${block.key || ''} ${block.conditionType} ${block.value || ''}`;
                            } else if (block.type === FilterType.Taint) {
                              blockContent = `污点 ${block.key || ''} ${block.conditionType} ${block.value || ''}`;
                            }

                            return (
                              <Tag
                                key={block.id}
                                closable
                                color={tagColor}
                                onClose={() => removeFilterBlock(group.id, block.id)}
                                style={{ margin: '4px' }}
                              >
                                {blockContent}
                              </Tag>
                            );
                          })}

                          {group.blocks.length === 0 && (
                            <div style={{ padding: '8px 0', color: '#999' }}>
                              请添加筛选条件
                            </div>
                          )}
                        </div>
                      ))
                    )}

                    <div style={{ marginTop: 16 }}>
                      <Button
                        type="dashed"
                        onClick={() => addFilterGroup()}
                        style={{ marginRight: 8 }}
                      >
                        添加筛选组
                      </Button>
                    </div>
                  </Card>
                </div>
              </>
            )
          ) : (
            <Alert
              message="请先选择资源池类型和动作类型"
              description="选择资源池类型和动作类型后，系统将自动加载可用的设备匹配策略。"
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />
          )}
        </Card>
      </Form>

      {/* 设备选择抽屉 */}
      <DeviceSelectionDrawer
        visible={drawerVisible}
        onClose={() => setDrawerVisible(false)}
        onSelectDevices={handleSelectDevices}
        filterGroups={filterGroups}
        selectedDevices={devices}
        loading={loading}
        simpleMode={useSimpleMode}
      />

      <Divider orientation="left">设备列表</Divider>

      {searchingDevices ? (
        <div style={{ textAlign: 'center', padding: '30px 0' }}>
          <Spin tip="正在查询设备..." />
        </div>
      ) : devices.length > 0 ? (
        <Table
          columns={deviceColumns}
          dataSource={devices}
          rowKey="id"
          pagination={false}
          size="small"
          scroll={{ y: 300 }}
          summary={() => (
            <Table.Summary fixed>
              <Table.Summary.Row>
                <Table.Summary.Cell index={0} colSpan={5}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', padding: '8px 0' }}>
                    <span>共选择 <strong>{devices.length}</strong> 台设备</span>
                  </div>
                </Table.Summary.Cell>
              </Table.Summary.Row>
            </Table.Summary>
          )}
        />
      ) : (
        <Empty
          description={
            <div>
              <p>暂无设备数据</p>
            </div>
          }
        />
      )}
    </Modal>
  );
};

export default CreateOrderModal;
