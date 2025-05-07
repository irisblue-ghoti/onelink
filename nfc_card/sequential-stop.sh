#!/bin/bash

# 设置颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # 无颜色

# 默认模式设置
FRONTEND_LOCAL=false

# 解析命令行参数
while [[ "$#" -gt 0 ]]; do
  case $1 in
    --frontend-local) FRONTEND_LOCAL=true ;;
    --help) 
      echo -e "${GREEN}用法: $0 [选项]${NC}"
      echo -e "选项:"
      echo -e "  ${YELLOW}--frontend-local${NC}  前端本地开发模式，不关闭前端Docker容器"
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
echo "║              NFC碰一碰应用 - 顺序关闭脚本                 ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# 显示当前模式
if [ "$FRONTEND_LOCAL" = true ]; then
  echo -e "${YELLOW}前端模式: 本地开发${NC}"
  echo -e "${CYAN}注意: 前端Docker容器将不会被关闭，因为前端使用本地开发模式${NC}"
else
  echo -e "${YELLOW}前端模式: Docker容器${NC}"
fi

# 检查服务状态
echo -e "${YELLOW}当前服务状态:${NC}"
docker-compose ps

# 确认关闭
read -p "确定要关闭所有服务吗? (y/n): " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
  echo -e "${YELLOW}操作已取消${NC}"
  exit 0
fi

# 顺序关闭各服务（与启动顺序相反）
echo -e "${YELLOW}开始顺序关闭服务...${NC}"

# 先关闭前端（如果不是本地开发模式）
if [ "$FRONTEND_LOCAL" = false ]; then
  echo -e "${GREEN}Step 1/5: 关闭前端服务${NC}"
  docker-compose stop frontend
else
  echo -e "${GREEN}Step 1/5: 跳过关闭前端服务 (本地开发模式)${NC}"
  echo -e "${CYAN}如果需要，请手动停止本地前端开发服务${NC}"
fi

# 关闭后端服务
echo -e "${GREEN}Step 2/5: 关闭后端服务${NC}"
for service in stats-service distribution-service nfc-service content-service merchant-service; do
  echo -e "${CYAN}关闭 $service...${NC}"
  docker-compose stop $service
done

# 关闭API网关
echo -e "${GREEN}Step 3/5: 关闭API网关 (APISIX)${NC}"
docker-compose stop apisix

# 关闭Nacos服务注册中心
echo -e "${GREEN}Step 4/5: 关闭Nacos服务注册中心${NC}"
docker-compose stop nacos

# 最后关闭基础服务
echo -e "${GREEN}Step 5/5: 关闭数据服务 (MinIO, Redis, PostgreSQL, Kafka, Zookeeper)${NC}"
docker-compose stop minio
docker-compose stop redis
docker-compose stop postgres
docker-compose stop kafka
docker-compose stop zookeeper

echo -e "${GREEN}所有服务已关闭!${NC}"

# 查看服务状态
echo -e "${BLUE}服务状态:${NC}"
docker-compose ps

# 如果是本地开发模式，提醒用户手动关闭前端
if [ "$FRONTEND_LOCAL" = true ]; then
  echo -e "${YELLOW}提示: 如果前端本地开发服务仍在运行，请手动停止它${NC}"
  echo -e "${CYAN}通常可以在前端开发终端中按 Ctrl+C 组合键停止${NC}"
fi 