#!/bin/bash

# 设置颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # 无颜色

# 默认模式为生产模式
DEV_MODE=false
FRONTEND_LOCAL=false

# 解析命令行参数
while [[ "$#" -gt 0 ]]; do
  case $1 in
    --dev) DEV_MODE=true ;;
    --frontend-local) FRONTEND_LOCAL=true ;;
    --help) 
      echo -e "${GREEN}用法: $0 [选项]${NC}"
      echo -e "选项:"
      echo -e "  ${YELLOW}--dev${NC}             开发模式，所有服务使用开发配置"
      echo -e "  ${YELLOW}--frontend-local${NC}  前端本地开发模式，不启动前端Docker容器"
      echo -e "  ${YELLOW}--help${NC}            显示此帮助信息"
      exit 0
      ;;
    *) echo "未知参数: $1"; exit 1 ;;
  esac
  shift
done

# 显示标题
echo -e "${BLUE}"
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║              NFC碰一碰应用 - 顺序启动脚本                 ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# 显示当前模式
if [ "$DEV_MODE" = true ]; then
  echo -e "${YELLOW}当前模式: 开发模式${NC}"
else
  echo -e "${YELLOW}当前模式: 生产模式${NC}"
fi

if [ "$FRONTEND_LOCAL" = true ]; then
  echo -e "${YELLOW}前端模式: 本地开发${NC}"
  echo -e "${CYAN}注意: 前端将不会通过Docker启动，请手动在本地启动前端${NC}"
  echo -e "${CYAN}前端本地启动命令: cd frontend && npm run dev${NC}"
else
  echo -e "${YELLOW}前端模式: Docker容器${NC}"
fi

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

# 初始化配置
if [ -f ./init-config.sh ]; then
  ./init-config.sh
else
  echo -e "${RED}错误: 找不到初始化配置脚本 init-config.sh${NC}"
  echo -e "${YELLOW}请确保 init-config.sh 文件存在于项目根目录${NC}"
  exit 1
fi

# 创建网络（如果不存在）
echo -e "${YELLOW}创建Docker网络...${NC}"
docker network create nfc_network 2>/dev/null || true

# 先构建所有镜像
echo -e "${CYAN}Step 0/7: 构建所需镜像${NC}"

# 构建前端镜像（如果不是本地开发模式）
if [ "$FRONTEND_LOCAL" = false ]; then
  echo -e "${GREEN}构建前端镜像...${NC}"
  if [ -d ./frontend ]; then
    echo -e "${CYAN}正在为前端构建Docker镜像，这可能需要几分钟时间...${NC}"
    cd ./frontend && docker build -t nfc-frontend . && cd ..
    if [ $? -ne 0 ]; then
      echo -e "${RED}前端镜像构建失败${NC}"
      echo -e "${YELLOW}请检查Dockerfile和前端代码${NC}"
      exit 1
    fi
    echo -e "${GREEN}前端镜像构建成功!${NC}"
  else
    echo -e "${RED}错误: 找不到前端目录${NC}"
    exit 1
  fi
else
  echo -e "${YELLOW}跳过前端镜像构建 (本地开发模式)${NC}"
fi

# 构建后端镜像
for service in content-service merchant-service nfc-service distribution-service stats-service; do
  echo -e "${GREEN}构建 $service 镜像 (Go 1.24.2)...${NC}"
  if [ -d ./backend/$service ]; then
    docker build -t nfc-$service -f ./backend/$service/Dockerfile . || { echo -e "${RED}$service 镜像构建失败${NC}"; exit 1; }
    echo -e "${GREEN}$service 镜像构建成功!${NC}"
  else
    echo -e "${RED}错误: 找不到 $service 目录${NC}"
    exit 1
  fi
done

# 顺序启动各服务
echo -e "${YELLOW}开始顺序启动服务...${NC}"

# 首先启动基础服务
echo -e "${GREEN}Step 1/7: 启动数据服务 (Zookeeper, Kafka, PostgreSQL, Redis, MinIO)${NC}"
docker-compose up -d zookeeper
sleep 3
docker-compose up -d kafka
sleep 3
docker-compose up -d postgres
sleep 2
docker-compose up -d redis
sleep 2
docker-compose up -d minio
sleep 2
echo -e "${CYAN}等待MinIO服务启动...${NC}"
# 检查MinIO健康状态
curl -s http://localhost:9000/minio/health/live > /dev/null
if [ $? -eq 0 ]; then
  echo -e "${GREEN}MinIO服务已成功启动!${NC}"
else
  echo -e "${YELLOW}MinIO服务可能尚未完全启动，将继续等待...${NC}"
  sleep 3
fi

# 启动Nacos服务注册中心
echo -e "${GREEN}Step 2/7: 启动Nacos服务注册中心${NC}"
docker-compose up -d nacos
sleep 5
echo -e "${CYAN}等待Nacos服务启动...${NC}"
# 检查Nacos健康状态
echo -e "${CYAN}检查Nacos服务状态...${NC}"
curl -s http://localhost:8848/nacos/v1/console/health/status 2>&1 > /dev/null
if [ $? -eq 0 ]; then
  echo -e "${GREEN}Nacos服务已成功启动!${NC}"
else
  echo -e "${YELLOW}Nacos服务可能尚未完全启动，将继续等待...${NC}"
  sleep 5
fi

# 启动API网关
echo -e "${GREEN}Step 3/7: 启动API网关 (APISIX)${NC}"
docker-compose up -d apisix
sleep 2

# 检查基础服务状态
echo -e "${GREEN}Step 4/7: 检查基础服务状态${NC}"
docker-compose ps zookeeper kafka postgres redis minio nacos apisix

# 启动后端服务
echo -e "${GREEN}Step 5/7: 启动后端服务${NC}"
for service in merchant-service content-service nfc-service distribution-service stats-service; do
  echo -e "${CYAN}启动 $service...${NC}"
  docker-compose up -d $service
  sleep 2
done

# 启动前端（如果不是本地开发模式）
if [ "$FRONTEND_LOCAL" = false ]; then
  echo -e "${GREEN}Step 6/7: 启动前端服务 (Docker容器)${NC}"
  docker-compose up -d frontend
  sleep 2
else
  echo -e "${GREEN}Step 6/7: 跳过前端服务启动 (本地开发模式)${NC}"
  echo -e "${CYAN}请在另一个终端窗口中手动启动前端:${NC}"
  echo -e "${YELLOW}cd frontend && npm run dev${NC}"
fi

echo -e "${GREEN}Step 7/7: 检查所有服务状态${NC}"
docker-compose ps

echo -e "${GREEN}所有服务已启动完成!${NC}"
echo -e "${YELLOW}=================================${NC}"
echo -e "${YELLOW}服务访问信息:${NC}"
if [ "$FRONTEND_LOCAL" = false ]; then
  echo -e "  前端应用: ${GREEN}http://localhost:3000${NC} (Docker容器)"
else
  echo -e "  前端应用: ${GREEN}http://localhost:3000${NC} (本地开发模式，需手动启动)"
fi
echo -e "  API网关: ${GREEN}http://localhost:9080${NC}"
echo -e "  Nacos控制台: ${GREEN}http://localhost:8848/nacos${NC}"
echo -e "  MinIO控制台: ${GREEN}http://localhost:9001${NC}"
echo -e "  内容服务: ${GREEN}http://localhost:8081${NC}"
echo -e "  商户服务: ${GREEN}http://localhost:8082${NC}"
echo -e "  NFC服务: ${GREEN}http://localhost:8083${NC}"
echo -e "  统计服务: ${GREEN}http://localhost:8084${NC}"
echo -e "  分发服务: ${GREEN}http://localhost:8085${NC}"
echo -e "  PostgreSQL: ${GREEN}localhost:5432${NC}"
echo -e "  Redis: ${GREEN}localhost:6379${NC}"
echo -e "  Kafka: ${GREEN}localhost:9092${NC}"
echo -e "${YELLOW}=================================${NC}" 