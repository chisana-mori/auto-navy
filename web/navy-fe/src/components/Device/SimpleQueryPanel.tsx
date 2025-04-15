import React from 'react';
import { Input, Button, Space, Typography } from 'antd';
import { SearchOutlined, InfoCircleOutlined } from '@ant-design/icons';

const { TextArea } = Input;
const { Text } = Typography;

interface SimpleQueryPanelProps {
  keyword: string;
  onKeywordChange: (keyword: string) => void;
  onSearch: () => void;
  loading: boolean;
}

const SimpleQueryPanel: React.FC<SimpleQueryPanelProps> = ({
  keyword,
  onKeywordChange,
  onSearch,
  loading,
}) => {
  const handleKeyPress = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Ctrl+Enter 执行查询
    if (e.key === 'Enter' && e.ctrlKey) {
      onSearch();
    }
  };

  return (
    <div className="simple-query-panel">
      <Space direction="vertical" style={{ width: '100%', marginBottom: 16 }}>
        <TextArea
          placeholder="输入关键字搜索设备ID、IP、集群等字段
支持多行查询，每行一个条件，多个条件之间是 OR 关系
例如：
192.168.1.1
192.168.1.2
或者：
BJ
SZ"
          value={keyword}
          onChange={(e) => onKeywordChange(e.target.value)}
          onKeyDown={handleKeyPress}
          allowClear
          style={{ width: '100%', minHeight: 120 }}
          autoSize={{ minRows: 4, maxRows: 10 }}
        />
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Button
              onClick={() => {
                onKeywordChange('');
              }}
            >
              重置
            </Button>
            <Button
              type="primary"
              icon={<SearchOutlined />}
              onClick={onSearch}
              loading={loading}
            >
              搜索
            </Button>
          </Space>
          <Text type="secondary">
            <InfoCircleOutlined style={{ marginRight: 4 }} />
            提示: 多行查询时，每行一个条件，多个条件之间是 OR 关系。可以使用 Ctrl+Enter 快捷键执行查询。
          </Text>
        </div>
      </Space>
    </div>
  );
};

export default SimpleQueryPanel;
