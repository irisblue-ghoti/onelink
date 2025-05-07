package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"distribution-service/internal/config"
)

// KafkaClient Kafka客户端
type KafkaClient struct {
	config         *config.KafkaConfig
	producer       sarama.SyncProducer
	consumerGroup  sarama.ConsumerGroup
	messageHandler MessageHandler
	ready          chan bool
	mutex          sync.Mutex
}

// MessagePayload 消息载荷
type MessagePayload struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Source    string      `json:"source"`
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	// HandleMessage 处理接收到的消息
	HandleMessage(topic string, payload *MessagePayload) error
}

// NewKafkaClient 创建新的Kafka客户端
func NewKafkaClient(cfg *config.KafkaConfig, handler MessageHandler) (*KafkaClient, error) {
	client := &KafkaClient{
		config:         cfg,
		messageHandler: handler,
		ready:          make(chan bool),
	}

	// 初始化生产者
	if err := client.initProducer(); err != nil {
		return nil, err
	}

	// 初始化消费者
	if err := client.initConsumer(); err != nil {
		client.producer.Close()
		return nil, err
	}

	return client, nil
}

// initProducer 初始化生产者
func (k *KafkaClient) initProducer() error {
	// 生产者配置
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	// 创建生产者
	producer, err := sarama.NewSyncProducer(k.config.Brokers, config)
	if err != nil {
		return fmt.Errorf("创建Kafka生产者失败: %w", err)
	}

	k.producer = producer
	return nil
}

// initConsumer 初始化消费者
func (k *KafkaClient) initConsumer() error {
	// 消费者配置
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	// 创建消费者组
	consumer, err := sarama.NewConsumerGroup(k.config.Brokers, k.config.ConsumerGroup, config)
	if err != nil {
		return fmt.Errorf("创建Kafka消费者组失败: %w", err)
	}

	k.consumerGroup = consumer
	return nil
}

// SendMessage 发送消息
func (k *KafkaClient) SendMessage(topic string, messageType string, data interface{}) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	// 构建消息载荷
	payload := MessagePayload{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
		Source:    "distribution-service",
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 创建Kafka消息
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(jsonData),
	}

	// 发送消息
	partition, offset, err := k.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	log.Printf("消息发送成功: topic=%s, partition=%d, offset=%d", topic, partition, offset)
	return nil
}

// StartConsumer 启动消费者
func (k *KafkaClient) StartConsumer(ctx context.Context) error {
	// 检查是否有要订阅的主题
	if len(k.config.ConsumerTopics) == 0 {
		log.Println("没有配置消费主题，跳过消费者启动")
		return nil
	}

	// 监听系统信号
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	// 消费者错误通道
	consumeErrors := make(chan error, 1)

	// 在单独的协程中启动消费者
	go func() {
		defer close(consumeErrors)
		for {
			// 创建消费者处理器
			handler := &consumerHandler{
				ready:          k.ready,
				messageHandler: k.messageHandler,
			}

			// 启动消费者组
			if err := k.consumerGroup.Consume(ctx, k.config.ConsumerTopics, handler); err != nil {
				consumeErrors <- fmt.Errorf("消费者组错误: %w", err)
				return
			}

			// 检查上下文是否已取消
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// 等待消费者就绪
	<-k.ready

	log.Printf("Kafka消费者已启动，正在监听主题: %v", k.config.ConsumerTopics)

	// 等待信号或错误
	select {
	case <-ctx.Done():
		log.Println("上下文取消，正在关闭消费者...")
	case <-sigterm:
		log.Println("收到终止信号，正在关闭消费者...")
	case err := <-consumeErrors:
		return err
	}

	return nil
}

// Close 关闭Kafka客户端
func (k *KafkaClient) Close() error {
	// 关闭生产者
	if err := k.producer.Close(); err != nil {
		log.Printf("关闭Kafka生产者失败: %v", err)
	}

	// 关闭消费者
	if err := k.consumerGroup.Close(); err != nil {
		return fmt.Errorf("关闭Kafka消费者失败: %w", err)
	}

	return nil
}

// consumerHandler 消费者处理器
type consumerHandler struct {
	ready          chan bool
	messageHandler MessageHandler
}

// Setup 消费者启动前的设置
func (h *consumerHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

// Cleanup 消费者关闭后的清理
func (h *consumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 消费消息
func (h *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		// 解析消息载荷
		var payload MessagePayload
		if err := json.Unmarshal(message.Value, &payload); err != nil {
			log.Printf("解析消息失败: %v, topic: %s, partition: %d, offset: %d",
				err, message.Topic, message.Partition, message.Offset)
			session.MarkMessage(message, "")
			continue
		}

		// 处理消息
		if err := h.messageHandler.HandleMessage(message.Topic, &payload); err != nil {
			log.Printf("处理消息失败: %v, topic: %s, partition: %d, offset: %d, type: %s",
				err, message.Topic, message.Partition, message.Offset, payload.Type)

			// 如果是发布任务消息，需要记录失败并尝试重试
			if isPublishTaskMessage(payload.Type) {
				if err := handlePublishTaskError(message, &payload, err); err != nil {
					log.Printf("处理任务失败记录失败: %v", err)
				}
			}
		}

		// 标记消息为已处理
		session.MarkMessage(message, "")
	}
	return nil
}

// isPublishTaskMessage 判断是否为发布任务消息
func isPublishTaskMessage(msgType string) bool {
	return msgType == "publish_job.created" ||
		msgType == "publish_job.retry" ||
		msgType == "publish_job.updated"
}

// handlePublishTaskError 处理发布任务错误
func handlePublishTaskError(msg *sarama.ConsumerMessage, payload *MessagePayload, processingErr error) error {
	// 如果是任务消息，需要尝试重试
	var publishJob map[string]interface{}
	if err := json.Unmarshal(payload.Data.([]byte), &publishJob); err != nil {
		return fmt.Errorf("解析任务数据失败: %w", err)
	}

	// 获取当前重试次数
	retryCount := 0
	if count, ok := publishJob["retryCount"].(float64); ok {
		retryCount = int(count)
	}

	// 增加重试次数
	retryCount++
	publishJob["retryCount"] = retryCount
	publishJob["lastError"] = processingErr.Error()

	// 检查是否超过最大重试次数
	maxRetries := 3
	if retryCount > maxRetries {
		// 超过最大重试次数，发送到死信队列
		publishJob["status"] = "failed"
		deadLetterData, _ := json.Marshal(publishJob)

		// 创建死信消息并发送到死信队列
		// 注意：这里仅记录日志，实际应该发送到死信队列
		log.Printf("任务失败，已达最大重试次数(%d)，发送到死信队列: %v, 数据: %s",
			maxRetries, publishJob["id"], string(deadLetterData))
		return nil
	}

	// 计算下次重试时间 (使用指数退避策略)
	backoff := 1 << uint(retryCount-1) // 1, 2, 4, 8, 16...分钟
	publishJob["nextRetryAt"] = time.Now().Add(time.Duration(backoff) * time.Minute).Format(time.RFC3339)
	publishJob["status"] = "retrying"

	retryData, _ := json.Marshal(publishJob)

	// 创建重试消息并发送到重试队列
	// 注意：这里仅记录日志，实际应该发送到重试队列
	log.Printf("任务失败，计划重试 #%d，任务ID: %v, 数据: %s",
		retryCount, publishJob["id"], string(retryData))
	return nil
}
