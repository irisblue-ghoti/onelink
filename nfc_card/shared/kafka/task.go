package kafka

import (
	"encoding/json"
	"fmt"
	"time"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	// TaskStatusPending 等待处理
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusProcessing 处理中
	TaskStatusProcessing TaskStatus = "processing"
	// TaskStatusCompleted 已完成
	TaskStatusCompleted TaskStatus = "completed"
	// TaskStatusFailed 失败
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusRetrying 重试中
	TaskStatusRetrying TaskStatus = "retrying"
)

// TaskMessage 任务消息
type TaskMessage struct {
	// ID 任务ID
	ID string `json:"id"`
	// Type 任务类型
	Type string `json:"type"`
	// Payload 任务载荷
	Payload json.RawMessage `json:"payload"`
	// Status 任务状态
	Status TaskStatus `json:"status"`
	// RetryCount 重试次数
	RetryCount int `json:"retryCount"`
	// MaxRetries
	MaxRetries int `json:"maxRetries"`
	// LastError 最后一次错误信息
	LastError string `json:"lastError,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
	// NextRetryAt 下次重试时间
	NextRetryAt *time.Time `json:"nextRetryAt,omitempty"`
	// Source 消息来源服务
	Source string `json:"source"`
}

// NewTaskMessage 创建新的任务消息
func NewTaskMessage(id, taskType string, payload interface{}, source string) (*TaskMessage, error) {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化任务载荷失败: %w", err)
	}

	now := time.Now()
	return &TaskMessage{
		ID:         id,
		Type:       taskType,
		Payload:    payloadData,
		Status:     TaskStatusPending,
		RetryCount: 0,
		MaxRetries: 3, // 默认最大重试次数
		CreatedAt:  now,
		UpdatedAt:  now,
		Source:     source,
	}, nil
}

// PrepareRetry 准备重试
func (t *TaskMessage) PrepareRetry(errorMsg string) bool {
	t.RetryCount++
	t.Status = TaskStatusRetrying
	t.LastError = errorMsg
	t.UpdatedAt = time.Now()

	// 检查是否超过最大重试次数
	if t.RetryCount > t.MaxRetries {
		t.Status = TaskStatusFailed
		return false
	}

	// 计算下次重试时间 (使用指数退避策略)
	backoff := time.Duration(1<<uint(t.RetryCount-1)) * time.Minute
	// 最大退避时间为1小时
	if backoff > time.Hour {
		backoff = time.Hour
	}
	nextRetry := time.Now().Add(backoff)
	t.NextRetryAt = &nextRetry

	return true
}

// MarkAsCompleted 标记为已完成
func (t *TaskMessage) MarkAsCompleted() {
	t.Status = TaskStatusCompleted
	t.UpdatedAt = time.Now()
	t.NextRetryAt = nil
}

// MarkAsFailed 标记为失败
func (t *TaskMessage) MarkAsFailed(errorMsg string) {
	t.Status = TaskStatusFailed
	t.LastError = errorMsg
	t.UpdatedAt = time.Now()
	t.NextRetryAt = nil
}

// MarkAsProcessing 标记为处理中
func (t *TaskMessage) MarkAsProcessing() {
	t.Status = TaskStatusProcessing
	t.UpdatedAt = time.Now()
}

// ShouldRetryNow 检查是否应该立即重试
func (t *TaskMessage) ShouldRetryNow() bool {
	if t.Status != TaskStatusRetrying {
		return false
	}

	if t.NextRetryAt == nil {
		return true
	}

	return time.Now().After(*t.NextRetryAt)
}
