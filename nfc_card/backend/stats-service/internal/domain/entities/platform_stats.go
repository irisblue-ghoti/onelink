package entities

import (
	"time"

	"github.com/google/uuid"
)

// PlatformStats 平台统计数据实体
type PlatformStats struct {
	ID            uuid.UUID              `json:"id" db:"I_d"`
	TenantID      uuid.UUID              `json:"tenantId" db:"Tenant_i_d"`
	VideoID       uuid.UUID              `json:"videoId" db:"Video_i_d"`
	NfcCardID     uuid.UUID              `json:"nfcCardId" db:"Nfc_card_i_d"`
	Platform      string                 `json:"platform" db:"Platform"`             // 平台名称：douyin, kuaishou, xiaohongshu, wechat
	PlatformID    uuid.UUID              `json:"platformId" db:"Platform_i_d"`       // 平台视频ID
	ViewCount     int64                  `json:"viewCount" db:"View_count"`          // 播放/观看次数
	LikeCount     int64                  `json:"likeCount" db:"Like_count"`          // 点赞数
	CommentCount  int64                  `json:"commentCount" db:"Comment_count"`    // 评论数
	ShareCount    int64                  `json:"shareCount" db:"Share_count"`        // 分享数
	CollectCount  int64                  `json:"collectCount" db:"Collect_count"`    // 收藏数
	RawData       map[string]interface{} `json:"rawData" db:"Raw_data"`              // 原始数据
	LastUpdatedAt time.Time              `json:"lastUpdatedAt" db:"Last_updated_at"` // 最后更新时间
	CreatedAt     time.Time              `json:"createdAt" db:"Created_at"`          // 创建时间
}

// NewPlatformStats 创建新的平台统计数据
func NewPlatformStats(tenantID, videoID, nfcCardID uuid.UUID, platform string, platformID string) *PlatformStats {
	now := time.Now()

	// 尝试解析平台ID
	platformUUID, err := uuid.Parse(platformID)
	if err != nil {
		// 如果无法解析，则生成一个基于字符串的UUID
		platformUUID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(platformID))
	}

	return &PlatformStats{
		ID:            uuid.New(),
		TenantID:      tenantID,
		VideoID:       videoID,
		NfcCardID:     nfcCardID,
		Platform:      platform,
		PlatformID:    platformUUID,
		RawData:       make(map[string]interface{}),
		LastUpdatedAt: now,
		CreatedAt:     now,
	}
}
