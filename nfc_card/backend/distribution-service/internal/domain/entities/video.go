package entities

import (
	"time"

	"github.com/google/uuid"
)

// Video 视频实体
type Video struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TenantID        uuid.UUID `json:"tenantId" db:"merchant_id"`
	Title           string    `json:"title" db:"title"`
	Description     string    `json:"description" db:"description"`
	FileName        string    `json:"fileName" db:"file_name,omitempty"`
	FileKey         string    `json:"fileKey" db:"file_key,omitempty"`
	FileType        string    `json:"fileType" db:"file_type,omitempty"`
	Size            int64     `json:"size" db:"size,omitempty"`
	Duration        float64   `json:"duration" db:"duration"`
	Width           int       `json:"width" db:"width,omitempty"`
	Height          int       `json:"height" db:"height,omitempty"`
	IsTranscoded    bool      `json:"isTranscoded" db:"is_transcoded,omitempty"`
	TranscodeStatus string    `json:"transcodeStatus" db:"transcode_status,omitempty"`
	StoragePath     string    `json:"storagePath" db:"storage_path"`
	CoverURL        string    `json:"coverUrl" db:"cover_url"`
	IsPublic        bool      `json:"isPublic" db:"is_public"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time `json:"updatedAt" db:"updated_at"`
}
