package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"nfc-service/internal/config"

	"github.com/IBM/sarama"
)

// KafkaClient Kafka客户端
type KafkaClient struct {
	config     *config.KafkaConfig
	producer   sarama.SyncProducer
	consumer   sarama.ConsumerGroup
	handlers   map[string]MessageHandler
	ctx        context.Context
	cancelFunc context.CancelFunc
	mutex      sync.Mutex
}

// MessagePayload 消息载荷
type MessagePayload struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	// HandleMessage 处理接收到的消息
	HandleMessage(topic string, msgType string, data []byte) error
}

// NewKafkaClient 创建Kafka客户端
func NewKafkaClient(cfg *config.KafkaConfig) (*KafkaClient, error) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	client := &KafkaClient{
		config:     cfg,
		handlers:   make(map[string]MessageHandler),
		ctx:        ctx,
		cancelFunc: cancel,
	}

	// 初始化生产者
	if err := client.initProducer(); err != nil {
		cancel()
		return nil, err
	}

	// 初始化消费者（如果有配置消费主题）
	if len(cfg.ConsumerTopics) > 0 {
		if err := client.initConsumer(); err != nil {
			client.producer.Close()
			cancel()
			return nil, err
		}
	}

	return client, nil
}

// initProducer 初始化生产者
func (k *KafkaClient) initProducer() error {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(k.config.Brokers, config)
	if err != nil {
		return fmt.Errorf("创建Kafka生产者失败: %w", err)
	}

	k.producer = producer
	return nil
}

// initConsumer 初始化消费者
func (k *KafkaClient) initConsumer() error {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumer, err := sarama.NewConsumerGroup(k.config.Brokers, k.config.ConsumerGroup, config)
	if err != nil {
		return fmt.Errorf("创建Kafka消费者组失败: %w", err)
	}

	k.consumer = consumer
	return nil
}

// RegisterHandler 注册消息处理器
func (k *KafkaClient) RegisterHandler(topic string, handler MessageHandler) {
	k.mutex.Lock()
	defer k.mutex.Unlock()
	k.handlers[topic] = handler
}

// SendMessage 发送消息
func (k *KafkaClient) SendMessage(topic, msgType string, data interface{}) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	// 构建消息负载
	payload := MessagePayload{
		Type:      msgType,
		Timestamp: time.Now(),
		Source:    "nfc-service",
	}

	// 序列化数据
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %w", err)
	}
	payload.Data = jsonData

	// 序列化整个消息
	msgData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 创建消息
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(msgData),
	}

	// 发送消息
	partition, offset, err := k.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	log.Printf("消息已发送: topic=%s, type=%s, partition=%d, offset=%d", topic, msgType, partition, offset)
	return nil
}

// StartConsumers 启动消费者
func (k *KafkaClient) StartConsumers() {
	if k.consumer == nil {
		log.Println("未配置消费者，跳过启动")
		return
	}

	// 启动消费者协程
	go func() {
		for {
			select {
			case <-k.ctx.Done():
				log.Println("消费者上下文被取消，停止消费")
				return
			default:
				// 创建处理器
				handler := &consumerHandler{
					handlers: k.handlers,
				}

				// 消费消息
				if err := k.consumer.Consume(k.ctx, k.config.ConsumerTopics, handler); err != nil {
					log.Printf("消费消息时出错: %v", err)
				}

				// 检查上下文是否已取消
				if k.ctx.Err() != nil {
					return
				}
			}
		}
	}()

	log.Printf("启动Kafka消费者，监听主题: %v", k.config.ConsumerTopics)
}

// Close 关闭Kafka客户端
func (k *KafkaClient) Close() {
	k.cancelFunc()

	if k.producer != nil {
		if err := k.producer.Close(); err != nil {
			log.Printf("关闭Kafka生产者失败: %v", err)
		}
	}

	if k.consumer != nil {
		if err := k.consumer.Close(); err != nil {
			log.Printf("关闭Kafka消费者失败: %v", err)
		}
	}
}

// consumerHandler 实现 sarama.ConsumerGroupHandler 接口
type consumerHandler struct {
	handlers map[string]MessageHandler
}

func (h *consumerHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		log.Printf("收到消息: topic=%s, partition=%d, offset=%d, key=%s",
			msg.Topic, msg.Partition, msg.Offset, string(msg.Key))

		// 解析消息
		var payload MessagePayload
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			log.Printf("解析消息失败: %v", err)
			session.MarkMessage(msg, "")
			continue
		}

		// 获取该主题的处理器
		handler, ok := h.handlers[msg.Topic]
		if !ok {
			log.Printf("未找到主题的处理器: %s", msg.Topic)
			session.MarkMessage(msg, "")
			continue
		}

		// 处理消息
		if err := handler.HandleMessage(msg.Topic, payload.Type, payload.Data); err != nil {
			log.Printf("处理消息失败: %v", err)
		}

		// 标记消息为已处理
		session.MarkMessage(msg, "")
	}
	return nil
}
