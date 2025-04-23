#!/bin/bash

# 切换到脚本所在目录
cd "$(dirname "$0")"

# 编译并运行预览生成器
echo "编译并运行资源报告预览生成器..."
go run *.go

# 检查是否生成了预览文件
if [ -f "preview.html" ]; then
    echo "预览文件已生成，正在打开..."
    
    # 根据操作系统打开预览文件
    case "$(uname)" in
        "Darwin") # macOS
            open preview.html
            ;;
        "Linux")
            if command -v xdg-open > /dev/null; then
                xdg-open preview.html
            else
                echo "请手动打开预览文件: $(pwd)/preview.html"
            fi
            ;;
        "MINGW"*|"MSYS"*|"CYGWIN"*) # Windows
            start preview.html
            ;;
        *)
            echo "请手动打开预览文件: $(pwd)/preview.html"
            ;;
    esac
else
    echo "预览文件生成失败，请检查错误信息。"
    exit 1
fi
