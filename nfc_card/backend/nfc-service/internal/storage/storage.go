package storage

import (
	"context"
	"fmt"
	"nfc-service/internal/config"
	"nfc-service/internal/domain/entities"
	"nfc-service/internal/domain/repositories"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Repositories 所有仓库的集合
type Repositories struct {
	db                  *sqlx.DB
	CardRepository      *CardRepository
	ShortlinkRepository *ShortlinkRepository
}

// NewDBConnection 创建数据库连接
func NewDBConnection(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	return sqlx.Connect("postgres", psqlInfo)
}

// NewRepositories 创建存储库集合
func NewRepositories(db *sqlx.DB) *Repositories {
	return &Repositories{
		db:                  db,
		CardRepository:      NewCardRepository(db),
		ShortlinkRepository: NewShortlinkRepository(db),
	}
}

// Close 关闭数据库连接
func (r *Repositories) Close() error {
	return r.db.Close()
}

// CardRepository NFC卡片存储库
type CardRepository struct {
	DB *sqlx.DB
}

// NewCardRepository 创建卡片存储库
func NewCardRepository(db *sqlx.DB) *CardRepository {
	return &CardRepository{
		DB: db,
	}
}

// Create 创建新的NFC卡片
func (r *CardRepository) Create(ctx context.Context, card *entities.CreateNfcCardDTO) (*entities.NfcCard, error) {
	// 使用领域存储库的实现
	repoImpl := repositories.NewNfcCardRepository(r.DB)
	return repoImpl.Create(ctx, card)
}

// FindByID 通过ID获取NFC卡片
func (r *CardRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.NfcCard, error) {
	// 使用领域存储库的实现
	repoImpl := repositories.NewNfcCardRepository(r.DB)
	return repoImpl.FindByID(ctx, id)
}

// FindByUID 通过UID获取NFC卡片
func (r *CardRepository) FindByUID(ctx context.Context, uid string) (*entities.NfcCard, error) {
	// 使用领域存储库的实现
	repoImpl := repositories.NewNfcCardRepository(r.DB)
	return repoImpl.FindByUID(ctx, uid)
}

// FindByMerchantID 获取商户的所有NFC卡片
func (r *CardRepository) FindByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]*entities.NfcCard, int, error) {
	// 使用领域存储库的实现
	repoImpl := repositories.NewNfcCardRepository(r.DB)
	return repoImpl.FindByMerchantID(ctx, merchantID, page, pageSize)
}

// Update 更新NFC卡片
func (r *CardRepository) Update(ctx context.Context, id uuid.UUID, card *entities.UpdateNfcCardDTO) (*entities.NfcCard, error) {
	// 使用领域存储库的实现
	repoImpl := repositories.NewNfcCardRepository(r.DB)
	return repoImpl.Update(ctx, id, card)
}

// Delete 删除NFC卡片
func (r *CardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// 使用领域存储库的实现
	repoImpl := repositories.NewNfcCardRepository(r.DB)
	return repoImpl.Delete(ctx, id)
}

// Activate 激活NFC卡片
func (r *CardRepository) Activate(ctx context.Context, uid string) (*entities.NfcCard, error) {
	// 使用领域存储库的实现
	repoImpl := repositories.NewNfcCardRepository(r.DB)
	return repoImpl.Activate(ctx, uid)
}

// ShortlinkRepository 短链接存储库
type ShortlinkRepository struct {
	DB *sqlx.DB
}

// NewShortlinkRepository 创建短链接存储库
func NewShortlinkRepository(db *sqlx.DB) *ShortlinkRepository {
	return &ShortlinkRepository{
		DB: db,
	}
}

// Create 创建短链接
func (r *ShortlinkRepository) Create(ctx context.Context, link *entities.CreateShortLinkDTO) (*entities.ShortLink, error) {
	// 直接实现SQL逻辑
	query := `
		INSERT INTO short_links (
			tenant_id, nfc_card_id, slug, target_url, expires_at, created_at, updated_at, clicks, active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $6, 0, true
		) RETURNING id, tenant_id, nfc_card_id, slug, target_url, clicks, active, created_at, updated_at, expires_at
	`

	now := time.Now()
	var result entities.ShortLink

	err := r.DB.QueryRowxContext(
		ctx,
		query,
		link.TenantID,
		link.NfcCardID,
		link.Slug,
		link.TargetURL,
		link.ExpiresAt,
		now,
	).StructScan(&result)

	if err != nil {
		return nil, fmt.Errorf("创建短链接失败: %w", err)
	}

	return &result, nil
}

// FindByID 通过ID获取短链接
func (r *ShortlinkRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.ShortLink, error) {
	query := `SELECT * FROM short_links WHERE id = $1`

	var link entities.ShortLink
	err := r.DB.GetContext(ctx, &link, query, id)
	if err != nil {
		return nil, fmt.Errorf("获取短链接失败: %w", err)
	}

	return &link, nil
}

// FindBySlug 通过Slug获取短链接
func (r *ShortlinkRepository) FindBySlug(ctx context.Context, slug string) (*entities.ShortLink, error) {
	query := `SELECT * FROM short_links WHERE slug = $1`

	var link entities.ShortLink
	err := r.DB.GetContext(ctx, &link, query, slug)
	if err != nil {
		return nil, fmt.Errorf("获取短链接失败: %w", err)
	}

	return &link, nil
}

// FindByTenantID 获取租户的所有短链接
func (r *ShortlinkRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]*entities.ShortLink, int, error) {
	offset := (page - 1) * pageSize

	query := `SELECT * FROM short_links WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	countQuery := `SELECT COUNT(*) FROM short_links WHERE tenant_id = $1`

	var links []*entities.ShortLink
	err := r.DB.SelectContext(ctx, &links, query, tenantID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("获取短链接列表失败: %w", err)
	}

	var count int
	err = r.DB.GetContext(ctx, &count, countQuery, tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("获取短链接数量失败: %w", err)
	}

	return links, count, nil
}

// FindByNfcCardID 获取NFC卡片的所有短链接
func (r *ShortlinkRepository) FindByNfcCardID(ctx context.Context, nfcCardID uuid.UUID) ([]*entities.ShortLink, error) {
	query := `SELECT * FROM short_links WHERE nfc_card_id = $1 ORDER BY created_at DESC`

	var links []*entities.ShortLink
	err := r.DB.SelectContext(ctx, &links, query, nfcCardID)
	if err != nil {
		return nil, fmt.Errorf("获取卡片短链接失败: %w", err)
	}

	return links, nil
}

// Update 更新短链接
func (r *ShortlinkRepository) Update(ctx context.Context, id uuid.UUID, link *entities.UpdateShortLinkDTO) (*entities.ShortLink, error) {
	query := `
		UPDATE short_links SET
			target_url = $1,
	`

	params := []interface{}{link.TargetURL}
	paramCount := 1

	// 如果提供了Active字段，则更新
	if link.Active != nil {
		query += fmt.Sprintf(", active = $%d", paramCount+1)
		params = append(params, *link.Active)
		paramCount++
	}

	// 如果提供了ExpiresAt字段，则更新
	if link.ExpiresAt != nil {
		query += fmt.Sprintf(", expires_at = $%d", paramCount+1)
		params = append(params, *link.ExpiresAt)
		paramCount++
	}

	query += fmt.Sprintf(", updated_at = $%d WHERE id = $%d RETURNING *", paramCount+1, paramCount+2)
	params = append(params, time.Now(), id)

	var updated entities.ShortLink
	err := r.DB.QueryRowxContext(ctx, query, params...).StructScan(&updated)

	if err != nil {
		return nil, fmt.Errorf("更新短链接失败: %w", err)
	}

	return &updated, nil
}

// Delete 删除短链接
func (r *ShortlinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM short_links WHERE id = $1`

	_, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("删除短链接失败: %w", err)
	}

	return nil
}

// IncrementClicks 增加点击次数
func (r *ShortlinkRepository) IncrementClicks(ctx context.Context, slug string) error {
	query := `UPDATE short_links SET clicks = clicks + 1 WHERE slug = $1`

	_, err := r.DB.ExecContext(ctx, query, slug)
	if err != nil {
		return fmt.Errorf("增加点击次数失败: %w", err)
	}

	return nil
}
