package shortlinks

import (
	"context"
	"nfc-service/internal/domain/entities"

	"github.com/google/uuid"
)

// Service 短链接服务接口
type Service interface {
	Create(ctx context.Context, link *entities.CreateShortLinkDTO) (*entities.ShortLink, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.ShortLink, error)
	GetBySlug(ctx context.Context, slug string) (*entities.ShortLink, error)
	GetByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]*entities.ShortLink, int, error)
	GetByNfcCardID(ctx context.Context, nfcCardID uuid.UUID) ([]*entities.ShortLink, error)
	Update(ctx context.Context, id uuid.UUID, link *entities.UpdateShortLinkDTO) (*entities.ShortLink, error)
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementClicks(ctx context.Context, slug string) error
	GetFullURL(baseURL, slug string) string
	UpdateDefaultForCard(ctx context.Context, cardID uuid.UUID, targetURL string) error
	CreateDefaultForCard(ctx context.Context, cardID uuid.UUID, name, targetURL string) (*entities.ShortLink, error)
	EnsureDefaultLinks(ctx context.Context) error
}
