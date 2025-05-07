package entities

import (
	"time"

	"github.com/google/uuid"
)

// MerchantApprovalStatus 商户审核状态
type MerchantApprovalStatus string

const (
	// MerchantApprovalPending 待审核
	MerchantApprovalPending MerchantApprovalStatus = "pending"
	// MerchantApprovalApproved 审核通过
	MerchantApprovalApproved MerchantApprovalStatus = "approved"
	// MerchantApprovalRejected 审核拒绝
	MerchantApprovalRejected MerchantApprovalStatus = "rejected"
	// MerchantApprovalSuspended 暂停
	MerchantApprovalSuspended MerchantApprovalStatus = "suspended"
)

// MerchantApproval 商户审核实体
type MerchantApproval struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	MerchantID uuid.UUID              `json:"merchantId" db:"merchant_id"`
	Status     MerchantApprovalStatus `json:"status" db:"status"`
	ReviewerID *uuid.UUID             `json:"reviewerId,omitempty" db:"reviewer_id"`
	Comments   string                 `json:"comments" db:"comments"`
	CreatedAt  time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time              `json:"updatedAt" db:"updated_at"`
}

// MerchantApprovalHistory 商户审核历史实体
type MerchantApprovalHistory struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	MerchantID uuid.UUID              `json:"merchantId" db:"merchant_id"`
	StatusFrom MerchantApprovalStatus `json:"statusFrom" db:"status_from"`
	StatusTo   MerchantApprovalStatus `json:"statusTo" db:"status_to"`
	ReviewerID *uuid.UUID             `json:"reviewerId,omitempty" db:"reviewer_id"`
	Comments   string                 `json:"comments" db:"comments"`
	CreatedAt  time.Time              `json:"createdAt" db:"created_at"`
}

// UpdateMerchantApprovalDTO 更新商户审核DTO
type UpdateMerchantApprovalDTO struct {
	Status   MerchantApprovalStatus `json:"status" binding:"required"`
	Comments string                 `json:"comments"`
}

// 更新Merchant实体，增加审核状态
func init() {
	// 扩展Merchant结构体，增加ApprovalStatus字段
	// 注意：这是一个虚拟操作，实际需要修改Merchant结构体
}
