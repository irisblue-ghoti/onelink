package entities

import (
	"time"

	"github.com/google/uuid"
)

// 商户状态枚举
type MerchantStatus string

const (
	MerchantStatusActive    MerchantStatus = "active"
	MerchantStatusInactive  MerchantStatus = "inactive"
	MerchantStatusSuspended MerchantStatus = "suspended"
)

// Merchant 商户实体
type Merchant struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	Name           string                 `json:"name" db:"name"`
	Description    string                 `json:"description" db:"description"`
	Email          string                 `json:"email" db:"email"`
	Phone          string                 `json:"phone" db:"phone"`
	LogoURL        string                 `json:"logoUrl" db:"logo_url"`
	Website        string                 `json:"website" db:"website"`
	Address        string                 `json:"address" db:"address"`
	Status         MerchantStatus         `json:"status" db:"status"`
	ApprovalStatus MerchantApprovalStatus `json:"approvalStatus" db:"approval_status"`
	ApiKey         string                 `json:"apiKey,omitempty" db:"api_key"`
	PlanID         uuid.UUID              `json:"planId" db:"plan_id"`
	CreatedAt      time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time              `json:"updatedAt" db:"updated_at"`
}

// CreateMerchantDTO 创建商户的数据传输对象
type CreateMerchantDTO struct {
	Name        string         `json:"name" binding:"required" db:"name"`
	Description string         `json:"description" db:"description"`
	Email       string         `json:"email" binding:"required,email" db:"email"`
	Phone       string         `json:"phone" db:"phone"`
	LogoURL     string         `json:"logoUrl" db:"logo_url"`
	Website     string         `json:"website" db:"website"`
	Address     string         `json:"address" db:"address"`
	Status      MerchantStatus `json:"status" binding:"required" db:"status"`
	PlanID      uuid.UUID      `json:"planId" binding:"required" db:"plan_id"`
}

// UpdateMerchantDTO 更新商户的数据传输对象
type UpdateMerchantDTO struct {
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`
	Email       string         `json:"email" binding:"omitempty,email" db:"email"`
	Phone       string         `json:"phone" db:"phone"`
	LogoURL     string         `json:"logoUrl" db:"logo_url"`
	Website     string         `json:"website" db:"website"`
	Address     string         `json:"address" db:"address"`
	Status      MerchantStatus `json:"status" db:"status"`
	PlanID      uuid.UUID      `json:"planId" db:"plan_id"`
}

// PaginationParams 分页参数
type PaginationParams struct {
	Page  int `form:"page" json:"page" db:"page"`
	Limit int `form:"limit" json:"limit" db:"limit"`
}

// PaginationMeta 分页元数据
type PaginationMeta struct {
	CurrentPage  int `json:"currentPage" db:"current_page"`
	ItemsPerPage int `json:"itemsPerPage" db:"items_per_page"`
	TotalItems   int `json:"totalItems" db:"total_items"`
	TotalPages   int `json:"totalPages" db:"total_pages"`
}

// PaginatedResponse 分页响应
type PaginatedResponse struct {
	Data []Merchant     `json:"data" db:"data"`
	Meta PaginationMeta `json:"meta" db:"meta"`
}
