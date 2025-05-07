# 统一日志系统

本文档描述了NFC卡片项目的统一日志系统，包括日志级别、格式和集中式日志收集机制。

## 1. 日志系统概述

我们实现了一个基于以下特性的统一日志系统：

1. 统一的日志格式和级别
2. 集中式日志收集（使用ELK - Elasticsearch, Logstash, Kibana）
3. 请求追踪功能，使用Trace ID在不同服务间跟踪请求

## 2. 日志级别

系统支持以下日志级别（从低到高）：

- `debug`: 调试信息，仅在开发环境使用
- `info`: 一般信息，记录系统正常操作
- `warn`: 警告信息，表示可能的问题但不影响系统运行
- `error`: 错误信息，表示发生了错误但系统能够继续运行
- `fatal`: 致命错误，会导致程序退出

## 3. 日志格式

所有日志都以JSON格式输出，包含以下标准字段：

```json
{
  "timestamp": "2023-05-01T12:34:56Z",  // ISO8601格式的时间戳
  "level": "info",                      // 日志级别
  "message": "请求处理成功",             // 日志消息
  "service": "nfc-service",             // 服务名称
  "trace_id": "550e8400-e29b-41d4-a716-446655440000", // 追踪ID
  "file": "handlers/card.go",           // 源代码文件
  "function": "GetCard",                // 函数名
  "line": 42                            // 行号
}
```

## 4. 如何使用

### 4.1 在Go服务中使用

1. 首先导入日志包：

```go
import (
    "github.com/nfc_card/shared/logger"
    "context"
)
```

2. 在服务启动时初始化日志系统：

```go
func main() {
    // 初始化日志系统
    log, err := logger.InitLogger("nfc-service", "config/logging.yaml")
    if err != nil {
        panic("初始化日志系统失败: " + err.Error())
    }
    
    // 使用日志
    log.Info("服务启动成功")
}
```

3. 使用带有上下文的日志方法传递追踪ID：

```go
func HandleRequest(ctx context.Context, req Request) {
    // 获取或生成追踪ID
    traceID := logger.GetTraceID(ctx)
    if traceID == "" {
        // 如果没有追踪ID，生成一个新的
        traceID = logger.GenerateTraceID()
        ctx = logger.WithTraceID(ctx, traceID)
    }
    
    // 记录带有追踪ID的日志
    logger.DefaultLogger.InfoContext(ctx, "处理请求: %s", req.ID)
    
    // 处理请求...
    
    // 记录错误
    if err := processRequest(req); err != nil {
        logger.DefaultLogger.ErrorContext(ctx, "处理请求失败: %v", err)
    }
}
```

### 4.2 在HTTP中间件中使用

在HTTP服务中，可以使用中间件自动为每个请求添加追踪ID：

```go
import (
    "github.com/nfc_card/shared/middleware"
    "net/http"
)

func main() {
    // 创建HTTP处理器
    handler := http.NewServeMux()
    
    // 注册路由
    handler.HandleFunc("/api/cards", handleCards)
    
    // 应用追踪中间件
    tracedHandler := middleware.TraceMiddleware(handler)
    
    // 启动HTTP服务器
    http.ListenAndServe(":8080", tracedHandler)
}
```

### 4.3 在Kafka消息中使用

在发送Kafka消息时，可以传递上下文以包含追踪ID：

```go
func handleRequest(ctx context.Context, req Request) {
    // 处理请求...
    
    // 发送带有追踪ID的Kafka消息
    kafkaClient.SendMessageWithContext(ctx, "topic", "message.type", data)
}
```

## 5. 集中式日志收集

项目使用ELK（Elasticsearch, Logstash, Kibana）进行集中式日志收集和分析。

### 5.1 启动ELK

可以使用Docker Compose启动ELK服务：

```bash
docker-compose -f docker-compose-elk.yaml up -d
```

### 5.2 访问Kibana

启动后，可以通过以下地址访问Kibana：

```
http://localhost:5601
```

### 5.3 配置索引模式

首次使用时，需要在Kibana中配置索引模式：

1. 进入Kibana，点击左侧菜单的"Stack Management"
2. 点击"Index Patterns"
3. 点击"Create index pattern"
4. 输入索引模式，例如`logs-*`
5. 选择时间字段为`@timestamp`
6. 点击"Create index pattern"完成配置

### 5.4 查看和分析日志

配置完成后，可以在Kibana的"Discover"页面查看和分析日志。可以使用以下查询示例：

- 查看特定服务的日志：`service: "nfc-service"`
- 查看特定级别的日志：`level: "error"`
- 查看特定追踪ID的日志：`trace_id: "550e8400-e29b-41d4-a716-446655440000"`

## 6. 环境变量配置

日志系统可以通过以下环境变量进行配置：

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