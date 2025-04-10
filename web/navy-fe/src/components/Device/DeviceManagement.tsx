import React, { useEffect, useState, useCallback } from 'react';
import { Table, Card, Input, Button, message, Space, Modal, Select, Form } from 'antd';
import { CloudServerOutlined, DownloadOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table/interface';
import { getDeviceList, downloadDeviceExcel, updateDeviceRole } from '../../services/deviceService';
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

  // 打开角色标记对话框
  const showRoleModal = (device: Device) => {
    setSelectedDevice(device);
    form.setFieldsValue({ role: device.role });
    setIsRoleModalVisible(true);
  };

  // 关闭角色标记对话框
  const handleRoleModalCancel = () => {
    setIsRoleModalVisible(false);
    setSelectedDevice(null);
    form.resetFields();
  };

  // 提交角色更新
  const handleRoleUpdate = async () => {
    if (!selectedDevice) return;

    try {
      const values = await form.validateFields();
      setRoleUpdateLoading(true);

      await updateDeviceRole(selectedDevice.id, values.role);
      message.success('设备角色更新成功');

      // 刷新数据
      fetchData(pagination.current as number, pagination.pageSize as number, searchKeyword);
      handleRoleModalCancel();
    } catch (error) {
      console.error('更新设备角色失败:', error);
      message.error('更新设备角色失败');
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
      title: '设备ID',
      dataIndex: 'deviceId',
      key: 'deviceId',
      width: 150,
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
      width: 120,
    },
    {
      title: '机器类型',
      dataIndex: 'machineType',
      key: 'machineType',
      width: 120,
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
      width: 100,
    },
    {
      title: '架构',
      dataIndex: 'arch',
      key: 'arch',
      width: 80,
    },
    {
      title: 'IDC',
      dataIndex: 'idc',
      key: 'idc',
      width: 80,
    },
    {
      title: 'Room',
      dataIndex: 'room',
      key: 'room',
      width: 100,
    },
    {
      title: '机房',
      dataIndex: 'datacenter',
      key: 'datacenter',
      width: 100,
    },
    {
      title: '机柜号',
      dataIndex: 'cabinet',
      key: 'cabinet',
      width: 80,
    },
    {
      title: '网络区域',
      dataIndex: 'network',
      key: 'network',
      width: 100,
    },
    {
      title: 'APPID',
      dataIndex: 'appId',
      key: 'appId',
      width: 100,
    },
    {
      title: '资源池/产品',
      dataIndex: 'resourcePool',
      key: 'resourcePool',
      width: 120,
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 150,
      render: (_, record) => (
        <div style={{ 
          display: 'flex',
          gap: '8px',
          justifyContent: 'flex-start',
          alignItems: 'center',
          height: '100%',
          padding: '0 8px',
          margin: '-12px -16px',
          minHeight: '46px'
        }}>
          <Button
            type="link"
            size="small"
            style={{ 
              padding: '4px 8px',
              height: '28px'
            }}
            onClick={() => navigate(`/device/${record.id}`)}
          >
            详情
          </Button>
          <Button
            type="link"
            size="small"
            style={{ 
              padding: '4px 8px',
              height: '28px'
            }}
            onClick={() => showRoleModal(record)}
          >
            标记
          </Button>
        </div>
      ),
    },
  ];

  return (
    <>
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <CloudServerOutlined style={{ marginRight: '12px', color: '#1677ff', fontSize: '18px' }} />
            <span>设备信息</span>
          </div>
        }
        className="device-management-card"
        extra={
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
        }
      >
        <div className="search-container">
          <Input
            placeholder="输入关键字搜索所有字段"
            value={searchKeyword}
            onChange={(e) => setSearchKeyword(e.target.value)}
            onKeyPress={handleKeyPress}
            prefix={<SearchOutlined />}
            allowClear
            style={{ width: 300, marginBottom: 0 }}
          />
          <Button
            type="primary"
            onClick={handleSearch}
            style={{ marginLeft: 8 }}
          >
            搜索
          </Button>
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
          scroll={{ x: 1500 }}
          size="middle"
        />
      </Card>

      {/* 角色标记对话框 */}
      <Modal
        title="标记设备角色"
        open={isRoleModalVisible}
        onOk={handleRoleUpdate}
        onCancel={handleRoleModalCancel}
        confirmLoading={roleUpdateLoading}
        okText="确认"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="role"
            label="角色"
            rules={[{ required: true, message: '请选择角色' }]}
          >
            <Select placeholder="请选择设备角色">
              <Select.Option value="x86">x86</Select.Option>
              <Select.Option value="ARM">ARM</Select.Option>
              <Select.Option value="master">master</Select.Option>
              <Select.Option value="worker">worker</Select.Option>
              <Select.Option value="etcd">etcd</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default DeviceManagement;
