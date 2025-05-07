package entities

import (
	"time"

	"github.com/google/uuid"
)

// 转码状态枚举
type TranscodeStatus string

const (
	TranscodeStatusPending    TranscodeStatus = "pending"
	TranscodeStatusProcessing TranscodeStatus = "processing"
	TranscodeStatusCompleted  TranscodeStatus = "completed"
	TranscodeStatusFailed     TranscodeStatus = "failed"
)

// Video 视频实体
type Video struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	TenantID        uuid.UUID       `json:"tenantId" db:"merchant_id"`
	Title           string          `json:"title" db:"title"`
	Description     string          `json:"description" db:"description"`
	FileName        string          `json:"fileName" db:"file_name,omitempty"`
	FileKey         string          `json:"fileKey" db:"file_key,omitempty"`
	CoverKey        string          `json:"coverKey" db:"cover_key"`
	FileType        string          `json:"fileType" db:"file_type,omitempty"`
	Size            int64           `json:"size" db:"size,omitempty"`
	Duration        float64         `json:"duration" db:"duration"`
	Width           int             `json:"width" db:"width,omitempty"`
	Height          int             `json:"height" db:"height,omitempty"`
	IsTranscoded    bool            `json:"isTranscoded" db:"is_transcoded,omitempty"`
	TranscodeStatus TranscodeStatus `json:"transcodeStatus" db:"transcode_status,omitempty"`
	StoragePath     string          `json:"storagePath" db:"storage_path"`
	CoverURL        string          `json:"coverUrl" db:"cover_url"`
	IsPublic        bool            `json:"isPublic" db:"is_public"`
	CreatedAt       time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time       `json:"updatedAt" db:"updated_at"`
}

// CreateVideoDTO 创建视频的数据传输对象
type CreateVideoDTO struct {
	Title       string `json:"title" binding:"required" db:"title"`
	Description string `json:"description" db:"description"`
}

// VideoResponse 视频响应对象
type VideoResponse struct {
	ID           uuid.UUID `json:"id" db:"I_d"`
	Title        string    `json:"title" db:"Title"`
	Description  string    `json:"description,omitempty" db:"Description"`
	URL          string    `json:"url" db:"U_r_l"`
	CoverURL     string    `json:"coverUrl,omitempty" db:"Cover_u_r_l"`
	Duration     float64   `json:"duration,omitempty" db:"Duration"`
	Width        int       `json:"width,omitempty" db:"Width"`
	Height       int       `json:"height,omitempty" db:"Height"`
	Size         int64     `json:"size,omitempty" db:"Size"`
	IsTranscoded bool      `json:"isTranscoded" db:"Is_transcoded"`
	CreatedAt    time.Time `json:"createdAt" db:"Created_at"`
}

// DetailedVideoResponse 详细视频响应对象
type DetailedVideoResponse struct {
	VideoResponse
	TranscodeStatus TranscodeStatus `json:"transcodeStatus" db:"Transcode_status"`
	UpdatedAt       time.Time       `json:"updatedAt" db:"Updated_at"`
}
