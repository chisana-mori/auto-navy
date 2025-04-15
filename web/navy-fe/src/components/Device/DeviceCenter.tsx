import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Tabs, Card, Table, Button, message, Space, Tooltip, Modal, Form, Input, Tag, Typography, Select, Spin, Pagination } from 'antd';
import type { SelectProps } from 'antd/es/select';
import {
  DatabaseOutlined,
  DownloadOutlined,
  ReloadOutlined,
  EditOutlined,
  PlayCircleOutlined,
  DeleteOutlined
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { Device, DeviceListResponse } from '../../types/device';
import { FilterGroup } from '../../types/deviceQuery';
import { getDeviceList, downloadDeviceExcel, updateDeviceRole, updateDeviceGroup } from '../../services/deviceService';
import { queryDevices, getQueryTemplates, getQueryTemplate, getDeviceFieldValues, deleteQueryTemplate } from '../../services/deviceQueryService';
import { generateQuerySummary } from '../../utils/queryUtils';
import SimpleQueryPanel from './SimpleQueryPanel';
import QuerySummary from './QuerySummary';
import AdvancedQueryPanel from './AdvancedQueryPanel';
import '../../styles/device-center.css';

const { TabPane } = Tabs;
const { Paragraph } = Typography;
const { Option } = Select;

// 查询状态类型
type QueryState = {
  mode: 'simple' | 'advanced' | 'template';
  simpleParams: {
    keyword: string;
  };
  advancedParams: {
    groups: FilterGroup[];
    sourceTemplateId?: number;  // 模板来源ID
    sourceTemplateName?: string;  // 模板来源名称
  };
  templateParams: {
    templateId: number | null;
    templateName: string;
  };
  results: {
    devices: Device[];
    pagination: {
      current: number;
      pageSize: number;
      total: number;
    };
    loading: boolean;
    lastUpdated: Date | null;
  };
};

// 初始状态
const initialQueryState: QueryState = {
  mode: 'simple',
  simpleParams: {
    keyword: '',
  },
  advancedParams: {
    groups: [],
  },
  templateParams: {
    templateId: null,
    templateName: '',
  },
  results: {
    devices: [],
    pagination: {
      current: 1,
      pageSize: 10,
      total: 0,
    },
    loading: false,
    lastUpdated: null,
  },
};

const DeviceCenter: React.FC = () => {
  const navigate = useNavigate();
  const [queryState, setQueryState] = useState<QueryState>(initialQueryState);
  const [roleEditVisible, setRoleEditVisible] = useState(false);
  const [editingDevice, setEditingDevice] = useState<Device | null>(null);
  const [groupOptions, setGroupOptions] = useState<string[]>([]);
  const [loadingGroupOptions, setLoadingGroupOptions] = useState(false);
  const [templates, setTemplates] = useState<any[]>([]);
  const [templateSearchKeyword, setTemplateSearchKeyword] = useState('');
  const [templatePagination, setTemplatePagination] = useState({
    current: 1,
    pageSize: 8,
    total: 0,
  });
  const [roleForm] = Form.useForm();

  // 加载模板列表
  const loadTemplates = async () => {
    try {
      const templatesData = await getQueryTemplates();
      setTemplates(templatesData);
      setTemplatePagination(prev => ({
        ...prev,
        total: templatesData.length
      }));
    } catch (error) {
      console.error('加载模板列表失败:', error);
      message.error('加载模板列表失败');
    }
  };

  // 过滤模板
  const filterTemplates = (keyword: string) => {
    if (!keyword.trim()) {
      return templates;
    }

    const lowerKeyword = keyword.toLowerCase();
    return templates.filter(template =>
      template.name.toLowerCase().includes(lowerKeyword) ||
      (template.description && template.description.toLowerCase().includes(lowerKeyword))
    );
  };

  // 处理模板搜索
  const handleTemplateSearch = (value: string) => {
    setTemplateSearchKeyword(value);
    setTemplatePagination(prev => ({
      ...prev,
      current: 1, // 重置到第一页
    }));
  };

  // 处理模板分页变化
  const handleTemplatePageChange = (page: number, pageSize?: number) => {
    setTemplatePagination(prev => ({
      ...prev,
      current: page,
      pageSize: pageSize || prev.pageSize,
    }));
  };

  // 加载模板列表和初始设备数据
  useEffect(() => {
    const loadInitialData = async () => {
      try {
        // 加载模板列表
        await loadTemplates();

        // 加载初始设备列表（无查询条件）
        setQueryState(prev => ({
          ...prev,
          results: {
            ...prev.results,
            loading: true,
          }
        }));

        const response = await getDeviceList({
          page: 1,
          size: 10,
          keyword: '',
        });

        if (response) {
          setQueryState(prev => ({
            ...prev,
            results: {
              devices: response.list || [],
              pagination: {
                current: 1,
                pageSize: 10,
                total: response.total || 0,
              },
              loading: false,
              lastUpdated: new Date(),
            }
          }));
        }
      } catch (error) {
        console.error('Failed to load initial data:', error);
        setQueryState(prev => ({
          ...prev,
          results: {
            ...prev.results,
            loading: false,
          }
        }));
      }
    };

    loadInitialData();
  }, []);

  // 处理多行查询关键字
  const processMultilineKeyword = (keyword: string): string => {
    // 如果关键字中包含换行符，则将其分割为多个关键字
    if (keyword.includes('\n')) {
      // 分割成多行，过滤空行，并用空格连接
      const lines = keyword.split('\n')
        .map(line => line.trim())
        .filter(line => line.length > 0);

      if (lines.length > 1) {
        // 多行查询时，使用 OR 连接多个关键字
        return lines.join(' OR ');
      } else if (lines.length === 1) {
        // 只有一行非空内容
        return lines[0];
      }
    }

    // 如果没有换行符或所有行都是空的，返回原始关键字
    return keyword;
  };

  // 执行查询的通用函数
  const executeQuery = useCallback(async () => {
    const { mode, simpleParams, advancedParams, templateParams, results } = queryState;
    const { pagination } = results;

    setQueryState(prev => ({
      ...prev,
      results: {
        ...prev.results,
        loading: true,
      }
    }));

    try {
      let response: DeviceListResponse | undefined;

      switch (mode) {
        case 'simple':
          // 处理多行查询
          const processedKeyword = processMultilineKeyword(simpleParams.keyword);
          response = await getDeviceList({
            page: pagination.current,
            size: pagination.pageSize,
            keyword: processedKeyword,
          });
          break;
        case 'advanced':
          response = await queryDevices({
            groups: advancedParams.groups,
            page: pagination.current,
            size: pagination.pageSize,
          });
          break;
        case 'template':
          if (templateParams.templateId) {
            const template = await getQueryTemplate(templateParams.templateId);
            response = await queryDevices({
              groups: template.groups,
              page: pagination.current,
              size: pagination.pageSize,
            });
          }
          break;
      }

      if (response) {
        // 使用类型断言来确保 TypeScript 知道 response 不会是 undefined
        const safeResponse = response as DeviceListResponse;
        setQueryState(prev => ({
          ...prev,
          results: {
            devices: safeResponse.list || [],
            pagination: {
              ...prev.results.pagination,
              total: safeResponse.total || 0,
            },
            loading: false,
            lastUpdated: new Date(),
          }
        }));

        message.success('查询成功');
      }
    } catch (error) {
      console.error('查询失败:', error);
      message.error('查询失败');

      setQueryState(prev => ({
        ...prev,
        results: {
          ...prev.results,
          loading: false,
        }
      }));
    }
  }, [queryState]);

  // 处理Tab切换
  const handleTabChange = (activeKey: string) => {
    setQueryState(prev => ({
      ...prev,
      mode: activeKey as 'simple' | 'advanced' | 'template',
    }));
  };

  // 切换到模板标签页
  const switchToTemplateTab = () => {
    setQueryState(prev => ({
      ...prev,
      mode: 'template',
    }));
  };

  // 处理简单查询
  const handleSimpleQuery = () => {
    setQueryState(prev => ({
      ...prev,
      results: {
        ...prev.results,
        pagination: {
          ...prev.results.pagination,
          current: 1, // 重置到第一页
        },
      }
    }));

    executeQuery();
  };

  // 处理高级查询
  const handleAdvancedQuery = () => {
    // 打印查询条件，用于调试
    console.log('Advanced query groups:', advancedParams.groups);

    setQueryState(prev => ({
      ...prev,
      results: {
        ...prev.results,
        pagination: {
          ...prev.results.pagination,
          current: 1, // 重置到第一页
        },
      }
    }));

    executeQuery();
  };

  // 更新高级查询条件
  const handleAdvancedQueryChange = (groups: FilterGroup[]) => {
    setQueryState(prev => ({
      ...prev,
      advancedParams: {
        groups,
      }
    }));
  };

  // 处理模板查询
  const handleTemplateQuery = (templateId: number, templateName: string) => {
    setQueryState(prev => ({
      ...prev,
      templateParams: {
        templateId,
        templateName,
      },
      results: {
        ...prev.results,
        pagination: {
          ...prev.results.pagination,
          current: 1, // 重置到第一页
        },
      }
    }));

    executeQuery();
  };

  // 删除模板
  const handleDeleteTemplate = async (templateId: number, templateName: string) => {
    Modal.confirm({
      title: '删除查询模板',
      content: `确定要删除模板「${templateName}」吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await deleteQueryTemplate(templateId);
          message.success(`模板「${templateName}」已删除`);

          // 刷新模板列表
          await loadTemplates();

          // 如果当前正在使用该模板进行查询，则重置查询状态
          if (queryState.mode === 'template' && queryState.templateParams.templateId === templateId) {
            setQueryState(prev => ({
              ...prev,
              templateParams: {
                templateId: null,
                templateName: '',
              },
            }));
          }
        } catch (error) {
          console.error('删除模板失败:', error);
          message.error('删除模板失败');
        }
      },
    });
  };

  // 加载模板到高级查询
  const handleEditTemplate = async (templateId: number) => {
    try {
      const template = await getQueryTemplate(templateId);
      if (template && template.groups) {
        // 处理模板数据，确保每个筛选块都有key字段
        const processedGroups = template.groups.map(group => ({
          ...group,
          blocks: group.blocks.map(block => {
            // 如果有field但没有key，将field的值赋给key
            if (block.field && !block.key) {
              return { ...block, key: block.field };
            }
            // 如果有key但没有field，将key的值赋给field
            if (block.key && !block.field) {
              return { ...block, field: block.key };
            }
            return block;
          })
        }));

        // 切换到高级查询模式
        setQueryState(prev => ({
          ...prev,
          mode: 'advanced',
          advancedParams: {
            groups: processedGroups,
            sourceTemplateId: templateId,  // 添加模板来源ID
            sourceTemplateName: template.name  // 添加模板来源名称
          }
        }));

        message.success(`已加载模板「${template.name}」到高级查询，可以进行编辑`);
      } else {
        message.warning('模板数据不完整');
      }
    } catch (error) {
      console.error('加载模板失败:', error);
      message.error('加载模板失败');
    }
  };

  // 处理分页变化
  const handlePaginationChange = (page: number, pageSize?: number) => {
    setQueryState(prev => ({
      ...prev,
      results: {
        ...prev.results,
        pagination: {
          ...prev.results.pagination,
          current: page,
          pageSize: pageSize || prev.results.pagination.pageSize,
        },
      }
    }));

    executeQuery();
  };

  // 导出设备信息
  const handleExport = async () => {
    try {
      const data = await downloadDeviceExcel();
      const blob = new Blob([data], { type: 'text/csv' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `设备列表_${new Date().toISOString().split('T')[0]}.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      message.success('导出成功');
    } catch (error) {
      console.error('导出失败:', error);
      message.error('导出失败');
    }
  };

  // 获取机器用途选项
  const fetchGroupOptions = async () => {
    if (groupOptions.length > 0) return; // 如果已经有选项，不再重复获取

    try {
      setLoadingGroupOptions(true);
      const values = await getDeviceFieldValues('group');
      setGroupOptions(values);
    } catch (error) {
      console.error('获取机器用途选项失败:', error);
      message.error('获取机器用途选项失败');
    } finally {
      setLoadingGroupOptions(false);
    }
  };

  // 处理用途编辑
  const handleRoleEdit = (device: Device) => {
    setEditingDevice(device);
    roleForm.setFieldsValue({ group: device.group });
    fetchGroupOptions(); // 获取机器用途选项
    setRoleEditVisible(true);
  };

  // 保存用途编辑
  const handleRoleSave = async () => {
    try {
      const values = await roleForm.validateFields();
      if (!editingDevice) return;

      // 处理 tags 模式下的值，只取第一个值
      let groupValue = values.group;
      if (Array.isArray(groupValue) && groupValue.length > 0) {
        groupValue = groupValue[0];
      } else if (Array.isArray(groupValue) && groupValue.length === 0) {
        message.warning('请输入或选择机器用途');
        return;
      }

      await updateDeviceGroup(editingDevice.id, groupValue);
      message.success('用途更新成功');

      // 更新本地数据
      setQueryState(prev => ({
        ...prev,
        results: {
          ...prev.results,
          devices: prev.results.devices.map(device =>
            device.id === editingDevice.id
              ? { ...device, group: groupValue }
              : device
          ),
        }
      }));

      // 如果是新的用途值，添加到选项中
      if (!groupOptions.includes(groupValue)) {
        setGroupOptions(prev => [...prev, groupValue]);
      }

      setRoleEditVisible(false);
    } catch (error) {
      console.error('用途更新失败:', error);
      message.error('用途更新失败');
    }
  };

  // 表格列定义
  const columns: ColumnsType<Device> = [
    {
      title: '设备编码',
      dataIndex: 'ciCode',
      key: 'ciCode',
      render: (text: string, record: Device) => (
        <Button
          type="link"
          onClick={() => navigate(`/device/${record.id}/detail`)}
          style={{ padding: 0 }}
        >
          {text}
        </Button>
      ),
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
    },
    {
      title: '机器用途',
      dataIndex: 'group',
      key: 'group',
      render: (text: string, record: Device) => (
        <Space>
          {text}
          <Tooltip title="编辑用途">
            <Button
              type="link"
              icon={<EditOutlined />}
              size="small"
              onClick={(e) => {
                e.stopPropagation();
                handleRoleEdit(record);
              }}
            />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: '所属集群',
      dataIndex: 'cluster',
      key: 'cluster',
    },
    {
      title: '集群角色',
      dataIndex: 'role',
      key: 'role',
    },
    {
      title: 'CPU架构',
      dataIndex: 'archType',
      key: 'archType',
    },
    {
      title: 'IDC',
      dataIndex: 'idc',
      key: 'idc',
    },
    {
      title: 'ROOM',
      dataIndex: 'room',
      key: 'room',
    },
    {
      title: '网络区域',
      dataIndex: 'netZone',
      key: 'netZone',
    },
    {
      title: 'APPID',
      dataIndex: 'appId',
      key: 'appId',
    },
    {
      title: '是否国产化',
      dataIndex: 'isLocalization',
      key: 'isLocalization',
      render: (value: boolean) => (
        <Tag color={value ? 'green' : 'default'}>
          {value ? '是' : '否'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Device) => (
        <Button
          type="link"
          onClick={() => navigate(`/device/${record.id}/detail`)}
          style={{ padding: 0 }}
        >
          详情
        </Button>
      ),
    },
  ];

  const {
    mode,
    simpleParams,
    advancedParams,
    templateParams,
    results
  } = queryState;

  const { devices, pagination, loading, lastUpdated } = results;

  // 渲染模板列表
  const renderTemplateList = () => {
    // 搜索框和标题
    const renderSearchHeader = () => (
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Typography.Title level={5} style={{ margin: 0 }}>查询模板</Typography.Title>
        <Input.Search
          placeholder="搜索模板名称或描述"
          allowClear
          style={{ width: 300 }}
          value={templateSearchKeyword}
          onChange={(e) => handleTemplateSearch(e.target.value)}
          onSearch={handleTemplateSearch}
        />
      </div>
    );

    if (templates.length === 0) {
      return (
        <>
          {renderSearchHeader()}
          <div style={{ textAlign: 'center', padding: '20px 0' }}>
            <p>暂无保存的查询模板</p>
          </div>
        </>
      );
    }

    // 过滤模板
    const filteredTemplates = filterTemplates(templateSearchKeyword);

    // 计算分页
    const { current, pageSize, total } = templatePagination;
    const paginatedTemplates = filteredTemplates.slice(
      (current - 1) * pageSize,
      current * pageSize
    );

    // 更新总数
    if (filteredTemplates.length !== total) {
      // 使用setTimeout避免在渲染过程中更新状态
      setTimeout(() => {
        setTemplatePagination(prev => ({
          ...prev,
          total: filteredTemplates.length
        }));
      }, 0);
    }

    return (
      <>
        {renderSearchHeader()}

        {filteredTemplates.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '20px 0' }}>
            <p>没有找到匹配的模板</p>
          </div>
        ) : (
          <>
            <div className="template-list">
              {paginatedTemplates.map(template => {
                // 生成条件组的缩略信息
                const querySummary = generateQuerySummary(template.groups || [], 200);

                return (
                  <Card
                    key={template.id}
                    title={template.name}
                    className="template-card"
                    extra={
                      <Space>
                        <Tooltip title="编辑查询参数">
                          <Button
                            type="default"
                            size="small"
                            icon={<EditOutlined />}
                            onClick={() => handleEditTemplate(template.id)}
                          />
                        </Tooltip>
                        <Tooltip title="执行查询">
                          <Button
                            type="primary"
                            size="small"
                            icon={<PlayCircleOutlined />}
                            onClick={() => handleTemplateQuery(template.id, template.name)}
                          >
                            执行
                          </Button>
                        </Tooltip>
                        <Tooltip title="删除模板">
                          <Button
                            type="default"
                            danger
                            size="small"
                            icon={<DeleteOutlined />}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteTemplate(template.id, template.name);
                            }}
                          />
                        </Tooltip>
                      </Space>
                    }
                  >
                    {template.description && (
                      <Paragraph style={{ marginBottom: 8 }}>
                        <strong>描述：</strong>{template.description}
                      </Paragraph>
                    )}

                    <Paragraph style={{ marginBottom: 8 }}>
                      <strong>条件组：</strong>{template.groups?.length || 0}个
                    </Paragraph>

                    <Paragraph
                      ellipsis={{ rows: 2, expandable: true, symbol: '展开' }}
                      style={{ background: '#f5f5f5', padding: '8px', borderRadius: '4px', marginBottom: 0 }}
                    >
                      <strong>查询条件：</strong>{querySummary}
                    </Paragraph>
                  </Card>
                );
              })}
            </div>

            {/* 分页器 */}
            <div className="template-pagination">
              <Pagination
                current={current}
                pageSize={pageSize}
                total={filteredTemplates.length}
                onChange={handleTemplatePageChange}
                showSizeChanger
                pageSizeOptions={['4', '8', '12', '16']}
                showTotal={(total) => `共 ${total} 个模板`}
              />
            </div>
          </>
        )}
      </>
    );
  };

  return (
    <div className="device-center-container">
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <DatabaseOutlined style={{ marginRight: '12px', color: '#1677ff', fontSize: '18px' }} />
            <span>设备中心</span>
          </div>
        }
        extra={
          <div></div>
        }
        className="device-center-card"
      >
        <Tabs activeKey={mode} onChange={handleTabChange}>
          <TabPane tab="基本查询" key="simple">
            <SimpleQueryPanel
              keyword={simpleParams.keyword}
              onKeywordChange={(keyword) =>
                setQueryState(prev => ({
                  ...prev,
                  simpleParams: { keyword }
                }))
              }
              onSearch={handleSimpleQuery}
              loading={loading}
            />
          </TabPane>
          <TabPane tab="高级查询" key="advanced">
            <AdvancedQueryPanel
              filterGroups={advancedParams.groups}
              onFilterGroupsChange={handleAdvancedQueryChange}
              onQuery={handleAdvancedQuery}
              loading={loading}
              sourceTemplateId={advancedParams.sourceTemplateId}
              sourceTemplateName={advancedParams.sourceTemplateName}
              onTemplateSaved={loadTemplates} // 保存模板后刷新模板列表
              onSwitchToTemplateTab={switchToTemplateTab} // 切换到模板标签页
            />
          </TabPane>
          <TabPane tab="查询模板" key="template">
            {renderTemplateList()}
          </TabPane>
        </Tabs>

        {/* 查询摘要 */}
        <QuerySummary
          mode={mode}
          simpleKeyword={simpleParams.keyword}
          advancedGroups={advancedParams.groups}
          templateName={templateParams.templateName}
          resultCount={pagination.total}
          lastUpdated={lastUpdated}
        />

        {/* 只有当有查询结果时才显示设备列表和操作按钮 */}
        {devices.length > 0 && (
          <>
            {/* 设备列表标题和操作按钮 */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <div>
                <Typography.Title level={5} style={{ margin: 0 }}>设备列表</Typography.Title>
              </div>
              <Space>
                <Tooltip title="刷新数据">
                  <Button
                    icon={<ReloadOutlined />}
                    onClick={executeQuery}
                    loading={loading}
                  >
                    刷新
                  </Button>
                </Tooltip>
                <Tooltip title="导出设备信息">
                  <Button
                    icon={<DownloadOutlined />}
                    onClick={handleExport}
                  >
                    导出
                  </Button>
                </Tooltip>
              </Space>
            </div>

            {/* 设备列表 */}
            <Table
              columns={columns}
              dataSource={devices}
              rowKey="id"
              pagination={{
                current: pagination.current,
                pageSize: pagination.pageSize,
                total: pagination.total,
                onChange: handlePaginationChange,
                showSizeChanger: true,
                showQuickJumper: true,
              }}
              loading={loading}
            />
          </>
        )}
      </Card>

      {/* 用途编辑对话框 */}
      <Modal
        title="编辑机器用途"
        open={roleEditVisible}
        onOk={handleRoleSave}
        onCancel={() => setRoleEditVisible(false)}
        destroyOnClose
        className="group-edit-modal"
        centered
      >
        <Form form={roleForm} layout="vertical">
          <Form.Item
            name="group"
            label="机器用途"
            rules={[{ required: true, message: '请输入机器用途' }]}
          >
            <Select<string, { value: string; children: React.ReactNode }>
              showSearch
              allowClear
              loading={loadingGroupOptions}
              placeholder="请选择或输入机器用途"
              style={{ width: '100%' }}
              showArrow
              notFoundContent={loadingGroupOptions ? <Spin size="small" /> : null}
              mode="tags"
              maxTagCount={1}
              tokenSeparators={[',']}
              optionFilterProp="value"
              filterOption={(input, option) =>
                (option?.value ?? '').toString().toLowerCase().includes(input.toLowerCase())
              }
            >
              {groupOptions.map(group => (
                <Option key={group} value={group}>{group}</Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DeviceCenter;
