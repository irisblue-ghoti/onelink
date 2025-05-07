package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// TaskQueueConfig 任务队列配置
type TaskQueueConfig struct {
	// TaskTopic 任务主题
	TaskTopic string
	// DeadLetterTopic 死信队列主题
	DeadLetterTopic string
	// RetryTopic 重试队列主题
	RetryTopic string
	// RetryInterval 重试扫描间隔
	RetryInterval time.Duration
	// MaxRetries 最大重试次数
	MaxRetries int
}

// DefaultTaskQueueConfig 默认任务队列配置
func DefaultTaskQueueConfig() *TaskQueueConfig {
	return &TaskQueueConfig{
		TaskTopic:       "tasks",
		DeadLetterTopic: "dead-letter",
		RetryTopic:      "retries",
		RetryInterval:   5 * time.Minute,
		MaxRetries:      3,
	}
}

// TaskHandler 任务处理器接口
type TaskHandler interface {
	// HandleTask 处理任务
	HandleTask(ctx context.Context, task *TaskMessage) error
	// GetTaskType 获取处理的任务类型
	GetTaskType() string
}

// TaskProcessor 任务处理器
type TaskProcessor struct {
	client      *Client
	config      *TaskQueueConfig
	handlers    map[string]TaskHandler
	retryTicker *time.Ticker
	stopChan    chan struct{}
	mutex       sync.RWMutex
	serviceName string
}

// NewTaskProcessor 创建任务处理器
func NewTaskProcessor(client *Client, config *TaskQueueConfig, serviceName string) *TaskProcessor {
	if config == nil {
		config = DefaultTaskQueueConfig()
	}

	return &TaskProcessor{
		client:      client,
		config:      config,
		handlers:    make(map[string]TaskHandler),
		stopChan:    make(chan struct{}),
		serviceName: serviceName,
	}
}

// RegisterHandler 注册任务处理器
func (tp *TaskProcessor) RegisterHandler(handler TaskHandler) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	taskType := handler.GetTaskType()
	tp.handlers[taskType] = handler
	log.Printf("已注册任务处理器: %s", taskType)
}

// GetHandler 获取任务处理器
func (tp *TaskProcessor) GetHandler(taskType string) (TaskHandler, bool) {
	tp.mutex.RLock()
	defer tp.mutex.RUnlock()

	handler, ok := tp.handlers[taskType]
	return handler, ok
}

// Start 启动任务处理器
func (tp *TaskProcessor) Start(ctx context.Context) error {
	// 注册主题处理器
	tp.registerTopicHandlers()

	// 启动重试扫描
	tp.startRetryScanner(ctx)

	// 修改Client配置以支持我们的主题
	tp.client.config.ConsumerTopics = append(
		tp.client.config.ConsumerTopics,
		tp.config.TaskTopic,
		tp.config.RetryTopic,
	)

	// 启动消费者
	return tp.client.StartConsumers()
}

// Stop 停止任务处理器
func (tp *TaskProcessor) Stop() {
	close(tp.stopChan)
	if tp.retryTicker != nil {
		tp.retryTicker.Stop()
	}
}

// EnqueueTask 将任务加入队列
func (tp *TaskProcessor) EnqueueTask(taskType string, payload interface{}, id string) error {
	// 创建任务消息
	task, err := NewTaskMessage(id, taskType, payload, tp.serviceName)
	if err != nil {
		return err
	}

	// 设置最大重试次数
	task.MaxRetries = tp.config.MaxRetries

	// 发送到任务队列
	return tp.sendTaskMessage(tp.config.TaskTopic, task)
}

// sendTaskMessage 发送任务消息
func (tp *TaskProcessor) sendTaskMessage(topic string, task *TaskMessage) error {
	return tp.client.SendMessage(topic, MessageType(task.Type), task)
}

// registerTopicHandlers 注册主题处理器
func (tp *TaskProcessor) registerTopicHandlers() {
	// 注册任务队列处理器
	tp.client.RegisterHandler(tp.config.TaskTopic, &taskMessageHandler{
		processor: tp,
	})

	// 注册重试队列处理器
	tp.client.RegisterHandler(tp.config.RetryTopic, &retryMessageHandler{
		processor: tp,
	})
}

// processSingleTask 处理单个任务
func (tp *TaskProcessor) processSingleTask(ctx context.Context, task *TaskMessage) error {
	// 获取任务处理器
	handler, ok := tp.GetHandler(task.Type)
	if !ok {
		err := fmt.Errorf("未找到任务类型 %s 的处理器", task.Type)
		// 发送到死信队列
		tp.sendToDeadLetter(task, err.Error())
		return err
	}

	// 标记为处理中
	task.MarkAsProcessing()

	// 处理任务
	err := handler.HandleTask(ctx, task)
	if err != nil {
		return tp.handleTaskError(task, err)
	}

	// 标记为已完成
	task.MarkAsCompleted()
	return nil
}

// handleTaskError 处理任务错误
func (tp *TaskProcessor) handleTaskError(task *TaskMessage, err error) error {
	errorMsg := err.Error()

	// 尝试重试
	if task.PrepareRetry(errorMsg) {
		// 发送到重试队列
		return tp.sendTaskMessage(tp.config.RetryTopic, task)
	}

	// 超过重试次数，发送到死信队列
	return tp.sendToDeadLetter(task, errorMsg)
}

// sendToDeadLetter 发送到死信队列
func (tp *TaskProcessor) sendToDeadLetter(task *TaskMessage, errorMsg string) error {
	task.MarkAsFailed(errorMsg)
	return tp.sendTaskMessage(tp.config.DeadLetterTopic, task)
}

// startRetryScanner 启动重试扫描
func (tp *TaskProcessor) startRetryScanner(ctx context.Context) {
	tp.retryTicker = time.NewTicker(tp.config.RetryInterval)

	go func() {
		for {
			select {
			case <-tp.retryTicker.C:
				tp.scanRetryQueue(ctx)
			case <-tp.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// scanRetryQueue 扫描重试队列
func (tp *TaskProcessor) scanRetryQueue(ctx context.Context) {
	// 此函数应该查询数据库中需要重试的任务
	// 这里只是一个示例，实际实现需要根据数据库设计来完成
	log.Println("扫描重试队列...")
}

// taskMessageHandler 任务消息处理器
type taskMessageHandler struct {
	processor *TaskProcessor
}

// HandleMessage 处理消息
func (h *taskMessageHandler) HandleMessage(topic string, message *Message) error {
	// 解析任务消息
	var task TaskMessage
	if err := json.Unmarshal(message.Data, &task); err != nil {
		return fmt.Errorf("解析任务消息失败: %w", err)
	}

	// 处理任务
	ctx := context.Background()
	return h.processor.processSingleTask(ctx, &task)
}

// retryMessageHandler 重试消息处理器
type retryMessageHandler struct {
	processor *TaskProcessor
}

// HandleMessage 处理消息
func (h *retryMessageHandler) HandleMessage(topic string, message *Message) error {
	// 解析任务消息
	var task TaskMessage
	if err := json.Unmarshal(message.Data, &task); err != nil {
		return fmt.Errorf("解析重试消息失败: %w", err)
	}

	// 检查是否应该立即重试
	if !task.ShouldRetryNow() {
		// 不需要立即重试，忽略消息
		return nil
	}

	// 重试任务
	ctx := context.Background()
	return h.processor.processSingleTask(ctx, &task)
}
