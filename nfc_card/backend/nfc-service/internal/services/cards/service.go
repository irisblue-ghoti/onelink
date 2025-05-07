package cards

import (
	"context"

	"nfc-service/internal/domain/entities"

	"github.com/google/uuid"
)

// Service 是NFC卡片服务的接口
type Service interface {
	Create(ctx context.Context, card *entities.CreateNfcCardDTO) (*entities.NfcCard, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.NfcCard, error)
	GetByUID(ctx context.Context, uid string) (*entities.NfcCard, error)
	GetByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]*entities.NfcCard, int, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.NfcCard, error)
	Update(ctx context.Context, id uuid.UUID, card *entities.UpdateNfcCardDTO) (*entities.NfcCard, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Activate(ctx context.Context, uid string) (*entities.NfcCard, error)
	Bind(ctx context.Context, bind *entities.BindNfcCardDTO) (*entities.NfcCard, error)
	Unbind(ctx context.Context, unbind *entities.UnbindNfcCardDTO) (*entities.NfcCard, error)
	Deactivate(ctx context.Context, id uuid.UUID) (*entities.NfcCard, error)
	Reactivate(ctx context.Context, id uuid.UUID) (*entities.NfcCard, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.CardStatus) (*entities.NfcCard, error)
}

// 移除原有重复的service实现，统一使用card_service.go中的实现
