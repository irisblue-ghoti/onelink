# NFC 碰一碰 - 一键分发短视频 SaaS 系统

基于 NFC 碰一碰技术的短视频多平台一键分发 SaaS 系统，支持抖音、快手、小红书和微信朋友圈。

## 系统概述

本系统是一个多租户 SaaS 平台，通过 NFC 碰一碰技术，实现短视频内容的快速多平台分发。

### 主要功能

- 多商户（租户）隔离，独立内容与用户后台
- 平台方的全局商户运营与计费管理
- NFC 卡片碰触触发短链落地页打开
- 支持抖音、快手（API全自动）、小红书与微信朋友圈（半自动分享）
- 视频上传、转码、内容安全、统计回流、分润计费等全链路功能

### 技术栈

- **前端**：Next.js + Ant Design Pro
- **后端**：NestJS (Node.js) + Go 微服务
- **数据库**：PostgreSQL + Row-Level Security
- **API网关**：APISIX
- **消息队列**：Kafka
- **身份认证**：Auth0 / Keycloak OIDC
- **内容服务**：对象存储 (S3/OSS)、FFmpeg 转码、阿里云/火山审核
- **短链服务**：Cloudflare Workers
- **部署**：Docker + Kubernetes + Helm

## 快速开始

详见 [部署文档](docs/deployment.md)。

## 项目结构

```
nfc_card/
├── apisix/                # API网关配置
├── backend/               # 后端微服务
│   ├── content-service/   # 内容服务(Go)
│   ├── merchant-service/  # 商户服务(Go)
│   ├── nfc-service/       # NFC卡服务(Go)
│   ├── distribution-service/ # 分发服务(Go)
│   ├── stats-service/     # 统计服务(Go)
│   └── start.sh           # 后端服务统一启动脚本
├── docs/                  # 项目文档
├── frontend/              # 前端应用
├── migrations/            # 数据库迁移脚本
├── shared/                # 共享代码
├── docker-compose.yaml    # Docker Compose配置
└── start.sh               # 项目启动脚本
```

## 开发环境搭建

### 启动项目

```bash
# 启动所有服务
./start.sh start

# 只启动后端服务
cd backend && ./start.sh start

# 停止所有服务
./start.sh stop

# 重启所有服务
./start.sh restart
```

### 后端微服务

所有后端微服务现已统一使用Go语言实现，并整合到backend目录下，可以使用统一的脚本进行管理：

```bash
# 进入后端目录
cd backend

# 启动所有后端服务
./start.sh start

# 停止所有后端服务
./start.sh stop

# 重启所有后端服务
./start.sh restart

# 查看后端服务状态
./start.sh status

# 可以单独操作某个服务
cd backend/merchant-service && ./start.sh
```

## 许可证

本项目采用 MIT 许可证。

# NFC 卡片系统

这是一个基于微服务架构的NFC卡片管理系统。

## 系统架构

该系统包含以下微服务：

- **商户服务 (Merchant Service)**: 管理商户、用户及订阅计划
- **内容服务 (Content Service)**: 管理视频内容和相关资源
- **NFC服务 (NFC Service)**: 管理NFC卡片和短链接
- **分发服务 (Distribution Service)**: 负责内容分发到各个渠道（抖音、快手等）
- **统计服务 (Stats Service)**: 收集和分析各种统计数据

以及以下基础设施服务：

- PostgreSQL: 关系型数据库
- Redis: 缓存和会话存储
- MinIO: 对象存储 (S3兼容)
- Kafka: 消息队列
- APISIX: API网关

## 启动系统

使用以下命令启动系统：

```bash
# 启动所有服务（前台模式）
./start.sh

# 后台模式启动
./start.sh --background

# 仅启动基础设施服务
./start.sh --infra-only

# 生产模式启动
./start.sh --prod
```

## 停止系统

使用以下命令停止系统：

```bash
# 停止所有服务
./stop.sh

# 仅停止基础设施服务
./stop.sh --infra-only

# 强制停止所有服务
./stop.sh --force
```

## 修复常见问题

### 修复商户服务

如果商户服务启动失败，可能是由于以下原因：

1. TypeScript编译错误
2. 实体定义与数据库表结构不匹配
3. 数据库中存在NULL值约束冲突

可以使用以下命令修复商户服务：

```bash
# 运行修复脚本
./fix-merchant-service.sh

# 然后执行SQL脚本修复数据库
docker exec -i nfc_postgres psql -U postgres -d nfc_card -f - < merchant-service/fix-merchants.sql
```

修复脚本会执行以下操作：

1. 清理缓存和构建文件
2. 安装/更新依赖
3. 创建环境配置文件
4. 修复数据库配置（禁用synchronize以防止自动表结构修改）
5. 修复实体定义文件
6. 生成SQL脚本，为数据库中的NULL值生成默认值
7. 重新编译项目

修复后，可以使用以下命令启动商户服务：

```bash
cd merchant-service && NODE_ENV=development npm run start:prod
```

或使用系统启动脚本重新启动整个系统：

```bash
./start.sh --background
```

# NFC碰一碰内容分发系统

NFC碰一碰是一个基于NFC技术的内容分发系统，用户可以通过碰一碰或扫描NFC卡片快速获取内容，商家可以管理和分发视频内容。

## 系统架构

该系统由以下服务组成：

- **前端服务 (Frontend)**: 基于Next.js的用户界面
- **API网关 (APISIX)**: 请求路由和认证
- **商户服务 (Merchant Service)**: 商户和用户管理
- **内容服务 (Content Service)**: 内容管理
- **NFC服务 (NFC Service)**: NFC卡片和短链接管理
- **分发服务 (Distribution Service)**: 内容分发到各平台
- **统计服务 (Stats Service)**: 数据统计和分析

## 环境要求

- Docker
- Docker Compose
- Git

## 快速开始

### 克隆仓库

```bash
git clone https://github.com/yourusername/nfc_card.git
cd nfc_card
```

### 使用启动脚本

我们提供了一个方便的启动脚本来管理服务：

```bash
# 给启动脚本添加执行权限
chmod +x start.sh

# 启动所有服务
./start.sh all

# 仅启动前端
./start.sh frontend

# 仅启动API和后端服务
./start.sh api

# 仅启动数据库服务
./start.sh db

# 仅启动Kafka服务
./start.sh kafka

# 停止所有服务
./start.sh stop

# 查看服务状态
./start.sh status

# 查看服务日志（例如前端服务）
./start.sh logs frontend
```

### 手动使用Docker Compose

如果你不想使用启动脚本，也可以直接使用Docker Compose命令：

```bash
# 启动所有服务
docker-compose up -d

# 停止所有服务
docker-compose down

# 查看服务日志
docker-compose logs -f [服务名]
```

## 访问服务

- 前端: http://localhost:3000
- API网关: http://localhost:9080

## 开发说明

### 目录结构

```
nfc_card/
├── apisix/              # API网关配置
├── content-service/     # 内容服务（NestJS）
├── merchant-service/    # 商户服务（NestJS）
├── frontend/            # 前端应用（Next.js）
├── distribution-service/# 分发服务（Go）
├── stats-service/       # 统计服务（Go）
├── nfc-service/         # NFC服务（Go）
├── shared/              # 共享库
├── docs/                # 文档
├── migrations/          # 数据库迁移脚本
├── docker-compose.yaml  # Docker Compose配置
└── start.sh             # 启动脚本
```

### 环境变量

各服务的环境变量在`docker-compose.yaml`文件中配置。如需自定义，可以创建`.env`文件或直接修改`docker-compose.yaml`。

### 数据持久化

所有数据都通过Docker卷进行持久化，包括：

- PostgreSQL数据：`postgres_data`
- Redis数据：`redis_data`
- Kafka数据：`kafka_data`
- 媒体文件：`media_data`

## 故障排除

如果遇到问题，请检查：

1. Docker和Docker Compose是否已正确安装
2. 端口是否被占用
3. 日志中是否有错误信息（使用`./start.sh logs <服务名>`）

## 维护与支持

如有问题，请联系项目维护者或提交Issue。

# NFC卡片项目统一日志系统

本项目实现了一个统一的日志系统，包括以下功能：

1. 统一的日志格式和级别
2. 集中式日志收集（基于ELK - Elasticsearch, Logstash, Kibana）
3. 请求追踪功能（Trace ID）

## 安装

使用Go Modules安装：

```bash
# 初始化模块（如果尚未初始化）
go mod init github.com/yourname/yourproject

# 添加依赖
go get github.com/nfc_card/shared/logger
go get github.com/nfc_card/shared/middleware
```

## 主要功能

### 1. 统一日志格式和级别

- 支持多种日志级别：debug, info, warn, error, fatal
- 标准化的JSON格式日志输出
- 支持文件和控制台输出
- 包含丰富的上下文信息（时间戳、服务名、文件、行号等）

### 2. 集中式日志收集

- 基于ELK（Elasticsearch, Logstash, Kibana）的集中式日志收集
- 批量日志发送，提高性能
- 支持多种日志来源（文件、TCP、UDP）
- 灵活的日志索引策略

### 3. 请求追踪功能

- 通过Trace ID跟踪请求在不同服务间的流转
- 支持HTTP请求的自动追踪ID注入
- 支持Kafka消息的追踪ID传递
- 在日志中自动关联追踪ID

## 使用示例

### 初始化日志系统

```go
package main

import (
    "github.com/nfc_card/shared/logger"
)

func main() {
    // 初始化日志系统
    log, err := logger.InitLogger("my-service", "")
    if err != nil {
        panic("初始化日志系统失败: " + err.Error())
    }
    
    // 使用日志
    log.Info("服务启动成功")
}
```

### 使用HTTP中间件自动处理追踪ID

```go
package main

import (
    "net/http"
    "github.com/nfc_card/shared/middleware"
    "github.com/nfc_card/shared/logger"
)

func main() {
    // 初始化日志
    logger.InitLogger("web-service", "")
    
    // 创建HTTP处理器
    mux := http.NewServeMux()
    mux.HandleFunc("/api/hello", handleHello)
    
    // 应用追踪中间件
    handler := middleware.TraceMiddleware(mux)
    
    // 启动服务器
    http.ListenAndServe(":8080", handler)
}

func handleHello(w http.ResponseWriter, r *http.Request) {
    // 从请求上下文获取追踪ID
    ctx := r.Context()
    
    // 记录带有追踪ID的日志
    logger.DefaultLogger.InfoContext(ctx, "处理/api/hello请求")
    
    w.Write([]byte("Hello, World!"))
}
```

### 使用Kafka传递追踪ID

```go
package main

import (
    "context"
    "github.com/nfc_card/shared/kafka"
    "github.com/nfc_card/shared/logger"
)

func main() {
    // 初始化日志
    logger.InitLogger("kafka-service", "")
    
    // 创建Kafka客户端
    cfg := &kafka.Config{
        Brokers:       []string{"localhost:9092"},
        ConsumerGroup: "my-group",
        ConsumerTopics: []string{"my-topic"},
        ServiceName:    "kafka-service",
    }
    
    client, _ := kafka.NewClient(cfg)
    defer client.Close()
    
    // 发送带有追踪ID的消息
    ctx := context.Background()
    ctx = logger.WithTraceID(ctx, logger.GenerateTraceID())
    
    client.SendMessageWithContext(ctx, "my-topic", "user.created", map[string]interface{}{
        "id": 123,
        "name": "测试用户",
    })
}
```

## 配置ELK环境

项目提供了Docker Compose配置，可以轻松启动ELK环境：

```bash
# 启动ELK服务
docker-compose -f docker-compose-elk.yaml up -d
```

启动后，可以通过以下地址访问Kibana：

```
http://localhost:5601
```

## 环境变量配置

日志系统支持通过环境变量进行配置：

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| LOG_LEVEL | 日志级别 | info |
| LOG_FILE_PATH | 日志文件路径 | logs/{service}.log |
| LOG_CONSOLE_OUTPUT | 是否输出到控制台 | true |
| LOG_JSON_FORMAT | 是否使用JSON格式 | true |
| LOG_REPORT_CALLER | 是否报告调用者信息 | true |
| ELASTICSEARCH_ENABLED | 是否启用Elasticsearch日志收集 | false |
| ELASTICSEARCH_URL | Elasticsearch地址 | http://elasticsearch:9200 |
| ELASTICSEARCH_INDEX_PREFIX | 索引名前缀 | logs-{service} |
| ELASTICSEARCH_USERNAME | Elasticsearch用户名 | |
| ELASTICSEARCH_PASSWORD | Elasticsearch密码 | |
| ELASTICSEARCH_BATCH_SIZE | 批量发送大小 | 100 |
| ELASTICSEARCH_FLUSH_INTERVAL | 刷新间隔（秒） | 5 |

## 详细文档

更多详细信息请参考：

- [日志系统文档](docs/logging.md)
- [API参考](docs/logger-api.md)
- [ELK配置指南](docs/elk-setup.md)

## 开发工具

### 数据库与后端实体对齐工具

我们提供了一个工具，用于检查和修复后端服务的Go实体模型与PostgreSQL数据库结构的不一致问题。

#### 检查数据库与实体的不一致

```bash
# 编译工具
cd cmd/db_align && go build

# 检查不一致
./db_align check
```

#### 自动修复不一致(演示模式)

```bash
./db_align align
```

#### 应用修复

```bash
./db_align align --apply
```

#### 可用选项

```
--apply          应用更改（默认为演示模式）
--merchant-id    使用merchantId作为租户ID命名（默认为tenantId）
--snake-case     使用snake_case作为JSON命名风格（默认为camelCase）
--no-id-convert  不转换ID类型
--no-tenant-unify 不统一租户ID命名
--no-json-unify  不统一JSON命名风格
--no-db-tags     不更新DB标签
--help           显示帮助信息
```

#### 示例用法

1. 仅检查不一致:
   ```bash
   ./db_align
   ```

2. 演示模式查看将要进行的更改:
   ```bash
   ./db_align align
   ```

3. 应用所有更改:
   ```bash
   ./db_align align --apply
   ```

4. 仅统一ID类型和JSON命名风格:
   ```bash
   ./db_align align --apply --no-tenant-unify --no-db-tags
   ```

更多信息请参见 [工具文档](cmd/db_align/README.md)。 