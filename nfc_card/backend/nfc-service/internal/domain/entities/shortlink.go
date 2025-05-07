package entities

import (
	"time"

	"github.com/google/uuid"
)

// ShortLink 表示短链接实体
type ShortLink struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	TenantID  uuid.UUID  `json:"tenantId" db:"merchant_id"`
	NfcCardID uuid.UUID  `json:"nfcCardId" db:"nfc_card_id"`
	Title     string     `json:"title" db:"title"`
	Slug      string     `json:"slug" db:"slug"`
	TargetURL string     `json:"targetUrl" db:"target_url"`
	Clicks    int        `json:"clicks" db:"clicks"`
	Active    bool       `json:"active" db:"active"`
	IsDefault bool       `json:"isDefault" db:"is_default"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	ExpiresAt *time.Time `json:"expiresAt" db:"expires_at"`
}

// CreateShortLinkDTO 创建短链接的数据传输对象
type CreateShortLinkDTO struct {
	TenantID  uuid.UUID  `json:"tenantId" binding:"required" db:"merchant_id"`
	NfcCardID uuid.UUID  `json:"nfcCardId" binding:"required" db:"nfc_card_id"`
	Title     string     `json:"title" binding:"required" db:"title"`
	Slug      string     `json:"slug" db:"slug"`
	TargetURL string     `json:"targetUrl" binding:"required,url" db:"target_url"`
	IsDefault bool       `json:"isDefault" db:"is_default"`
	ExpiresAt *time.Time `json:"expiresAt" db:"expires_at"`
}

// UpdateShortLinkDTO 更新短链接的数据传输对象
type UpdateShortLinkDTO struct {
	Title     string     `json:"title" binding:"omitempty" db:"title"`
	TargetURL string     `json:"targetUrl" binding:"omitempty,url" db:"target_url"`
	Active    *bool      `json:"active,omitempty" db:"active"`
	IsDefault *bool      `json:"isDefault,omitempty" db:"is_default"`
	ExpiresAt *time.Time `json:"expiresAt" db:"expires_at"`
}

// IncrementClicksDTO 增加点击次数的数据传输对象
type IncrementClicksDTO struct {
	Slug string `json:"slug" binding:"required" db:"slug"`
}
