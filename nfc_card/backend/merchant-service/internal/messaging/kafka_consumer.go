package messaging

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"

	"merchant-service/internal/config"
	"merchant-service/internal/services"
)

// KafkaConsumer Kafka消费者结构
type KafkaConsumer struct {
	config          *config.Config
	consumerGroup   sarama.ConsumerGroup
	merchantService *services.MerchantService
	topics          []string
}

// NewKafkaConsumer 创建新的Kafka消费者
func NewKafkaConsumer(cfg *config.Config, merchantService *services.MerchantService) (*KafkaConsumer, error) {
	// 配置Sarama
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_1_0 // 使用Kafka 2.8.1
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	// 创建消费者组
	topics := strings.Split(cfg.Kafka.Topic, ",")
	consumerGroup, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, "merchant-service", saramaConfig)
	if err != nil {
		return nil, err
	}

	return &KafkaConsumer{
		config:          cfg,
		consumerGroup:   consumerGroup,
		merchantService: merchantService,
		topics:          topics,
	}, nil
}

// Start 启动Kafka消费者
func (k *KafkaConsumer) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理系统信号
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// 启动消费者
	consumer := &Consumer{
		ready:           make(chan bool),
		merchantService: k.merchantService,
	}

	go func() {
		for {
			log.Printf("开始消费主题: %v", k.topics)
			if err := k.consumerGroup.Consume(ctx, k.topics, consumer); err != nil {
				log.Printf("消费出错: %v", err)
				time.Sleep(5 * time.Second) // 重试间隔
				continue
			}
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready
	log.Println("Kafka消费者已就绪")

	// 等待终止信号
	<-signals
	log.Println("正在关闭Kafka消费者...")
	cancel()
	if err := k.consumerGroup.Close(); err != nil {
		log.Panicf("关闭消费者组时出错: %v", err)
	}
}

// Consumer 实现sarama.ConsumerGroupHandler接口
type Consumer struct {
	ready           chan bool
	merchantService *services.MerchantService
}

// Setup 是在消费者会话开始时运行的
func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	close(c.ready)
	return nil
}

// Cleanup 是在消费者会话结束时运行的
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// 事件类型常量
const (
	EventTypeMerchantCreated = "merchant.created"
	EventTypeMerchantUpdated = "merchant.updated"
	EventTypePlanSubscribed  = "plan.subscribed"
	EventTypeUserCreated     = "user.created"
)

// MessageEvent Kafka消息事件结构
type MessageEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// MerchantCreatedPayload 商户创建事件载荷
type MerchantCreatedPayload struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Email       string `json:"email"`
}

// MerchantUpdatedPayload 商户更新事件载荷
type MerchantUpdatedPayload struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// PlanSubscribedPayload 订阅计划事件载荷
type PlanSubscribedPayload struct {
	MerchantID string `json:"merchantId"`
	PlanID     string `json:"planId"`
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
}

// UserCreatedPayload 用户创建事件载荷
type UserCreatedPayload struct {
	ID         string   `json:"id"`
	MerchantID string   `json:"merchantId"`
	Username   string   `json:"username"`
	Email      string   `json:"email"`
	Roles      []string `json:"roles"`
}

// ConsumeClaim 处理消息
func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		log.Printf("收到消息 %s: %s", message.Topic, string(message.Value))

		// 解析消息
		var event MessageEvent
		if err := json.Unmarshal(message.Value, &event); err != nil {
			log.Printf("解析消息失败: %v", err)
			session.MarkMessage(message, "")
			continue
		}

		// 根据事件类型处理消息
		switch event.Type {
		case EventTypeMerchantCreated:
			c.handleMerchantCreated(event.Payload)
		case EventTypeMerchantUpdated:
			c.handleMerchantUpdated(event.Payload)
		case EventTypePlanSubscribed:
			c.handlePlanSubscribed(event.Payload)
		case EventTypeUserCreated:
			c.handleUserCreated(event.Payload)
		default:
			log.Printf("未知事件类型: %s", event.Type)
		}

		// 标记消息已处理
		session.MarkMessage(message, "")
	}
	return nil
}

// 处理商户创建事件
func (c *Consumer) handleMerchantCreated(payload json.RawMessage) {
	var data MerchantCreatedPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("解析商户创建事件失败: %v", err)
		return
	}

	log.Printf("处理商户创建事件: %+v", data)
	// 业务逻辑处理（如更新本地数据库、发送通知等）
}

// 处理商户更新事件
func (c *Consumer) handleMerchantUpdated(payload json.RawMessage) {
	var data MerchantUpdatedPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("解析商户更新事件失败: %v", err)
		return
	}

	log.Printf("处理商户更新事件: %+v", data)
	// 业务逻辑处理
}

// 处理订阅计划事件
func (c *Consumer) handlePlanSubscribed(payload json.RawMessage) {
	var data PlanSubscribedPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("解析订阅计划事件失败: %v", err)
		return
	}

	log.Printf("处理订阅计划事件: %+v", data)
	// 业务逻辑处理
}

// 处理用户创建事件
func (c *Consumer) handleUserCreated(payload json.RawMessage) {
	var data UserCreatedPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("解析用户创建事件失败: %v", err)
		return
	}

	log.Printf("处理用户创建事件: %+v", data)
	// 业务逻辑处理
}
