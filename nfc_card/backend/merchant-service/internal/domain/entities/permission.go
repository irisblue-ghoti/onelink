package entities

import (
	"time"

	"github.com/google/uuid"
)

// Permission 权限实体
type Permission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Resource    string    `json:"resource" db:"resource"`
	Action      string    `json:"action" db:"action"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// Role 角色实体
type Role struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// RolePermission 角色权限关联实体
type RolePermission struct {
	RoleID       uuid.UUID `json:"roleId" db:"role_id"`
	PermissionID uuid.UUID `json:"permissionId" db:"permission_id"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
}

// UserRole 用户角色关联实体
type UserRole struct {
	UserID    string    `json:"userId" db:"user_id"`
	RoleID    uuid.UUID `json:"roleId" db:"role_id"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// Department 部门实体
type Department struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	MerchantID  uuid.UUID  `json:"merchantId" db:"merchant_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	ParentID    *uuid.UUID `json:"parentId,omitempty" db:"parent_id"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`
}

// UserDepartment 用户部门关联实体
type UserDepartment struct {
	UserID       string    `json:"userId" db:"user_id"`
	DepartmentID uuid.UUID `json:"departmentId" db:"department_id"`
	IsManager    bool      `json:"isManager" db:"is_manager"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

// CreateRoleDTO 创建角色DTO
type CreateRoleDTO struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateRoleDTO 更新角色DTO
type UpdateRoleDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AssignRolePermissionDTO 分配角色权限DTO
type AssignRolePermissionDTO struct {
	PermissionIDs []uuid.UUID `json:"permissionIds" binding:"required"`
}

// AssignUserRoleDTO 分配用户角色DTO
type AssignUserRoleDTO struct {
	RoleIDs []uuid.UUID `json:"roleIds" binding:"required"`
}

// CreateDepartmentDTO 创建部门DTO
type CreateDepartmentDTO struct {
	MerchantID  uuid.UUID  `json:"merchantId" binding:"required"`
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parentId"`
}

// UpdateDepartmentDTO 更新部门DTO
type UpdateDepartmentDTO struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parentId"`
}

// AssignUserDepartmentDTO 分配用户部门DTO
type AssignUserDepartmentDTO struct {
	DepartmentID uuid.UUID `json:"departmentId" binding:"required"`
	IsManager    bool      `json:"isManager"`
}
