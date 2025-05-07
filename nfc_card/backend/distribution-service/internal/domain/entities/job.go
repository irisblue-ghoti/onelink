package entities

import (
	"time"

	"github.com/google/uuid"
)

// PublishJob 分发任务实体
type PublishJob struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	TenantID    uuid.UUID              `json:"tenantId" db:"merchant_id"`
	VideoID     uuid.UUID              `json:"videoId" db:"video_id"`
	NfcCardID   uuid.UUID              `json:"nfcCardId" db:"nfc_card_id"`
	Channel     string                 `json:"channel" db:"channel"` // 'douyin', 'kuaishou', 'xiaohongshu', 'wechat'
	Status      string                 `json:"status" db:"status"`   // 'pending', 'processing', 'completed', 'failed', 'retrying'
	Result      map[string]interface{} `json:"result" db:"result"`
	Params      map[string]interface{} `json:"params,omitempty" db:"-"` // 发布参数，如标签、@用户等
	ErrorMsg    string                 `json:"errorMsg" db:"error_message"`
	CreatedAt   time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time              `json:"updatedAt" db:"updated_at"`
	CompletedAt time.Time              `json:"completedAt,omitempty" db:"completed_at"`
	// 重试相关字段
	RetryCount  int        `json:"retryCount" db:"retry_count"`    // 当前重试次数
	MaxRetries  int        `json:"maxRetries" db:"max_retries"`    // 最大重试次数
	NextRetryAt *time.Time `json:"nextRetryAt" db:"next_retry_at"` // 下次重试时间
	LastError   string     `json:"lastError" db:"last_error"`      // 最后一次错误
}

// NewPublishJob 创建新的分发任务
func NewPublishJob(tenantID, videoID, nfcCardID uuid.UUID, channel string) *PublishJob {
	now := time.Now()
	return &PublishJob{
		ID:         uuid.New(),
		TenantID:   tenantID,
		VideoID:    videoID,
		NfcCardID:  nfcCardID,
		Channel:    channel,
		Status:     "pending",
		Result:     make(map[string]interface{}),
		Params:     make(map[string]interface{}),
		CreatedAt:  now,
		UpdatedAt:  now,
		RetryCount: 0,
		MaxRetries: 3,
	}
}
