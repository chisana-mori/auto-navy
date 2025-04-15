import React from 'react';
import { Tag, Space, Typography } from 'antd';
import { FilterGroup } from '../../types/deviceQuery';

const { Text } = Typography;

interface QuerySummaryProps {
  mode: 'simple' | 'advanced' | 'template';
  simpleKeyword?: string;
  advancedGroups?: FilterGroup[];
  templateName?: string;
  resultCount: number;
  lastUpdated: Date | null;
}

const QuerySummary: React.FC<QuerySummaryProps> = ({
  mode,
  simpleKeyword,
  advancedGroups,
  templateName,
  resultCount,
  lastUpdated,
}) => {
  if (!lastUpdated) {
    return null; // 没有执行过查询，不显示摘要
  }

  const renderSummary = () => {
    switch (mode) {
      case 'simple':
        return (
          <Space>
            <Text>关键字:</Text>
            <Tag color="blue">{simpleKeyword || '(空)'}</Tag>
          </Space>
        );
      case 'advanced':
        return (
          <Space>
            <Text>高级查询:</Text>
            <Tag color="purple">{advancedGroups?.length || 0} 个条件组</Tag>
          </Space>
        );
      case 'template':
        return (
          <Space>
            <Text>查询模板:</Text>
            <Tag color="green">{templateName}</Tag>
          </Space>
        );
      default:
        return null;
    }
  };

  return (
    <div className="query-summary" style={{ marginBottom: 16, padding: '8px 12px', background: '#f5f5f5', borderRadius: 4 }}>
      <Space size="large">
        {renderSummary()}
        <Text>
          查询结果: <Text strong>{resultCount}</Text> 条记录
        </Text>
        <Text type="secondary">
          更新时间: {lastUpdated.toLocaleString()}
        </Text>
      </Space>
    </div>
  );
};

export default QuerySummary;
