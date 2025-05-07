#!/bin/bash

# 设置颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # 无颜色

echo -e "${BLUE}正在初始化配置文件...${NC}"

# 创建配置目录
mkdir -p config

# 初始化内容服务配置
if [ ! -f config/content-service.yaml ]; then
  echo -e "${YELLOW}创建内容服务配置文件${NC}"
  cat > config/content-service.yaml << EOF
server:
  port: 8081
  
database:
  postgres:
    host: postgres
    port: 5432
    user: postgres
    password: postgres
    dbname: nfc_card
    sslmode: disable
    
redis:
  addr: redis:6379
  password: ""
  db: 0
  
kafka:
  brokers:
    - kafka:9092
  
log:
  level: debug
  output: stdout

nacos:
  server_addr: nacos:8848
  namespace_id: public
  group: DEFAULT_GROUP
  enable: true
EOF
fi

# 初始化商户服务配置
if [ ! -f config/merchant-service.yaml ]; then
  echo -e "${YELLOW}创建商户服务配置文件${NC}"
  cat > config/merchant-service.yaml << EOF
server:
  port: 8082
  
database:
  postgres:
    host: postgres
    port: 5432
    user: postgres
    password: postgres
    dbname: nfc_card
    sslmode: disable
    
redis:
  addr: redis:6379
  password: ""
  db: 0
  
log:
  level: debug
  output: stdout

nacos:
  server_addr: nacos:8848
  namespace_id: public
  group: DEFAULT_GROUP
  enable: true
EOF
fi

# 初始化NFC服务配置
if [ ! -f config/nfc-service.yaml ]; then
  echo -e "${YELLOW}创建NFC服务配置文件${NC}"
  cat > config/nfc-service.yaml << EOF
server:
  port: 8083
  
database:
  postgres:
    host: postgres
    port: 5432
    user: postgres
    password: postgres
    dbname: nfc_card
    sslmode: disable
    
redis:
  addr: redis:6379
  password: ""
  db: 0
  
log:
  level: debug
  output: stdout

nacos:
  server_addr: nacos:8848
  namespace_id: public
  group: DEFAULT_GROUP
  enable: true
EOF
fi

# 初始化统计服务配置
if [ ! -f config/stats-service.yaml ]; then
  echo -e "${YELLOW}创建统计服务配置文件${NC}"
  cat > config/stats-service.yaml << EOF
server:
  port: 8084
  
database:
  postgres:
    host: postgres
    port: 5432
    user: postgres
    password: postgres
    dbname: nfc_card
    sslmode: disable
    
redis:
  addr: redis:6379
  password: ""
  db: 0
  
kafka:
  brokers:
    - kafka:9092
  
log:
  level: debug
  output: stdout

nacos:
  server_addr: nacos:8848
  namespace_id: public
  group: DEFAULT_GROUP
  enable: true
EOF
fi

# 初始化分发服务配置
if [ ! -f config/distribution-service.yaml ]; then
  echo -e "${YELLOW}创建分发服务配置文件${NC}"
  cat > config/distribution-service.yaml << EOF
server:
  port: 8085
  
database:
  postgres:
    host: postgres
    port: 5432
    user: postgres
    password: postgres
    dbname: nfc_card
    sslmode: disable
    
redis:
  addr: redis:6379
  password: ""
  db: 0
  
kafka:
  brokers:
    - kafka:9092
  
log:
  level: debug
  output: stdout

nacos:
  server_addr: nacos:8848
  namespace_id: public
  group: DEFAULT_GROUP
  enable: true
EOF
fi

echo -e "${GREEN}配置文件初始化完成！${NC}" 