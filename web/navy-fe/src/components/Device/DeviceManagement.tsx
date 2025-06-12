import React, { useEffect, useState, useCallback } from 'react';
import { Table, Card, Input, Button, message, Space, Modal, Select, Form, Tag, Spin } from 'antd';
import { CloudServerOutlined, DownloadOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table/interface';
import { getDeviceList, downloadDeviceExcel, updateDeviceGroup } from '../../services/deviceService';
import { getDeviceFieldValues } from '../../services/deviceQueryService';
import type { Device, DeviceQuery } from '../../types/device';
import { useNavigate } from 'react-router-dom';
import '../../styles/device-management.css';

const DeviceManagement: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Device[]>([]);
  const [pagination, setPagination] = useState<TablePaginationConfig>({
    current: 1,
    pageSize: 10,
    total: 0,
  });
  const [searchKeyword, setSearchKeyword] = useState('');
  const [isRoleModalVisible, setIsRoleModalVisible] = useState(false);
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);
  const [form] = Form.useForm();
  const [roleUpdateLoading, setRoleUpdateLoading] = useState(false);
  const [groupOptions, setGroupOptions] = useState<string[]>([]);
  const [loadingGroupOptions, setLoadingGroupOptions] = useState(false);

  // 获取设备列表数据
  const fetchData = useCallback(async (page = 1, size = 10, keyword = '') => {
    setLoading(true);
    try {
      const params: DeviceQuery = {
        page,
        size,
        keyword,
      };

      const response = await getDeviceList(params);

      if (response && response.list) {
        setData(response.list);
        setPagination({
          current: page,
          pageSize: size,
          total: response.total || 0,
        });
      } else {
        console.error('API返回数据格式不正确:', response);
        setData([]);
      }
    } catch (error) {
      console.error('获取数据失败:', error);
      message.error('获取设备列表失败');
      setData([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // 初始加载数据
  useEffect(() => {
    fetchData(1, 10);
  }, [fetchData]);

  // 处理表格分页、排序、筛选变化
  const handleTableChange = (newPagination: TablePaginationConfig) => {
    fetchData(
      newPagination.current || 1,
      newPagination.pageSize || 10,
      searchKeyword
    );
  };

  // 处理搜索
  const handleSearch = () => {
    fetchData(1, pagination.pageSize || 10, searchKeyword);
  };

  // 处理搜索框按下回车
  const handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  // 获取机器用途选项
  const fetchGroupOptions = async () => {
    if (groupOptions.length > 0) return; // 如果已经有选项，不再重复获取

    try {
      setLoadingGroupOptions(true);
      const values = await getDeviceFieldValues('group', 10000);
      setGroupOptions(values);
    } catch (error) {
      console.error('获取机器用途选项失败:', error);
      message.error('获取机器用途选项失败');
    } finally {
      setLoadingGroupOptions(false);
    }
  };

  // 打开用途标记对话框
  const showRoleModal = (device: Device) => {
    setSelectedDevice(device);
    // 确保当前用途被选中
    form.setFieldsValue({ group: device.group });
    fetchGroupOptions(); // 获取机器用途选项
    setIsRoleModalVisible(true);
  };

  // 关闭用途标记对话框
  const handleRoleModalCancel = () => {
    setIsRoleModalVisible(false);
    setSelectedDevice(null);
    form.resetFields();
  };

  // 提交用途更新
  const handleRoleUpdate = async () => {
    if (!selectedDevice) return;

    try {
      const values = await form.validateFields();
      setRoleUpdateLoading(true);

      const groupValue = values.group;
      await updateDeviceGroup(selectedDevice.id, groupValue);
      message.success('设备用途更新成功');

      // 如果是新的用途值，添加到选项中
      if (groupValue && !groupOptions.includes(groupValue)) {
        setGroupOptions(prev => [...prev, groupValue]);
      }

      // 刷新数据
      fetchData(pagination.current as number, pagination.pageSize as number, searchKeyword);
      handleRoleModalCancel();
    } catch (error) {
      console.error('更新设备用途失败:', error);
      message.error('更新设备用途失败');
    } finally {
      setRoleUpdateLoading(false);
    }
  };

  // 处理下载Excel
  const handleDownload = async () => {
    try {
      setLoading(true);
      const blob = await downloadDeviceExcel();

      // 创建URL并下载
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `设备信息_${new Date().toISOString().split('T')[0]}.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      message.success('设备信息导出成功');
    } catch (error) {
      console.error('导出设备信息失败:', error);
      message.error('导出设备信息失败');
    } finally {
      setLoading(false);
    }
  };

  // 表格列定义
  const columns: ColumnsType<Device> = [
    {
      title: '设备编码',
      dataIndex: 'ciCode',
      key: 'ciCode',
      width: 150,
      ellipsis: true,
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
      width: 130,
      ellipsis: true,
    },
    {
      title: '机器用途',
      dataIndex: 'group',
      key: 'group',
      width: 120,
      ellipsis: true,
      render: (text: string, record: Device) => (
        <Space>
          {text}
          <Button
            type="link"
            size="small"
            className="action-button"
            onClick={(e) => {
              e.stopPropagation();
              showRoleModal(record);
            }}
          >
            编辑
          </Button>
        </Space>
      ),
    },
    {
      title: '所属集群',
      dataIndex: 'cluster',
      key: 'cluster',
      width: 150,
      ellipsis: true,
    },
    {
      title: '集群角色',
      dataIndex: 'role',
      key: 'role',
      width: 100,
      ellipsis: true,
    },
    {
      title: 'CPU架构',
      dataIndex: 'archType',
      key: 'archType',
      width: 80,
      ellipsis: true,
    },
    {
      title: 'IDC',
      dataIndex: 'idc',
      key: 'idc',
      width: 80,
      ellipsis: true,
    },
    {
      title: 'ROOM',
      dataIndex: 'room',
      key: 'room',
      width: 120,
      ellipsis: { showTitle: false },
      render: (text) => (
        <span title={text}>{text}</span>
      ),
    },
    {
      title: '网络区域',
      dataIndex: 'netZone',
      key: 'netZone',
      width: 100,
      ellipsis: true,
    },
    {
      title: 'APPID',
      dataIndex: 'appId',
      key: 'appId',
      width: 100,
      ellipsis: true,
    },
    {
      title: '是否国产化',
      dataIndex: 'isLocalization',
      key: 'isLocalization',
      width: 100,
      ellipsis: true,
      render: (value: boolean) => (
        <Tag color={value ? 'green' : 'default'}>
          {value ? '是' : '否'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 80,
      align: 'center',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          className="action-button"
          onClick={() => navigate(`/device/${record.id}`)}
        >
          详情
        </Button>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <CloudServerOutlined style={{ marginRight: '12px', color: '#1677ff', fontSize: '18px' }} />
            <span>设备信息</span>
          </div>
        }
        className="device-management-card"
      >
        <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div className="search-container" style={{ display: 'flex', alignItems: 'center' }}>
            <Input
              placeholder="输入关键字搜索所有字段"
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              onKeyPress={handleKeyPress}
              prefix={<SearchOutlined />}
              allowClear
              style={{ width: 300 }}
            />
            <Button
              type="primary"
              onClick={handleSearch}
              style={{ marginLeft: 8 }}
            >
              搜索
            </Button>
          </div>
          <Space>
            <Button
              icon={<DownloadOutlined />}
              onClick={handleDownload}
              loading={loading}
            >
              下载
            </Button>
            <Button
              type="primary"
              icon={<ReloadOutlined />}
              onClick={() => fetchData(pagination.current || 1, pagination.pageSize || 10, searchKeyword)}
              loading={loading}
            >
              刷新
            </Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          pagination={{
            ...pagination,
            showTotal: (total) => `共 ${total} 条记录`,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            size: 'default',
            showQuickJumper: true,
          }}
          onChange={handleTableChange}
          size="middle"
          scroll={{ x: 1500 }}
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
            };
          }}
        />
      </Card>

      {/* 用途编辑对话框 */}
      <Modal
        title="编辑机器用途"
        open={isRoleModalVisible}
        onOk={handleRoleUpdate}
        onCancel={handleRoleModalCancel}
        confirmLoading={roleUpdateLoading}
        okText="确认"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="group"
            label="机器用途"
            rules={[]}
          >
            <Select
              placeholder="请选择或输入机器用途"
              showSearch
              allowClear
              loading={loadingGroupOptions}
              notFoundContent={loadingGroupOptions ? <Spin size="small" /> : null}
              optionFilterProp="children"
              filterOption={(input, option) =>
                (option?.children as unknown as string)?.toLowerCase().includes(input.toLowerCase())
              }
            >
              {/* 动态生成选项 */}
              {groupOptions.map(group => (
                <Select.Option key={group} value={group}>{group}</Select.Option>
              ))}

              {/* 如果当前用途不在选项中，添加当前用途作为选项 */}
              {selectedDevice?.group && groupOptions.indexOf(selectedDevice.group) === -1 ? (
                <Select.Option key={selectedDevice.group} value={selectedDevice.group}>
                  {selectedDevice.group}
                </Select.Option>
              ) : null}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DeviceManagement;
