#!/bin/bash

# 修复弹性伸缩SQL文件中的表名不一致问题
# 将SQL文件中的表名修改为与Go模型中定义的实际表名一致

echo "开始修复弹性伸缩SQL文件中的表名..."

# 定义需要修复的SQL文件列表
SQL_FILES=(
    "test_data_scenario2_pool_exit.sql"
    "test_data_scenario3_threshold_not_met.sql"
    "test_data_scenario4_pool_entry_no_devices.sql"
    "test_data_scenario5_pool_exit_no_devices.sql"
    "test_data_scenario6_pool_entry_partial_devices.sql"
    "test_data_scenario7_pool_exit_partial_devices.sql"
)

# 定义表名映射关系（旧表名 -> 新表名）
declare -A TABLE_MAPPINGS=(
    ["k8s_clusters"]="k8s_cluster"
    ["devices"]="device"
    ["query_templates"]="query_template"
    ["elastic_scaling_strategies"]="elastic_scaling_strategy"
    ["strategy_cluster_associations"]="strategy_cluster_association"
    ["resource_snapshots"]="k8s_cluster_resource_snapshot"
)

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 修复每个SQL文件
for file in "${SQL_FILES[@]}"; do
    file_path="$SCRIPT_DIR/$file"
    
    if [[ ! -f "$file_path" ]]; then
        echo "警告: 文件 $file 不存在，跳过..."
        continue
    fi
    
    echo "正在修复文件: $file"
    
    # 创建备份
    cp "$file_path" "$file_path.backup"
    
    # 应用表名映射
    for old_table in "${!TABLE_MAPPINGS[@]}"; do
        new_table="${TABLE_MAPPINGS[$old_table]}"
        
        # 替换表名（考虑各种SQL语句格式）
        sed -i.tmp \
            -e "s/FROM ${old_table}/FROM ${new_table}/g" \
            -e "s/INTO ${old_table}/INTO ${new_table}/g" \
            -e "s/DELETE FROM ${old_table}/DELETE FROM ${new_table}/g" \
            -e "s/INSERT INTO ${old_table}/INSERT INTO ${new_table}/g" \
            -e "s/UPDATE ${old_table}/UPDATE ${new_table}/g" \
            -e "s/TABLE IF NOT EXISTS ${old_table}/TABLE IF NOT EXISTS ${new_table}/g" \
            -e "s/TABLE ${old_table}/TABLE ${new_table}/g" \
            -e "s/REFERENCES ${old_table}/REFERENCES ${new_table}/g" \
            -e "s/'${old_table}'/'${new_table}'/g" \
            -e "s/\"${old_table}\"/\"${new_table}\"/g" \
            "$file_path"
        
        # 删除临时文件
        rm -f "$file_path.tmp"
    done
    
    echo "✓ 完成修复: $file"
done

echo ""
echo "表名修复完成！"
echo ""
echo "修复的表名映射："
for old_table in "${!TABLE_MAPPINGS[@]}"; do
    new_table="${TABLE_MAPPINGS[$old_table]}"
    echo "  $old_table -> $new_table"
done

echo ""
echo "备份文件已创建（.backup后缀），如需回滚可使用备份文件。"
echo "建议运行测试验证修复结果。"
