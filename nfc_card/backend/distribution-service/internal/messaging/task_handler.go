package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"distribution-service/internal/config"
	"distribution-service/internal/domain/entities"
	"distribution-service/internal/messaging/kafka"
	"distribution-service/internal/services"
)

// 任务类型常量
const (
	TaskTypePublishContent = "publish_content"
)

// PublishTaskHandler 发布任务处理器
type PublishTaskHandler struct {
	publishService *services.PublishService
}

// NewPublishTaskHandler 创建发布任务处理器
func NewPublishTaskHandler(publishService *services.PublishService) *PublishTaskHandler {
	return &PublishTaskHandler{
		publishService: publishService,
	}
}

// HandleTask 处理任务
func (h *PublishTaskHandler) HandleTask(ctx context.Context, task *kafka.TaskMessage) error {
	log.Printf("处理发布任务: ID=%s, 类型=%s", task.ID, task.Type)

	// 解析任务数据
	var job entities.PublishJob
	if err := json.Unmarshal(task.Payload, &job); err != nil {
		return fmt.Errorf("解析任务数据失败: %w", err)
	}

	// 更新任务状态为处理中
	if err := h.publishService.UpdateJobStatus(ctx, &job, "processing", ""); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 查询视频信息
	video, err := h.publishService.GetVideo(ctx, job.TenantID, job.VideoID)
	if err != nil {
		errorMsg := fmt.Sprintf("查询视频信息失败: %v", err)
		h.publishService.UpdateJobStatus(ctx, &job, "failed", errorMsg)
		return errors.New(errorMsg)
	}

	// 查询NFC卡信息
	// 这里根据实际业务需求添加

	// 执行发布操作
	err = h.publishContent(ctx, &job, video)
	if err != nil {
		errorMsg := fmt.Sprintf("发布内容失败: %v", err)
		h.publishService.UpdateJobStatus(ctx, &job, "failed", errorMsg)
		return errors.New(errorMsg)
	}

	// 更新任务状态为已完成
	job.CompletedAt = time.Now()
	if err := h.publishService.UpdateJobStatus(ctx, &job, "completed", ""); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	return nil
}

// GetTaskType 获取处理的任务类型
func (h *PublishTaskHandler) GetTaskType() string {
	return TaskTypePublishContent
}

// publishContent 发布内容到指定渠道
func (h *PublishTaskHandler) publishContent(ctx context.Context, job *entities.PublishJob, video *entities.Video) error {
	// 根据渠道执行发布
	switch job.Channel {
	case "douyin":
		return h.publishService.PublishToDouyin(ctx, job, video)
	case "kuaishou":
		return h.publishService.PublishToKuaishou(ctx, job, video)
	case "xiaohongshu":
		return h.publishService.PublishToXiaohongshu(ctx, job, video)
	case "wechat":
		return h.publishService.PublishToWechat(ctx, job, video)
	default:
		return fmt.Errorf("不支持的渠道: %s", job.Channel)
	}
}

// TaskProcessor 任务处理器
type TaskProcessor struct {
	processor      *kafka.TaskProcessor
	kafkaClient    *kafka.Client
	publishService *services.PublishService
}

// NewTaskProcessor 创建任务处理器
func NewTaskProcessor(cfg *config.Config, publishService *services.PublishService) (*TaskProcessor, error) {
	// 创建Kafka配置
	kafkaConfig := &kafka.Config{
		Brokers:        cfg.Kafka.Brokers,
		ConsumerGroup:  cfg.Kafka.ConsumerGroup,
		ConsumerTopics: []string{},
		ProducerTopics: map[string]string{
			"tasks":      fmt.Sprintf("tasks-%s", "distribution-service"),
			"retries":    fmt.Sprintf("retries-%s", "distribution-service"),
			"deadLetter": "dead-letter",
		},
		ServiceName: "distribution-service",
	}

	// 创建Kafka客户端
	client, err := kafka.NewClient(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("创建Kafka客户端失败: %w", err)
	}

	// 创建任务队列配置
	taskConfig := &kafka.TaskQueueConfig{
		TaskTopic:       fmt.Sprintf("tasks-%s", "distribution-service"),
		RetryTopic:      fmt.Sprintf("retries-%s", "distribution-service"),
		DeadLetterTopic: "dead-letter",
		RetryInterval:   time.Duration(cfg.TaskProcessing.RetryIntervalMinutes) * time.Minute,
		MaxRetries:      cfg.TaskProcessing.MaxRetries,
	}

	// 创建任务处理器
	processor := kafka.NewTaskProcessor(client, taskConfig, "distribution-service")

	// 创建并返回任务处理器
	return &TaskProcessor{
		processor:      processor,
		kafkaClient:    client,
		publishService: publishService,
	}, nil
}

// Start 启动任务处理器
func (tp *TaskProcessor) Start(ctx context.Context) error {
	// 注册任务处理器
	tp.registerTaskHandlers()

	// 启动任务处理器
	return tp.processor.Start(ctx)
}

// Stop 停止任务处理器
func (tp *TaskProcessor) Stop() {
	tp.processor.Stop()
	tp.kafkaClient.Close()
}

// EnqueuePublishJob 将发布任务加入队列
func (tp *TaskProcessor) EnqueuePublishJob(job *entities.PublishJob) error {
	return tp.processor.EnqueueTask(TaskTypePublishContent, job, job.ID.String())
}

// registerTaskHandlers 注册任务处理器
func (tp *TaskProcessor) registerTaskHandlers() {
	// 注册发布任务处理器
	publishHandler := NewPublishTaskHandler(tp.publishService)
	tp.processor.RegisterHandler(publishHandler)
}
