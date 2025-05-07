# Kafka消息规范

## 1. 概述

本文档定义了NFC卡系统中使用Kafka进行微服务间通信的标准和规范。遵循这些规范可以确保系统各组件之间的消息交换稳定、可靠和一致。

## 2. 消息格式

### 2.1 基础消息结构

所有通过Kafka传输的消息必须遵循以下JSON格式：

```json
{
  "type": "消息类型标识符",
  "data": {
    // 消息负载，根据消息类型不同而不同
  },
  "timestamp": "2023-07-12T10:15:30Z", // ISO 8601格式
  "source": "发送服务标识符",
  "traceId": "唯一跟踪ID"
}
```

字段说明：
- `type`: 消息类型，使用点分隔的字符串，例如 `merchant.created`
- `data`: 消息的具体内容，根据业务需求定义
- `timestamp`: 消息创建时间，使用ISO 8601格式
- `source`: 发送消息的服务名称，例如 `merchant-service`
- `traceId`: 可选，用于跟踪消息流转的唯一标识符

### 2.2 错误处理

处理消息时出现错误，应记录详细日志，并根据业务需求决定是否重试或丢弃。对于关键业务消息，建议实现重试机制。

## 3. 主题命名规则

主题命名应遵循以下格式：`{业务域}-{事件类型}-events`

标准主题包括：
- `merchant-events`: 商户相关事件
- `card-events`: NFC卡相关事件
- `video-events`: 视频内容相关事件
- `publish-events`: 内容发布相关事件
- `stats-events`: 统计数据相关事件

## 4. 消息类型

### 4.1 商户服务消息类型
- `merchant.created`: 商户创建
- `merchant.updated`: 商户信息更新

### 4.2 NFC卡服务消息类型
- `card.created`: NFC卡创建
- `card.updated`: NFC卡更新
- `card.bound`: NFC卡绑定
- `card.unbound`: NFC卡解绑

### 4.3 内容服务消息类型
- `video.created`: 视频创建
- `video.updated`: 视频更新
- `video.deleted`: 视频删除

### 4.4 分发服务消息类型
- `publish_job.created`: 发布任务创建
- `publish_job.updated`: 发布任务更新
- `publish_job.completed`: 发布任务完成

## 5. 消费者组设计

每个服务应使用唯一的消费者组ID，格式为：`{服务名}-group`

标准消费者组包括：
- `merchant-service-group`
- `nfc-service-group`
- `content-service-group`
- `distribution-service-group`
- `stats-service-group`

## 6. 服务间消息流

```
商户服务 ─────────┐
                  ↓
                merchant-events
                  ↓
                  ├─── NFC卡服务
                  └─── 内容服务

NFC卡服务 ─────────┐
                   ↓
                 card-events
                   ↓
                   ├─── 商户服务
                   ├─── 内容服务
                   └─── 分发服务

内容服务 ──────────┐
                   ↓
                 video-events
                   ↓
                   ├─── 分发服务
                   └─── 统计服务

分发服务 ──────────┐
                   ↓
                publish-events
                   ↓
                   ├─── 内容服务
                   └─── 统计服务
```

## 7. 消息可靠性保证

### 7.1 消息生产者配置
- 使用同步发送确保消息已被写入
- 配置适当的重试次数
- 确保 `acks=all` 以获得最强的持久性保证

### 7.2 消息消费者配置
- 手动提交偏移量，确保消息被成功处理后再提交
- 实现幂等性处理，确保重复消息不会导致数据不一致
- 适当的错误处理和重试机制

## 8. 监控和运维

### 8.1 监控指标
- 消息生产和消费的延迟
- 消息处理错误率
- 消费者组的消费延迟

### 8.2 日志规范
所有与Kafka相关的操作都应该记录适当的日志，包括：
- 消息发送成功/失败
- 消息接收和处理
- 消费者启动和关闭

## 9. 实现示例

参考各服务中的Kafka实现代码，确保遵循本规范进行集成和开发。 