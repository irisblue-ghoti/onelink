package repositories

import (
	"merchant-service/internal/domain/entities"

	"github.com/google/uuid"
)

// MerchantRepository 商户仓库接口
type MerchantRepository interface {
	// Create 创建商户
	Create(merchant entities.Merchant) (entities.Merchant, error)

	// FindByID 通过ID查找商户
	FindByID(id string) (entities.Merchant, error)

	// FindAll 查找所有商户（分页）
	FindAll(params entities.PaginationParams) ([]entities.Merchant, int, error)

	// Update 更新商户
	Update(merchant entities.Merchant) (entities.Merchant, error)

	// Delete 删除商户
	Delete(id string) error

	// UpdateApiKey 更新商户API密钥
	UpdateApiKey(id string, apiKey string) error

	// CountAll 获取商户总数
	CountAll() (int, error)

	// FindByEmail 通过邮箱查找商户
	FindByEmail(email string) (entities.Merchant, error)

	// FindByPlanID 通过套餐ID查找商户
	FindByPlanID(planID uuid.UUID) ([]entities.Merchant, error)
}
