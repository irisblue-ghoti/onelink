#!/bin/bash

# 设置颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # 无颜色

# 显示标题
echo -e "${BLUE}"
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║              NFC碰一碰应用 - 镜像构建脚本                 ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# 检查Docker是否安装和运行
if ! command -v docker &> /dev/null; then
  echo -e "${RED}错误: Docker未安装${NC}"
  echo "请先安装Docker: https://docs.docker.com/get-docker/"
  exit 1
fi

if ! docker info &> /dev/null; then
  echo -e "${RED}错误: Docker未运行${NC}"
  echo "请启动Docker服务后再试"
  exit 1
fi

# 创建网络（如果不存在）
echo -e "${YELLOW}创建Docker网络...${NC}"
docker network create nfc_network 2>/dev/null || true

# 先构建所有镜像
echo -e "${CYAN}开始构建镜像${NC}"

# 构建前端镜像
echo -e "${GREEN}构建前端镜像...${NC}"
if [ -d ./frontend ]; then
  docker build -t nfc-frontend -f ./frontend/Dockerfile .
  if [ $? -eq 0 ]; then
    echo -e "${GREEN}前端镜像构建成功!${NC}"
  else
    echo -e "${RED}前端镜像构建失败${NC}"
    exit 1
  fi
else
  echo -e "${RED}错误: 找不到前端目录${NC}"
  exit 1
fi

# 构建后端镜像
for service in content-service merchant-service nfc-service distribution-service stats-service; do
  echo -e "${GREEN}构建 $service 镜像 (Go 1.24.2)...${NC}"
  if [ -d ./backend/$service ]; then
    docker build -t nfc-$service -f ./backend/$service/Dockerfile .
    if [ $? -eq 0 ]; then
      echo -e "${GREEN}$service 镜像构建成功!${NC}"
    else
      echo -e "${RED}$service 镜像构建失败${NC}"
      exit 1
    fi
  else
    echo -e "${RED}错误: 找不到 $service 目录${NC}"
    exit 1
  fi
done

echo -e "${GREEN}所有镜像构建完成!${NC}"
echo -e "${YELLOW}现在可以使用 ./docker-start.sh 启动服务${NC}" 