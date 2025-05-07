# NFC 碰一碰分发系统架构文档

## 架构图

```
+----------------------------------------------------------------------------------------------------------+
|                                           客户端                                                           |
|  +----------------+   +----------------+   +----------------+   +----------------+   +----------------+   |
|  |   商户后台      |   |   平台后台      |   |   用户前端      |   |   抖音/快手     |   |   小红书/微信   |   |
|  | (Next.js+AntD) |   | (Next.js+AntD) |   | (Next.js+React)|   |     App        |   |     App        |   |
|  +-------+--------+   +-------+--------+   +-------+--------+   +-------+--------+   +-------+--------+   |
+----------|--------------------|-------------------|------------------|------------------|----------------+
           |                    |                   |                  |                  |
           |                    |                   |                  |                  |
+----------v--------------------v-------------------v------------------v------------------v----------------+
|                                         API 网关层 (APISIX)                                              |
|  +----------------------------------------------------------------------------------------+             |
|  |                                                                                        |             |
|  |  +----------------+   +----------------+   +----------------+   +----------------+     |             |
|  |  | 认证插件        |   | 限流插件        |   | 租户识别插件     |   | 熔断/灰度插件    |     |             |
|  |  | (JWT/OIDC)     |   | (Rate Limit)   |   | (Tenant ID)    |   | (Circuit Break) |     |             |
|  |  +----------------+   +----------------+   +----------------+   +----------------+     |             |
|  +----------------------------------------------------------------------------------------+             |
+---------------------------------------+---------------+----------------+-------------------------+-------+
                                        |               |                |                         |
+-----------+    +-------------------+  |               |                |  +-----------------+    |
| Auth 服务  |<-->|  身份认证服务       |  |               |                |  |  计费服务        |<-->|
| (Keycloak) |    | (Auth0/Keycloak)  |  |               |                |  | (Stripe API)    |    |
+-----------+    +-------------------+  |               |                |  +-----------------+    |
                                        v               v                v                         v
+---------------------------------------+---------------+----------------+-------------------------+-------+
|                                           应用服务层                                                      |
|  +----------------+   +----------------+   +----------------+   +----------------+   +----------------+   |
|  |   商户服务      |   |   内容服务      |   |   分发服务      |   |   统计服务      |   |   NFC服务      |   |
|  | (Go)           |   | (Go)           |   | (Go)           |   | (Go)           |   | (Go)           |   |
|  +-------+--------+   +-------+--------+   +-------+--------+   +-------+--------+   +-------+--------+   |
+----------|--------------------|-------------------|------------------|------------------|----------------+
           |                    |                   |                  |                  |
           v                    v                   v                  v                  v
+-------------------------------------------------------------------------------------------------------+
|                                      服务注册与发现中心 (Nacos)                                            |
+-------------------------------------------------------------------------------------------------------+
           |                    |                   |                  |                  |
           v                    v                   v                  v                  v
+-------------------------------------------------------------------------------------------------------+
|                                          消息/任务队列 (Kafka)                                           |
+-------------------------------------------------------------------------------------------------------+
           |                    |                   |                  |                  |
           v                    v                   v                  v                  v
+----------|--------------------|-------------------|------------------|------------------|----------------+
|                                           微服务层                                                       |
|  +----------------+   +----------------+   +-------------------------+   +----------------+              |
|  |   转码服务      |   |   内容审核      |   |   平台分发适配器          |   |   短链服务      |              |
|  | (FFmpeg)       |   | (阿里云/火山)   |   | (抖音/快手/小红书/微信)    |   | (CF Workers)   |              |
|  +-------+--------+   +-------+--------+   +------------+------------+   +-------+--------+              |
+----------|--------------------|--------------------------|--------------------------|--------------------+
           |                    |                          |                          |
           v                    v                          v                          v
+-------------------------------------------------------------------------------------------------------+
|                                           存储层                                                        |
|  +----------------+   +----------------+   +----------------+   +----------------+                     |
|  |   PostgreSQL   |   |   对象存储      |   |   Redis缓存     |   |   时序数据库     |                     |
|  | (RLS多租户)     |   | (S3/OSS)       |   | (API缓存)       |   | (监控/统计)      |                     |
|  +----------------+   +----------------+   +----------------+   +----------------+                     |
+-------------------------------------------------------------------------------------------------------+
```

## 服务注册与发现

本系统使用Nacos作为服务注册与发现中心，实现微服务之间的动态服务发现和负载均衡。Nacos提供了服务注册、健康检查、配置管理等功能，为系统提供了高可用的服务治理能力。

### Nacos架构

```
+-------------------------------------------------------------------+
|                         服务调用方                                  |
|  +------------------+      +----------------------+               |
|  | 服务发现客户端     |----->| 注册中心SDK           |               |
|  +------------------+      +----------------------+               |
+-------------------------------------------------------------------+
                |                      |
                v                      v
+-------------------------------------------------------------------+
|                        Nacos注册中心                                |
|  +------------------+      +----------------------+               |
|  | 服务列表          |<-----| 健康检查机制           |               |
|  +------------------+      +----------------------+               |
|  +------------------+      +----------------------+               |
|  | 配置管理          |      | 集群管理              |               |
|  +------------------+      +----------------------+               |
+-------------------------------------------------------------------+
                ^                      ^
                |                      |
+-------------------------------------------------------------------+
|                         服务提供方                                  |
|  +------------------+      +----------------------+               |
|  | 服务实例          |----->| 注册中心SDK           |               |
|  +------------------+      +----------------------+               |
+-------------------------------------------------------------------+
```

### 工作原理

1. **服务注册**: 各微服务启动时，通过Nacos SDK向Nacos服务端注册自身实例信息，包括IP、端口、服务名等。
2. **健康检查**: Nacos定期对已注册的服务实例进行健康检查，确保服务列表中的实例都是可用的。
3. **服务发现**: 当服务需要调用其他服务时，通过Nacos SDK查询目标服务的实例列表，而不是硬编码的服务地址。
4. **负载均衡**: 基于从Nacos获取的服务实例列表，客户端可以实现负载均衡策略，如轮询、随机、权重等。
5. **动态感知**: 当服务实例发生变化时（如新增、下线、健康状态变更），Nacos会通知订阅该服务的消费者，实现动态服务发现。

### 配置示例

#### Docker Compose配置

```yaml
# Nacos服务注册中心
nacos:
  image: nacos/nacos-server:v2.2.3
  ports:
    - "8848:8848"
    - "9848:9848"
    - "9849:9849"
  environment:
    - MODE=standalone
    - NACOS_AUTH_ENABLE=false
    - PREFER_HOST_MODE=hostname
    - NACOS_APPLICATION_PORT=8848
  volumes:
    - nacos_data:/home/nacos/data
  networks:
    - nfc_network
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8848/nacos/v1/console/health/status"]
    interval: 10s
    timeout: 5s
    retries: 3
    start_period: 30s
```

#### 服务配置示例

##### 服务配置文件 (merchant-service.yaml)
```yaml
# Nacos服务注册与发现配置
nacos:
  server_addr: "nacos:8848"              # Nacos服务地址
  namespace_id: "public"                 # 命名空间ID
  group: "DEFAULT_GROUP"                 # 分组
  service_name: "merchant-service"       # 服务名称
  enable: true                           # 是否启用服务发现
  weight: 10                             # 服务权重
  metadata:                              # 服务元数据
    version: "1.0.0"
    env: "dev"
  log_dir: "/tmp/nacos/log"              # Nacos日志目录
  cache_dir: "/tmp/nacos/cache"          # Nacos缓存目录
```

## 数据库 ERD

```mermaid
erDiagram
    MERCHANTS {
        uuid id PK
        string name
        string domain
        string logo_url
        boolean is_active
        string api_key
        timestamp created_at
        timestamp updated_at
        uuid plan_id FK
    }
    
    PLANS {
        uuid id PK
        string name
        float price
        boolean is_metered
        json features
        int max_videos
        int max_channels
        int max_storage_gb
        timestamp created_at
        timestamp updated_at
    }
    
    USERS {
        uuid id PK
        string email
        string password_hash
        string role
        uuid merchant_id FK
        boolean is_active
        timestamp created_at
        timestamp updated_at
    }
    
    NFC_CARDS {
        uuid id PK
        uuid merchant_id FK
        string uid
        string name
        string description
        uuid default_video_id FK
        timestamp activated_at
        timestamp created_at
        timestamp updated_at
    }
    
    VIDEOS {
        uuid id PK
        uuid merchant_id FK
        string title
        string description
        string status
        string storage_path
        json metadata
        int duration
        string cover_url
        boolean is_public
        timestamp created_at
        timestamp updated_at
    }
    
    PUBLISH_JOBS {
        uuid id PK
        uuid video_id FK
        uuid merchant_id FK
        uuid nfc_card_id FK
        string channel
        string status
        json result
        string error_message
        timestamp started_at
        timestamp completed_at
        timestamp created_at
        timestamp updated_at
    }
    
    CHANNEL_ACCOUNTS {
        uuid id PK
        uuid merchant_id FK
        string channel
        string name
        json credentials
        boolean is_active
        timestamp expires_at
        timestamp created_at
        timestamp updated_at
    }
    
    STATS {
        uuid id PK
        uuid video_id FK
        uuid merchant_id FK
        uuid nfc_card_id FK
        string channel
        int views
        int likes
        int shares
        int comments
        timestamp recorded_at
    }
    
    BILLING_RECORDS {
        uuid id PK
        uuid merchant_id FK
        float amount
        string currency
        string status
        string invoice_id
        string description
        timestamp billing_date
        timestamp created_at
        timestamp updated_at
    }
    
    SHORT_LINKS {
        uuid id PK
        uuid merchant_id FK
        uuid nfc_card_id FK
        string slug
        string target_url
        int clicks
        timestamp created_at
        timestamp updated_at
    }

    MERCHANTS ||--o{ USERS : "has"
    MERCHANTS ||--o{ NFC_CARDS : "owns"
    MERCHANTS ||--o{ VIDEOS : "owns"
    MERCHANTS ||--o{ CHANNEL_ACCOUNTS : "has"
    MERCHANTS ||--o{ BILLING_RECORDS : "has"
    MERCHANTS }|--|| PLANS : "subscribes to"
    
    NFC_CARDS ||--o{ PUBLISH_JOBS : "triggers"
    NFC_CARDS ||--o{ SHORT_LINKS : "has"
    NFC_CARDS }o--|| VIDEOS : "defaults to"
    
    VIDEOS ||--o{ PUBLISH_JOBS : "has"
    VIDEOS ||--o{ STATS : "has"
    
    PUBLISH_JOBS }|--|| CHANNEL_ACCOUNTS : "uses"
    CHANNEL_ACCOUNTS ||--o{ STATS : "records"
```

## 服务目录

### 共享基础库 (Shared)

```
shared/
├── auth/                        # 统一认证
│   ├── jwt.go                   # JWT工具
│   └── middleware.go            # 认证中间件
├── kafka/                       # Kafka消息队列
│   ├── client.go                # Kafka客户端
│   ├── producer.go              # 生产者
│   └── consumer.go              # 消费者
├── logger/                      # 日志工具
│   ├── init.go                  # 日志初始化
│   └── elk.go                   # ELK日志收集
├── middleware/                  # 共享中间件
│   ├── auth.go                  # 认证中间件
│   └── tenant.go                # 租户中间件
├── nacos/                       # Nacos服务注册发现
│   ├── client.go                # Nacos客户端
│   └── discovery.go             # 服务发现工具
└── service/                     # 服务调用
    └── client.go                # 服务HTTP客户端
```

### API 网关 (APISIX)

```
apisix/
├── config.yaml             # APISIX 配置文件
├── routes/                 # 路由配置
│   ├── admin.yaml          # 管理后台路由
│   ├── merchant.yaml       # 商户 API 路由
│   ├── public.yaml         # 公开 API 路由
│   └── nfc.yaml            # NFC 相关路由
└── plugins/                # 自定义插件
    ├── tenant-injector.lua # 租户 ID 注入插件
    ├── auth-validator.lua  # 认证验证插件
    └── rate-limiter.lua    # 限流插件
```

### 商户服务 (Go)

```
merchant-service/
├── cmd/
│   └── server/
│       └── main.go                   # 应用入口
├── internal/
│   ├── api/                          # API 处理
│   │   ├── handlers/
│   │   │   ├── merchants.go
│   │   │   ├── users.go
│   │   │   └── plans.go
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   └── tenant.go
│   │   └── router.go
│   ├── config/                       # 配置
│   │   └── config.go
│   ├── domain/                       # 领域模型
│   │   ├── entities/
│   │   │   ├── merchant.go
│   │   │   ├── user.go
│   │   │   └── plan.go
│   │   └── repositories/
│   │       ├── merchant_repository.go
│   │       ├── user_repository.go
│   │       └── plan_repository.go
│   ├── auth/                         # 认证
│   │   ├── jwt.go
│   │   └── middleware.go
│   └── services/                     # 业务服务
│       ├── merchant/
│       │   └── service.go
│       ├── user/
│       │   └── service.go
│       └── plan/
│           └── service.go
├── pkg/                              # 公共包
│   ├── kafka/
│   │   └── producer.go
│   ├── errors/
│   │   └── errors.go
│   └── database/
│       └── postgres.go
└── test/                             # 测试
```

### 内容服务 (Go)

```
content-service/
├── cmd/
│   └── server/
│       └── main.go                   # 应用入口
├── internal/
│   ├── api/                          # API 处理
│   │   ├── handlers/
│   │   │   └── videos.go
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   └── tenant.go
│   │   └── router.go
│   ├── config/                       # 配置
│   │   └── config.go
│   ├── domain/                       # 领域模型
│   │   ├── entities/
│   │   │   └── video.go
│   │   └── repositories/
│   │       └── video_repository.go
│   ├── storage/                      # 存储模块
│   │   ├── service.go
│   │   └── adapters/
│   │       ├── s3.go
│   │       └── minio.go
│   └── services/                     # 业务服务
│       ├── video/
│       │   └── service.go
│       └── transcoding/
│           └── service.go
├── pkg/                              # 公共包
│   ├── kafka/
│   │   └── consumer.go
│   └── ffmpeg/
│       └── processor.go
└── test/                             # 测试
```

### 分发服务 (Go)

```
distribution-service/
├── cmd/
│   └── server/
│       └── main.go                   # 应用入口
├── internal/
│   ├── api/                          # API 处理
│   │   ├── handlers/
│   │   │   ├── publish.go
│   │   │   └── jobs.go
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   └── tenant.go
│   │   └── router.go
│   ├── config/                       # 配置
│   │   └── config.go
│   ├── domain/                       # 领域模型
│   │   ├── entities/
│   │   │   ├── job.go
│   │   │   └── video.go
│   │   └── repositories/
│   │       ├── job_repository.go
│   │       └── video_repository.go
│   ├── adapters/                     # 平台适配器
│   │   ├── douyin/
│   │   │   ├── client.go
│   │   │   └── uploader.go
│   │   ├── kuaishou/
│   │   │   ├── client.go
│   │   │   └── uploader.go
│   │   ├── xiaohongshu/
│   │   │   ├── client.go
│   │   │   └── share.go
│   │   └── wechat/
│   │       ├── client.go
│   │       └── jssdk.go
│   └── services/                     # 业务服务
│       ├── publish/
│       │   └── service.go
│       └── orchestrator/
│           └── service.go
├── pkg/                              # 公共包
│   ├── kafka/
│   │   └── producer.go
│   ├── errors/
│   │   └── errors.go
│   └── auth/
│       └── jwt.go
└── test/                             # 测试
```

### NFC 服务 (Go)

```
nfc-service/
├── cmd/
│   └── server/
│       └── main.go                   # 应用入口
├── internal/
│   ├── api/                          # API 处理
│   │   ├── handlers/
│   │   │   ├── cards.go
│   │   │   └── shortlinks.go
│   │   ├── middleware/
│   │   │   └── auth.go
│   │   └── router.go
│   ├── config/                       # 配置
│   │   └── config.go
│   ├── domain/                       # 领域模型
│   │   ├── entities/
│   │   │   ├── card.go
│   │   │   └── shortlink.go
│   │   └── repositories/
│   │       ├── card_repository.go
│   │       └── shortlink_repository.go
│   └── services/                     # 业务服务
│       ├── cards/
│       │   └── service.go
│       └── shortlinks/
│           └── service.go
├── pkg/                              # 公共包
│   └── cloudflare/
│       └── workers.go
└── test/                             # 测试
```

### 统计服务 (Go)

```
stats-service/
├── cmd/
│   └── server/
│       └── main.go                   # 应用入口
├── internal/
│   ├── api/                          # API 处理
│   │   ├── handlers/
│   │   │   └── stats.go
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   └── tenant.go
│   │   └── router.go
│   ├── config/                       # 配置
│   │   └── config.go
│   ├── domain/                       # 领域模型
│   │   ├── entities/
│   │   │   └── stat.go
│   │   └── repositories/
│   │       └── stat_repository.go
│   ├── adapters/                     # 平台适配器
│   │   ├── douyin/
│   │   │   └── collector.go
│   │   ├── kuaishou/
│   │   │   └── collector.go
│   │   └── xiaohongshu/
│   │       └── collector.go
│   └── services/                     # 业务服务
│       └── stats/
│           └── service.go
├── pkg/                              # 公共包
│   ├── kafka/
│   │   └── consumer.go
│   └── timeseries/
│       └── influxdb.go
└── test/                             # 测试
```

### 前端项目 (Next.js)

```
frontend/
├── public/                           # 静态资源
├── src/
│   ├── app/                          # Next.js App Router
│   │   ├── layout.tsx
│   │   ├── page.tsx
│   │   ├── admin/                    # 平台管理后台
│   │   │   └── [...]
│   │   ├── merchant/                 # 商户后台
│   │   │   └── [...]
│   │   └── landing/                  # NFC 落地页
│   │       └── [shortLink]/page.tsx
│   ├── components/                   # 组件
│   │   ├── common/
│   │   ├── admin/
│   │   ├── merchant/
│   │   └── landing/
│   ├── hooks/                        # 自定义钩子
│   ├── services/                     # API 服务
│   │   ├── merchant.ts
│   │   ├── video.ts
│   │   ├── nfc.ts
│   │   └── publish.ts
│   ├── store/                        # 状态管理
│   ├── styles/                       # 样式
│   ├── types/                        # 类型定义
│   └── utils/                        # 工具函数
└── next.config.js                    # Next.js 配置
```

## API 设计

### 商户服务 API (/api/v1/merchants)

```yaml
openapi: 3.0.0
info:
  title: 商户服务 API
  version: 1.0.0
  description: NFC 碰一碰分发系统商户服务 API

paths:
  /api/v1/merchants:
    get:
      summary: 获取商户列表
      tags:
        - Merchants
      parameters:
        - name: page
          in: query
          schema:
            type: integer
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        '200':
          description: 商户列表
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/Merchant'
                  meta:
                    $ref: '#/components/schemas/PaginationMeta'
      security:
        - BearerAuth: []
      x-roles:
        - admin
    
    post:
      summary: 创建商户
      tags:
        - Merchants
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateMerchantDto'
      responses:
        '201':
          description: 商户创建成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Merchant'
      security:
        - BearerAuth: []
      x-roles:
        - admin
  
  /api/v1/merchants/{id}:
    get:
      summary: 获取商户详情
      tags:
        - Merchants
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: 商户详情
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Merchant'
      security:
        - BearerAuth: []
      x-roles:
        - admin
        - merchant
    
    put:
      summary: 更新商户
      tags:
        - Merchants
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateMerchantDto'
      responses:
        '200':
          description: 商户更新成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Merchant'
      security:
        - BearerAuth: []
      x-roles:
        - admin
        - merchant
    
    delete:
      summary: 删除商户
      tags:
        - Merchants
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '204':
          description: 商户删除成功
      security:
        - BearerAuth: []
      x-roles:
        - admin
  
  /api/v1/merchants/{id}/api-key:
    post:
      summary: 重新生成 API Key
      tags:
        - Merchants
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: API Key 重新生成成功
          content:
            application/json:
              schema:
                type: object
                properties:
                  apiKey:
                    type: string
      security:
        - BearerAuth: []
      x-roles:
        - admin
        - merchant

components:
  schemas:
    Merchant:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        domain:
          type: string
        logoUrl:
          type: string
        isActive:
          type: boolean
        apiKey:
          type: string
        planId:
          type: string
          format: uuid
        plan:
          $ref: '#/components/schemas/Plan'
        createdAt:
          type: string
          format: date-time
        updatedAt:
          type: string
          format: date-time
    
    CreateMerchantDto:
      type: object
      required:
        - name
        - planId
      properties:
        name:
          type: string
        domain:
          type: string
        logoUrl:
          type: string
        isActive:
          type: boolean
          default: true
        planId:
          type: string
          format: uuid
    
    UpdateMerchantDto:
      type: object
      properties:
        name:
          type: string
        domain:
          type: string
        logoUrl:
          type: string
        isActive:
          type: boolean
        planId:
          type: string
          format: uuid
    
    Plan:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        price:
          type: number
        isMetered:
          type: boolean
        features:
          type: object
        maxVideos:
          type: integer
        maxChannels:
          type: integer
        maxStorageGb:
          type: integer
        createdAt:
          type: string
          format: date-time
        updatedAt:
          type: string
          format: date-time
    
    PaginationMeta:
      type: object
      properties:
        currentPage:
          type: integer
        itemsPerPage:
          type: integer
        totalItems:
          type: integer
        totalPages:
          type: integer
  
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

### 视频服务 API (/api/v1/videos)

```yaml
openapi: 3.0.0
info:
  title: 视频服务 API
  version: 1.0.0
  description: NFC 碰一碰分发系统视频服务 API

paths:
  /api/v1/videos:
    get:
      summary: 获取视频列表
      tags:
        - Videos
      parameters:
        - name: page
          in: query
          schema:
            type: integer
        - name: limit
          in: query
          schema:
            type: integer
        - name: status
          in: query
          schema:
            type: string
            enum: [draft, processing, ready, failed]
      responses:
        '200':
          description: 视频列表
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/Video'
                  meta:
                    $ref: '#/components/schemas/PaginationMeta'
      security:
        - BearerAuth: []
      x-tenant: true
    
    post:
      summary: 创建视频
      tags:
        - Videos
      requestBody:
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                title:
                  type: string
                description:
                  type: string
                file:
                  type: string
                  format: binary
                isPublic:
                  type: boolean
      responses:
        '201':
          description: 视频创建成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Video'
      security:
        - BearerAuth: []
      x-tenant: true
  
  /api/v1/videos/{id}:
    get:
      summary: 获取视频详情
      tags:
        - Videos
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: 视频详情
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Video'
      security:
        - BearerAuth: []
      x-tenant: true
    
    put:
      summary: 更新视频
      tags:
        - Videos
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateVideoDto'
      responses:
        '200':
          description: 视频更新成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Video'
      security:
        - BearerAuth: []
      x-tenant: true
    
    delete:
      summary: 删除视频
      tags:
        - Videos
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '204':
          description: 视频删除成功
      security:
        - BearerAuth: []
      x-tenant: true
  
  /api/v1/videos/{id}/upload-url:
    get:
      summary: 获取分片上传 URL
      tags:
        - Videos
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
        - name: partNumber
          in: query
          required: true
          schema:
            type: integer
        - name: uploadId
          in: query
          required: true
          schema:
            type: string
      responses:
        '200':
          description: 上传 URL
          content:
            application/json:
              schema:
                type: object
                properties:
                  url:
                    type: string
      security:
        - BearerAuth: []
      x-tenant: true
  
  /api/v1/videos/{id}/complete-upload:
    post:
      summary: 完成分片上传
      tags:
        - Videos
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                uploadId:
                  type: string
                parts:
                  type: array
                  items:
                    type: object
                    properties:
                      ETag:
                        type: string
                      PartNumber:
                        type: integer
      responses:
        '200':
          description: 上传完成
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Video'
      security:
        - BearerAuth: []
      x-tenant: true

components:
  schemas:
    Video:
      type: object
      properties:
        id:
          type: string
          format: uuid
        merchantId:
          type: string
          format: uuid
        title:
          type: string
        description:
          type: string
        status:
          type: string
          enum: [draft, processing, ready, failed]
        storagePath:
          type: string
        metadata:
          type: object
          properties:
            width:
              type: integer
            height:
              type: integer
            format:
              type: string
            bitrate:
              type: integer
        duration:
          type: integer
        coverUrl:
          type: string
        isPublic:
          type: boolean
        createdAt:
          type: string
          format: date-time
        updatedAt:
          type: string
          format: date-time
    
    UpdateVideoDto:
      type: object
      properties:
        title:
          type: string
        description:
          type: string
        isPublic:
          type: boolean
    
    PaginationMeta:
      type: object
      properties:
        currentPage:
          type: integer
        itemsPerPage:
          type: integer
        totalItems:
          type: integer
        totalPages:
          type: integer
  
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

### 分发服务 API (/api/v1/publish/jobs)

```yaml
openapi: 3.0.0
info:
  title: 分发服务 API
  version: 1.0.0
  description: NFC 碰一碰分发系统分发服务 API

paths:
  /api/v1/publish/jobs:
    get:
      summary: 获取分发任务列表
      tags:
        - PublishJobs
      parameters:
        - name: page
          in: query
          schema:
            type: integer
        - name: limit
          in: query
          schema:
            type: integer
        - name: status
          in: query
          schema:
            type: string
            enum: [pending, processing, completed, failed]
        - name: videoId
          in: query
          schema:
            type: string
            format: uuid
        - name: nfcCardId
          in: query
          schema:
            type: string
            format: uuid
        - name: channel
          in: query
          schema:
            type: string
            enum: [douyin, kuaishou, xiaohongshu, wechat]
      responses:
        '200':
          description: 分发任务列表
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/PublishJob'
                  meta:
                    $ref: '#/components/schemas/PaginationMeta'
      security:
        - BearerAuth: []
      x-tenant: true
    
    post:
      summary: 创建分发任务
      tags:
        - PublishJobs
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreatePublishJobDto'
      responses:
        '201':
          description: 分发任务创建成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/PublishJob'
      security:
        - BearerAuth: []
      x-tenant: true
  
  /api/v1/publish/jobs/{id}:
    get:
      summary: 获取分发任务详情
      tags:
        - PublishJobs
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: 分发任务详情
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/PublishJob'
      security:
        - BearerAuth: []
      x-tenant: true
  
  /api/v1/publish/jobs/{id}/retry:
    post:
      summary: 重试分发任务
      tags:
        - PublishJobs
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: 分发任务重试成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/PublishJob'
      security:
        - BearerAuth: []
      x-tenant: true
  
  /api/v1/publish/jobs/{id}/cancel:
    post:
      summary: 取消分发任务
      tags:
        - PublishJobs
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: 分发任务取消成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/PublishJob'
      security:
        - BearerAuth: []
      x-tenant: true
  
  /api/v1/publish/batch:
    post:
      summary: 批量创建分发任务
      tags:
        - PublishJobs
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                videoId:
                  type: string
                  format: uuid
                nfcCardId:
                  type: string
                  format: uuid
                channels:
                  type: array
                  items:
                    type: string
                    enum: [douyin, kuaishou, xiaohongshu, wechat]
      responses:
        '201':
          description: 批量分发任务创建成功
          content:
            application/json:
              schema:
                type: object
                properties:
                  jobs:
                    type: array
                    items:
                      $ref: '#/components/schemas/PublishJob'
      security:
        - BearerAuth: []
      x-tenant: true

components:
  schemas:
    PublishJob:
      type: object
      properties:
        id:
          type: string
          format: uuid
        videoId:
          type: string
          format: uuid
        merchantId:
          type: string
          format: uuid
        nfcCardId:
          type: string
          format: uuid
        channel:
          type: string
          enum: [douyin, kuaishou, xiaohongshu, wechat]
        status:
          type: string
          enum: [pending, processing, completed, failed]
        result:
          type: object
          properties:
            platformId:
              type: string
            url:
              type: string
            shareKey:
              type: string
        errorMessage:
          type: string
        startedAt:
          type: string
          format: date-time
        completedAt:
          type: string
          format: date-time
        createdAt:
          type: string
          format: date-time
        updatedAt:
          type: string
          format: date-time
        video:
          $ref: '#/components/schemas/VideoSummary'
        nfcCard:
          $ref: '#/components/schemas/NfcCardSummary'
    
    CreatePublishJobDto:
      type: object
      required:
        - videoId
        - channel
      properties:
        videoId:
          type: string
          format: uuid
        nfcCardId:
          type: string
          format: uuid
        channel:
          type: string
          enum: [douyin, kuaishou, xiaohongshu, wechat]
    
    VideoSummary:
      type: object
      properties:
        id:
          type: string
          format: uuid
        title:
          type: string
        coverUrl:
          type: string
        duration:
          type: integer
        status:
          type: string
    
    NfcCardSummary:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        uid:
          type: string
    
    PaginationMeta:
      type: object
      properties:
        currentPage:
          type: integer
        itemsPerPage:
          type: integer
        totalItems:
          type: integer
        totalPages:
          type: integer
  
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

## 核心代码示例

### Nacos服务注册与发现实现

#### Nacos客户端 (client.go)

```go
// shared/nacos/client.go

package nacos

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// Config Nacos配置
type Config struct {
	ServerAddr  string `mapstructure:"server_addr"`  // Nacos服务地址，如localhost:8848
	NamespaceID string `mapstructure:"namespace_id"` // 命名空间ID，默认为public
	Group       string `mapstructure:"group"`        // 分组，默认为DEFAULT_GROUP
	DataID      string `mapstructure:"data_id"`      // 配置ID
	Username    string `mapstructure:"username"`     // 用户名
	Password    string `mapstructure:"password"`     // 密码
	LogDir      string `mapstructure:"log_dir"`      // 日志目录
	CacheDir    string `mapstructure:"cache_dir"`    // 缓存目录
}

// Client Nacos客户端
type Client struct {
	config       *Config
	namingClient naming_client.INamingClient
}

// NewClient 创建Nacos客户端
func NewClient(config *Config) (*Client, error) {
	// 设置默认值
	if config.NamespaceID == "" {
		config.NamespaceID = "public"
	}
	if config.Group == "" {
		config.Group = "DEFAULT_GROUP"
	}
	if config.LogDir == "" {
		config.LogDir = "/tmp/nacos/log"
	}
	if config.CacheDir == "" {
		config.CacheDir = "/tmp/nacos/cache"
	}

	// 解析服务器地址
	serverAddrs := strings.Split(config.ServerAddr, ",")
	serverConfigs := make([]constant.ServerConfig, 0, len(serverAddrs))
	
	for _, addr := range serverAddrs {
		parts := strings.Split(addr, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("无效的服务器地址格式: %s", addr)
		}
		
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("无效的端口号: %s", parts[1])
		}
		
		serverConfigs = append(serverConfigs, constant.ServerConfig{
			IpAddr: parts[0],
			Port:   uint64(port),
		})
	}

	// 创建客户端配置
	clientConfig := constant.ClientConfig{
		NamespaceId:         config.NamespaceID,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              config.LogDir,
		CacheDir:            config.CacheDir,
		Username:            config.Username,
		Password:            config.Password,
		LogLevel:            "info",
	}

	// 创建命名服务客户端
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("创建Nacos命名服务客户端失败: %w", err)
	}

	return &Client{
		config:       config,
		namingClient: namingClient,
	}, nil
}

// RegisterService 注册服务实例
func (c *Client) RegisterService(serviceName, ip string, port int, metadata map[string]string) (bool, error) {
	// 如果未指定IP，则尝试获取本机IP
	if ip == "" {
		localIP, err := c.getLocalIP()
		if err != nil {
			return false, fmt.Errorf("无法获取本机IP: %w", err)
		}
		ip = localIP
	}

	// 注册服务实例
	success, err := c.namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        uint64(port),
		ServiceName: serviceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    metadata,
		GroupName:   c.config.Group,
	})

	if err != nil {
		return false, fmt.Errorf("注册服务实例失败: %w", err)
	}

	return success, nil
}

// GetRandomServiceInstance 随机获取一个服务实例
func (c *Client) GetRandomServiceInstance(serviceName string) (*model.Instance, error) {
	instance, err := c.namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		GroupName:   c.config.Group,
	})

	if err != nil {
		return nil, fmt.Errorf("获取服务实例失败: %w", err)
	}

	return &instance, nil
}

// StartHealthCheck 开始健康检查
func (c *Client) StartHealthCheck(serviceName, ip string, port int, checkInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		for range ticker.C {
			_, err := c.namingClient.UpdateInstance(vo.UpdateInstanceParam{
				Ip:          ip,
				Port:        uint64(port),
				ServiceName: serviceName,
				Weight:      10,
				Enable:      true,
				Healthy:     true,
				Ephemeral:   true,
				GroupName:   c.config.Group,
			})

			if err != nil {
				log.Printf("更新服务实例状态失败: %v", err)
			}
		}
	}()
}

// getLocalIP 获取本机IP
func (c *Client) getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("无法获取本机IP地址")
}
```

#### 服务发现客户端 (discovery.go)

```go
// shared/nacos/discovery.go

package nacos

import (
	"fmt"
	"log"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

// ServiceDiscovery 服务发现客户端
type ServiceDiscovery struct {
	client              *Client
	cachedInstances     map[string][]model.Instance
	cachedInstancesLock sync.RWMutex
	subscriptions       map[string]bool
	subscriptionsLock   sync.RWMutex
}

// NewServiceDiscovery 创建服务发现客户端
func NewServiceDiscovery(client *Client) *ServiceDiscovery {
	return &ServiceDiscovery{
		client:          client,
		cachedInstances: make(map[string][]model.Instance),
		subscriptions:   make(map[string]bool),
	}
}

// GetServiceURL 获取服务URL
func (sd *ServiceDiscovery) GetServiceURL(serviceName string) (string, error) {
	instance, err := sd.client.GetRandomServiceInstance(serviceName)
	if err != nil {
		return "", err
	}

	// 构建URL
	schema := "http"
	if _, ok := instance.Metadata["secure"]; ok {
		schema = "https"
	}

	return fmt.Sprintf("%s://%s:%d", schema, instance.Ip, instance.Port), nil
}

// GetServiceInstance 获取服务实例
func (sd *ServiceDiscovery) GetServiceInstance(serviceName string) (*model.Instance, error) {
	return sd.client.GetRandomServiceInstance(serviceName)
}

// SubscribeService 订阅服务变更
func (sd *ServiceDiscovery) SubscribeService(serviceName string) error {
	sd.subscriptionsLock.Lock()
	defer sd.subscriptionsLock.Unlock()

	// 如果已经订阅，则直接返回
	if _, ok := sd.subscriptions[serviceName]; ok {
		return nil
	}

	// 订阅服务变更
	err := sd.client.Subscribe(serviceName, func(instances []model.Instance) {
		// 更新缓存
		sd.cachedInstancesLock.Lock()
		sd.cachedInstances[serviceName] = instances
		sd.cachedInstancesLock.Unlock()

		log.Printf("服务[%s]实例列表已更新，共%d个实例", serviceName, len(instances))
	})

	if err != nil {
		return err
	}

	sd.subscriptions[serviceName] = true
	return nil
}
```

#### 服务注册和发现使用示例 (main.go)

```go
// merchant-service/cmd/server/main.go

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nfc_card/shared/nacos"
	
	"merchant-service/internal/api"
	"merchant-service/internal/config"
)

func main() {
	fmt.Println("服务启动中...")

	// 初始化日志
	logger := log.New(os.Stdout, "[SERVICE] ", log.LstdFlags)

	// 加载配置
	cfg, err := config.LoadConfig("./config.yaml")
	if err != nil {
		logger.Fatalf("加载配置失败: %v", err)
	}

	// 获取服务端口
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = cfg.Server.Port
	}

	// 初始化API路由
	router := api.NewRouter(cfg)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: router,
	}

	// 初始化并注册Nacos服务
	var nacosClient *nacos.Client
	if cfg.Nacos.Enable {
		nacosConfig := &nacos.Config{
			ServerAddr:  cfg.Nacos.ServerAddr,
			NamespaceID: cfg.Nacos.NamespaceID,
			Group:       cfg.Nacos.Group,
			LogDir:      cfg.Nacos.LogDir,
			CacheDir:    cfg.Nacos.CacheDir,
		}

		nacosClient, err = nacos.NewClient(nacosConfig)
		if err != nil {
			logger.Printf("初始化Nacos客户端失败: %v", err)
		} else {
			// 获取本机IP并注册服务
			port, _ := strconv.Atoi(serverPort)
			success, err := nacosClient.RegisterService(
				cfg.Nacos.ServiceName,
				"", // 空字符串表示自动获取本机IP
				port,
				cfg.Nacos.Metadata,
			)
			if err != nil {
				logger.Printf("注册服务到Nacos失败: %v", err)
			} else if success {
				logger.Printf("已成功注册到Nacos，服务名: %s, 端口: %d", cfg.Nacos.ServiceName, port)
				
				// 启动健康检查
				nacosClient.StartHealthCheck(cfg.Nacos.ServiceName, "", port, 5*time.Second)
			}
		}
	}

	// 在goroutine中启动服务器，以便不阻塞信号处理
	go func() {
		logger.Printf("服务已启动，端口: %s", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("监听错误: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("正在关闭服务...")

	// 从Nacos注销服务
	if cfg.Nacos.Enable && nacosClient != nil {
		port, _ := strconv.Atoi(serverPort)
		_, err := nacosClient.DeregisterService(cfg.Nacos.ServiceName, "", port)
		if err != nil {
			logger.Printf("从Nacos注销服务失败: %v", err)
		} else {
			logger.Println("已从Nacos注销服务")
		}
	}

	// 优雅关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("服务器关闭错误: %v", err)
	}

	logger.Println("服务已关闭")
}
```

#### 服务调用客户端 (client.go)

```go
// shared/service/client.go

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/nfc_card/shared/nacos"
)

// 服务名称常量
const (
	MerchantService     = "merchant-service"
	NFCService          = "nfc-service"
	ContentService      = "content-service"
	StatsService        = "stats-service"
	DistributionService = "distribution-service"
)

// Client 服务客户端
type Client struct {
	discovery    *nacos.ServiceDiscovery
	httpClient   *http.Client
	serviceURLs  map[string]string
	mutex        sync.RWMutex
	enableNacos  bool
	defaultPorts map[string]int
	logger       *log.Logger
}

// NewClient 创建服务客户端
func NewClient(nacosClient *nacos.Client, logger *log.Logger) *Client {
	client := &Client{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		serviceURLs: make(map[string]string),
		enableNacos: nacosClient != nil,
		defaultPorts: map[string]int{
			MerchantService:     8082,
			NFCService:          8083,
			ContentService:      8081,
			StatsService:        8084,
			DistributionService: 8085,
		},
		logger: logger,
	}

	// 如果启用了Nacos，则创建服务发现客户端
	if client.enableNacos {
		client.discovery = nacos.NewServiceDiscovery(nacosClient)
	}

	return client
}

// GetServiceURL 获取服务URL
func (c *Client) GetServiceURL(serviceName string) string {
	// 尝试从缓存获取URL
	c.mutex.RLock()
	url, ok := c.serviceURLs[serviceName]
	c.mutex.RUnlock()

	if ok {
		return url
	}

	// 如果启用了Nacos，则从Nacos获取服务实例
	if c.enableNacos {
		instance, err := c.discovery.GetServiceInstance(serviceName)
		if err == nil {
			// 构建URL
			url = fmt.Sprintf("http://%s:%d", instance.Ip, instance.Port)
			
			// 缓存URL
			c.mutex.Lock()
			c.serviceURLs[serviceName] = url
			c.mutex.Unlock()
			
			return url
		}
		
		c.logger.Printf("从Nacos获取服务[%s]地址失败: %v，将使用默认地址", serviceName, err)
	}

	// 如果未启用Nacos或从Nacos获取失败，则使用默认地址
	port, ok := c.defaultPorts[serviceName]
	if !ok {
		c.logger.Printf("未知的服务名称: %s", serviceName)
		port = 8080
	}

	// 使用服务名作为主机名（适用于Docker环境）
	url = fmt.Sprintf("http://%s:%d", serviceName, port)
	
	// 缓存URL
	c.mutex.Lock()
	c.serviceURLs[serviceName] = url
	c.mutex.Unlock()

	return url
}

// GetJSON 发送GET请求并解析JSON响应
func (c *Client) GetJSON(serviceName, path string, result interface{}) error {
	serviceURL := c.GetServiceURL(serviceName)
	url := fmt.Sprintf("%s%s", serviceURL, path)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务[%s]返回非200状态码: %d", serviceName, resp.StatusCode)
	}

	// 解析JSON
	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	return nil
}

// PostJSON 发送POST请求并解析JSON响应
func (c *Client) PostJSON(serviceName, path string, data, result interface{}) error {
	// 序列化请求数据
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// 发送请求
	serviceURL := c.GetServiceURL(serviceName)
	url := fmt.Sprintf("%s%s", serviceURL, path)
	
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("服务[%s]返回非成功状态码: %d", serviceName, resp.StatusCode)
	}

	// 解析响应
	if result != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		
		if err := json.Unmarshal(body, result); err != nil {
			return err
		}
	}

	return nil
}

// SubscribeService 订阅服务变更
func (c *Client) SubscribeService(serviceName string) error {
	if !c.enableNacos {
		return nil
	}

	// 订阅服务变更
	return c.discovery.SubscribeService(serviceName)
}
```

### Go中间件实现 RLS （自动注入 tenant_id）

Go中间件用于提取租户ID并自动应用 PostgreSQL 的行级安全（Row-Level Security）：

```go
// merchant-service/internal/middleware/tenant.go

package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/nfc_card/pkg/errors"
)

// TenantMiddleware 租户中间件
type TenantMiddleware struct {
	db        *pgxpool.Pool
	jwtSecret string
}

// NewTenantMiddleware 创建新的租户中间件
func NewTenantMiddleware(db *pgxpool.Pool, jwtSecret string) *TenantMiddleware {
	return &TenantMiddleware{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

// Middleware 租户中间件处理函数
func (m *TenantMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从Authorization头中提取JWT
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// 验证JWT
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(m.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 从JWT中提取商户ID
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		merchantID, ok := claims["merchant_id"].(string)
		if !ok || merchantID == "" {
			next.ServeHTTP(w, r)
			return
		}

		// 将merchantID存储在请求上下文中
		ctx := context.WithValue(r.Context(), "merchant_id", merchantID)
		r = r.WithContext(ctx)

		// 在数据库连接上设置RLS变量
		conn, err := m.db.Acquire(r.Context())
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer conn.Release()

		// 设置当前请求的租户ID
		_, err = conn.Exec(r.Context(), `SET LOCAL "app.current_tenant" = $1`, merchantID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// 调用下一个处理器
		next.ServeHTTP(w, r)
	})
}

// GetTenantID 从上下文中获取租户ID
func GetTenantID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value("merchant_id").(string)
	return id, ok
}
```

配置RLS策略的数据库迁移文件：

```sql
-- migrations/001_setup_rls.sql

-- 创建schema
CREATE SCHEMA IF NOT EXISTS auth;

-- 创建应用RLS所需的函数
CREATE OR REPLACE FUNCTION auth.current_tenant_id()
RETURNS UUID AS $$
BEGIN
  RETURN current_setting('app.current_tenant', TRUE)::UUID;
EXCEPTION
  WHEN OTHERS THEN
    RETURN NULL;
END
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 为所有表添加tenant_id列并设置RLS策略的函数
CREATE OR REPLACE FUNCTION auth.create_tenant_schema_for_table(
  table_name text,
  schema_name text DEFAULT 'public'
)
RETURNS void AS $$
BEGIN
  -- 为表添加tenant_id列（如果不存在）
  EXECUTE format(
    'ALTER TABLE %I.%I ADD COLUMN IF NOT EXISTS merchant_id UUID NOT NULL',
    schema_name,
    table_name
  );
  
  -- 启用RLS
  EXECUTE format(
    'ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY',
    schema_name,
    table_name
  );
  
  -- 创建RLS策略
  EXECUTE format(
    'DROP POLICY IF EXISTS tenant_isolation_policy ON %I.%I',
    schema_name,
    table_name
  );
  
  EXECUTE format(
    'CREATE POLICY tenant_isolation_policy ON %I.%I
     USING (merchant_id = auth.current_tenant_id())
     WITH CHECK (merchant_id = auth.current_tenant_id())',
    schema_name,
    table_name
  );
END;
$$ LANGUAGE plpgsql;

-- 对需要RLS的表应用策略
SELECT auth.create_tenant_schema_for_table('videos');
SELECT auth.create_tenant_schema_for_table('nfc_cards');
SELECT auth.create_tenant_schema_for_table('publish_jobs');
SELECT auth.create_tenant_schema_for_table('channel_accounts');
SELECT auth.create_tenant_schema_for_table('stats');
SELECT auth.create_tenant_schema_for_table('short_links');
```

Go实现的数据库连接管理器：

```go
// merchant-service/pkg/database/postgres.go

package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/nfc_card/pkg/config"
)

// PostgresDB 是PostgreSQL数据库连接管理器
type PostgresDB struct {
	pool *pgxpool.Pool
}

// NewPostgresDB 创建一个新的数据库连接池
func NewPostgresDB(cfg *config.Config) (*PostgresDB, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("无法解析数据库连接字符串: %w", err)
	}

	// 设置连接池大小
	config.MaxConns = 20
	config.MinConns = 5

	// 创建连接池
	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("无法连接到数据库: %w", err)
	}

	return &PostgresDB{
		pool: pool,
	}, nil
}

// Pool 返回底层的连接池
func (db *PostgresDB) Pool() *pgxpool.Pool {
	return db.pool
}

// Close 关闭数据库连接
func (db *PostgresDB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// EnableRLS 在连接上启用行级安全性
func (db *PostgresDB) EnableRLS(ctx context.Context) (*pgxpool.Conn, error) {
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("无法获取数据库连接: %w", err)
	}

	_, err = conn.Exec(ctx, `SET LOCAL "app.enable_row_level_security" = true`)
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("无法设置RLS启用变量: %w", err)
	}

	return conn, nil
}

// GetTenantContext 返回一个设置了租户ID的上下文
func GetTenantContext(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, "tenant_id", tenantID)
}
``` 