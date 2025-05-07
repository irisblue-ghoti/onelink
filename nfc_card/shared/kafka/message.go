package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MessageType 定义消息类型常量
type MessageType string

// 消息类型常量定义
const (
	// 商户相关消息类型
	TypeMerchantCreated MessageType = "merchant.created"
	TypeMerchantUpdated MessageType = "merchant.updated"

	// NFC卡相关消息类型
	TypeCardCreated MessageType = "card.created"
	TypeCardUpdated MessageType = "card.updated"
	TypeCardBound   MessageType = "card.bound"
	TypeCardUnbound MessageType = "card.unbound"

	// 视频内容相关消息类型
	TypeVideoCreated MessageType = "video.created"
	TypeVideoUpdated MessageType = "video.updated"
	TypeVideoDeleted MessageType = "video.deleted"

	// 发布任务相关消息类型
	TypePublishJobCreated   MessageType = "publish_job.created"
	TypePublishJobUpdated   MessageType = "publish_job.updated"
	TypePublishJobCompleted MessageType = "publish_job.completed"
)

// Topic 定义Kafka主题常量
type Topic string

// 主题常量定义
const (
	TopicMerchantEvents Topic = "merchant-events"
	TopicCardEvents     Topic = "card-events"
	TopicVideoEvents    Topic = "video-events"
	TopicPublishEvents  Topic = "publish-events"
	TopicStatsEvents    Topic = "stats-events"
)

// Message 定义标准Kafka消息结构
type Message struct {
	Type      MessageType     `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	TraceID   string          `json:"trace_id"`
}

// NewMessage 创建新的消息
func NewMessage(msgType MessageType, data interface{}, source string) (*Message, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("序列化消息数据失败: %w", err)
	}

	return &Message{
		Type:      msgType,
		Data:      jsonData,
		Timestamp: time.Now(),
		Source:    source,
		TraceID:   uuid.New().String(),
	}, nil
}

// NewMessageWithTraceID 创建带追踪ID的新消息
func NewMessageWithTraceID(msgType MessageType, data interface{}, source, traceID string) (*Message, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("序列化消息数据失败: %w", err)
	}

	if traceID == "" {
		traceID = uuid.New().String()
	}

	return &Message{
		Type:      msgType,
		Data:      jsonData,
		Timestamp: time.Now(),
		Source:    source,
		TraceID:   traceID,
	}, nil
}

// GetPayload 获取消息数据
func (m *Message) GetPayload(v interface{}) error {
	return json.Unmarshal(m.Data, v)
}

// UnmarshalData 将消息数据反序列化为指定类型
func (m *Message) UnmarshalData(v interface{}) error {
	return json.Unmarshal(m.Data, v)
}

// generateTraceID 生成唯一的跟踪ID
func generateTraceID() string {
	return time.Now().Format("20060102-150405") + "-" + randomString(8)
}

// randomString 生成指定长度的随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
