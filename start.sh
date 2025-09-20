#!/bin/bash

# AutoBot 启动脚本

echo "=== AutoBot 计划任务平台 ==="
echo "正在启动服务..."

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到 Go 环境，请先安装 Go 1.21 或更高版本"
    exit 1
fi

# 检查 Python 环境
if ! command -v python3 &> /dev/null; then
    echo "错误: 未找到 Python3 环境，请先安装 Python 3.6 或更高版本"
    exit 1
fi

echo "✓ Go 版本: $(go version)"
echo "✓ Python 版本: $(python3 --version)"

# 下载依赖
echo "正在下载依赖包..."
go mod download

# 构建项目
echo "正在构建项目..."
go build -o autobot main.go

if [ $? -ne 0 ]; then
    echo "构建失败，请检查代码"
    exit 1
fi

echo "✓ 构建完成"

# 启动服务
echo "正在启动服务..."
echo "访问地址: http://localhost:8080"
echo "按 Ctrl+C 停止服务"
echo "========================"

./autobot



