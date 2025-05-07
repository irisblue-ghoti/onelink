package services

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"merchant-service/internal/domain/entities"
	"merchant-service/internal/domain/repositories"

	"github.com/google/uuid"
)

// MerchantService 商户服务
type MerchantService struct {
	repo   repositories.MerchantRepository
	logger *log.Logger
}

// NewMerchantService 创建商户服务
func NewMerchantService(repo repositories.MerchantRepository, logger *log.Logger) *MerchantService {
	return &MerchantService{
		repo:   repo,
		logger: logger,
	}
}

// Create 创建新商户
func (s *MerchantService) Create(dto entities.CreateMerchantDTO) (entities.Merchant, error) {
	merchant := entities.Merchant{
		ID:          uuid.New(),
		Name:        dto.Name,
		Description: dto.Description,
		Email:       dto.Email,
		Phone:       dto.Phone,
		LogoURL:     dto.LogoURL,
		Website:     dto.Website,
		Address:     dto.Address,
		Status:      dto.Status,
		PlanID:      dto.PlanID,
		ApiKey:      generateApiKey(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return s.repo.Create(merchant)
}

// FindAll 获取所有商户（分页）
func (s *MerchantService) FindAll(params entities.PaginationParams) ([]entities.Merchant, int, error) {
	return s.repo.FindAll(params)
}

// FindOne 获取单个商户
func (s *MerchantService) FindOne(id string) (entities.Merchant, error) {
	return s.repo.FindByID(id)
}

// Update 更新商户
func (s *MerchantService) Update(id string, dto entities.UpdateMerchantDTO) (entities.Merchant, error) {
	merchant, err := s.FindOne(id)
	if err != nil {
		return entities.Merchant{}, err
	}

	// 更新字段
	if dto.Name != "" {
		merchant.Name = dto.Name
	}
	if dto.Description != "" {
		merchant.Description = dto.Description
	}
	if dto.Email != "" {
		merchant.Email = dto.Email
	}
	if dto.Phone != "" {
		merchant.Phone = dto.Phone
	}
	if dto.LogoURL != "" {
		merchant.LogoURL = dto.LogoURL
	}
	if dto.Website != "" {
		merchant.Website = dto.Website
	}
	if dto.Address != "" {
		merchant.Address = dto.Address
	}
	if dto.Status != "" {
		merchant.Status = dto.Status
	}
	if dto.PlanID != uuid.Nil {
		merchant.PlanID = dto.PlanID
	}

	merchant.UpdatedAt = time.Now()

	return s.repo.Update(merchant)
}

// Remove 删除商户
func (s *MerchantService) Remove(id string) error {
	return s.repo.Delete(id)
}

// RegenerateApiKey 重新生成API Key
func (s *MerchantService) RegenerateApiKey(id string) (string, error) {
	// 检查商户是否存在
	_, err := s.FindOne(id)
	if err != nil {
		return "", err
	}

	// 生成新的API Key
	apiKey := generateApiKey()

	// 更新商户API Key
	err = s.repo.UpdateApiKey(id, apiKey)
	if err != nil {
		return "", err
	}

	return apiKey, nil
}

// 生成API Key
func generateApiKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// 如果发生错误，回退到不太安全但可用的方法
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(bytes)
}

// UpdateApproval 更新商户审核状态
func (s *MerchantService) UpdateApproval(id string, dto entities.UpdateMerchantApprovalDTO, reviewerID string) (entities.Merchant, error) {
	// 查找商户
	merchant, err := s.FindOne(id)
	if err != nil {
		return entities.Merchant{}, err
	}

	// 转换UUID
	reviewerUUID, err := uuid.Parse(reviewerID)
	if err != nil {
		return entities.Merchant{}, err
	}

	// 创建审核记录和历史
	// 由于目前没有实现完整的审核存储逻辑，先记录在日志中
	s.logger.Printf("创建审核记录: 商户ID=%s, 状态=%s, 审核人=%s",
		merchant.ID, dto.Status, reviewerUUID)
	s.logger.Printf("记录审核历史: 商户ID=%s, 状态从%s变更为%s, 审核人=%s",
		merchant.ID, merchant.ApprovalStatus, dto.Status, reviewerUUID)

	// 更新商户状态
	merchant.ApprovalStatus = dto.Status
	merchant.UpdatedAt = time.Now()

	// TODO: 这里应该使用事务来确保所有操作原子性
	// 1. 保存审核记录
	// 2. 保存审核历史
	// 3. 更新商户状态

	// 目前临时只更新商户状态
	updatedMerchant, err := s.repo.Update(merchant)
	if err != nil {
		return entities.Merchant{}, err
	}

	// 如果状态变更为已批准，并且商户状态为非活跃，则自动激活商户
	if dto.Status == entities.MerchantApprovalApproved && merchant.Status != entities.MerchantStatusActive {
		updatedMerchant.Status = entities.MerchantStatusActive
		return s.repo.Update(updatedMerchant)
	}

	return updatedMerchant, nil
}
