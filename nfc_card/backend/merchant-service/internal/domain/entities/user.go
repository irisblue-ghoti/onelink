package entities

import (
	"time"
)

// User 用户实体
type User struct {
	ID         string    `json:"id" db:"id"`
	MerchantID string    `json:"merchant_id" db:"merchant_id"`
	Name       string    `json:"name" db:"name"`
	Email      string    `json:"email" db:"email"`
	Password   string    `json:"-" db:"password_hash"` // 密码不返回给客户端
	Role       string    `json:"role" db:"role"`
	Status     string    `json:"status" db:"status"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// CreateUserDTO 创建用户DTO
type CreateUserDTO struct {
	MerchantID string `json:"merchant_id"`
	Name       string `json:"name" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=6"`
	Role       string `json:"role" binding:"required"`
}

// UpdateUserDTO 更新用户DTO
type UpdateUserDTO struct {
	Name     string `json:"name"`
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"omitempty,min=6"`
	Status   string `json:"status"`
}
