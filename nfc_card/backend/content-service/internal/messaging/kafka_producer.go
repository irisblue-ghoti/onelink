package messaging

import (
	"encoding/json"
	"log"
	"time"

	"content-service/internal/config"

	"github.com/IBM/sarama"
)

// KafkaProducer Kafka生产者结构
type KafkaProducer struct {
	config   *config.Config
	producer sarama.SyncProducer
}

// NewKafkaProducer 创建新的Kafka生产者
func NewKafkaProducer(cfg *config.Config) (*KafkaProducer, error) {
	// 配置Sarama
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_1_0                 // 使用Kafka 2.8.1
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll // 等待所有副本确认
	saramaConfig.Producer.Retry.Max = 5                    // 重试次数
	saramaConfig.Producer.Return.Successes = true

	// 创建生产者
	producer, err := sarama.NewSyncProducer(cfg.Kafka.Brokers, saramaConfig)
	if err != nil {
		return nil, err
	}

	return &KafkaProducer{
		config:   cfg,
		producer: producer,
	}, nil
}

// Close 关闭Kafka生产者
func (k *KafkaProducer) Close() error {
	return k.producer.Close()
}

// 事件类型常量
const (
	EventTypeVideoUploaded        = "video.uploaded"
	EventTypeVideoProcessing      = "video.processing"
	EventTypeVideoProcessed       = "video.processed"
	EventTypeVideoPublished       = "video.published"
	EventTypeContentSecurityCheck = "content.security.check"
)

// MessageEvent Kafka消息事件结构
type MessageEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// VideoUploadedPayload 视频上传事件载荷
type VideoUploadedPayload struct {
	ID         string `json:"id"`
	TenantID   string `json:"tenantId"`
	Title      string `json:"title"`
	FileKey    string `json:"fileKey"`
	FileType   string `json:"fileType"`
	Size       int64  `json:"size"`
	UploadedAt string `json:"uploadedAt"`
}

// VideoProcessingPayload 视频处理中事件载荷
type VideoProcessingPayload struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenantId"`
	Status       string `json:"status"`
	ProcessingAt string `json:"processingAt"`
}

// VideoProcessedPayload 视频处理完成事件载荷
type VideoProcessedPayload struct {
	ID          string  `json:"id"`
	TenantID    string  `json:"tenantId"`
	Status      string  `json:"status"`
	Duration    float64 `json:"duration"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	ProcessedAt string  `json:"processedAt"`
}

// VideoPublishedPayload 视频发布事件载荷
type VideoPublishedPayload struct {
	ID          string   `json:"id"`
	TenantID    string   `json:"tenantId"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Platforms   []string `json:"platforms"`
	PublishedAt string   `json:"publishedAt"`
}

// SecurityCheckPayload 内容安全检查事件载荷
type SecurityCheckPayload struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenantId"`
	ContentType string            `json:"contentType"`
	Result      string            `json:"result"`
	Violations  map[string]string `json:"violations"`
	CheckedAt   string            `json:"checkedAt"`
}

// SendVideoUploaded 发送视频上传事件
func (k *KafkaProducer) SendVideoUploaded(payload VideoUploadedPayload) error {
	return k.SendEvent(EventTypeVideoUploaded, payload)
}

// SendVideoProcessing 发送视频处理中事件
func (k *KafkaProducer) SendVideoProcessing(payload VideoProcessingPayload) error {
	return k.SendEvent(EventTypeVideoProcessing, payload)
}

// SendVideoProcessed 发送视频处理完成事件
func (k *KafkaProducer) SendVideoProcessed(payload VideoProcessedPayload) error {
	return k.SendEvent(EventTypeVideoProcessed, payload)
}

// SendVideoPublished 发送视频发布事件
func (k *KafkaProducer) SendVideoPublished(payload VideoPublishedPayload) error {
	return k.SendEvent(EventTypeVideoPublished, payload)
}

// SendSecurityCheck 发送内容安全检查事件
func (k *KafkaProducer) SendSecurityCheck(payload SecurityCheckPayload) error {
	return k.SendEvent(EventTypeContentSecurityCheck, payload)
}

// SendContentSecurityCheck 发送内容安全检查事件
func (k *KafkaProducer) SendContentSecurityCheck(payload SecurityCheckPayload) error {
	return k.SendEvent(EventTypeContentSecurityCheck, payload)
}

// ContentModerationResult 内容审核结果接口
type ContentModerationResult struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenantId"`
	ContentType    string                 `json:"contentType"` // "video", "image", "text"
	Status         string                 `json:"status"`      // "pass", "review", "reject"
	Categories     map[string]float64     `json:"categories"`  // 包含各内容类别的置信度分数
	Details        map[string]interface{} `json:"details"`     // 详细审核信息
	ModeratedAt    string                 `json:"moderatedAt"`
	Recommendation string                 `json:"recommendation"` // 处理建议
}

// 内容类别常量
const (
	ContentCategoryViolence    = "violence"
	ContentCategoryPornography = "pornography"
	ContentCategoryPolitics    = "politics"
	ContentCategoryHate        = "hate_speech"
	ContentCategoryDrugs       = "drugs"
	ContentCategoryAlcohol     = "alcohol"
	ContentCategoryCopyright   = "copyright"
)

// 内容审核状态
const (
	ContentModerationStatusPending   = "pending"
	ContentModerationStatusReviewing = "reviewing"
	ContentModerationStatusPassed    = "passed"
	ContentModerationStatusRejected  = "rejected"
	ContentModerationStatusError     = "error"
)

// SendContentModeration 发送内容审核事件
func (k *KafkaProducer) SendContentModeration(result ContentModerationResult) error {
	return k.SendEvent("content.moderation.result", result)
}

// SendEvent 发送事件
func (k *KafkaProducer) SendEvent(eventType string, payload interface{}) error {
	event := MessageEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Payload:   payload,
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: k.config.Kafka.Topic,
		Value: sarama.StringEncoder(jsonData),
	}

	partition, offset, err := k.producer.SendMessage(msg)
	if err != nil {
		return err
	}

	log.Printf("消息发送成功: 主题=%s, 分区=%d, 偏移量=%d, 类型=%s",
		k.config.Kafka.Topic, partition, offset, eventType)
	return nil
}
