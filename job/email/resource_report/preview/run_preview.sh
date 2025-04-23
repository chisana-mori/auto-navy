#!/bin/bash

# 切换到脚本所在目录
cd "$(dirname "$0")"

# 编译并运行预览生成器
echo "编译并运行资源报告预览生成器..."
go run *.go

# 检查是否生成了HTML预览文件
if [ -f "preview.html" ]; then
    echo "HTML预览文件已生成，正在打开..."

    # 根据操作系统打开HTML预览文件
    case "$(uname)" in
        "Darwin") # macOS
            open preview.html
            ;;
        "Linux")
            if command -v xdg-open > /dev/null; then
                xdg-open preview.html
            else
                echo "请手动打开HTML预览文件: $(pwd)/preview.html"
            fi
            ;;
        "MINGW"*|"MSYS"*|"CYGWIN"*) # Windows
            start preview.html
            ;;
        *)
            echo "请手动打开HTML预览文件: $(pwd)/preview.html"
            ;;
    esac
else
    echo "HTML预览文件生成失败，请检查错误信息。"
    exit 1
fi

# 检查是否生成了Excel预览文件
EXCEL_FILE=$(ls k8s_resource_report_*.xlsx 2>/dev/null | head -n 1)
if [ -n "$EXCEL_FILE" ]; then
    echo "Excel预览文件已生成: $EXCEL_FILE"
    echo "是否打开Excel文件? (y/n)"
    read -r answer
    if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
        # 根据操作系统打开Excel预览文件
        case "$(uname)" in
            "Darwin") # macOS
                open "$EXCEL_FILE"
                ;;
            "Linux")
                if command -v xdg-open > /dev/null; then
                    xdg-open "$EXCEL_FILE"
                else
                    echo "请手动打开Excel预览文件: $(pwd)/$EXCEL_FILE"
                fi
                ;;
            "MINGW"*|"MSYS"*|"CYGWIN"*) # Windows
                start "$EXCEL_FILE"
                ;;
            *)
                echo "请手动打开Excel预览文件: $(pwd)/$EXCEL_FILE"
                ;;
        esac
    fi
else
    echo "Excel预览文件生成失败，请检查错误信息。"
fi
