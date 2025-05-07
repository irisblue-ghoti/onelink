package cards

import (
	"nfc-service/internal/domain/repositories"
	"nfc-service/internal/storage"
)

// CardRepositoryAdapter 适配器将storage.CardRepository转换为repositories.NfcCardRepository
type CardRepositoryAdapter struct {
	repo *storage.CardRepository
}

// NewCardRepositoryAdapter 创建新的适配器
func NewCardRepositoryAdapter(repo *storage.CardRepository) *repositories.NfcCardRepository {
	// 直接使用DB字段创建领域仓库实例
	return repositories.NewNfcCardRepository(repo.DB)
}
