#!/bin/bash

# 弹性伸缩测试数据验证脚本
# 用于验证所有SQL脚本的语法正确性和数据完整性

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 创建临时数据库进行测试
create_temp_db() {
    local temp_db="/tmp/elastic_scaling_test_$(date +%s).db"
    echo "$temp_db"
}

# 创建测试数据库表结构
create_test_schema() {
    local temp_db=$1
    local script_dir=$2
    local schema_file="$script_dir/create_test_schema.sql"

    if [ -f "$schema_file" ]; then
        sqlite3 "$temp_db" < "$schema_file" 2>/dev/null
        return 0
    else
        print_warning "表结构文件不存在: $schema_file"
        return 1
    fi
}

# 验证SQL脚本语法
validate_sql_syntax() {
    local sql_file=$1
    local temp_db=$2
    local script_dir=$3

    print_info "验证 $sql_file 语法..."

    # 先创建表结构
    if ! create_test_schema "$temp_db" "$script_dir"; then
        print_error "无法创建测试表结构"
        return 1
    fi

    if sqlite3 "$temp_db" < "$sql_file" 2>/dev/null; then
        print_success "$sql_file 语法正确"
        return 0
    else
        print_error "$sql_file 语法错误"
        return 1
    fi
}

# 验证数据完整性
validate_data_integrity() {
    local temp_db=$1
    local scenario_name=$2
    
    print_info "验证 $scenario_name 数据完整性..."
    
    # 检查基础表数据
    local clusters=$(sqlite3 "$temp_db" "SELECT COUNT(*) FROM k8s_clusters;")
    local devices=$(sqlite3 "$temp_db" "SELECT COUNT(*) FROM devices;")
    local strategies=$(sqlite3 "$temp_db" "SELECT COUNT(*) FROM elastic_scaling_strategies;")
    local snapshots=$(sqlite3 "$temp_db" "SELECT COUNT(*) FROM resource_snapshots;")
    
    print_info "数据统计："
    echo "  - 集群数量: $clusters"
    echo "  - 设备数量: $devices"
    echo "  - 策略数量: $strategies"
    echo "  - 资源快照: $snapshots"
    
    # 验证数据完整性
    if [ "$clusters" -eq 0 ] || [ "$devices" -eq 0 ] || [ "$strategies" -eq 0 ] || [ "$snapshots" -eq 0 ]; then
        print_error "$scenario_name 数据不完整"
        return 1
    fi
    
    # 验证外键关系
    local invalid_devices=$(sqlite3 "$temp_db" "SELECT COUNT(*) FROM devices WHERE cluster_id NOT IN (SELECT id FROM k8s_clusters);")
    local invalid_associations=$(sqlite3 "$temp_db" "SELECT COUNT(*) FROM ng_strategy_cluster_association WHERE cluster_id NOT IN (SELECT id FROM k8s_clusters) OR strategy_id NOT IN (SELECT id FROM ng_elastic_scaling_strategy);")
    
    if [ "$invalid_devices" -gt 0 ] || [ "$invalid_associations" -gt 0 ]; then
        print_error "$scenario_name 外键关系错误"
        return 1
    fi
    
    print_success "$scenario_name 数据完整性验证通过"
    return 0
}

# 主验证函数
main() {
    print_info "开始验证弹性伸缩测试数据..."
    
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local total_tests=0
    local passed_tests=0
    
    # 测试场景列表
    local scenarios=(
        "scenario1_pool_entry:场景1-入池订单生成"
        "scenario2_pool_exit:场景2-退池订单生成"
        "scenario3_threshold_not_met:场景3-不满足条件"
        "scenario4_pool_entry_no_devices:场景4-入池无法匹配到设备"
        "scenario5_pool_exit_no_devices:场景5-退池无法匹配到设备"
        "scenario6_pool_entry_partial_devices:场景6-入池只能匹配部分设备"
        "scenario7_pool_exit_partial_devices:场景7-退池只能匹配部分设备"
    )
    
    for scenario in "${scenarios[@]}"; do
        IFS=':' read -r scenario_file scenario_name <<< "$scenario"
        local sql_file="$script_dir/test_data_${scenario_file}.sql"
        
        if [ ! -f "$sql_file" ]; then
            print_error "文件不存在: $sql_file"
            continue
        fi
        
        total_tests=$((total_tests + 1))
        
        # 创建临时数据库
        local temp_db=$(create_temp_db)
        
        # 验证SQL语法
        if validate_sql_syntax "$sql_file" "$temp_db" "$script_dir"; then
            # 验证数据完整性
            if validate_data_integrity "$temp_db" "$scenario_name"; then
                passed_tests=$((passed_tests + 1))
            fi
        fi
        
        # 清理临时数据库
        rm -f "$temp_db"
        echo ""
    done
    
    # 输出总结
    echo "=================================="
    print_info "验证完成"
    echo "总测试数: $total_tests"
    echo "通过测试: $passed_tests"
    echo "失败测试: $((total_tests - passed_tests))"
    
    if [ "$passed_tests" -eq "$total_tests" ]; then
        print_success "所有测试数据验证通过！"
        exit 0
    else
        print_error "部分测试数据验证失败！"
        exit 1
    fi
}

# 显示使用说明
show_usage() {
    echo "弹性伸缩测试数据验证脚本"
    echo ""
    echo "使用方法:"
    echo "  $0                    # 验证所有测试数据"
    echo "  $0 --help            # 显示帮助信息"
    echo ""
    echo "功能:"
    echo "  - 验证SQL脚本语法正确性"
    echo "  - 验证测试数据完整性"
    echo "  - 验证外键关系正确性"
    echo "  - 统计数据量信息"
}

# 检查参数
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    show_usage
    exit 0
fi

# 执行主函数
main
