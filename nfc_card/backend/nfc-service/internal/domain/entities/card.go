package entities

import (
	"time"

	"github.com/google/uuid"
)

// CardStatus 表示NFC卡片的状态
type CardStatus string

const (
	// CardStatusNew 新建卡片
	CardStatusNew CardStatus = "new"
	// CardStatusActivated 已激活卡片
	CardStatusActivated CardStatus = "activated"
	// CardStatusBound 已绑定卡片
	CardStatusBound CardStatus = "bound"
	// CardStatusDeactivated 已停用卡片
	CardStatusDeactivated CardStatus = "deactivated"
	// CardStatusExpired 已过期卡片
	CardStatusExpired CardStatus = "expired"
)

// NfcCard 表示NFC卡片实体
type NfcCard struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TenantID       uuid.UUID  `json:"tenantId" db:"merchant_id"`
	MerchantID     uuid.UUID  `json:"merchantId" db:"merchant_id"`
	UID            string     `json:"uid" db:"uid"`
	Name           string     `json:"name" db:"name"`
	Description    string     `json:"description,omitempty" db:"description"`
	DefaultVideoID *uuid.UUID `json:"defaultVideoId" db:"default_video_id"`
	Status         CardStatus `json:"status" db:"status"`
	UserID         *uuid.UUID `json:"userId" db:"user_id"`
	ActivatedAt    *time.Time `json:"activatedAt" db:"activated_at"`
	BoundAt        *time.Time `json:"boundAt" db:"bound_at"`
	DeactivatedAt  *time.Time `json:"deactivatedAt" db:"deactivated_at"`
	ExpiresAt      *time.Time `json:"expiresAt" db:"expires_at"`
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time  `json:"updatedAt" db:"updated_at"`
}

// CreateNfcCardDTO 创建NFC卡片的数据传输对象
type CreateNfcCardDTO struct {
	TenantID       uuid.UUID  `json:"tenantId" binding:"required" db:"merchant_id"`
	MerchantID     uuid.UUID  `json:"merchantId" db:"merchant_id"`
	UID            string     `json:"uid" binding:"required" db:"uid"`
	Name           string     `json:"name" binding:"required" db:"name"`
	Description    string     `json:"description,omitempty" db:"description"`
	DefaultVideoID *uuid.UUID `json:"defaultVideoId" db:"default_video_id"`
	Status         CardStatus `json:"status" db:"status"`
	ExpiresAt      *time.Time `json:"expiresAt" db:"expires_at"`
}

// UpdateNfcCardDTO 更新NFC卡片的数据传输对象
type UpdateNfcCardDTO struct {
	Name           string     `json:"name,omitempty" db:"name"`
	Description    string     `json:"description,omitempty" db:"description"`
	DefaultVideoID *uuid.UUID `json:"defaultVideoId" db:"default_video_id"`
	Status         CardStatus `json:"status,omitempty" db:"status"`
	UserID         *uuid.UUID `json:"userId,omitempty" db:"user_id"`
	ActivatedAt    *time.Time `json:"activatedAt" db:"activated_at"`
	BoundAt        *time.Time `json:"boundAt,omitempty" db:"bound_at"`
	DeactivatedAt  *time.Time `json:"deactivatedAt,omitempty" db:"deactivated_at"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty" db:"expires_at"`
}

// ActivateNfcCardDTO 激活NFC卡片的数据传输对象
type ActivateNfcCardDTO struct {
	UID string `json:"uid" binding:"required" db:"uid"`
}

// BindNfcCardDTO 绑定NFC卡片的数据传输对象
type BindNfcCardDTO struct {
	ID     uuid.UUID `json:"id" binding:"required" db:"id"`
	UserID uuid.UUID `json:"userId" binding:"required" db:"user_id"`
}

// UnbindNfcCardDTO 解绑NFC卡片的数据传输对象
type UnbindNfcCardDTO struct {
	ID uuid.UUID `json:"id" binding:"required" db:"id"`
}

// DeactivateNfcCardDTO 停用NFC卡片的数据传输对象
type DeactivateNfcCardDTO struct {
	ID uuid.UUID `json:"id" binding:"required" db:"id"`
}
