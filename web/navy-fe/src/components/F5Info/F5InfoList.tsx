import { useEffect, useState, useCallback } from 'react';
import { Table, Space, Button, message, Popconfirm, Card, Input, Tag } from 'antd';
import { SearchOutlined, CheckCircleFilled, CloseCircleFilled, WarningFilled, CloudServerOutlined, ReloadOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table/interface';
import request from '../../utils/request';
import type { F5Info, F5InfoListResponse, F5InfoQuery } from '../../types/f5';
import Highlighter from 'react-highlight-words';

// 获取状态对应的图标和颜色
const getStatusInfo = (status: string) => {
  const lowerStatus = status.toLowerCase();
  if (lowerStatus === 'active' || lowerStatus === 'online' || lowerStatus === 'running' || lowerStatus === 'healthy') {
    return {
      icon: <CheckCircleFilled style={{ color: '#52c41a' }} />,
      color: '#f6ffed',
      textColor: '#52c41a',
      tagColor: 'success'
    };
  } else if (lowerStatus === 'inactive' || lowerStatus === 'offline' || lowerStatus === 'stopped') {
    return {
      icon: <CloseCircleFilled style={{ color: '#ff4d4f' }} />,
      color: '#fff1f0',
      textColor: '#ff4d4f',
      tagColor: 'error'
    };
  } else if (lowerStatus === 'degraded' || lowerStatus === 'warning') {
    return {
      icon: <WarningFilled style={{ color: '#faad14' }} />,
      color: '#fffbe6',
      textColor: '#faad14',
      tagColor: 'warning'
    };
  }
  
  return {
    icon: null,
    color: 'transparent',
    textColor: 'inherit',
    tagColor: 'default' as 'default'
  };
};

const F5InfoList: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<F5Info[]>([]);
  const [pagination, setPagination] = useState<TablePaginationConfig>({
    current: 1,
    pageSize: 10,
    total: 0,
  });
  const [filters, setFilters] = useState<Partial<F5InfoQuery>>({});
  const [searchText, setSearchText] = useState<Record<string, string>>({});
  const [searchedColumn, setSearchedColumn] = useState<string>('');
  const navigate = useNavigate();

  const fetchData = useCallback(async (page = 1, size = 10, query: Partial<F5InfoQuery> = {}) => {
    setLoading(true);
    try {
      const params: F5InfoQuery = {
        page,
        size,
        ...query
      };
      
      const response = await request.get<any, F5InfoListResponse>('/f5', {
        params
      });
      console.log('API响应数据:', response);
      
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
      message.error('获取数据失败');
      setData([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData(1, 10, filters);
  }, [fetchData, filters]);

  const handleIgnore = async (id: number, shouldIgnore: boolean) => {
    try {
      // 获取当前记录以保留必填字段
      const currentRecord = data.find(item => item.id === id);
      if (!currentRecord) {
        message.error('未找到记录信息');
        return;
      }
      
      // 构建更新数据，保留所有必填字段
      const updateData = {
        ignored: shouldIgnore,
        name: currentRecord.name,
        vip: currentRecord.vip,
        port: currentRecord.port,
        appid: currentRecord.appid,
        // 保留其他原始值，避免后端验证失败
        instance_group: currentRecord.instance_group,
        status: currentRecord.status,
        pool_name: currentRecord.pool_name,
        pool_status: currentRecord.pool_status,
        pool_members: currentRecord.pool_members,
        k8s_cluster_id: currentRecord.k8s_cluster_id,
        domains: currentRecord.domains,
        grafana_params: currentRecord.grafana_params
      };
      
      await request.put(`/f5/${id}`, updateData);
      message.success(shouldIgnore ? '已忽略' : '已取消忽略');
      fetchData(pagination.current || 1, pagination.pageSize || 10, filters);
    } catch (error) {
      console.error(shouldIgnore ? '忽略失败:' : '取消忽略失败:', error);
      message.error(shouldIgnore ? '忽略失败' : '取消忽略失败');
    }
  };

  const handleTableChange = (newPagination: TablePaginationConfig) => {
    if (newPagination.current && newPagination.pageSize) {
      fetchData(newPagination.current, newPagination.pageSize, filters);
    }
  };

  // 创建搜索筛选器
  const getColumnSearchProps = (dataIndex: keyof F5Info) => ({
    filterDropdown: ({ setSelectedKeys, selectedKeys, confirm, clearFilters }: any) => (
      <div style={{ padding: 8 }}>
        <Input
          placeholder={`搜索${dataIndex}`}
          value={selectedKeys[0]}
          onChange={e => setSelectedKeys(e.target.value ? [e.target.value] : [])}
          onPressEnter={() => {
            // 关闭筛选器下拉框
            confirm();
            
            // 更新搜索文本状态
            setSearchText({
              ...searchText,
              [dataIndex]: selectedKeys[0] as string
            });
            setSearchedColumn(dataIndex as string);

            // 更新过滤条件
            const newFilters = { ...filters } as any;
            if (selectedKeys[0]) {
              newFilters[dataIndex] = selectedKeys[0];
            } else {
              delete newFilters[dataIndex];
            }
            
            // 使用新的过滤条件直接调用fetchData，而不是等待状态更新
            setFilters(newFilters);
            fetchData(1, pagination.pageSize || 10, newFilters);
          }}
          style={{ width: 188, marginBottom: 8, display: 'block' }}
        />
        <Space>
          <Button
            type="primary"
            onClick={() => {
              // 关闭筛选器下拉框
              confirm();
              
              // 更新搜索文本状态
              setSearchText({
                ...searchText,
                [dataIndex]: selectedKeys[0] as string
              });
              setSearchedColumn(dataIndex as string);
              
              // 更新过滤条件
              const newFilters = { ...filters } as any;
              if (selectedKeys[0]) {
                newFilters[dataIndex] = selectedKeys[0];
              } else {
                delete newFilters[dataIndex];
              }
              
              // 使用新的过滤条件直接调用fetchData，而不是等待状态更新
              setFilters(newFilters);
              fetchData(1, pagination.pageSize || 10, newFilters);
            }}
            icon={<SearchOutlined />}
            size="small"
            style={{ width: 90 }}
          >
            搜索
          </Button>
          <Button
            onClick={() => {
              // 清空搜索框内容
              setSelectedKeys([]);
              // 调用表格的清除筛选器方法
              clearFilters && clearFilters();
              // 清除此列的搜索文本
              const newSearchText = { ...searchText };
              delete newSearchText[dataIndex as string];
              setSearchText(newSearchText);
              
              // 从过滤条件中移除此列
              const newFilters = { ...filters } as any;
              delete newFilters[dataIndex];
              
              // 使用新的过滤条件直接调用fetchData
              setFilters(newFilters);
              fetchData(1, pagination.pageSize || 10, newFilters);
              
              // 关闭筛选器下拉框
              confirm();
            }}
            size="small"
            style={{ width: 90 }}
          >
            重置
          </Button>
        </Space>
      </div>
    ),
    filterIcon: (filtered: boolean) => (
      <SearchOutlined style={{ color: filtered ? '#1890ff' : undefined }} />
    ),
    render: (text: any) =>
      searchedColumn === dataIndex ? (
        <Highlighter
          highlightStyle={{ backgroundColor: '#ffc069', padding: 0 }}
          searchWords={[searchText[dataIndex as string] || '']}
          autoEscape
          textToHighlight={text ? text.toString() : ''}
        />
      ) : (
        text
      ),
  });

  const columns: ColumnsType<F5Info> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 70,
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      ...getColumnSearchProps('name'),
      render: (text, record) => (
        <div style={{ fontWeight: 500 }}>{text}</div>
      ),
    },
    {
      title: 'VIP',
      dataIndex: 'vip',
      key: 'vip',
      ...getColumnSearchProps('vip'),
    },
    {
      title: '端口',
      dataIndex: 'port',
      key: 'port',
      width: 80,
      ...getColumnSearchProps('port'),
    },
    {
      title: 'appid',
      dataIndex: 'appid',
      key: 'appid',
      ...getColumnSearchProps('appid'),
    },
    {
      title: '实例组',
      dataIndex: 'instance_group',
      key: 'instance_group',
      ...getColumnSearchProps('instance_group'),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      ...getColumnSearchProps('status'),
      render: (status) => {
        const statusInfo = getStatusInfo(status);
        return (
          <Tag color={statusInfo.tagColor} icon={statusInfo.icon}>
            {status}
          </Tag>
        );
      },
      width: 120,
    },
    {
      title: 'Pool状态',
      dataIndex: 'pool_status',
      key: 'pool_status',
      ...getColumnSearchProps('pool_status'),
      render: (status) => {
        const statusInfo = getStatusInfo(status);
        return (
          <Tag color={statusInfo.tagColor} icon={statusInfo.icon}>
            {status}
          </Tag>
        );
      },
      width: 120,
    },
    {
      title: 'K8s集群',
      dataIndex: 'k8s_cluster_name',
      key: 'k8s_cluster_name',
      ...getColumnSearchProps('k8s_cluster_name'),
    },
    {
      title: '是否忽略',
      dataIndex: 'ignored',
      key: 'ignored',
      render: (ignored) => (
        <Tag color={ignored ? 'default' : 'green'}>
          {ignored ? '是' : '否'}
        </Tag>
      ),
      filters: [
        { text: '是', value: true },
        { text: '否', value: false },
      ],
      onFilter: (value, record) => record.ignored === value,
      width: 100,
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size={8}>
          <Button
            type="link"
            size="small"
            onClick={() => navigate(`/f5/${record.id}`)}
          >
            详情
          </Button>
          {record.ignored ? (
            <Popconfirm
              title="确定要取消忽略此条记录吗？"
              onConfirm={() => handleIgnore(record.id, false)}
              okText="确定"
              cancelText="取消"
            >
              <Button
                type="link"
                size="small"
                icon={<CheckCircleFilled />}
              >
                取消忽略
              </Button>
            </Popconfirm>
          ) : (
            <Popconfirm
              title="确定要忽略此条记录吗？"
              onConfirm={() => handleIgnore(record.id, true)}
              okText="确定"
              cancelText="取消"
            >
              <Button
                type="link"
                size="small"
                icon={<CloseCircleFilled />}
              >
                忽略
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
      width: 180,
      fixed: 'right',
    },
  ];

  return (
    <Card 
      title={
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <CloudServerOutlined style={{ marginRight: '12px', color: '#1677ff', fontSize: '18px' }} />
          <span>F5 信息列表</span>
        </div>
      } 
      className="f5-info-list-card"
      extra={
        <Button 
          type="primary"
          icon={<ReloadOutlined />}
          onClick={() => fetchData(pagination.current || 1, pagination.pageSize || 10, filters)}
        >
          刷新数据
        </Button>
      }
    >
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
        scroll={{ x: 1300 }}
        rowClassName={(record) => {
          return record.ignored ? 'ignored-row' : `status-${record.status.toLowerCase()}-row`;
        }}
        onRow={(record) => {
          const statusInfo = getStatusInfo(record.status);
          return {
            style: {
              backgroundColor: record.ignored ? '#fffbe6' : statusInfo.color,
              transition: 'background-color 0.3s',
            }
          };
        }}
      />
    </Card>
  );
};

export default F5InfoList;