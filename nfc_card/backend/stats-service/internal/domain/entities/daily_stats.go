package entities

import (
	"time"

	"github.com/google/uuid"
)

// DailyStats 每日统计数据实体
type DailyStats struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TenantID     uuid.UUID `json:"tenantId" db:"merchant_id"`
	VideoID      uuid.UUID `json:"videoId" db:"video_id"`
	NfcCardID    uuid.UUID `json:"nfcCardId" db:"nfc_card_id"`
	Platform     string    `json:"platform" db:"platform"`     // 平台名称：douyin, kuaishou, xiaohongshu, wechat, all
	Date         time.Time `json:"date" db:"date"`             // 统计日期
	ViewCount    int64     `json:"viewCount" db:"views"`       // 当日播放/观看次数
	LikeCount    int64     `json:"likeCount" db:"likes"`       // 当日点赞数
	CommentCount int64     `json:"commentCount" db:"comments"` // 当日评论数
	ShareCount   int64     `json:"shareCount" db:"shares"`     // 当日分享数
	CollectCount int64     `json:"collectCount" db:"collects"` // 当日收藏数
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`  // 创建时间
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`  // 更新时间
}

// NewDailyStats 创建新的每日统计数据
func NewDailyStats(tenantID, videoID, nfcCardID uuid.UUID, platform string, date time.Time) *DailyStats {
	now := time.Now()
	return &DailyStats{
		ID:        uuid.New(),
		TenantID:  tenantID,
		VideoID:   videoID,
		NfcCardID: nfcCardID,
		Platform:  platform,
		Date:      date,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
