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
    --frontend-local) 
      FRONTEND_LOCAL=true
      shift
      ;;
    all|backend|frontend|db|help|*)
      break
      ;;
  esac
done

# 显示标题
echo -e "${BLUE}"
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║              NFC碰一碰应用 - 服务重启脚本                 ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# 服务列表
SERVICES=("frontend" "apisix" "nacos" "minio" "content-service" "merchant-service" "nfc-service" "distribution-service" "stats-service" "postgres" "redis" "kafka" "zookeeper")

# 显示帮助信息
show_help() {
  echo -e "${YELLOW}用法: $0 [选项] [服务名]${NC}"
  echo
  echo -e "选项:"
  echo -e "  ${GREEN}--frontend-local${NC} 前端本地开发模式，跳过前端Docker操作"
  echo -e "  ${GREEN}all${NC}       重启所有服务"
  echo -e "  ${GREEN}backend${NC}   重启所有后端服务 (不包括数据库)"
  echo -e "  ${GREEN}frontend${NC}  仅重启前端服务"
  echo -e "  ${GREEN}db${NC}        重启数据库服务 (postgres, redis, kafka, zookeeper, minio)"
  echo -e "  ${GREEN}help${NC}      显示此帮助信息"
  echo
  echo -e "单个服务重启:"
  echo -e "  ${CYAN}frontend${NC}             - 前端服务"
  echo -e "  ${CYAN}apisix${NC}               - API网关"
  echo -e "  ${CYAN}nacos${NC}                - 服务注册中心"
  echo -e "  ${CYAN}minio${NC}                - 对象存储服务"
  echo -e "  ${CYAN}content-service${NC}      - 内容服务"
  echo -e "  ${CYAN}merchant-service${NC}     - 商户服务"
  echo -e "  ${CYAN}nfc-service${NC}          - NFC服务"
  echo -e "  ${CYAN}distribution-service${NC} - 分发服务"
  echo -e "  ${CYAN}stats-service${NC}        - 统计服务"
  echo -e "  ${CYAN}postgres${NC}             - PostgreSQL数据库"
  echo -e "  ${CYAN}redis${NC}                - Redis缓存"
  echo -e "  ${CYAN}kafka${NC}                - Kafka消息队列"
  echo -e "  ${CYAN}zookeeper${NC}            - Zookeeper"
  echo
  echo -e "示例:"
  echo -e "  $0 all               ${YELLOW}# 重启所有服务${NC}"
  echo -e "  $0 --frontend-local all ${YELLOW}# 重启所有服务，但前端使用本地开发模式${NC}"
  echo -e "  $0 backend           ${YELLOW}# 仅重启后端服务${NC}"
  echo -e "  $0 content-service   ${YELLOW}# 仅重启内容服务${NC}"
  echo -e "  $0 nacos             ${YELLOW}# 仅重启Nacos服务${NC}"
  echo -e "  $0 minio             ${YELLOW}# 仅重启MinIO服务${NC}"
}

# 检查Docker是否安装和运行
check_docker() {
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

  if ! command -v docker-compose &> /dev/null; then
    if ! docker compose version &> /dev/null; then
      echo -e "${RED}错误: Docker Compose未安装${NC}"
      echo "请安装Docker Compose: https://docs.docker.com/compose/install/"
      exit 1
    fi
  fi
}

# 确认重启
confirm_restart() {
  local service=$1
  if [ "$service" == "all" ]; then
    read -p "确定要重启所有服务吗? (y/n): " confirm
  else
    read -p "确定要重启 $service 吗? (y/n): " confirm
  fi
  
  if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo -e "${YELLOW}操作已取消${NC}"
    exit 0
  fi
}

# 重启单个服务
restart_service() {
  local service=$1
  
  # 检查服务是否有效
  if [[ ! " ${SERVICES[@]} " =~ " ${service} " ]]; then
    echo -e "${RED}错误: 未知服务 '$service'${NC}"
    show_help
    exit 1
  fi
  
  # 如果是前端服务且设置了本地开发模式，则跳过
  if [ "$service" == "frontend" ] && [ "$FRONTEND_LOCAL" = true ]; then
    echo -e "${YELLOW}前端设置为本地开发模式，跳过Docker容器重启${NC}"
    echo -e "${CYAN}请在本地手动重启前端开发服务:${NC}"
    echo -e "${YELLOW}cd frontend && npm run dev${NC}"
    return 0
  fi
  
  # 重建镜像（如果服务不是基础设施服务）
  if [[ ! " postgres redis kafka zookeeper nacos minio " =~ " ${service} " ]]; then
    echo -e "${CYAN}正在重新构建 $service 镜像...${NC}"
    
    # 确定构建上下文路径
    local build_context=""
    if [ "$service" == "frontend" ]; then
      build_context="./frontend"
    elif [ "$service" == "apisix" ]; then
      # APISIX通常使用预构建镜像，不需要本地构建
      echo -e "${YELLOW}APISIX服务通常使用预构建镜像，跳过构建步骤${NC}"
    else
      build_context="./backend/$service"
    fi
    
    # 如果需要构建，则执行构建
    if [ -n "$build_context" ] && [ -d "$build_context" ]; then
      echo -e "${YELLOW}正在构建 $service，使用最新代码...${NC}"
      docker-compose build $service
      
      if [ $? -ne 0 ]; then
        echo -e "${RED}构建 $service 失败，将使用现有镜像重启${NC}"
      else
        echo -e "${GREEN}$service 镜像构建成功!${NC}"
      fi
    fi
  fi
  
  echo -e "${CYAN}正在重启 $service...${NC}"
  docker-compose restart $service
  
  # 检查重启是否成功
  if [ $? -eq 0 ]; then
    echo -e "${GREEN}$service 重启成功!${NC}"
    
    # 对于特殊服务，等待它完全启动
    check_special_service_status "$service"
  else
    echo -e "${RED}$service 重启失败!${NC}"
    echo -e "${YELLOW}尝试停止再启动 $service...${NC}"
    
    docker-compose stop $service
    sleep 2
    docker-compose up -d $service
    
    if [ $? -eq 0 ]; then
      echo -e "${GREEN}$service 已成功停止并重新启动!${NC}"
      
      # 对于特殊服务，等待它完全启动
      check_special_service_status "$service"
    else
      echo -e "${RED}$service 重启失败，请检查日志!${NC}"
      exit 1
    fi
  fi
}

# 检查特殊服务的状态
check_special_service_status() {
  local service=$1
  
  # 检查Nacos健康状态
  if [ "$service" == "nacos" ]; then
    echo -e "${CYAN}等待Nacos服务完全启动...${NC}"
    sleep 5
    curl -s http://localhost:8848/nacos/v1/console/health/status > /dev/null
    if [ $? -eq 0 ]; then
      echo -e "${GREEN}Nacos服务已成功启动!${NC}"
    else
      echo -e "${YELLOW}Nacos服务可能尚未完全启动，请稍后手动检查${NC}"
    fi
  fi
  
  # 检查MinIO健康状态
  if [ "$service" == "minio" ]; then
    echo -e "${CYAN}等待MinIO服务完全启动...${NC}"
    sleep 3
    curl -s http://localhost:9000/minio/health/live > /dev/null
    if [ $? -eq 0 ]; then
      echo -e "${GREEN}MinIO服务已成功启动!${NC}"
    else
      echo -e "${YELLOW}MinIO服务可能尚未完全启动，请稍后手动检查${NC}"
    fi
  fi
}

# 重启所有服务
restart_all() {
  echo -e "${YELLOW}正在重启所有服务...${NC}"
  
  # 首先重新构建所有服务
  echo -e "${BLUE}步骤 1/2: 重新构建所有服务，使用最新代码...${NC}"
  
  if [ "$FRONTEND_LOCAL" = true ]; then
    echo -e "${YELLOW}前端设置为本地开发模式，跳过前端Docker镜像构建${NC}"
    # 构建除前端外的所有服务
    for service in apisix content-service merchant-service nfc-service distribution-service stats-service; do
      echo -e "${CYAN}构建 $service...${NC}"
      docker-compose build $service
    done
  else
    # 构建所有服务
    docker-compose build
  fi
  
  if [ $? -ne 0 ]; then
    echo -e "${RED}警告: 部分服务构建失败，将尝试继续重启${NC}"
  else
    echo -e "${GREEN}所有服务构建成功!${NC}"
  fi
  
  # 重启所有服务，保留容器
  echo -e "${BLUE}步骤 2/2: 正在重启所有服务，不删除容器...${NC}"
  
  if [ "$FRONTEND_LOCAL" = true ]; then
    echo -e "${YELLOW}前端设置为本地开发模式，跳过前端Docker容器重启${NC}"
    # 重启除前端外的所有服务
    for service in apisix nacos minio content-service merchant-service nfc-service distribution-service stats-service postgres redis kafka zookeeper; do
      echo -e "${CYAN}重启 $service...${NC}"
      docker-compose restart $service
      sleep 1
    done
    echo -e "${CYAN}请在本地手动重启前端开发服务:${NC}"
    echo -e "${YELLOW}cd frontend && npm run dev${NC}"
  else
    # 重启所有服务
    docker-compose restart
  fi
  
  if [ $? -eq 0 ]; then
    echo -e "${GREEN}所有服务已成功重启!${NC}"
    
    # 检查特殊服务状态
    echo -e "${CYAN}等待特殊服务完全启动...${NC}"
    
    # 检查Nacos健康状态
    sleep 5
    curl -s http://localhost:8848/nacos/v1/console/health/status > /dev/null
    if [ $? -eq 0 ]; then
      echo -e "${GREEN}Nacos服务已成功启动!${NC}"
    else
      echo -e "${YELLOW}Nacos服务可能尚未完全启动，请稍后手动检查${NC}"
    fi
    
    # 检查MinIO健康状态
    curl -s http://localhost:9000/minio/health/live > /dev/null
    if [ $? -eq 0 ]; then
      echo -e "${GREEN}MinIO服务已成功启动!${NC}"
    else
      echo -e "${YELLOW}MinIO服务可能尚未完全启动，请稍后手动检查${NC}"
    fi
  else
    echo -e "${RED}重启服务失败!${NC}"
    echo -e "${YELLOW}尝试逐个重启服务...${NC}"
    
    # 按特定顺序重启服务
    restart_zookeeper_kafka_first
    
    # 然后重启Nacos和MinIO
    for service in minio nacos; do
      echo -e "${CYAN}重启 $service...${NC}"
      docker-compose restart $service
      sleep 3
      check_special_service_status "$service"
    done
    
    # 最后重启其他服务
    for service in apisix content-service merchant-service nfc-service distribution-service stats-service; do
      echo -e "${CYAN}重启 $service...${NC}"
      docker-compose restart $service
      sleep 2
    done
    
    # 如果不是本地开发模式，也重启前端
    if [ "$FRONTEND_LOCAL" = false ]; then
      echo -e "${CYAN}重启 frontend...${NC}"
      docker-compose restart frontend
      sleep 2
    else
      echo -e "${YELLOW}前端设置为本地开发模式，跳过Docker容器重启${NC}"
      echo -e "${CYAN}请在本地手动重启前端开发服务:${NC}"
      echo -e "${YELLOW}cd frontend && npm run dev${NC}"
    fi
    
    echo -e "${GREEN}所有服务重启完成!${NC}"
  fi
}

# 重启所有后端服务
restart_backend() {
  echo -e "${YELLOW}正在重启所有后端服务...${NC}"
  
  local backend_services=("apisix" "nacos" "minio" "content-service" "merchant-service" "nfc-service" "distribution-service" "stats-service")
  
  # 先构建后端服务
  echo -e "${BLUE}步骤 1/4: 重新构建后端服务，使用最新代码...${NC}"
  for service in "${backend_services[@]}"; do
    if [[ ! " apisix nacos minio " =~ " ${service} " ]]; then # 跳过apisix、nacos和minio构建
      echo -e "${CYAN}构建 $service...${NC}"
      docker-compose build $service
    fi
  done
  
  # 先重启MinIO服务
  echo -e "${BLUE}步骤 2/4: 重启MinIO服务...${NC}"
  echo -e "${CYAN}重启 minio...${NC}"
  docker-compose restart minio
  sleep 3
  check_special_service_status "minio"
  
  # 然后重启Nacos服务
  echo -e "${BLUE}步骤 3/4: 重启Nacos服务...${NC}"
  echo -e "${CYAN}重启 nacos...${NC}"
  docker-compose restart nacos
  sleep 5
  check_special_service_status "nacos"
  
  # 然后重启其他后端服务
  echo -e "${BLUE}步骤 4/4: 重启其他后端服务...${NC}"
  for service in "${backend_services[@]}"; do
    if [[ ! " nacos minio " =~ " ${service} " ]]; then
      echo -e "${CYAN}重启 $service...${NC}"
      docker-compose restart $service
      sleep 2
    fi
  done
  
  echo -e "${GREEN}所有后端服务已重启!${NC}"
}

# 重启数据库服务
restart_db() {
  echo -e "${YELLOW}正在重启数据库服务...${NC}"
  
  local db_services=("redis" "postgres" "kafka" "zookeeper" "minio")
  
  for service in "${db_services[@]}"; do
    echo -e "${CYAN}重启 $service...${NC}"
    docker-compose restart $service
    sleep 2
    
    # 对于MinIO，检查健康状态
    if [ "$service" == "minio" ]; then
      check_special_service_status "minio"
    fi
  done
  
  echo -e "${GREEN}所有数据库服务已重启!${NC}"
}

# 辅助函数：按照特定顺序重启zookeeper和kafka
restart_zookeeper_kafka_first() {
  echo -e "${CYAN}重启 zookeeper...${NC}"
  docker-compose restart zookeeper
  sleep 3
  
  echo -e "${CYAN}重启 kafka...${NC}"
  docker-compose restart kafka
  sleep 3
  
  echo -e "${CYAN}重启 postgres...${NC}"
  docker-compose restart postgres
  sleep 2
  
  echo -e "${CYAN}重启 redis...${NC}"
  docker-compose restart redis
  sleep 2
}

# 检查服务状态
check_status() {
  echo -e "${BLUE}服务状态:${NC}"
  docker-compose ps
}

# 主函数
main() {
  # 检查Docker环境
  check_docker
  
  # 显示当前模式
  if [ "$FRONTEND_LOCAL" = true ]; then
    echo -e "${YELLOW}前端模式: 本地开发${NC}"
    echo -e "${CYAN}注意: 前端将不会通过Docker启动，请手动在本地启动前端${NC}"
  else
    echo -e "${YELLOW}前端模式: Docker容器${NC}"
  fi
  
  # 如果没有参数或者参数是help，显示帮助
  if [ $# -eq 0 ] || [ "$1" == "help" ]; then
    show_help
    exit 0
  fi
  
  # 处理参数
  case "$1" in
    all)
      confirm_restart "all"
      restart_all
      ;;
    backend)
      confirm_restart "backend"
      restart_backend
      ;;
    frontend)
      confirm_restart "frontend"
      restart_service "frontend"
      ;;
    db)
      confirm_restart "db"
      restart_db
      ;;
    *)
      # 重启单个服务
      confirm_restart "$1"
      restart_service "$1"
      ;;
  esac
  
  # 显示服务状态
  echo -e "${YELLOW}重启后的服务状态:${NC}"
  check_status
}

# 执行主函数并传递所有参数
main "$@" 