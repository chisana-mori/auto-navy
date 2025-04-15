import { FilterGroup, FilterBlock, FilterType, ConditionType, LogicalOperator } from '../types/deviceQuery';

/**
 * 生成条件组的缩略信息
 * @param groups 条件组列表
 * @param maxLength 最大长度，默认为100
 * @returns 条件组的缩略信息
 */
export function generateQuerySummary(groups: FilterGroup[], maxLength: number = 100): string {
  if (!groups || groups.length === 0) {
    return '无查询条件';
  }

  const summaries = groups.map((group, groupIndex) => {
    const blockSummaries = group.blocks.map((block) => {
      return generateBlockSummary(block);
    });

    const groupSummary = blockSummaries.join(` ${getOperatorText(group.operator)} `);
    
    // 如果不是最后一个组，添加组间操作符
    const isLastGroup = groupIndex === groups.length - 1;
    return isLastGroup ? `(${groupSummary})` : `(${groupSummary}) ${getOperatorText(group.operator)}`;
  });

  let summary = summaries.join(' ');
  
  // 如果摘要太长，截断并添加省略号
  if (summary.length > maxLength) {
    summary = summary.substring(0, maxLength - 3) + '...';
  }
  
  return summary;
}

/**
 * 生成单个筛选块的摘要
 * @param block 筛选块
 * @returns 筛选块的摘要
 */
function generateBlockSummary(block: FilterBlock): string {
  const fieldName = getFieldName(block);
  const conditionText = getConditionText(block.conditionType);
  
  // 对于存在和不存在条件，不需要显示值
  if (block.conditionType === ConditionType.Exists) {
    return `${fieldName}存在`;
  }
  
  if (block.conditionType === ConditionType.NotExists) {
    return `${fieldName}不存在`;
  }
  
  // 处理数组值
  let valueText = '';
  if (Array.isArray(block.value)) {
    valueText = block.value.length > 2 
      ? `[${block.value.slice(0, 2).join(', ')}...]` 
      : `[${block.value.join(', ')}]`;
  } else {
    valueText = block.value || '';
  }
  
  return `${fieldName}${conditionText}${valueText}`;
}

/**
 * 获取字段名称
 * @param block 筛选块
 * @returns 字段名称
 */
function getFieldName(block: FilterBlock): string {
  // 使用field或key，优先使用field
  const fieldName = block.field || block.key || '';
  
  // 根据筛选类型添加前缀
  switch (block.type) {
    case FilterType.NodeLabel:
      return `标签[${fieldName}]`;
    case FilterType.Taint:
      return `污点[${fieldName}]`;
    case FilterType.Device:
      return fieldName;
    default:
      return fieldName;
  }
}

/**
 * 获取条件文本
 * @param conditionType 条件类型
 * @returns 条件文本
 */
function getConditionText(conditionType?: ConditionType): string {
  switch (conditionType) {
    case ConditionType.Equal:
      return '=';
    case ConditionType.NotEqual:
      return '≠';
    case ConditionType.Contains:
      return '包含';
    case ConditionType.NotContains:
      return '不包含';
    case ConditionType.In:
      return '∈';
    case ConditionType.NotIn:
      return '∉';
    default:
      return '';
  }
}

/**
 * 获取操作符文本
 * @param operator 操作符
 * @returns 操作符文本
 */
function getOperatorText(operator?: LogicalOperator): string {
  return operator === LogicalOperator.Or ? 'OR' : 'AND';
}
