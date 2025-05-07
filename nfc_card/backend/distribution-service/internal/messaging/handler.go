package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"distribution-service/internal/config"
	"distribution-service/internal/domain/entities"
	"distribution-service/internal/services"
)

// MessageTypes 消息类型常量
const (
	MessageTypeVideoCreated        = "video.created"
	MessageTypeVideoUpdated        = "video.updated"
	MessageTypePublishJobCreated   = "publish_job.created"
	MessageTypePublishJobUpdated   = "publish_job.updated"
	MessageTypePublishJobCompleted = "publish_job.completed"
)

// Topics 主题常量
const (
	TopicVideoEvents   = "video-events"
	TopicPublishEvents = "publish-events"
	TopicCardEvents    = "card-events"
)

// MessageProcessor 消息处理器
type MessageProcessor struct {
	publishService *services.PublishService
	config         *config.Config
}

// NewMessageProcessor 创建消息处理器
func NewMessageProcessor(publishService *services.PublishService, cfg *config.Config) *MessageProcessor {
	return &MessageProcessor{
		publishService: publishService,
		config:         cfg,
	}
}

// SetPublishService 设置发布服务
func (mp *MessageProcessor) SetPublishService(service *services.PublishService) {
	mp.publishService = service
}

// HandleMessage 处理接收到的消息
func (mp *MessageProcessor) HandleMessage(topic string, payload *MessagePayload) error {
	log.Printf("收到消息: topic=%s, type=%s", topic, payload.Type)

	switch topic {
	case TopicVideoEvents:
		return mp.handleVideoEvents(payload)
	case TopicPublishEvents:
		return mp.handlePublishEvents(payload)
	case TopicCardEvents:
		return mp.handleCardEvents(payload)
	default:
		return fmt.Errorf("未知主题: %s", topic)
	}
}

// handleVideoEvents 处理视频相关事件
func (mp *MessageProcessor) handleVideoEvents(payload *MessagePayload) error {
	switch payload.Type {
	case MessageTypeVideoCreated:
		return mp.handleVideoCreated(payload.Data)
	case MessageTypeVideoUpdated:
		return mp.handleVideoUpdated(payload.Data)
	default:
		return fmt.Errorf("未知的视频事件类型: %s", payload.Type)
	}
}

// handlePublishEvents 处理发布相关事件
func (mp *MessageProcessor) handlePublishEvents(payload *MessagePayload) error {
	switch payload.Type {
	case MessageTypePublishJobCreated:
		return mp.handlePublishJobCreated(payload.Data)
	case MessageTypePublishJobUpdated:
		return mp.handlePublishJobUpdated(payload.Data)
	default:
		return fmt.Errorf("未知的发布事件类型: %s", payload.Type)
	}
}

// handleCardEvents 处理NFC卡相关事件
func (mp *MessageProcessor) handleCardEvents(payload *MessagePayload) error {
	// 根据具体业务逻辑处理NFC卡相关事件
	log.Printf("处理NFC卡事件: %s", payload.Type)
	return nil
}

// handleVideoCreated 处理视频创建事件
func (mp *MessageProcessor) handleVideoCreated(data interface{}) error {
	// 将data转换为Video对象
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化视频数据失败: %w", err)
	}

	var video entities.Video
	if err := json.Unmarshal(jsonData, &video); err != nil {
		return fmt.Errorf("反序列化视频数据失败: %w", err)
	}

	log.Printf("处理视频创建事件: videoID=%s, title=%s", video.ID, video.Title)

	// 在这里可以添加处理视频创建的业务逻辑
	// 例如：自动创建分发任务等

	return nil
}

// handleVideoUpdated 处理视频更新事件
func (mp *MessageProcessor) handleVideoUpdated(data interface{}) error {
	// 将data转换为Video对象
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化视频数据失败: %w", err)
	}

	var video entities.Video
	if err := json.Unmarshal(jsonData, &video); err != nil {
		return fmt.Errorf("反序列化视频数据失败: %w", err)
	}

	log.Printf("处理视频更新事件: videoID=%s, title=%s", video.ID, video.Title)

	// 在这里可以添加处理视频更新的业务逻辑
	// 例如：更新已有的分发任务等

	return nil
}

// handlePublishJobCreated 处理分发任务创建事件
func (mp *MessageProcessor) handlePublishJobCreated(data interface{}) error {
	// 将data转换为PublishJob对象
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化任务数据失败: %w", err)
	}

	var job entities.PublishJob
	if err := json.Unmarshal(jsonData, &job); err != nil {
		return fmt.Errorf("反序列化任务数据失败: %w", err)
	}

	log.Printf("处理分发任务创建事件: jobID=%s, channel=%s", job.ID, job.Channel)

	// 在这里可以添加处理分发任务创建的业务逻辑
	// 例如：开始异步处理分发任务等

	return mp.publishService.CreateJob(context.Background(), &job)
}

// handlePublishJobUpdated 处理分发任务更新事件
func (mp *MessageProcessor) handlePublishJobUpdated(data interface{}) error {
	// 将data转换为PublishJob对象
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化任务数据失败: %w", err)
	}

	var job entities.PublishJob
	if err := json.Unmarshal(jsonData, &job); err != nil {
		return fmt.Errorf("反序列化任务数据失败: %w", err)
	}

	log.Printf("处理分发任务更新事件: jobID=%s, status=%s", job.ID, job.Status)

	// 在这里可以添加处理分发任务更新的业务逻辑
	// 例如：更新任务状态等

	return nil
}
