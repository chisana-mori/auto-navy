#!/bin/bash

# 弹性伸缩策略评估前端测试执行脚本
# 使用方法: ./run_frontend_tests.sh [scenario_number] [database_path]

set -e

# 默认配置
DEFAULT_DB_PATH="./data/navy.db"
DOCS_DIR="./docs/elastic_scaling"

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

# 显示使用说明
show_usage() {
    echo "弹性伸缩策略评估前端测试脚本"
    echo ""
    echo "使用方法:"
    echo "  $0 [scenario_number] [database_path]"
    echo ""
    echo "参数说明:"
    echo "  scenario_number  测试场景编号 (1-7, 或 'all')"
    echo "    1: 入池订单生成测试"
    echo "    2: 退池订单生成测试"
    echo "    3: 不满足条件测试"
    echo "    4: 入池无法匹配到设备"
    echo "    5: 退池无法匹配到设备"
    echo "    6: 入池只能匹配部分设备"
    echo "    7: 退池只能匹配部分设备"
    echo "    all: 运行所有场景"
    echo "  database_path    SQLite数据库文件路径 (默认: $DEFAULT_DB_PATH)"
    echo ""
    echo "示例:"
    echo "  $0 1                    # 运行场景1，使用默认数据库路径"
    echo "  $0 all ./data/test.db   # 运行所有场景，使用指定数据库"
    echo "  $0 4                    # 运行场景4"
}

# 检查数据库文件
check_database() {
    local db_path=$1
    if [ ! -f "$db_path" ]; then
        print_error "数据库文件不存在: $db_path"
        print_info "请确保数据库文件存在，或者先启动后端服务创建数据库"
        exit 1
    fi
    print_success "数据库文件检查通过: $db_path"
}

# 执行SQL脚本
execute_sql_script() {
    local script_path=$1
    local db_path=$2
    local scenario_name=$3
    
    if [ ! -f "$script_path" ]; then
        print_error "SQL脚本文件不存在: $script_path"
        return 1
    fi
    
    print_info "执行 $scenario_name 测试数据脚本..."
    sqlite3 "$db_path" < "$script_path"
    
    if [ $? -eq 0 ]; then
        print_success "$scenario_name 测试数据初始化完成"
    else
        print_error "$scenario_name 测试数据初始化失败"
        return 1
    fi
}

# 运行场景1：入池订单生成测试
run_scenario1() {
    local db_path=$1
    print_info "=== 场景1：入池订单生成测试 ==="
    print_info "测试目标：验证当CPU使用率连续3天超过80%时，系统能够正确生成入池订单"
    
    execute_sql_script "$DOCS_DIR/test_data_scenario1_pool_entry.sql" "$db_path" "场景1"
    
    print_info "测试数据已准备完成，请按以下步骤进行前端测试："
    echo ""
    echo "1. 访问策略管理页面: http://localhost:3000/elastic-scaling/strategies"
    echo "   - 验证策略 'CPU High Usage Scale Out' 状态为启用"
    echo "   - 查看策略配置：CPU阈值80%，持续时间3天"
    echo ""
    echo "2. 访问资源监控页面: http://localhost:3000/elastic-scaling/dashboard"
    echo "   - 选择 'production-cluster' 集群"
    echo "   - 查看CPU使用率趋势图，应显示连续3天超过80%"
    echo ""
    echo "3. 手动触发策略评估（可选）:"
    echo "   curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/1/evaluate"
    echo ""
    echo "4. 访问订单管理页面: http://localhost:3000/elastic-scaling/orders"
    echo "   - 验证生成新的入池订单"
    echo "   - 查看订单详情，确认设备分配正确"
    echo ""
    echo "5. 查看策略执行历史:"
    echo "   - 访问策略详情页面查看执行历史"
    echo "   - 验证执行结果为 'order_created'"
}

# 运行场景2：退池订单生成测试
run_scenario2() {
    local db_path=$1
    print_info "=== 场景2：退池订单生成测试 ==="
    print_info "测试目标：验证当内存分配率连续2天低于20%时，系统能够正确生成退池订单"
    
    execute_sql_script "$DOCS_DIR/test_data_scenario2_pool_exit.sql" "$db_path" "场景2"
    
    print_info "测试数据已准备完成，请按以下步骤进行前端测试："
    echo ""
    echo "1. 访问策略管理页面: http://localhost:3000/elastic-scaling/strategies"
    echo "   - 验证策略 'Memory Low Usage Scale In' 状态为启用"
    echo "   - 查看策略配置：内存阈值20%，持续时间2天，动作为退池"
    echo ""
    echo "2. 访问资源监控页面: http://localhost:3000/elastic-scaling/dashboard"
    echo "   - 选择 'staging-cluster' 集群"
    echo "   - 查看内存分配率趋势，应显示连续2天低于20%"
    echo ""
    echo "3. 手动触发策略评估（可选）:"
    echo "   curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/2/evaluate"
    echo ""
    echo "4. 访问订单管理页面: http://localhost:3000/elastic-scaling/orders"
    echo "   - 验证生成新的退池订单"
    echo "   - 查看订单详情，确认退池设备信息"
    echo ""
    echo "5. 验证设备状态:"
    echo "   - 在设备管理页面查看在池设备"
    echo "   - 确认退池订单包含正确的设备"
}

# 运行场景3：不满足条件测试
run_scenario3() {
    local db_path=$1
    print_info "=== 场景3：不满足条件测试 ==="
    print_info "测试目标：验证当资源使用率未连续满足阈值要求时，系统不生成订单"
    
    execute_sql_script "$DOCS_DIR/test_data_scenario3_threshold_not_met.sql" "$db_path" "场景3"
    
    print_info "测试数据已准备完成，请按以下步骤进行前端测试："
    echo ""
    echo "1. 访问策略管理页面: http://localhost:3000/elastic-scaling/strategies"
    echo "   - 验证策略 'CPU Threshold Not Met Test' 状态为启用"
    echo "   - 查看策略配置：CPU阈值80%，持续时间3天"
    echo ""
    echo "2. 访问资源监控页面: http://localhost:3000/elastic-scaling/dashboard"
    echo "   - 选择 'test-cluster' 集群"
    echo "   - 查看CPU使用率趋势，注意第2天低于阈值（中断连续性）"
    echo ""
    echo "3. 手动触发策略评估（可选）:"
    echo "   curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/3/evaluate"
    echo ""
    echo "4. 访问订单管理页面: http://localhost:3000/elastic-scaling/orders"
    echo "   - 验证没有生成新订单"
    echo ""
    echo "5. 查看策略执行历史:"
    echo "   - 访问策略详情页面查看执行历史"
    echo "   - 验证执行结果为 'failure_threshold_not_met'"
    echo "   - 查看失败原因描述"
}

# 运行场景4：入池无法匹配到设备
run_scenario4() {
    local db_path=$1
    print_info "=== 场景4：入池无法匹配到设备测试 ==="
    print_info "测试目标：验证当满足入池条件但无可用设备时，系统生成提醒订单的处理逻辑"

    execute_sql_script "$DOCS_DIR/test_data_scenario4_pool_entry_no_devices.sql" "$db_path" "场景4"

    print_info "测试数据已准备完成，请按以下步骤进行前端测试："
    echo ""
    echo "1. 访问设备管理页面，确认无可用设备（所有设备状态为非in_stock）"
    echo ""
    echo "2. 访问策略管理页面: http://localhost:3000/elastic-scaling/strategies"
    echo "   - 验证策略 'CPU High No Devices Available' 状态为启用"
    echo "   - 查看策略配置：要求2台设备"
    echo ""
    echo "3. 手动触发策略评估:"
    echo "   curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/4/evaluate"
    echo ""
    echo "4. 访问订单管理页面: http://localhost:3000/elastic-scaling/orders"
    echo "   - 验证生成提醒订单（设备数量为0）"
    echo "   - 查看订单详情，确认显示'找不到要处理的设备，请自行协调处理'"
    echo ""
    echo "5. 查看策略执行历史和邮件通知:"
    echo "   - 验证执行结果为 'order_created_no_devices'"
    echo "   - 查看邮件通知内容，确认包含设备申请提醒"
}

# 运行场景5：退池无法匹配到设备
run_scenario5() {
    local db_path=$1
    print_info "=== 场景5：退池无法匹配到设备测试 ==="
    print_info "测试目标：验证当满足退池条件但无在池设备时，系统生成提醒订单的处理逻辑"

    execute_sql_script "$DOCS_DIR/test_data_scenario5_pool_exit_no_devices.sql" "$db_path" "场景5"

    print_info "测试数据已准备完成，请按以下步骤进行前端测试："
    echo ""
    echo "1. 访问设备管理页面，确认无在池设备（所有设备状态为非in_pool）"
    echo ""
    echo "2. 访问策略管理页面: http://localhost:3000/elastic-scaling/strategies"
    echo "   - 验证策略 'Memory Low No Pool Devices' 状态为启用"
    echo "   - 查看策略配置：要求1台设备，动作为退池"
    echo ""
    echo "3. 手动触发策略评估:"
    echo "   curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/5/evaluate"
    echo ""
    echo "4. 访问订单管理页面: http://localhost:3000/elastic-scaling/orders"
    echo "   - 验证生成提醒订单（设备数量为0）"
    echo "   - 查看订单详情，确认显示'找不到要处理的设备，请自行协调处理'"
    echo ""
    echo "5. 查看策略执行历史和邮件通知:"
    echo "   - 验证执行结果为 'order_created_no_devices'"
    echo "   - 查看邮件通知内容，确认包含协调处理提醒"
}

# 运行场景6：入池只能匹配部分设备
run_scenario6() {
    local db_path=$1
    print_info "=== 场景6：入池只能匹配部分设备测试 ==="
    print_info "测试目标：验证当满足入池条件但只能匹配到部分设备时，系统的处理逻辑"

    execute_sql_script "$DOCS_DIR/test_data_scenario6_pool_entry_partial_devices.sql" "$db_path" "场景6"

    print_info "测试数据已准备完成，请按以下步骤进行前端测试："
    echo ""
    echo "1. 访问设备管理页面，确认只有2台可用设备"
    echo ""
    echo "2. 访问策略管理页面: http://localhost:3000/elastic-scaling/strategies"
    echo "   - 验证策略 'CPU High Partial Devices' 状态为启用"
    echo "   - 查看策略配置：要求5台设备，但只有2台可用"
    echo ""
    echo "3. 手动触发策略评估:"
    echo "   curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/6/evaluate"
    echo ""
    echo "4. 访问订单管理页面: http://localhost:3000/elastic-scaling/orders"
    echo "   - 验证生成部分订单（包含2台设备）"
    echo "   - 查看订单详情，确认设备数量和配置"
    echo ""
    echo "5. 查看策略执行历史:"
    echo "   - 验证执行结果为 'order_created_partial'"
    echo "   - 查看部分匹配的说明"
}

# 运行场景7：退池只能匹配部分设备
run_scenario7() {
    local db_path=$1
    print_info "=== 场景7：退池只能匹配部分设备测试 ==="
    print_info "测试目标：验证当满足退池条件但只能匹配到部分设备时，系统的处理逻辑"

    execute_sql_script "$DOCS_DIR/test_data_scenario7_pool_exit_partial_devices.sql" "$db_path" "场景7"

    print_info "测试数据已准备完成，请按以下步骤进行前端测试："
    echo ""
    echo "1. 访问设备管理页面，确认只有2台在池设备"
    echo ""
    echo "2. 访问策略管理页面: http://localhost:3000/elastic-scaling/strategies"
    echo "   - 验证策略 'Memory Low Partial Pool Devices' 状态为启用"
    echo "   - 查看策略配置：要求4台设备，但只有2台在池"
    echo ""
    echo "3. 手动触发策略评估:"
    echo "   curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/7/evaluate"
    echo ""
    echo "4. 访问订单管理页面: http://localhost:3000/elastic-scaling/orders"
    echo "   - 验证生成部分订单（包含2台设备）"
    echo "   - 查看订单详情，确认退池设备信息"
    echo ""
    echo "5. 查看策略执行历史:"
    echo "   - 验证执行结果为 'order_created_partial'"
    echo "   - 查看部分匹配的说明"
}

# 清理测试数据
cleanup_test_data() {
    local db_path=$1
    print_warning "清理所有测试数据..."
    
    cat << EOF | sqlite3 "$db_path"
DELETE FROM ng_strategy_execution_history;
DELETE FROM ng_order_device;
DELETE FROM ng_elastic_scaling_order_details;
DELETE FROM ng_orders;
DELETE FROM resource_snapshots;
DELETE FROM ng_strategy_cluster_association;
DELETE FROM elastic_scaling_strategies;
DELETE FROM query_templates;
DELETE FROM devices;
DELETE FROM k8s_clusters;
DELETE FROM sqlite_sequence WHERE name IN (
    'ng_strategy_execution_history', 'ng_order_device', 'ng_elastic_scaling_order_details',
'ng_orders', 'resource_snapshots', 'ng_strategy_cluster_association',
    'elastic_scaling_strategies', 'query_templates', 'devices', 'k8s_clusters'
);
EOF
    
    print_success "测试数据清理完成"
}

# 主函数
main() {
    local scenario=${1:-""}
    local db_path=${2:-$DEFAULT_DB_PATH}
    
    # 显示脚本信息
    print_info "弹性伸缩策略评估前端测试脚本"
    print_info "数据库路径: $db_path"
    
    # 检查参数
    if [ -z "$scenario" ]; then
        show_usage
        exit 1
    fi
    
    # 检查数据库
    check_database "$db_path"
    
    # 根据场景执行测试
    case "$scenario" in
        "1")
            run_scenario1 "$db_path"
            ;;
        "2")
            run_scenario2 "$db_path"
            ;;
        "3")
            run_scenario3 "$db_path"
            ;;
        "4")
            run_scenario4 "$db_path"
            ;;
        "5")
            run_scenario5 "$db_path"
            ;;
        "6")
            run_scenario6 "$db_path"
            ;;
        "7")
            run_scenario7 "$db_path"
            ;;
        "all")
            print_info "运行所有测试场景..."
            run_scenario1 "$db_path"
            echo ""
            read -p "按回车键继续场景2测试..."
            run_scenario2 "$db_path"
            echo ""
            read -p "按回车键继续场景3测试..."
            run_scenario3 "$db_path"
            echo ""
            read -p "按回车键继续场景4测试..."
            run_scenario4 "$db_path"
            echo ""
            read -p "按回车键继续场景5测试..."
            run_scenario5 "$db_path"
            echo ""
            read -p "按回车键继续场景6测试..."
            run_scenario6 "$db_path"
            echo ""
            read -p "按回车键继续场景7测试..."
            run_scenario7 "$db_path"
            ;;
        "clean")
            cleanup_test_data "$db_path"
            ;;
        *)
            print_error "无效的场景编号: $scenario"
            show_usage
            exit 1
            ;;
    esac
    
    echo ""
    print_success "测试准备完成！"
    print_info "提示：测试完成后可以运行 '$0 clean $db_path' 清理测试数据"
}

# 执行主函数
main "$@"
