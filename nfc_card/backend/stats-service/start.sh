#!/bin/bash

# 统计服务启动脚本

echo "正在启动统计服务..."

# 设置环境变量
export CONFIG_PATH="./config.yaml"
export SERVER_PORT="8084"

# 检查是否为测试模式
if [ "$1" = "test" ]; then
  echo "统计服务启动脚本测试成功"
  exit 0
fi

# 创建日志目录
mkdir -p ./logs

# 进入服务目录
cd "$(dirname "$0")"

# 检查配置文件是否存在
if [ ! -f "$CONFIG_PATH" ]; then
  echo "警告: 配置文件不存在，将使用默认配置"
fi

# 构建项目
echo "构建统计服务..."
go build -o ./bin/stats-service ./cmd/server/main.go

# 检查构建是否成功
if [ $? -ne 0 ]; then
  echo "构建失败，请检查错误信息"
  exit 1
fi

# 运行服务
echo "统计服务构建成功，正在启动..."
./bin/stats-service 2>&1 | tee ./logs/stats-service.log

# 捕获退出信号
trap "echo '正在关闭统计服务...'; exit 0" SIGINT SIGTERM

# 等待服务退出
wait $!
