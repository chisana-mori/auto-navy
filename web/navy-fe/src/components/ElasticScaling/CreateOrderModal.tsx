import React, { useState, useEffect, useCallback } from 'react';
import {
  Modal, Form, Input, Select, Button, Card, Alert, Empty, Spin,
  Tag, message, Checkbox, Row, Col, Space
} from 'antd';
import {
  PlusOutlined, CloudUploadOutlined, CloudDownloadOutlined, SearchOutlined,
  CloseOutlined,
  ClusterOutlined, DatabaseOutlined, InfoCircleOutlined,
  DeleteOutlined, CheckCircleOutlined
} from '@ant-design/icons';
import { Device } from '../../types/device';
import { FilterGroup, FilterType, ConditionType, LogicalOperator, FilterBlock } from '../../types/deviceQuery';
import { ResourcePoolDeviceMatchingPolicy, getResourcePoolDeviceMatchingPoliciesByType } from '../../services/resourcePoolDeviceMatchingPolicyService';
import { queryDevices, getFilterOptions } from '../../services/deviceQueryService';
import { statsApi } from '../../services/elasticScalingService';
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
  initialValues?: any;
}

const CreateOrderModal = React.forwardRef<
  { open(values?: any): void },
  CreateOrderModalProps
>((
  {
    visible,
    onCancel,
    onSubmit,
    clusters,
    resourcePools: initialResourcePools,
    initialValues
  },
  ref
) => {
  // 本地状态存储资源池列表
  const [resourcePools, setResourcePools] = useState<any[]>([]);
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
  const [useSimpleMode] = useState(false);
  // 新增选中状态追踪
  const [checkedFilters, setCheckedFilters] = useState<{
    idc: boolean;
    zone: boolean;
    room: boolean;
  }>({
    idc: false, 
    zone: false,
    room: false
  });

  // Watch essential form values for reactive updates
  const actionType = Form.useWatch('actionType', form);
  const clusterId = Form.useWatch('clusterId', form);
  const resourcePoolType = Form.useWatch('resourcePoolType', form);

  // Helper function to map condition types to simple symbols or raw values
  const mapConditionTypeToSymbol = (conditionType: ConditionType | string): string => {
    switch (conditionType) {
      case ConditionType.Equal: return '=';
      case ConditionType.NotEqual: return '!=';
      case ConditionType.Contains: return 'contains';
      case ConditionType.IsEmpty: return 'is empty';
      case ConditionType.IsNotEmpty: return 'is not empty';
      case ConditionType.In: return 'in';
      case ConditionType.NotIn: return 'not in';
      default: return String(conditionType); // Return raw value if not matched
    }
  };

  // Centralized function to build and display query groups
  const buildAndDisplayQueryGroups = useCallback((
    policy: ResourcePoolDeviceMatchingPolicy | null,
    currentCluster: any,
    currentActionType: string,
    currentCheckedFilters: { idc: boolean; zone: boolean; room: boolean }
  ) => {
    if (!currentCluster || !currentActionType) { // Removed filterOptions dependency for initial call if not strictly needed for labels
      setFilterGroups([]);
      return;
    }

    let newQueryGroups: FilterGroup[] = [];
    if (policy && policy.queryGroups) {
      newQueryGroups = JSON.parse(JSON.stringify(policy.queryGroups));
      newQueryGroups.forEach(group => {
        group.blocks?.forEach(block => {
          const conditionSymbol = mapConditionTypeToSymbol(block.conditionType);
          // For Device types from policy, always try to format label if key/field exists, overriding existing label.
          if (block.type === FilterType.Device) {
            const keyForDevice = block.key || block.field; // Prioritize key if available, else field
            if (keyForDevice) {
              block.label = `${keyForDevice} ${conditionSymbol} ${block.value || ''}`.trim();
            } else if (!block.label) {
              // Fallback for Device type if no key/field, and no pre-existing label
              block.label = `Device Attr: ${conditionSymbol} ${block.value || 'N/A'}`;
            }
            // If it had a non-overridden label and no key/field, that label persists.
          } 
          // For other types (NodeLabel, Taint), only generate label if one doesn't already exist.
          else if (!block.label) {
            if (block.type === FilterType.NodeLabel && block.key) {
              block.label = `label:${block.key} ${conditionSymbol} ${block.value || ''}`.trim();
            } else if (block.type === FilterType.Taint && block.key) {
              block.label = `taint:${block.key} ${conditionSymbol} ${block.value || ''}`.trim();
            } else {
              // Generic fallback for other types if no label and cannot determine specifics
              block.label = `Policy Condition (Type: ${block.type || 'N/A'})`;
            }
          }
        });
      });
    }

    // Add / ensure "同集群" for pool_exit (This logic will apply its label after the above processing)
    if (currentActionType === 'pool_exit') {
      const clusterName = currentCluster.name || currentCluster.clusterName || currentCluster.clusterNameCn || currentCluster.alias || `集群-${currentCluster.id}`;
      const poolExitBlock: FilterBlock = { id: uuidv4(), type: FilterType.Device, field: 'cluster', conditionType: ConditionType.Equal, value: clusterName, operator: LogicalOperator.And, label: '同集群' };
      if (!newQueryGroups.some(g => g.blocks.some(b => b.field === 'cluster' && b.conditionType === ConditionType.Equal && b.label === '同集群'))) {
         newQueryGroups.push({ id: uuidv4(), blocks: [poolExitBlock], operator: LogicalOperator.And });
      }
    }
    // Add / ensure "未入池设备" for pool_entry (This logic will apply its label after the above processing)
    else { 
      const unpooledDeviceBlock: FilterBlock = { id: uuidv4(), type: FilterType.Device, field: 'cluster', conditionType: ConditionType.IsEmpty, operator: LogicalOperator.And, label: '未入池设备' };
      if (!newQueryGroups.some(g => g.blocks.some(b => b.field === 'cluster' && b.conditionType === ConditionType.IsEmpty && b.label === '未入池设备'))) {
        newQueryGroups.unshift({ id: uuidv4(), blocks: [unpooledDeviceBlock], operator: LogicalOperator.And });
      }
      // Add location-based filters from checkboxes
      const additionalLocationBlocks: FilterBlock[] = [];
      if (currentCheckedFilters.idc && currentCluster.idc) {
        additionalLocationBlocks.push({ id: uuidv4(), type: FilterType.Device, field: 'idc', conditionType: ConditionType.Equal, value: currentCluster.idc, operator: LogicalOperator.And, label: `idc = ${currentCluster.idc}` });
      }
      if (currentCheckedFilters.zone && currentCluster.zone) {
        additionalLocationBlocks.push({ id: uuidv4(), type: FilterType.Device, field: 'zone', conditionType: ConditionType.Equal, value: currentCluster.zone, operator: LogicalOperator.And, label: `zone = ${currentCluster.zone}` });
      }
      const roomValue = currentCluster.room || currentCluster.idc || '';
      if (currentCheckedFilters.room && roomValue) {
        additionalLocationBlocks.push({ id: uuidv4(), type: FilterType.Device, field: 'room', conditionType: ConditionType.Equal, value: roomValue, operator: LogicalOperator.And, label: `room = ${roomValue}` });
      }
      if (additionalLocationBlocks.length > 0) {
        newQueryGroups.push({ id: uuidv4(), blocks: additionalLocationBlocks, operator: LogicalOperator.And });
      }
    }
    setFilterGroups(newQueryGroups);
  }, []);

  // 暴露组件方法
  React.useImperativeHandle(ref, () => ({
    open: (values?: any) => {
      if (values) {
        const { name, devices, ...rest } = values;
        form.setFieldsValue({
          ...rest,
          name: `克隆自${name}`,
          devices: devices
        });
        // 同步更新设备列表状态
        if (devices && Array.isArray(devices)) {
          setDevices(devices);
        }
      }
    }
  }));

  // 处理初始值
  useEffect(() => {
    if (initialValues) {
      const { name, devices, ...rest } = initialValues;
      form.setFieldsValue({
        ...rest,
        name: `克隆自${name}`,
        devices: devices
      });
      // 同步更新设备列表状态
      if (devices && Array.isArray(devices)) {
        setDevices(devices);
      }
    }
  }, [initialValues, form]);

  // Effect to fetch initial filter options
  useEffect(() => {
    const fetchInitialData = async () => {
      try {
        const options = await getFilterOptions();
        setFilterOptions(options);
        // Removed call to buildAndDisplayQueryGroups from here
      } catch (error) {
        console.error('获取筛选选项失败:', error);
        message.error('获取筛选选项失败');
      }
      
      // 从后端接口获取资源池类型
      try {
        const resourceTypes = await statsApi.getResourcePoolTypes();
        const poolTypes = resourceTypes.map((type: string) => {
          // 直接使用原始值作为名称，不进行中文翻译
          return { type, name: type };
        });
        setResourcePools(poolTypes);
      } catch (error) {
        console.error('获取资源池类型失败:', error);
        message.error('获取资源池类型失败，请刷新重试');
        // 如果获取失败，不使用任何默认数据，保持空状态
        setResourcePools([]);
      }
    };
    fetchInitialData();
  }, []); // Empty deps: run once on mount

  // Effect to react to changes and update displayed query groups
  useEffect(() => {
    if (!actionType || !clusterId) {
      setFilterGroups([]);
      return;
    }
    const currentCluster = clusters.find(c => c.id === clusterId);
    if (!currentCluster) {
      setFilterGroups([]);
      return;
    }
    // Pass the Form.useWatch values and state directly
    buildAndDisplayQueryGroups(selectedPolicy, currentCluster, actionType, checkedFilters);
  }, [
    selectedPolicy, // Policy object from state
    actionType,     // Watched from form
    clusterId,      // Watched from form
    clusters,       // Prop
    checkedFilters, // State for checkboxes
    buildAndDisplayQueryGroups // The useCallback function itself
  ]);

  // Effect to fetch matching policies when resourcePoolType or actionType change
  useEffect(() => {
    if (resourcePoolType && actionType) {
      const fetchPolicies = async () => {
        setLoading(true);
        // Reset policy and dependent filters before fetching new ones
        setSelectedPolicy(null);
        // setFilterGroups([]); // Covered by the main reactive useEffect
        // setCheckedFilters({ idc: false, zone: false, room: false }); // Covered by the main reactive useEffect via selectedPolicy=null

        try {
          const policies = await getResourcePoolDeviceMatchingPoliciesByType(resourcePoolType, actionType);
          const enabledPolicies = policies.filter(policy => policy.status === 'enabled');
          setMatchingPolicies(enabledPolicies);
          if (enabledPolicies.length > 0) {
            const firstPolicy = enabledPolicies[0];
            setSelectedPolicy(firstPolicy); 
            if (actionType === 'pool_entry' && firstPolicy.additionConds) {
              setCheckedFilters({
                idc: firstPolicy.additionConds.some(cond => cond === 'idc' || cond === 'same_idc'),
                zone: firstPolicy.additionConds.some(cond => cond === 'zone' || cond === 'same_zone'),
                room: firstPolicy.additionConds.some(cond => cond === 'room' || cond === 'same_room'),
              });
            } else { // Policy doesn't have additionConds or not pool_entry
              setCheckedFilters({ idc: false, zone: false, room: false });
            }
          } else {
            setSelectedPolicy(null); // No policy found
            setCheckedFilters({ idc: false, zone: false, room: false }); // Reset checkboxes
          }
        } catch (error) {
          console.error('获取匹配策略失败:', error);
          setMatchingPolicies([]);
          setSelectedPolicy(null); // Ensure reset on error
          setCheckedFilters({ idc: false, zone: false, room: false });
        } finally {
          setLoading(false);
        }
      };
      fetchPolicies();
    } else {
      setMatchingPolicies([]);
      setSelectedPolicy(null);
      setCheckedFilters({ idc: false, zone: false, room: false });
    }
  }, [resourcePoolType, actionType]); // Removed form dependency

  // Update filter groups based on checkbox changes (for location filters)
  const updateFilterGroups = useCallback((newCheckedState: { idc: boolean; zone: boolean; room: boolean }, currentClusterDetails: any) => {
    // This function is primarily for *manual* checkbox interactions.
    // The main reactive useEffect should handle most updates.
    // However, if checkboxes directly modify filterGroups, this logic needs to be robust
    // or integrated into buildAndDisplayQueryGroups / reactive useEffect.
    // For now, let's assume buildAndDisplayQueryGroups is the source of truth for filterGroups.
    // So, when checkboxes change, we update checkedFilters state, and the reactive useEffect handles the rest.
    
    // The `onChange` for checkboxes should only call `setCheckedFilters`.
    // The reactive `useEffect` will then pick up `checkedFilters` and call `buildAndDisplayQueryGroups`.
    // The original `updateFilterGroups` which directly manipulated `filterGroups` for checkboxes is removed.
  }, []);


  // Simplified handleFormValuesChange - now mostly for policy fetching logic trigger
  const handleFormValuesChange = async (changedValues: any) => {
    if (changedValues.resourcePoolType || changedValues.actionType) {
      setDevices([]);
    }
  };


  // Re-define handleSubmit
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (devices.length === 0) {
        message.error('请先选择设备');
        return;
      }
      const formData = {
        ...values,
        devices: devices.map(device => device.id),
        deviceCount: devices.length // Use actual selected device count
      };
      setLoading(true);
      await onSubmit(formData);
      // onCancel(); // Optionally close modal on success
    } catch (error) {
      console.error('表单验证失败或提交失败:', error);
      // message.error('订单创建失败，请检查表单信息。'); // Provide user feedback
    } finally {
      setLoading(false);
    }
  };

  const handleSearchDevices = async () => {
    const currentClusterId = form.getFieldValue('clusterId');   // Use watched value if available

    if (!currentClusterId) {
      message.error('请先选择集群');
      return;
    }
    const currentCluster = clusters.find(c => c.id === currentClusterId);
    if (!currentCluster) {
      message.error('选择的集群无效');
      return;
    }
    const currentActionType = form.getFieldValue('actionType'); // Use watched value if available

    let finalQueryGroups: FilterGroup[] = [];
    if (selectedPolicy && selectedPolicy.queryGroups) {
      finalQueryGroups = JSON.parse(JSON.stringify(selectedPolicy.queryGroups));
      finalQueryGroups.forEach(group => {
        group.blocks?.forEach(block => {
          const conditionSymbol = mapConditionTypeToSymbol(block.conditionType);
          if (block.type === FilterType.Device) {
            const keyForDevice = block.key || block.field;
            if (keyForDevice) {
              block.label = `${keyForDevice} ${conditionSymbol} ${block.value || ''}`.trim();
            } else if (!block.label) {
              block.label = `Device Attr: ${conditionSymbol} ${block.value || 'N/A'}`;
            }
          } 
          else if (!block.label) {
            if (block.type === FilterType.NodeLabel && block.key) {
              block.label = `label:${block.key} ${conditionSymbol} ${block.value || ''}`.trim();
            } else if (block.type === FilterType.Taint && block.key) {
              block.label = `taint:${block.key} ${conditionSymbol} ${block.value || ''}`.trim();
            } else {
              block.label = `Policy Condition (Type: ${block.type || 'N/A'})`;
            }
          }
        });
      });
    }

    // Add / ensure "同集群" for pool_exit
    if (currentActionType === 'pool_exit') { // use watched actionType
      const clusterName = currentCluster.name || currentCluster.clusterName || currentCluster.clusterNameCn || currentCluster.alias || `集群-${currentCluster.id}`;
      const poolExitBlock: FilterBlock = { id: uuidv4(), type: FilterType.Device, field: 'cluster', conditionType: ConditionType.Equal, value: clusterName, operator: LogicalOperator.And, label: '同集群' };
      if (!finalQueryGroups.some(g => g.blocks.some(b => b.field === 'cluster' && b.conditionType === ConditionType.Equal && b.label === '同集群'))) {
         finalQueryGroups.push({ id: uuidv4(), blocks: [poolExitBlock], operator: LogicalOperator.And });
      }
    } 
    // Add / ensure "未入池设备" for pool_entry
    else { 
      const unpooledDeviceBlock: FilterBlock = { id: uuidv4(), type: FilterType.Device, field: 'cluster', conditionType: ConditionType.IsEmpty, operator: LogicalOperator.And, label: '未入池设备' };
      if (!finalQueryGroups.some(g => g.blocks.some(b => b.field === 'cluster' && b.conditionType === ConditionType.IsEmpty && b.label === '未入池设备'))) {
        finalQueryGroups.unshift({ id: uuidv4(), blocks: [unpooledDeviceBlock], operator: LogicalOperator.And });
      }
      // Add location-based filters from checkboxes
      const additionalBlocksFromCheckboxes: FilterBlock[] = [];
       if (checkedFilters.idc && currentCluster.idc) {
        additionalBlocksFromCheckboxes.push({ id: uuidv4(), type: FilterType.Device, field: 'idc', conditionType: ConditionType.Equal, value: currentCluster.idc, operator: LogicalOperator.And, label: `idc = ${currentCluster.idc}` });
      }
      if (checkedFilters.zone && currentCluster.zone) {
        additionalBlocksFromCheckboxes.push({ id: uuidv4(), type: FilterType.Device, field: 'zone', conditionType: ConditionType.Equal, value: currentCluster.zone, operator: LogicalOperator.And, label: `zone = ${currentCluster.zone}` });
      }
      const roomValue = currentCluster.room || currentCluster.idc || '';
      if (checkedFilters.room && roomValue) {
        additionalBlocksFromCheckboxes.push({ id: uuidv4(), type: FilterType.Device, field: 'room', conditionType: ConditionType.Equal, value: roomValue, operator: LogicalOperator.And, label: `room = ${roomValue}` });
      }
      if (additionalBlocksFromCheckboxes.length > 0) {
          finalQueryGroups.push({ id: uuidv4(), blocks: additionalBlocksFromCheckboxes, operator: LogicalOperator.And });
      }
    }
    
    setFilterGroups(finalQueryGroups);

    try {
      setSearchingDevices(true);
      const response = await queryDevices({ groups: finalQueryGroups, page: 1, size: 20 });
      
      // 这里只是获取初始数据，完整分页逻辑在DeviceSelectionDrawer中处理
      const initialDevices = response.list || [];
      setDevices(initialDevices);
      
      // 将查询条件传递给抽屉组件
      setDrawerVisible(true);
    } catch (error) {
      console.error('查询设备失败:', error);
      message.error('查询设备失败');
    } finally {
      setSearchingDevices(false);
    }
  };

  // 处理设备选择
  const handleSelectDevices = (selectedDevices: Device[]) => {
    setDevices(selectedDevices);
    message.success(`已选择 ${selectedDevices.length} 台设备`);
  };

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
      width={800}
      okText="确定" // 修改按钮文本
      cancelText="取消" // 修改按钮文本
      style={{ 
        zIndex: drawerVisible ? 999 : 1000 
      }}
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
            <Select
              placeholder="请选择集群"
              showSearch
              optionFilterProp="children"
              filterOption={(input, option) =>
                (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
              }
              showArrow
              style={{ width: '100%' }}
            >
              {clusters.map(cluster => (
                <Option key={cluster.id} value={cluster.id}>
                  <ClusterOutlined style={{ marginRight: 4, color: '#52c41a' }} />
                  {cluster.name || cluster.clusterName || cluster.clusterNameCn || cluster.alias || `集群-${cluster.id}`}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="resourcePoolType"
            label="资源池类型"
            rules={[{ required: true, message: '请选择资源池类型' }]}
          >
            <Select
              placeholder="请选择资源池类型"
              showSearch
              optionFilterProp="children"
              filterOption={(input, option) =>
                (option?.children?.toString().toLowerCase().indexOf(input.toLowerCase()) ?? -1) >= 0
              }
              showArrow
              style={{ width: '100%' }}
            >
              {resourcePools.map(pool => (
                <Option key={pool.type} value={pool.type}>
                  <DatabaseOutlined style={{ marginRight: 4, color: '#1890ff' }} />
                  {pool.type === 'compute' ? '' : pool.name}
                </Option>
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
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <DatabaseOutlined style={{ marginRight: 8, color: '#722ed1' }} />
              <span style={{ fontSize: '14px', fontWeight: 500 }}>设备选择</span>
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
                {selectedPolicy ? (
                  <>
                    <div style={{ marginBottom: 16 }}>
                      <div style={{ marginBottom: 8, fontWeight: 500 }}>
                        <span>设备匹配策略：</span>
                        <Tag color="blue" style={{ marginLeft: 4 }}>{selectedPolicy?.name || '未命名策略'}</Tag>
                      </div>

                      {/* 显示额外动态条件 (additionConds) */}
                      {form.getFieldValue('actionType') === 'pool_entry' && (
                        <div style={{ marginTop: 16 }}>
                          <div style={{ marginBottom: 8, fontWeight: 500 }}>额外动态条件：</div>
                          <div className="info-box" style={{ 
                            backgroundColor: '#f0f7ff', 
                            border: '1px solid #d6e8ff', 
                            borderRadius: '8px', 
                            padding: '12px', 
                            marginBottom: '16px' 
                          }}>
                            <div style={{ display: 'flex', alignItems: 'center', marginBottom: '8px' }}>
                              <div style={{ 
                                backgroundColor: '#1677ff', 
                                color: 'white', 
                                borderRadius: '50%', 
                                width: '20px', 
                                height: '20px', 
                                display: 'flex', 
                                justifyContent: 'center', 
                                alignItems: 'center',
                                marginRight: '8px'
                              }}>
                                i
                              </div>
                              <span>这些条件将在入池时自动添加到查询条件中</span>
                            </div>
                            <div style={{ marginLeft: '28px', color: '#666' }}>
                              选中的条件将确保设备与目标集群的位置信息匹配
                            </div>
                          </div>
                          <div>
                            <Row>
                              <Col span={8}>
                                <Checkbox 
                                  checked={checkedFilters.idc}
                                  style={{ fontSize: '14px', padding: '8px 0' }}
                                  onChange={(e) => {
                                    // 更新选中状态
                                    const newCheckedFilters = {
                                      ...checkedFilters,
                                      idc: e.target.checked
                                    };
                                    setCheckedFilters(newCheckedFilters);
                                    
                                    // 更新筛选条件
                                    const clusterId = form.getFieldValue('clusterId');
                                    const selectedCluster = clusters.find(c => c.id === clusterId);
                                    if (selectedCluster) {
                                      updateFilterGroups(newCheckedFilters, selectedCluster);
                                    }
                                  }}
                                >
                                  同集群IDC
                                </Checkbox>
                              </Col>
                              <Col span={8}>
                                <Checkbox 
                                  checked={checkedFilters.zone}
                                  style={{ fontSize: '14px', padding: '8px 0' }}
                                  onChange={(e) => {
                                    // 更新选中状态
                                    const newCheckedFilters = {
                                      ...checkedFilters,
                                      zone: e.target.checked
                                    };
                                    setCheckedFilters(newCheckedFilters);
                                    
                                    // 更新筛选条件
                                    const clusterId = form.getFieldValue('clusterId');
                                    const selectedCluster = clusters.find(c => c.id === clusterId);
                                    if (selectedCluster) {
                                      updateFilterGroups(newCheckedFilters, selectedCluster);
                                    }
                                  }}
                                >
                                  同集群Zone
                                </Checkbox>
                              </Col>
                              <Col span={8}>
                                <Checkbox 
                                  checked={checkedFilters.room}
                                  style={{ fontSize: '14px', padding: '8px 0' }}
                                  onChange={(e) => {
                                    // 更新选中状态
                                    const newCheckedFilters = {
                                      ...checkedFilters,
                                      room: e.target.checked
                                    };
                                    setCheckedFilters(newCheckedFilters);
                                    
                                    // 更新筛选条件
                                    const clusterId = form.getFieldValue('clusterId');
                                    const selectedCluster = clusters.find(c => c.id === clusterId);
                                    if (selectedCluster) {
                                      updateFilterGroups(newCheckedFilters, selectedCluster);
                                    }
                                  }}
                                >
                                  同集群Room
                                </Checkbox>
                              </Col>
                            </Row>
                          </div>
                          
                          {/* 已选条件卡片 */}
                          {filterGroups.length > 0 && (
                            <div style={{ marginTop: 16 }}>
                              <div style={{ marginBottom: 8, fontWeight: 500 }}>已选条件：</div>
                              <Card size="small" style={{ marginBottom: 16 }}>
                                <div style={{ margin: '8px 0' }}>
                                  {filterGroups.flatMap(group => group.blocks).map((block) => {
                                    // 根据筛选块类型显示不同的内容
                                    let blockContent = '';
                                    let tagColor = 'default';
    
                                    // 特殊处理入池和退池条件的显示
                                    const actionType = form.getFieldValue('actionType');
                                    if (block.type === FilterType.Device) {
                                      // 优先使用自定义标签
                                      if (block.label) {
                                        blockContent = block.label;
                                        tagColor = 'blue';
                                      } else if (block.field === 'cluster') {
                                        if (actionType === 'pool_entry' && block.conditionType === ConditionType.IsEmpty) {
                                          blockContent = '未入池设备';
                                          tagColor = 'blue';
                                        } else if (actionType === 'pool_exit' && block.conditionType === ConditionType.Equal) {
                                          blockContent = '已入池设备';
                                          tagColor = 'orange';
                                        } else {
                                          // 常规集群字段处理
                                          const fieldLabel = filterOptions.deviceFields?.find((f: any) => f.value === block.field)?.label || block.field;
                                          const conditionLabel = block.conditionType === ConditionType.Equal ? '等于' :
                                                                block.conditionType === ConditionType.NotEqual ? '不等于' :
                                                                block.conditionType === ConditionType.Contains ? '包含' :
                                                                block.conditionType === ConditionType.IsEmpty ? '为空' :
                                                                block.conditionType === ConditionType.IsNotEmpty ? '不为空' :
                                                                block.conditionType;
                                          blockContent = `${fieldLabel || block.field} ${conditionLabel} ${block.value || ''}`;
                                          tagColor = 'blue';
                                        }
                                      } else if (['idc', 'zone', 'room'].includes(block.field || '') && block.conditionType === ConditionType.Equal) {
                                        blockContent = `${block.field} = ${block.value}`;
                                        tagColor = 'blue';
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
                                        blockContent = `${fieldLabel || block.field} ${conditionLabel} ${block.value || ''}`;
                                        tagColor = 'blue';
                                      }
                                    } else if (block.type === FilterType.NodeLabel) {
                                      blockContent = `标签 ${block.key || ''} ${block.conditionType} ${block.value || ''}`;
                                      tagColor = 'green';
                                    } else if (block.type === FilterType.Taint) {
                                      blockContent = `污点 ${block.key || ''} ${block.conditionType} ${block.value || ''}`;
                                      tagColor = 'orange';
                                    }
    
                                    return (
                                      <Tag
                                        key={block.id}
                                        color={tagColor}
                                        style={{ margin: '4px' }}
                                      >
                                        {['idc', 'zone', 'room'].includes(block.field || '') && block.conditionType === ConditionType.Equal 
                                          ? `${block.field} = ${block.value}` 
                                          : (blockContent || '无参数')}
                                      </Tag>
                                    );
                                  })}
                                </div>
                              </Card>
                            </div>
                          )}
                        </div>
                      )}
                    </div>

                    <Button
                      type="primary"
                      icon={<SearchOutlined />}
                      onClick={handleSearchDevices}
                      loading={searchingDevices}
                      style={{ width: '100%' }}
                    >
                      查询设备
                    </Button>
                  </>
                ) : (
                  <Alert
                    message="未找到匹配的设备策略"
                    description="当前资源池类型和动作类型没有可用的设备匹配策略，请尝试选择其他资源池类型或动作类型。"
                    type="info"
                    showIcon
                  />
                )}
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
                    {form.getFieldValue('actionType') === 'pool_exit' ? (
                      <div style={{ padding: '12px 0' }}>
                        <Tag color="blue" style={{ fontSize: '14px', padding: '4px 8px' }}>
                          <CheckCircleOutlined style={{ marginRight: 8 }} />
                          同集群（默认）
                        </Tag>
                        <div style={{ marginTop: 12, color: '#666', marginBottom: 16 }}>
                          退池操作默认只能选择同集群设备
                        </div>
                        <Button
                          type="primary"
                          icon={<SearchOutlined />}
                          onClick={handleSearchDevices}
                          loading={searchingDevices}
                          style={{ width: '100%' }}
                        >
                          查询设备
                        </Button>
                      </div>
                    ) : (
                      <>
                        <div style={{ marginBottom: 16 }}>
                          <Row>
                            <Col span={8}>
                              <Checkbox 
                                checked={checkedFilters.idc}
                                style={{ fontSize: '14px', padding: '8px 0' }}
                                onChange={(e) => {
                                  // 更新选中状态
                                  const newCheckedFilters = {
                                    ...checkedFilters,
                                    idc: e.target.checked
                                  };
                                  setCheckedFilters(newCheckedFilters);
                                  
                                  // 更新筛选条件
                                  const clusterId = form.getFieldValue('clusterId');
                                  const selectedCluster = clusters.find(c => c.id === clusterId);
                                  if (selectedCluster) {
                                    updateFilterGroups(newCheckedFilters, selectedCluster);
                                  }
                                }}
                              >
                                同集群IDC
                              </Checkbox>
                            </Col>
                            <Col span={8}>
                              <Checkbox 
                                checked={checkedFilters.zone}
                                style={{ fontSize: '14px', padding: '8px 0' }}
                                onChange={(e) => {
                                  // 更新选中状态
                                  const newCheckedFilters = {
                                    ...checkedFilters,
                                    zone: e.target.checked
                                  };
                                  setCheckedFilters(newCheckedFilters);
                                  
                                  // 更新筛选条件
                                  const clusterId = form.getFieldValue('clusterId');
                                  const selectedCluster = clusters.find(c => c.id === clusterId);
                                  if (selectedCluster) {
                                    updateFilterGroups(newCheckedFilters, selectedCluster);
                                  }
                                }}
                              >
                                同集群Zone
                              </Checkbox>
                            </Col>
                            <Col span={8}>
                              <Checkbox 
                                checked={checkedFilters.room}
                                style={{ fontSize: '14px', padding: '8px 0' }}
                                onChange={(e) => {
                                  // 更新选中状态
                                  const newCheckedFilters = {
                                    ...checkedFilters,
                                    room: e.target.checked
                                  };
                                  setCheckedFilters(newCheckedFilters);
                                  
                                  // 更新筛选条件
                                  const clusterId = form.getFieldValue('clusterId');
                                  const selectedCluster = clusters.find(c => c.id === clusterId);
                                  if (selectedCluster) {
                                    updateFilterGroups(newCheckedFilters, selectedCluster);
                                  }
                                }}
                              >
                                同集群Room
                              </Checkbox>
                            </Col>
                          </Row>
                        </div>
                        <div style={{ marginTop: 16 }}>
                          {filterGroups.length > 0 ? (
                            filterGroups.map((group, groupIndex) => (
                              <div key={group.id} className="filter-group">
                                <div className="filter-group-header">
                                  <span>筛选组 {groupIndex + 1}</span>
                                </div>
                                <div style={{ margin: '8px 0' }}>
                                  {group.blocks.map((block) => {
                                    // 根据筛选块类型显示不同的内容
                                    let blockContent = '';
                                    let tagColor = 'default';

                                    // 特殊处理入池和退池条件的显示
                                    const actionType = form.getFieldValue('actionType');
                                    if (block.type === FilterType.Device) {
                                      // 优先使用自定义标签
                                      if (block.label) {
                                        blockContent = block.label;
                                        tagColor = 'blue';
                                      } else if (block.field === 'cluster') {
                                        if (actionType === 'pool_entry' && block.conditionType === ConditionType.IsEmpty) {
                                          blockContent = '未入池设备';
                                          tagColor = 'blue';
                                        } else if (actionType === 'pool_exit' && block.conditionType === ConditionType.Equal) {
                                          blockContent = '已入池设备';
                                          tagColor = 'orange';
                                        } else {
                                          // 常规集群字段处理
                                          const fieldLabel = filterOptions.deviceFields?.find((f: any) => f.value === block.field)?.label || block.field;
                                          const conditionLabel = block.conditionType === ConditionType.Equal ? '等于' :
                                                                block.conditionType === ConditionType.NotEqual ? '不等于' :
                                                                block.conditionType === ConditionType.Contains ? '包含' :
                                                                block.conditionType === ConditionType.IsEmpty ? '为空' :
                                                                block.conditionType === ConditionType.IsNotEmpty ? '不为空' :
                                                                block.conditionType;
                                          blockContent = `${fieldLabel || block.field} ${conditionLabel} ${block.value || ''}`;
                                          tagColor = 'blue';
                                        }
                                      } else if (['idc', 'zone', 'room'].includes(block.field || '') && block.conditionType === ConditionType.Equal) {
                                        blockContent = `${block.field} = ${block.value}`;
                                        tagColor = 'blue';
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
                                        blockContent = `${fieldLabel || block.field} ${conditionLabel} ${block.value || ''}`;
                                        tagColor = 'blue';
                                      }
                                    } else if (block.type === FilterType.NodeLabel) {
                                      blockContent = `标签 ${block.key || ''} ${block.conditionType} ${block.value || ''}`;
                                      tagColor = 'green';
                                    } else if (block.type === FilterType.Taint) {
                                      blockContent = `污点 ${block.key || ''} ${block.conditionType} ${block.value || ''}`;
                                      tagColor = 'orange';
                                    }

                                    return (
                                                                              <Tag
                                          key={block.id}
                                          color={tagColor}
                                          style={{ margin: '4px' }}
                                        >
                                          {['idc', 'zone', 'room'].includes(block.field || '') && block.conditionType === ConditionType.Equal 
                                            ? `${block.field} = ${block.value}` 
                                            : blockContent}
                                        </Tag>
                                    );
                                  })}
                                </div>
                              </div>
                            ))
                          ) : (
                            <Empty
                              description="暂无筛选条件"
                              image={Empty.PRESENTED_IMAGE_SIMPLE}
                            />
                          )}
                        </div>
                        <Button
                          type="primary"
                          icon={<SearchOutlined />}
                          style={{ width: '100%', marginTop: '16px' }}
                          onClick={() => setDrawerVisible(true)}
                        >
                          查询设备
                        </Button>
                      </>
                    )}
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
        appliedFilters={filterGroups} // 将filterGroups传递给appliedFilters
        selectedDevices={devices}
        loading={loading}
        simpleMode={useSimpleMode}
      />

      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <DatabaseOutlined style={{ marginRight: 8, color: '#1890ff' }} />
            <span style={{ fontSize: '14px', fontWeight: 500 }}>设备列表</span>
          </div>
        }
        size="small"
        style={{ marginBottom: '24px' }}
        headStyle={{ backgroundColor: '#f5f7fa', padding: '0 16px' }} // Adjusted padding for headStyle
        bodyStyle={{ padding: '0px 24px 16px 24px' }} // Adjusted padding for bodyStyle
                  extra={
            <Space align="center">
              <Tag color="success" style={{ fontWeight: 600 }}>
                已选择 {devices.length} 台设备
              </Tag>
              {devices.length > 0 && (
                <Button 
                  size="small" 
                  type="text"
                  danger 
                  icon={<DeleteOutlined />}
                  onClick={() => setDevices([])}
                  style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '0 4px' }}
                />
              )}
            </Space>
          }
      >
        {searchingDevices ? (
          <div style={{ 
            textAlign: 'center', 
            padding: '30px 0'
          }}>
            <Spin tip="正在查询设备..." />
          </div>
        ) : devices.length === 0 ? (
          <Empty
            description="暂无选择的设备，请使用上方筛选条件查询设备"
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            style={{ margin: '24px 0' }}
          />
        ) : (
          <div className="device-tags-container" style={{ 
            padding: '16px',
            background: '#fff',
            border: 'none',
            boxShadow: 'none',
            borderRadius: '0',
            minHeight: 'auto'
          }}>
            <div style={{ 
              display: 'flex', 
              flexWrap: 'wrap',
              margin: '8px 0'
            }}>
              {devices.map((device) => {
                // Determine background color based on device properties
                let bgColor = '#f5f7fa'; // Default background
                
                // Special device (has group or special features)
                if (device.isSpecial || 
                    (device.group && device.group.trim() !== '') || 
                    (device.featureCount && device.featureCount > 0)) {
                  bgColor = '#fffbe6'; // Light yellow for special devices
                }
                // Has cluster but not special
                else if (device.cluster && device.cluster.trim() !== '') {
                  bgColor = '#f6ffed'; // Light green for cluster devices
                }
                
                return (
                  <Tag
                    key={device.id}
                    className="device-tag"
                    style={{
                      margin: '0 8px 8px 0',
                      padding: '8px 12px',
                      paddingRight: '24px', // Add space for the close button
                      borderRadius: '4px',
                      backgroundColor: bgColor,
                      border: '1px solid #d9d9d9',
                      color: '#595959',
                      boxShadow: '0 2px 0 rgba(0, 0, 0, 0.015)',
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'flex-start',
                      position: 'relative' // Needed for absolute positioning of the close icon
                    }}
                  >
                    <span style={{ fontWeight: 'bold', fontSize: '13px', color: '#333' }}>{device.ciCode || '未知设备'}</span>
                    <span style={{ fontSize: '11px', color: '#666', marginTop: '3px' }}>{device.ip || '未知IP'}</span>
                    <CloseOutlined
                      style={{
                        position: 'absolute',
                        top: '6px', // Adjust as needed
                        right: '6px', // Adjust as needed
                        fontSize: '10px', // Adjust as needed
                        color: '#8c8c8c', // Adjust as needed
                        cursor: 'pointer'
                      }}
                      onClick={(e) => {
                        e.stopPropagation(); // Prevent tag click event if any
                        setDevices(prevDevices => prevDevices.filter(d => d.id !== device.id));
                      }}
                    />
                  </Tag>
                );
              })}
            </div>

          </div>
        )}

      </Card>
    </Modal>
  );
});

export default CreateOrderModal;
