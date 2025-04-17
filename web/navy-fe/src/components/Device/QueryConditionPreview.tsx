import React from 'react';
import { Tag, Space, Typography } from 'antd';
import { FilterGroup, FilterBlock, FilterType, ConditionType, LogicalOperator } from '../../types/deviceQuery';

const { Text } = Typography;

interface QueryConditionPreviewProps {
  groups: FilterGroup[];
  maxBlocks?: number; // 最多显示的条件块数量
  showGroupOperator?: boolean; // 是否显示组间操作符
}

const QueryConditionPreview: React.FC<QueryConditionPreviewProps> = ({
  groups,
  maxBlocks = 3,
  showGroupOperator = true,
}) => {
  // 调试输出
  console.log('QueryConditionPreview received groups:', groups);

  // 确保 groups 是数组
  if (!groups || !Array.isArray(groups) || groups.length === 0) {
    console.log('No valid groups found');
    return <Text type="secondary">无查询条件</Text>;
  }

  // 检查每个组是否有有效的 blocks
  const validGroups = groups.filter(group =>
    group && Array.isArray(group.blocks) && group.blocks.length > 0
  );

  if (validGroups.length === 0) {
    console.log('No valid blocks found in groups');
    return <Text type="secondary">无有效查询条件</Text>;
  }

  // 获取条件类型的标签
  const getConditionTag = (conditionType: ConditionType) => {
    switch (conditionType) {
      case ConditionType.Equal:
        return <Tag color="blue" style={{ padding: '0 4px' }}>=</Tag>;
      case ConditionType.NotEqual:
        return <Tag color="red" style={{ padding: '0 4px' }}>≠</Tag>;
      case ConditionType.Contains:
        return <Tag color="green" style={{ padding: '0 4px' }}>⊃</Tag>;
      case ConditionType.NotContains:
        return <Tag color="orange" style={{ padding: '0 4px' }}>⊅</Tag>;
      case ConditionType.In:
        return <Tag color="purple" style={{ padding: '0 4px' }}>∈</Tag>;
      case ConditionType.NotIn:
        return <Tag color="magenta" style={{ padding: '0 4px' }}>∉</Tag>;
      case ConditionType.Exists:
        return <Tag color="cyan" style={{ padding: '0 4px' }}>∃</Tag>;
      case ConditionType.NotExists:
        return <Tag color="volcano" style={{ padding: '0 4px' }}>∄</Tag>;
      default:
        return null;
    }
  };

  // 获取操作符标签
  const getOperatorTag = (operator: LogicalOperator) => {
    switch (operator) {
      case LogicalOperator.And:
        return <Tag color="blue" style={{ padding: '0 4px' }}>AND</Tag>;
      case LogicalOperator.Or:
        return <Tag color="orange" style={{ padding: '0 4px' }}>OR</Tag>;
      default:
        return null;
    }
  };

  // 获取筛选类型标签
  const getFilterTypeTag = (type: FilterType) => {
    switch (type) {
      case FilterType.Device:
        return <Tag color="blue" style={{ padding: '0 4px' }}>设备</Tag>;
      case FilterType.NodeLabel:
        return <Tag color="green" style={{ padding: '0 4px' }}>标签</Tag>;
      case FilterType.Taint:
        return <Tag color="orange" style={{ padding: '0 4px' }}>污点</Tag>;
      case FilterType.NodeInfo:
        return <Tag color="purple" style={{ padding: '0 4px' }}>节点信息</Tag>;
      default:
        return null;
    }
  };

  // 渲染单个条件块
  const renderBlock = (block: FilterBlock, isLast: boolean, groupOperator: LogicalOperator) => {
    // 对于存在/不存在条件，不显示值
    const showValue = block.conditionType !== ConditionType.Exists &&
                      block.conditionType !== ConditionType.NotExists;

    // 处理数组值
    let displayValue = block.value;
    if (Array.isArray(displayValue)) {
      displayValue = displayValue.join(', ');
    }

    // 如果值太长，截断显示
    if (typeof displayValue === 'string' && displayValue.length > 20) {
      displayValue = displayValue.substring(0, 17) + '...';
    }

    return (
      <div className="query-condition-block">
        <Space size={4} wrap>
          {getFilterTypeTag(block.type)}
          <Text>{block.key || block.field}</Text>
          {getConditionTag(block.conditionType)}
          {showValue && <Text>{displayValue}</Text>}
        </Space>

        {!isLast && (
          <div className="query-condition-operator">
            {getOperatorTag(block.operator || groupOperator)}
          </div>
        )}
      </div>
    );
  };

  // 计算总条件块数
  const totalBlocks = validGroups.reduce((sum, group) => sum + (group.blocks?.length || 0), 0);

  // 计算要显示的条件块数
  let blocksToShow = 0;
  let remainingBlocks = 0;

  if (totalBlocks > maxBlocks) {
    blocksToShow = maxBlocks;
    remainingBlocks = totalBlocks - maxBlocks;
  } else {
    blocksToShow = totalBlocks;
  }

  // 收集要显示的条件块
  const visibleGroups: {
    group: FilterGroup;
    blocks: FilterBlock[];
    isLastGroup: boolean
  }[] = [];

  let count = 0;

  for (let i = 0; i < validGroups.length; i++) {
    const group = validGroups[i];
    if (!group.blocks || group.blocks.length === 0) continue;

    const visibleBlocks = [];
    for (let j = 0; j < group.blocks.length && count < blocksToShow; j++) {
      visibleBlocks.push(group.blocks[j]);
      count++;
    }

    if (visibleBlocks.length > 0) {
      visibleGroups.push({
        group,
        blocks: visibleBlocks,
        isLastGroup: i === validGroups.length - 1
      });
    }

    if (count >= blocksToShow) break;
  }

  return (
    <div className="query-condition-preview">
      {visibleGroups.map((item, groupIndex) => (
        <div key={item.group.id || groupIndex} className="query-condition-group">
          {/* 显示组操作符标签 */}
          <div className="query-condition-group-header">
            {getOperatorTag(item.group.operator || LogicalOperator.And)}
          </div>

          {/* 显示组内的条件块 */}
          <div className="query-condition-group-blocks">
            {item.blocks.map((block, blockIndex) => (
              <React.Fragment key={block.id || blockIndex}>
                {renderBlock(
                  block,
                  blockIndex === item.blocks.length - 1,
                  item.group.operator || LogicalOperator.And
                )}
              </React.Fragment>
            ))}
          </div>

          {/* 显示组间操作符 */}
          {showGroupOperator && !item.isLastGroup && (
            <div className="query-condition-group-separator">
              {getOperatorTag(item.group.operator || LogicalOperator.And)}
            </div>
          )}
        </div>
      ))}

      {/* 显示剩余条件数量 */}
      {remainingBlocks > 0 && (
        <div className="query-condition-more">
          <Text type="secondary">还有 {remainingBlocks} 个条件...</Text>
        </div>
      )}
    </div>
  );
};

export default QueryConditionPreview;
