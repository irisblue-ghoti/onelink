#!/bin/bash

# 设置环境变量
export PORT=3001

# 检查日志目录是否存在，不存在则创建
if [ ! -d "logs" ]; then
  mkdir -p logs
fi

# 构建应用
echo "构建应用..."
go build -o bin/content-service ./cmd/server/main.go

# 判断是否构建成功
if [ $? -ne 0 ]; then
  echo "构建失败"
  exit 1
fi

# 检查是否存在config.yaml
if [ ! -f "config.yaml" ]; then
  echo "警告: 未找到config.yaml文件，请确保配置正确"
fi

# 启动应用
echo "启动内容服务..."
./bin/content-service > ./logs/content-service.log 2>&1 &

# 保存进程ID
echo $! > ./logs/content-service.pid

echo "内容服务已启动，PID: $(cat ./logs/content-service.pid)"
echo "日志文件: ./logs/content-service.log"

# 显示日志
echo "输出日志..."
tail -f ./logs/content-service.log 