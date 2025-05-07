package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/IBM/sarama"
)

// ConsumerHandler Kafka消费者处理器
type ConsumerHandler struct {
	client *Client
	ready  chan bool
}

// Setup 是ConsumerGroupHandler接口要求的方法
func (h *ConsumerHandler) Setup(sarama.ConsumerGroupSession) error {
	// 标记消费者已准备好
	close(h.ready)
	return nil
}

// Cleanup 是ConsumerGroupHandler接口要求的方法
func (h *ConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 处理消息
func (h *ConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// 处理消息通道中的消息
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

		// 获取主题处理器
		h.client.mutex.Lock()
		handler, exists := h.client.handlers[msg.Topic]
		h.client.mutex.Unlock()

		if !exists {
			log.Printf("未找到主题的处理器: %s", msg.Topic)
			session.MarkMessage(msg, "")
			continue
		}

		// 创建带有追踪ID的上下文
		ctx := context.Background()
		if message.TraceID != "" {
			// 使用TraceID键将追踪ID添加到上下文
			ctx = context.WithValue(ctx, "trace_id", message.TraceID)
		}

		// 调用处理器处理消息
		// 注意：我们暂时忽略上下文，因为当前的接口不支持
		if err := handler.HandleMessage(msg.Topic, &message); err != nil {
			log.Printf("处理消息失败: %v", err)
			// 消息处理失败，但仍标记为已处理
			// 在实际应用中，可以根据错误类型决定是否重试
			session.MarkMessage(msg, "")
			continue
		}

		// 标记消息已处理
		session.MarkMessage(msg, "")
	}

	return nil
}

// 注意：不要在这里重复声明MessageHandler接口
// MessageHandler接口已在client.go中定义
