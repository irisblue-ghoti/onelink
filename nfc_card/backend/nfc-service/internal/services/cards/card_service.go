package cards

import (
	"context"
	"log"
	"nfc-service/internal/domain/entities"
	"nfc-service/internal/domain/repositories"

	"github.com/google/uuid"
)

// 保留服务接口定义在service.go中，此文件实现该接口

// cardService 是NFC卡片服务的实现
type cardService struct {
	cardRepo *repositories.NfcCardRepository
	logger   *log.Logger
}

// NewCardService 创建新的NFC卡片服务
func NewCardService(cardRepo *repositories.NfcCardRepository, logger *log.Logger) Service {
	return &cardService{
		cardRepo: cardRepo,
		logger:   logger,
	}
}

// Create 创建新的NFC卡片
func (s *cardService) Create(ctx context.Context, card *entities.CreateNfcCardDTO) (*entities.NfcCard, error) {
	s.logger.Printf("创建NFC卡片: %+v", card)
	return s.cardRepo.Create(ctx, card)
}

// GetByID 通过ID获取NFC卡片
func (s *cardService) GetByID(ctx context.Context, id uuid.UUID) (*entities.NfcCard, error) {
	s.logger.Printf("获取NFC卡片，ID: %s", id)
	return s.cardRepo.FindByID(ctx, id)
}

// GetByUID 通过UID获取NFC卡片
func (s *cardService) GetByUID(ctx context.Context, uid string) (*entities.NfcCard, error) {
	s.logger.Printf("获取NFC卡片，UID: %s", uid)
	return s.cardRepo.FindByUID(ctx, uid)
}

// GetByMerchantID 获取商户的所有NFC卡片
func (s *cardService) GetByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]*entities.NfcCard, int, error) {
	s.logger.Printf("获取商户NFC卡片列表，商户ID: %s", merchantID)
	return s.cardRepo.FindByMerchantID(ctx, merchantID, page, pageSize)
}

// Update 更新NFC卡片
func (s *cardService) Update(ctx context.Context, id uuid.UUID, card *entities.UpdateNfcCardDTO) (*entities.NfcCard, error) {
	s.logger.Printf("更新NFC卡片，ID: %s", id)
	// 验证更新的卡片是否存在
	existingCard, err := s.cardRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existingCard == nil {
		return nil, repositories.ErrCardNotFound
	}

	return s.cardRepo.Update(ctx, id, card)
}

// Delete 删除NFC卡片
func (s *cardService) Delete(ctx context.Context, id uuid.UUID) error {
	s.logger.Printf("删除NFC卡片，ID: %s", id)
	return s.cardRepo.Delete(ctx, id)
}

// Activate 激活NFC卡片
func (s *cardService) Activate(ctx context.Context, uid string) (*entities.NfcCard, error) {
	s.logger.Printf("激活NFC卡片，UID: %s", uid)
	card, err := s.cardRepo.Activate(ctx, uid)
	if err != nil {
		return nil, err
	}

	// 生成激活事件，通知其他服务
	s.logger.Printf("NFC卡片已激活，触发后续流程")
	// TODO: 发送激活事件到消息队列

	return card, nil
}

// GetByUserID 获取用户的所有NFC卡片
func (s *cardService) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.NfcCard, error) {
	s.logger.Printf("获取用户NFC卡片列表，用户ID: %s", userID)
	return s.cardRepo.FindByUserID(ctx, userID)
}

// Bind 绑定NFC卡片到用户
func (s *cardService) Bind(ctx context.Context, bind *entities.BindNfcCardDTO) (*entities.NfcCard, error) {
	s.logger.Printf("绑定NFC卡片，ID: %s, 用户ID: %s", bind.ID, bind.UserID)
	card, err := s.cardRepo.BindToUser(ctx, bind.ID, bind.UserID)
	if err != nil {
		return nil, err
	}

	// 触发绑定事件
	s.logger.Printf("NFC卡片已绑定到用户，触发后续流程")
	// TODO: 发送绑定事件到消息队列

	return card, nil
}

// Unbind 解绑NFC卡片
func (s *cardService) Unbind(ctx context.Context, unbind *entities.UnbindNfcCardDTO) (*entities.NfcCard, error) {
	s.logger.Printf("解绑NFC卡片，ID: %s", unbind.ID)
	card, err := s.cardRepo.UnbindFromUser(ctx, unbind.ID)
	if err != nil {
		return nil, err
	}

	// 触发解绑事件
	s.logger.Printf("NFC卡片已解绑，触发后续流程")
	// TODO: 发送解绑事件到消息队列

	return card, nil
}

// Deactivate 停用NFC卡片
func (s *cardService) Deactivate(ctx context.Context, id uuid.UUID) (*entities.NfcCard, error) {
	s.logger.Printf("停用NFC卡片，ID: %s", id)
	card, err := s.cardRepo.Deactivate(ctx, id)
	if err != nil {
		return nil, err
	}

	// 触发停用事件
	s.logger.Printf("NFC卡片已停用，触发后续流程")
	// TODO: 发送停用事件到消息队列

	return card, nil
}

// Reactivate 重新激活NFC卡片
func (s *cardService) Reactivate(ctx context.Context, id uuid.UUID) (*entities.NfcCard, error) {
	s.logger.Printf("重新激活NFC卡片，ID: %s", id)
	card, err := s.cardRepo.Reactivate(ctx, id)
	if err != nil {
		return nil, err
	}

	// 触发重新激活事件
	s.logger.Printf("NFC卡片已重新激活，触发后续流程")
	// TODO: 发送重新激活事件到消息队列

	return card, nil
}

// UpdateStatus 更新NFC卡片状态
func (s *cardService) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.CardStatus) (*entities.NfcCard, error) {
	s.logger.Printf("更新NFC卡片状态，ID: %s, 状态: %s", id, status)
	return s.cardRepo.UpdateStatus(ctx, id, status)
}
