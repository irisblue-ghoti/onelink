#!/bin/bash

# 设置颜色
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# 显示标题
echo -e "${BLUE}==================================================${NC}"
echo -e "${BLUE}          NFC 后端微服务统一启动脚本            ${NC}"
echo -e "${BLUE}==================================================${NC}"

# 定义服务列表
SERVICES=("merchant-service" "content-service" "nfc-service" "distribution-service" "stats-service")

# 检查参数
if [ "$1" == "start" ]; then
    ACTION="start"
elif [ "$1" == "stop" ]; then
    ACTION="stop"
elif [ "$1" == "restart" ]; then
    ACTION="restart"
elif [ "$1" == "status" ]; then
    ACTION="status"
else
    echo -e "${YELLOW}用法: $0 {start|stop|restart|status}${NC}"
    echo -e "${YELLOW}示例: $0 start${NC}"
    exit 1
fi

# 启动服务函数
start_service() {
    SERVICE=$1
    echo -e "${GREEN}正在启动 $SERVICE...${NC}"
    
    # 检查服务目录是否存在
    if [ ! -d "$SERVICE" ]; then
        echo -e "${RED}错误: $SERVICE 目录不存在${NC}"
        return 1
    fi
    
    # 进入服务目录并执行启动脚本
    (cd $SERVICE && ./start.sh) &
    
    # 等待几秒确保服务启动
    sleep 2
    
    # 检查是否成功启动
    if [ -f "$SERVICE/logs/$SERVICE.pid" ]; then
        PID=$(cat $SERVICE/logs/$SERVICE.pid)
        if ps -p $PID > /dev/null; then
            echo -e "${GREEN}$SERVICE 成功启动，PID: $PID${NC}"
            return 0
        else
            echo -e "${RED}$SERVICE 启动失败${NC}"
            return 1
        fi
    else
        echo -e "${RED}$SERVICE 启动失败，未找到PID文件${NC}"
        return 1
    fi
}

# 停止服务函数
stop_service() {
    SERVICE=$1
    echo -e "${YELLOW}正在停止 $SERVICE...${NC}"
    
    # 检查PID文件是否存在
    if [ -f "$SERVICE/logs/$SERVICE.pid" ]; then
        PID=$(cat $SERVICE/logs/$SERVICE.pid)
        if ps -p $PID > /dev/null; then
            kill $PID
            sleep 2
            if ps -p $PID > /dev/null; then
                echo -e "${RED}$SERVICE 停止失败，尝试强制终止...${NC}"
                kill -9 $PID
            fi
            echo -e "${GREEN}$SERVICE 已停止${NC}"
        else
            echo -e "${YELLOW}$SERVICE 不在运行状态${NC}"
        fi
        rm -f $SERVICE/logs/$SERVICE.pid
    else
        echo -e "${YELLOW}$SERVICE 不在运行状态，未找到PID文件${NC}"
    fi
}

# 重启服务函数
restart_service() {
    SERVICE=$1
    stop_service $SERVICE
    sleep 2
    start_service $SERVICE
}

# 检查服务状态函数
check_service_status() {
    SERVICE=$1
    echo -e "${BLUE}检查 $SERVICE 状态...${NC}"
    
    if [ -f "$SERVICE/logs/$SERVICE.pid" ]; then
        PID=$(cat $SERVICE/logs/$SERVICE.pid)
        if ps -p $PID > /dev/null; then
            echo -e "${GREEN}$SERVICE 运行中，PID: $PID${NC}"
        else
            echo -e "${RED}$SERVICE 未运行，但PID文件存在${NC}"
        fi
    else
        echo -e "${YELLOW}$SERVICE 未运行${NC}"
    fi
}

# 根据操作执行相应的动作
case $ACTION in
    start)
        echo -e "${GREEN}开始启动所有后端微服务...${NC}"
        for SERVICE in "${SERVICES[@]}"; do
            start_service $SERVICE
        done
        ;;
    stop)
        echo -e "${YELLOW}开始停止所有后端微服务...${NC}"
        for SERVICE in "${SERVICES[@]}"; do
            stop_service $SERVICE
        done
        ;;
    restart)
        echo -e "${BLUE}开始重启所有后端微服务...${NC}"
        for SERVICE in "${SERVICES[@]}"; do
            restart_service $SERVICE
        done
        ;;
    status)
        echo -e "${BLUE}检查所有后端微服务状态...${NC}"
        for SERVICE in "${SERVICES[@]}"; do
            check_service_status $SERVICE
        done
        ;;
esac

echo -e "${BLUE}==================================================${NC}"
echo -e "${GREEN}操作完成${NC}"
echo -e "${BLUE}==================================================${NC}" 