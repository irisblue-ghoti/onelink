package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/IBM/sarama"
)

// Config Kafka配置
type Config struct {
	Brokers        []string
	ConsumerGroup  string
	ConsumerTopics []string
	ProducerTopics map[string]string
	ServiceName    string
}

// Client Kafka客户端
type Client struct {
	config     *Config
	producer   sarama.SyncProducer
	consumer   sarama.ConsumerGroup
	handlers   map[string]MessageHandler
	ctx        context.Context
	cancelFunc context.CancelFunc
	ready      chan bool
	mutex      sync.Mutex
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	// HandleMessage 处理接收到的消息
	// 注意：未来我们将添加上下文参数，用于传递追踪ID
	HandleMessage(topic string, message *Message) error
}

// NewClient 创建Kafka客户端
func NewClient(cfg *Config) (*Client, error) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config:     cfg,
		handlers:   make(map[string]MessageHandler),
		ctx:        ctx,
		cancelFunc: cancel,
		ready:      make(chan bool),
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
func (c *Client) initProducer() error {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(c.config.Brokers, config)
	if err != nil {
		return fmt.Errorf("创建Kafka生产者失败: %w", err)
	}

	c.producer = producer
	return nil
}

// initConsumer 初始化消费者
func (c *Client) initConsumer() error {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumer, err := sarama.NewConsumerGroup(c.config.Brokers, c.config.ConsumerGroup, config)
	if err != nil {
		return fmt.Errorf("创建Kafka消费者组失败: %w", err)
	}

	c.consumer = consumer
	return nil
}

// RegisterHandler 注册消息处理器
func (c *Client) RegisterHandler(topic string, handler MessageHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.handlers[topic] = handler
}

// SendMessage 发送消息
func (c *Client) SendMessage(topic string, msgType MessageType, data interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 创建消息
	message, err := NewMessage(msgType, data, c.config.ServiceName)
	if err != nil {
		return fmt.Errorf("创建消息失败: %w", err)
	}

	// 序列化消息
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 创建Kafka消息
	kafkaMsg := &sarama.ProducerMessage{
		Topic: string(topic),
		Value: sarama.StringEncoder(jsonData),
	}

	// 发送消息
	partition, offset, err := c.producer.SendMessage(kafkaMsg)
	if err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	log.Printf("消息已发送: topic=%s, type=%s, partition=%d, offset=%d", topic, msgType, partition, offset)
	return nil
}

// SendMessageWithContext 发送消息，并从上下文中获取追踪ID
func (c *Client) SendMessageWithContext(ctx context.Context, topic string, msgType MessageType, data interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 从上下文获取追踪ID
	var traceID string
	if ctx != nil {
		// 尝试从上下文获取追踪ID
		// 兼容多种可能的键名
		traceIDValues := []interface{}{
			ctx.Value("trace_id"),
			ctx.Value("traceID"),
			ctx.Value("TraceID"),
		}

		for _, val := range traceIDValues {
			if v, ok := val.(string); ok && v != "" {
				traceID = v
				break
			}
		}
	}

	// 创建消息，使用从上下文获取的追踪ID
	var message *Message
	var err error
	if traceID != "" {
		message, err = NewMessageWithTraceID(msgType, data, c.config.ServiceName, traceID)
	} else {
		message, err = NewMessage(msgType, data, c.config.ServiceName)
	}

	if err != nil {
		return fmt.Errorf("创建消息失败: %w", err)
	}

	// 序列化消息
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 创建Kafka消息
	kafkaMsg := &sarama.ProducerMessage{
		Topic: string(topic),
		Value: sarama.StringEncoder(jsonData),
	}

	// 发送消息
	partition, offset, err := c.producer.SendMessage(kafkaMsg)
	if err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	log.Printf("消息已发送: topic=%s, type=%s, trace_id=%s, partition=%d, offset=%d",
		topic, msgType, message.TraceID, partition, offset)
	return nil
}

// StartConsumers 启动消费者
func (c *Client) StartConsumers() error {
	if c.consumer == nil || len(c.config.ConsumerTopics) == 0 {
		log.Println("未配置消费者或主题，跳过启动")
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
				ready:    c.ready,
				handlers: c.handlers,
			}

			// 启动消费者组
			topics := make([]string, len(c.config.ConsumerTopics))
			for i, topic := range c.config.ConsumerTopics {
				topics[i] = string(topic)
			}

			if err := c.consumer.Consume(c.ctx, topics, handler); err != nil {
				consumeErrors <- fmt.Errorf("消费者组错误: %w", err)
				return
			}

			// 检查上下文是否已取消
			if c.ctx.Err() != nil {
				return
			}
		}
	}()

	// 等待消费者就绪
	<-c.ready

	log.Printf("Kafka消费者已启动，正在监听主题: %v", c.config.ConsumerTopics)

	// 等待信号或错误
	select {
	case <-c.ctx.Done():
		log.Println("上下文取消，正在关闭消费者...")
	case <-sigterm:
		log.Println("收到终止信号，正在关闭消费者...")
	case err := <-consumeErrors:
		return err
	}

	return nil
}

// Close 关闭Kafka客户端
func (c *Client) Close() {
	c.cancelFunc()

	if c.producer != nil {
		if err := c.producer.Close(); err != nil {
			log.Printf("关闭Kafka生产者失败: %v", err)
		}
	}

	if c.consumer != nil {
		if err := c.consumer.Close(); err != nil {
			log.Printf("关闭Kafka消费者失败: %v", err)
		}
	}
}

// consumerHandler 实现 sarama.ConsumerGroupHandler 接口
type consumerHandler struct {
	ready    chan bool
	handlers map[string]MessageHandler
}

func (h *consumerHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

func (h *consumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		log.Printf("收到消息: topic=%s, partition=%d, offset=%d",
			msg.Topic, msg.Partition, msg.Offset)

		// 解析消息
		var message Message
		if err := json.Unmarshal(msg.Value, &message); err != nil {
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
		if err := handler.HandleMessage(msg.Topic, &message); err != nil {
			log.Printf("处理消息失败: %v", err)
		}

		// 标记消息为已处理
		session.MarkMessage(msg, "")
	}
	return nil
}
