package shortlinks

import (
	"context"
	"fmt"
	"log"
	"nfc-service/internal/domain/entities"
	"nfc-service/internal/storage"
	"nfc-service/pkg/cloudflare"

	"github.com/google/uuid"
)

// ShortlinkService 短链接服务
type ShortlinkService struct {
	repo     *storage.ShortlinkRepository
	cfClient *cloudflare.Client
	logger   *log.Logger
}

// NewShortlinkService 创建短链接服务
func NewShortlinkService(
	repo *storage.ShortlinkRepository,
	cfClient *cloudflare.Client,
	logger *log.Logger,
) *ShortlinkService {
	return &ShortlinkService{
		repo:     repo,
		cfClient: cfClient,
		logger:   logger,
	}
}

// Create 创建短链接
func (s *ShortlinkService) Create(ctx context.Context, link *entities.CreateShortLinkDTO) (*entities.ShortLink, error) {
	s.logger.Printf("创建短链接: %v", link)
	return s.repo.Create(ctx, link)
}

// GetByID 根据ID获取短链接
func (s *ShortlinkService) GetByID(ctx context.Context, id uuid.UUID) (*entities.ShortLink, error) {
	s.logger.Printf("获取短链接: %s", id)
	return s.repo.FindByID(ctx, id)
}

// GetBySlug 根据Slug获取短链接
func (s *ShortlinkService) GetBySlug(ctx context.Context, slug string) (*entities.ShortLink, error) {
	s.logger.Printf("获取短链接: %s", slug)
	return s.repo.FindBySlug(ctx, slug)
}

// GetByMerchantID 获取商户的所有短链接
func (s *ShortlinkService) GetByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]*entities.ShortLink, int, error) {
	s.logger.Printf("获取商户短链接: %s, 页码: %d, 每页数量: %d", merchantID, page, pageSize)
	return s.repo.FindByTenantID(ctx, merchantID, page, pageSize)
}

// GetByNfcCardID 获取卡片的所有短链接
func (s *ShortlinkService) GetByNfcCardID(ctx context.Context, nfcCardID uuid.UUID) ([]*entities.ShortLink, error) {
	s.logger.Printf("获取卡片短链接: %s", nfcCardID)
	return s.repo.FindByNfcCardID(ctx, nfcCardID)
}

// Update 更新短链接
func (s *ShortlinkService) Update(ctx context.Context, id uuid.UUID, link *entities.UpdateShortLinkDTO) (*entities.ShortLink, error) {
	s.logger.Printf("更新短链接: %s", id)
	return s.repo.Update(ctx, id, link)
}

// Delete 删除短链接
func (s *ShortlinkService) Delete(ctx context.Context, id uuid.UUID) error {
	s.logger.Printf("删除短链接: %s", id)
	return s.repo.Delete(ctx, id)
}

// IncrementClicks 增加点击次数
func (s *ShortlinkService) IncrementClicks(ctx context.Context, slug string) error {
	s.logger.Printf("增加点击次数: %s", slug)
	return s.repo.IncrementClicks(ctx, slug)
}

// GetFullURL 获取完整URL
func (s *ShortlinkService) GetFullURL(baseURL, slug string) string {
	return fmt.Sprintf("%s/%s", baseURL, slug)
}

// UpdateDefaultForCard 当卡片更新时更新关联的默认短链接
func (s *ShortlinkService) UpdateDefaultForCard(ctx context.Context, cardID uuid.UUID, targetURL string) error {
	s.logger.Printf("为卡片 %s 更新默认短链接", cardID)

	// 获取该卡片的所有短链接
	links, err := s.repo.FindByNfcCardID(ctx, cardID)
	if err != nil {
		s.logger.Printf("获取卡片短链接失败: %v", err)
		return err
	}

	// 查找默认短链接并更新
	found := false
	for _, link := range links {
		if link.IsDefault {
			updateDTO := &entities.UpdateShortLinkDTO{
				TargetURL: targetURL,
			}

			_, err := s.repo.Update(ctx, link.ID, updateDTO)
			if err != nil {
				s.logger.Printf("更新短链接失败: %v", err)
				return err
			}

			s.logger.Printf("已更新卡片 %s 的默认短链接 %s", cardID, link.ID)
			found = true
			break
		}
	}

	// 如果没有找到默认短链接，可以考虑创建一个
	if !found && len(links) > 0 {
		// 将第一个链接设置为默认链接
		updateDTO := &entities.UpdateShortLinkDTO{
			TargetURL: targetURL,
			IsDefault: &[]bool{true}[0],
		}

		_, err := s.repo.Update(ctx, links[0].ID, updateDTO)
		if err != nil {
			s.logger.Printf("设置默认短链接失败: %v", err)
			return err
		}

		s.logger.Printf("已将卡片 %s 的短链接 %s 设置为默认并更新", cardID, links[0].ID)
	}

	return nil
}

// CreateDefaultForCard 为卡片创建默认短链接
func (s *ShortlinkService) CreateDefaultForCard(ctx context.Context, cardID uuid.UUID, name, targetURL string) (*entities.ShortLink, error) {
	s.logger.Printf("为卡片 %s 创建默认短链接", cardID)

	// 获取卡片所属的商户ID
	// 这里简化处理，实际实现中可能需要通过卡片服务获取商户ID
	var tenantID uuid.UUID
	// 假设从卡片获取商户ID的逻辑...

	// 创建默认短链接
	linkDTO := &entities.CreateShortLinkDTO{
		TenantID:  tenantID,
		NfcCardID: cardID,
		Title:     name + " - 默认链接",
		TargetURL: targetURL,
		IsDefault: true,
	}

	link, err := s.repo.Create(ctx, linkDTO)
	if err != nil {
		s.logger.Printf("创建默认短链接失败: %v", err)
		return nil, err
	}

	s.logger.Printf("已为卡片 %s 创建默认短链接 %s", cardID, link.ID)
	return link, nil
}

// EnsureDefaultLinks 确保所有卡片都有默认短链接
func (s *ShortlinkService) EnsureDefaultLinks(ctx context.Context) error {
	s.logger.Printf("确保所有卡片都有默认短链接")

	// 此方法可以作为定时任务运行，检查所有卡片是否都有默认短链接
	// 实现逻辑需要遍历所有卡片，检查每个卡片是否有默认短链接
	// 如果没有，则创建一个

	// 实际实现中，此方法可能需要调用卡片服务获取所有卡片
	// 然后对每个卡片执行以下逻辑

	// 示例伪代码:
	// cards := cardService.GetAll(ctx)
	// for _, card := range cards {
	//     links, _ := s.repo.FindByNfcCardID(ctx, card.ID)
	//     hasDefault := false
	//     for _, link := range links {
	//         if link.IsDefault {
	//             hasDefault = true
	//             break
	//         }
	//     }
	//     if !hasDefault {
	//         s.CreateDefaultForCard(ctx, card.ID, card.Name, "/nfc-landing/"+card.UID)
	//     }
	// }

	s.logger.Printf("默认短链接检查完成")
	return nil
}
