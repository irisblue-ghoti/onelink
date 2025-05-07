# Kafka实现指南

本文档提供了如何在各个微服务中集成和使用Kafka消息队列的详细指南。遵循这些指南可以确保微服务之间的消息通信稳定、可靠和一致。

## 目录

1. [准备工作](#1-准备工作)
2. [Go服务集成](#2-go服务集成)
3. [NestJS服务集成](#3-nestjs服务集成)
4. [消息处理器实现](#4-消息处理器实现)
5. [错误处理和恢复机制](#5-错误处理和恢复机制)
6. [监控和日志](#6-监控和日志)
7. [常见问题](#7-常见问题)

## 1. 准备工作

### 1.1 依赖安装

对于Go服务，确保安装Sarama库：

```bash
go get github.com/IBM/sarama
```

对于NestJS服务，确保安装Kafka客户端：

```bash
npm install @nestjs/microservices kafkajs
```

### 1.2 配置文件

确保在各服务的配置文件中包含Kafka相关配置，参考[共享配置示例](../shared/kafka/config.yaml)。

## 2. Go服务集成

### 2.1 引入共享库

在Go服务中引入共享Kafka库：

```go
import (
    "github.com/nfc_card/shared/kafka"
)
```

### 2.2 初始化Kafka客户端

```go
// 创建Kafka配置
kafkaConfig := &kafka.Config{
    Brokers:        cfg.Kafka.Brokers,
    ConsumerGroup:  cfg.Kafka.ConsumerGroup,
    ConsumerTopics: cfg.Kafka.ConsumerTopics,
    ServiceName:    "your-service-name",
}

// 创建Kafka客户端
kafkaClient, err := kafka.NewClient(kafkaConfig)
if err != nil {
    log.Fatalf("初始化Kafka客户端失败: %v", err)
}

// 注册消息处理器
cardHandler := NewCardMessageHandler()
for _, topic := range cfg.Kafka.ConsumerTopics {
    kafkaClient.RegisterHandler(topic, cardHandler)
}

// 启动消费者（在单独的协程中）
go func() {
    if err := kafkaClient.StartConsumers(); err != nil {
        log.Printf("Kafka消费者错误: %v", err)
    }
}()

// 设置关闭钩子
defer kafkaClient.Close()
```

### 2.3 发送消息

```go
// 发送NFC卡创建事件
if err := kafkaClient.SendMessage(
    string(kafka.TopicCardEvents),
    kafka.TypeCardCreated,
    cardData,
); err != nil {
    log.Printf("发送卡片创建消息失败: %v", err)
}
```

## 3. NestJS服务集成

### 3.1 配置Kafka模块

在`app.module.ts`中：

```typescript
import { Module } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { ClientsModule, Transport } from '@nestjs/microservices';
import { KafkaModule } from './messaging/kafka.module';

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
      load: [kafkaConfig],
    }),
    KafkaModule,
    // 其他模块...
  ],
  controllers: [],
  providers: [],
})
export class AppModule {}
```

### 3.2 创建Kafka模块和服务

在`kafka.module.ts`中：

```typescript
import { Module, Global } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { ClientsModule, Transport } from '@nestjs/microservices';
import { KafkaService } from './kafka.service';
// 导入消息处理器
import { VideoEventHandler } from './handlers/video-event.handler';

@Global()
@Module({
  imports: [
    ConfigModule,
    ClientsModule.registerAsync([
      {
        name: 'KAFKA_CLIENT',
        inject: [ConfigService],
        useFactory: (configService: ConfigService) => ({
          transport: Transport.KAFKA,
          options: {
            client: {
              clientId: configService.get('kafka.clientId'),
              brokers: configService.get('kafka.brokers'),
            },
            consumer: {
              groupId: configService.get('kafka.groupId'),
            },
          },
        }),
      },
    ]),
  ],
  providers: [KafkaService, VideoEventHandler],
  exports: [KafkaService],
})
export class KafkaModule {}
```

### 3.3 实现Kafka服务

在`kafka.service.ts`中：

```typescript
import { Injectable, OnModuleInit, Inject } from '@nestjs/common';
import { ClientKafka } from '@nestjs/microservices';
import { ConfigService } from '@nestjs/config';
import { lastValueFrom } from 'rxjs';

export enum MessageType {
  // 定义消息类型常量
  VIDEO_CREATED = 'video.created',
  VIDEO_UPDATED = 'video.updated',
  // 其他消息类型...
}

@Injectable()
export class KafkaService implements OnModuleInit {
  constructor(
    @Inject('KAFKA_CLIENT') private kafkaClient: ClientKafka,
    private configService: ConfigService,
  ) {}

  async onModuleInit() {
    // 订阅主题
    const consumerTopics = this.configService.get<string[]>('kafka.consumerTopics') || [];
    
    for (const topic of consumerTopics) {
      this.kafkaClient.subscribeToResponseOf(topic);
    }

    await this.kafkaClient.connect();
  }

  /**
   * 发送消息到Kafka
   */
  async send(topic: string, type: MessageType, data: any) {
    const message = {
      type,
      data,
      timestamp: new Date(),
      source: 'your-service-name',
      traceId: this.generateTraceId(),
    };

    try {
      await lastValueFrom(this.kafkaClient.emit(topic, message));
      console.log(`消息已发送: ${topic}, 类型: ${type}`);
      return true;
    } catch (error) {
      console.error(`发送消息失败: ${topic}, 类型: ${type}`, error);
      return false;
    }
  }

  /**
   * 发送特定类型的事件（根据需要添加特定的方法）
   */
  sendVideoEvent(type: MessageType, data: any) {
    const topicName = this.configService.get('kafka.producerTopics.videoEvents');
    return this.send(topicName, type, data);
  }

  /**
   * 生成跟踪ID
   */
  private generateTraceId(): string {
    return new Date().toISOString().replace(/[-:.]/g, '') + '-' + 
      Math.random().toString(36).substring(2, 10);
  }
}
```

## 4. 消息处理器实现

### 4.1 Go服务消息处理器

```go
type CardMessageHandler struct {
    // 依赖服务或存储库
    cardService *services.CardService
}

func NewCardMessageHandler(cardService *services.CardService) *CardMessageHandler {
    return &CardMessageHandler{
        cardService: cardService,
    }
}

func (h *CardMessageHandler) HandleMessage(topic string, message *kafka.Message) error {
    log.Printf("处理消息: topic=%s, type=%s", topic, message.Type)

    switch message.Type {
    case kafka.TypeCardCreated:
        var cardData map[string]interface{}
        if err := message.UnmarshalData(&cardData); err != nil {
            return err
        }
        return h.handleCardCreated(cardData)
        
    case kafka.TypeCardUpdated:
        var cardData map[string]interface{}
        if err := message.UnmarshalData(&cardData); err != nil {
            return err
        }
        return h.handleCardUpdated(cardData)
        
    default:
        log.Printf("未知的消息类型: %s", message.Type)
        return nil
    }
}

func (h *CardMessageHandler) handleCardCreated(data map[string]interface{}) error {
    // 实现具体的业务逻辑
    log.Printf("处理卡片创建事件: %v", data["id"])
    return nil
}
```

### 4.2 NestJS服务消息处理器

```typescript
import { Injectable } from '@nestjs/common';
import { MessagePattern, Payload } from '@nestjs/microservices';
import { VideoService } from '../videos/video.service';

@Injectable()
export class VideoEventHandler {
  constructor(private videoService: VideoService) {}

  @MessagePattern('video-events')
  async handleVideoEvents(@Payload() message: any) {
    console.log(`收到视频事件: ${message.type}`);
    
    try {
      switch (message.type) {
        case 'video.created':
          await this.handleVideoCreated(message.data);
          break;
        case 'video.updated':
          await this.handleVideoUpdated(message.data);
          break;
        default:
          console.log(`未知的视频事件类型: ${message.type}`);
      }
    } catch (error) {
      console.error(`处理视频事件失败: ${error.message}`);
    }
  }

  private async handleVideoCreated(data: any) {
    // 实现具体的业务逻辑
    console.log(`处理视频创建事件: ${data.id}`);
  }
}
```

## 5. 错误处理和恢复机制

### 5.1 重试机制

对于重要的消息，应实现重试机制：

```go
// Go示例
func (h *Handler) handleWithRetry(fn func() error) error {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        
        log.Printf("操作失败，尝试重试 (%d/%d): %v", i+1, maxRetries, err)
        time.Sleep(time.Second * time.Duration(i+1)) // 退避策略
    }
    return fmt.Errorf("达到最大重试次数")
}
```

### 5.2 死信队列

考虑配置死信队列来处理无法处理的消息：

```yaml
# Kafka配置示例
kafka:
  # ... 其他配置
  deadLetterTopic: "dead-letter-queue"
```

## 6. 监控和日志

### 6.1 监控指标

应监控以下Kafka指标：

- 消息生产和消费的延迟
- 消息处理错误率
- 消费者组的消费延迟

### 6.2 日志记录

确保记录详细的Kafka操作日志：

```go
// Go示例
log.Printf("消息已发送: topic=%s, type=%s, traceId=%s", topic, msgType, message.TraceID)
```

```typescript
// TypeScript示例
console.log(`消息已发送: ${topic}, 类型: ${type}, traceId: ${message.traceId}`);
```

## 7. 常见问题

### 7.1 消费者不接收消息

- 检查消费者组配置
- 验证主题名称是否正确
- 确保Kafka服务可访问

### 7.2 消息序列化问题

- 确保所有服务使用一致的消息格式
- 检查JSON序列化/反序列化代码

### 7.3 消息丢失

- 配置适当的确认级别 (`acks=all`)
- 手动提交偏移量
- 增加副本因子 