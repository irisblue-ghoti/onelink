#!/bin/bash

# NFC碰一碰应用Docker启动脚本

# 设置颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # 无颜色

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

# 初始化配置文件
init_config() {
  echo -e "${BLUE}正在检查配置文件...${NC}"
  
  # 运行初始化配置脚本
  if [ -f ./init-config.sh ]; then
    echo -e "${GREEN}执行配置文件初始化...${NC}"
    ./init-config.sh
  else
    echo -e "${RED}错误: 找不到初始化配置脚本 init-config.sh${NC}"
    echo -e "${YELLOW}请确保 init-config.sh 文件存在于项目根目录${NC}"
    exit 1
  fi
}

# 显示标题和菜单
show_menu() {
  clear
  echo -e "${BLUE}"
  echo "╔═══════════════════════════════════════════════════════════╗"
  echo "║                NFC碰一碰应用 - 服务管理系统               ║"
  echo "╚═══════════════════════════════════════════════════════════╝"
  echo -e "${NC}"
  echo -e "${YELLOW}请选择操作:${NC}"
  echo
  echo -e "${CYAN}[ 服务管理 ]${NC}"
  echo -e "  ${GREEN}1.${NC} 启动所有服务           ${YELLOW}(启动全部服务并在终端显示日志)${NC}"
  echo -e "  ${GREEN}2.${NC} 后台启动所有服务       ${YELLOW}(启动全部服务在后台运行)${NC}"
  echo -e "  ${GREEN}3.${NC} 停止所有服务           ${YELLOW}(关闭并移除所有容器)${NC}"
  echo
  echo -e "${CYAN}[ 单独服务 ]${NC}"
  echo -e "  ${GREEN}4.${NC} 启动前端服务           ${YELLOW}(http://localhost:3000)${NC}"
  echo -e "  ${GREEN}5.${NC} 启动后端服务           ${YELLOW}(包含API网关和所有微服务)${NC}"
  echo -e "  ${GREEN}6.${NC} 启动数据库服务         ${YELLOW}(PostgreSQL, Redis, Kafka)${NC}"
  echo
  echo -e "${CYAN}[ 监控与日志 ]${NC}"
  echo -e "  ${GREEN}7.${NC} 查看服务状态           ${YELLOW}(显示所有运行中的容器)${NC}"
  echo -e "  ${GREEN}8.${NC} 查看服务日志           ${YELLOW}(查看指定服务的日志)${NC}"
  echo -e "  ${GREEN}9.${NC} 重新初始化配置         ${YELLOW}(重新生成服务配置文件)${NC}"
  echo
  echo -e "  ${GREEN}0.${NC} 退出"
  echo
  echo -e "${YELLOW}也可以直接使用命令:${NC}"
  echo -e "  ./docker-start.sh up      ${YELLOW}(启动所有服务)${NC}"
  echo -e "  ./docker-start.sh up-d    ${YELLOW}(后台启动所有服务)${NC}"
  echo -e "  ./docker-start.sh down    ${YELLOW}(停止所有服务)${NC}"
  echo -e "  ./docker-start.sh ps      ${YELLOW}(查看服务状态)${NC}"
  echo -e "  ./docker-start.sh logs 服务名  ${YELLOW}(查看指定服务日志)${NC}"
  echo -e "  ./docker-start.sh init    ${YELLOW}(初始化配置文件)${NC}"
  echo
  read -p "请输入选项 [0-9]: " choice
  handle_choice $choice
}

# 启动所有服务
start_all() {
  # 先初始化配置
  init_config
  
  echo -e "${GREEN}正在启动所有服务...${NC}"
  echo -e "${YELLOW}提示: 按 Ctrl+C 停止查看日志，服务会继续在后台运行${NC}"
  echo -e "${YELLOW}等待3秒钟，按Ctrl+C可取消启动...${NC}"
  sleep 3
  docker-compose up
  echo -e "${GREEN}所有服务已启动！${NC}"
}

# 后台启动所有服务
start_all_detached() {
  # 先初始化配置
  init_config
  
  echo -e "${GREEN}正在后台启动所有服务...${NC}"
  docker-compose up -d
  echo -e "${GREEN}所有服务已在后台启动！${NC}"
  echo -e "${YELLOW}=================================${NC}"
  echo -e "${YELLOW}服务访问信息:${NC}"
  echo -e "  前端: ${GREEN}http://localhost:3000${NC}"
  echo -e "  API网关: ${GREEN}http://localhost:9080${NC}"
  echo -e "  PostgreSQL: ${GREEN}localhost:5432${NC}"
  echo -e "  Redis: ${GREEN}localhost:6379${NC}"
  echo -e "  Kafka: ${GREEN}localhost:9092${NC}"
  echo -e "${YELLOW}=================================${NC}"
  echo
  echo -e "按任意键返回主菜单..."
  read -n 1
  show_menu
}

# 启动前端服务
start_frontend() {
  # 先初始化配置
  init_config
  
  echo -e "${GREEN}正在启动前端服务...${NC}"
  docker-compose up -d frontend
  echo -e "${GREEN}前端服务已启动！${NC}"
  echo -e "${YELLOW}前端访问地址: ${GREEN}http://localhost:3000${NC}"
  echo
  echo -e "按任意键返回主菜单..."
  read -n 1
  show_menu
}

# 启动后端服务
start_backend() {
  # 先初始化配置
  init_config
  
  echo -e "${GREEN}正在启动后端服务...${NC}"
  docker-compose up -d apisix content-service merchant-service nfc-service distribution-service stats-service
  echo -e "${GREEN}后端服务已启动！${NC}"
  echo -e "${YELLOW}=================================${NC}"
  echo -e "${YELLOW}服务访问信息:${NC}"
  echo -e "  API网关: ${GREEN}http://localhost:9080${NC}"
  echo -e "  内容服务: ${GREEN}http://localhost:8081${NC}"
  echo -e "  商户服务: ${GREEN}http://localhost:8082${NC}"
  echo -e "  NFC服务: ${GREEN}http://localhost:8083${NC}"
  echo -e "  统计服务: ${GREEN}http://localhost:8084${NC}"
  echo -e "  分发服务: ${GREEN}http://localhost:8085${NC}"
  echo -e "${YELLOW}=================================${NC}"
  echo
  echo -e "按任意键返回主菜单..."
  read -n 1
  show_menu
}

# 启动数据库服务
start_db() {
  echo -e "${GREEN}正在启动数据库服务...${NC}"
  docker-compose up -d postgres redis kafka zookeeper
  echo -e "${GREEN}数据库服务已启动！${NC}"
  echo -e "${YELLOW}=================================${NC}"
  echo -e "${YELLOW}服务访问信息:${NC}"
  echo -e "  PostgreSQL: ${GREEN}localhost:5432${NC}"
  echo -e "    用户名: postgres"
  echo -e "    密码: postgres"
  echo -e "    数据库: nfc_card"
  echo -e "  Redis: ${GREEN}localhost:6379${NC}"
  echo -e "  Kafka: ${GREEN}localhost:9092${NC}"
  echo -e "${YELLOW}=================================${NC}"
  echo
  echo -e "按任意键返回主菜单..."
  read -n 1
  show_menu
}

# 停止所有服务
stop_all() {
  echo -e "${YELLOW}正在停止所有服务...${NC}"
  docker-compose down
  echo -e "${GREEN}所有服务已停止${NC}"
  echo
  echo -e "按任意键返回主菜单..."
  read -n 1
  show_menu
}

# 查看服务状态
check_status() {
  echo -e "${BLUE}服务状态:${NC}"
  docker-compose ps
  echo
  echo -e "按任意键返回主菜单..."
  read -n 1
  show_menu
}

# 查看服务日志
view_logs() {
  clear
  echo -e "${BLUE}查看服务日志${NC}"
  echo -e "${YELLOW}可用服务列表:${NC}"
  echo -e "  ${GREEN}1.${NC} frontend          ${YELLOW}(前端服务)${NC}"
  echo -e "  ${GREEN}2.${NC} apisix            ${YELLOW}(API网关)${NC}"
  echo -e "  ${GREEN}3.${NC} content-service   ${YELLOW}(内容服务)${NC}"
  echo -e "  ${GREEN}4.${NC} merchant-service  ${YELLOW}(商户服务)${NC}"
  echo -e "  ${GREEN}5.${NC} nfc-service       ${YELLOW}(NFC服务)${NC}"
  echo -e "  ${GREEN}6.${NC} distribution-service ${YELLOW}(分发服务)${NC}"
  echo -e "  ${GREEN}7.${NC} stats-service     ${YELLOW}(统计服务)${NC}"
  echo -e "  ${GREEN}8.${NC} postgres          ${YELLOW}(PostgreSQL数据库)${NC}"
  echo -e "  ${GREEN}9.${NC} redis             ${YELLOW}(Redis缓存)${NC}"
  echo -e "  ${GREEN}10.${NC} kafka            ${YELLOW}(Kafka消息队列)${NC}"
  echo -e "  ${GREEN}11.${NC} zookeeper        ${YELLOW}(Zookeeper)${NC}"
  echo -e "  ${GREEN}0.${NC} 返回主菜单"
  echo
  read -p "请选择要查看日志的服务 [0-11]: " log_choice
  
  case $log_choice in
    1) service="frontend" ;;
    2) service="apisix" ;;
    3) service="content-service" ;;
    4) service="merchant-service" ;;
    5) service="nfc-service" ;;
    6) service="distribution-service" ;;
    7) service="stats-service" ;;
    8) service="postgres" ;;
    9) service="redis" ;;
    10) service="kafka" ;;
    11) service="zookeeper" ;;
    0) show_menu
       return ;;
    *) echo -e "${RED}无效选项${NC}"
       sleep 2
       view_logs
       return ;;
  esac
  
  echo -e "${BLUE}正在查看 $service 的日志:${NC}"
  echo -e "${YELLOW}(按 Ctrl+C 退出日志查看)${NC}"
  echo -e "${YELLOW}等待3秒钟，按Ctrl+C可取消...${NC}"
  sleep 3
  
  docker-compose logs -f "$service"
  
  echo
  echo -e "按任意键返回主菜单..."
  read -n 1
  show_menu
}

handle_choice() {
  case $1 in
    1) start_all ;;
    2) start_all_detached ;;
    3) stop_all ;;
    4) start_frontend ;;
    5) start_backend ;;
    6) start_db ;;
    7) check_status ;;
    8) view_logs ;;
    9) init_config
       echo -e "按任意键返回主菜单..."
       read -n 1
       show_menu ;;
    0) exit 0 ;;
    *) 
      echo -e "${RED}无效选项${NC}"
      sleep 2
      show_menu
      ;;
  esac
}

# 命令行模式处理
handle_command_line() {
  case "$1" in
    up)
      start_all
      ;;
    up-d)
      start_all_detached
      ;;
    frontend)
      start_frontend
      ;;
    backend)
      start_backend
      ;;
    db)
      start_db
      ;;
    down)
      stop_all
      ;;
    ps)
      check_status
      ;;
    logs)
      if [ -z "$2" ]; then
        view_logs
      else
        service=$2
        echo -e "${BLUE}查看 $service 的日志:${NC}"
        docker-compose logs -f "$service"
      fi
      ;;
    init)
      init_config
      echo -e "${GREEN}配置初始化完成！${NC}"
      ;;
    *)
      show_menu
      ;;
  esac
}

# 主函数
main() {
  check_docker
  
  # 如果有命令行参数，则按命令行处理
  if [ $# -gt 0 ]; then
    handle_command_line "$@"
  else
    # 否则显示交互式菜单
    show_menu
  fi
}

# 执行主函数
main "$@" 