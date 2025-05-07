package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"nfc-service/internal/domain/entities"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrShortLinkNotFound = errors.New("短链接未找到")
	ErrSlugAlreadyExists = errors.New("短链接slug已存在")
)

// ShortLinkRepository 实现短链接的数据库访问
type ShortLinkRepository struct {
	db *sqlx.DB
}

// NewShortLinkRepository 创建短链接仓库的实例
func NewShortLinkRepository(db *sqlx.DB) *ShortLinkRepository {
	return &ShortLinkRepository{
		db: db,
	}
}

// Create 创建新的短链接
func (r *ShortLinkRepository) Create(ctx context.Context, link *entities.CreateShortLinkDTO) (*entities.ShortLink, error) {
	// 如果提供了slug，检查是否已存在
	if link.Slug != "" {
		var count int
		err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM short_links WHERE slug = $1", link.Slug)
		if err != nil {
			return nil, fmt.Errorf("检查slug是否存在失败: %w", err)
		}

		if count > 0 {
			return nil, ErrSlugAlreadyExists
		}
	} else {
		// 生成唯一的slug
		link.Slug = generateSlug()

		// 确保slug唯一
		for {
			var count int
			err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM short_links WHERE slug = $1", link.Slug)
			if err != nil {
				return nil, fmt.Errorf("检查slug是否存在失败: %w", err)
			}

			if count == 0 {
				break
			}

			link.Slug = generateSlug()
		}
	}

	// 创建新短链接
	newLink := &entities.ShortLink{
		ID:        uuid.New(),
		TenantID:  link.TenantID,
		NfcCardID: link.NfcCardID,
		Slug:      link.Slug,
		TargetURL: link.TargetURL,
		Clicks:    0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO short_links (id, merchant_id, nfc_card_id, slug, target_url, clicks, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		newLink.ID,
		newLink.TenantID,
		newLink.NfcCardID,
		newLink.Slug,
		newLink.TargetURL,
		newLink.Clicks,
		newLink.CreatedAt,
		newLink.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("创建短链接失败: %w", err)
	}

	return newLink, nil
}

// FindByID 通过ID查找短链接
func (r *ShortLinkRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.ShortLink, error) {
	var link entities.ShortLink
	query := `
		SELECT id, merchant_id, nfc_card_id, slug, target_url, clicks, created_at, updated_at
		FROM short_links
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &link, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrShortLinkNotFound
		}
		return nil, fmt.Errorf("查找短链接失败: %w", err)
	}

	return &link, nil
}

// FindBySlug 通过Slug查找短链接
func (r *ShortLinkRepository) FindBySlug(ctx context.Context, slug string) (*entities.ShortLink, error) {
	var link entities.ShortLink
	query := `
		SELECT id, merchant_id, nfc_card_id, slug, target_url, clicks, created_at, updated_at
		FROM short_links
		WHERE slug = $1
	`

	err := r.db.GetContext(ctx, &link, query, slug)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrShortLinkNotFound
		}
		return nil, fmt.Errorf("查找短链接失败: %w", err)
	}

	return &link, nil
}

// FindByTenantID 查找商户的所有短链接
func (r *ShortLinkRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]*entities.ShortLink, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// 获取总数
	var total int
	countQuery := "SELECT COUNT(*) FROM short_links WHERE merchant_id = $1"
	err := r.db.GetContext(ctx, &total, countQuery, tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("获取短链接总数失败: %w", err)
	}

	// 获取分页数据
	query := `
		SELECT id, merchant_id, nfc_card_id, slug, target_url, clicks, created_at, updated_at
		FROM short_links
		WHERE merchant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var links []*entities.ShortLink
	err = r.db.SelectContext(ctx, &links, query, tenantID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("获取短链接列表失败: %w", err)
	}

	return links, total, nil
}

// FindByNfcCardID 查找NFC卡片的所有短链接
func (r *ShortLinkRepository) FindByNfcCardID(ctx context.Context, nfcCardID uuid.UUID) ([]*entities.ShortLink, error) {
	query := `
		SELECT id, merchant_id, nfc_card_id, slug, target_url, clicks, created_at, updated_at
		FROM short_links
		WHERE nfc_card_id = $1
		ORDER BY created_at DESC
	`

	var links []*entities.ShortLink
	err := r.db.SelectContext(ctx, &links, query, nfcCardID)
	if err != nil {
		return nil, fmt.Errorf("获取NFC卡片的短链接列表失败: %w", err)
	}

	return links, nil
}

// Update 更新短链接
func (r *ShortLinkRepository) Update(ctx context.Context, id uuid.UUID, link *entities.UpdateShortLinkDTO) (*entities.ShortLink, error) {
	// 先检查短链接是否存在
	existingLink, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 更新需要修改的字段
	if link.TargetURL != "" {
		existingLink.TargetURL = link.TargetURL
	}
	existingLink.UpdatedAt = time.Now()

	query := `
		UPDATE short_links
		SET target_url = $1, updated_at = $2
		WHERE id = $3
	`

	_, err = r.db.ExecContext(ctx, query, existingLink.TargetURL, existingLink.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("更新短链接失败: %w", err)
	}

	return existingLink, nil
}

// Delete 删除短链接
func (r *ShortLinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM short_links WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("删除短链接失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return ErrShortLinkNotFound
	}

	return nil
}

// IncrementClicks 增加短链接的点击次数
func (r *ShortLinkRepository) IncrementClicks(ctx context.Context, slug string) error {
	query := `
		UPDATE short_links
		SET clicks = clicks + 1
		WHERE slug = $1
	`

	result, err := r.db.ExecContext(ctx, query, slug)
	if err != nil {
		return fmt.Errorf("增加点击次数失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return ErrShortLinkNotFound
	}

	return nil
}

// 生成随机的6位短链slug
func generateSlug() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 8

	result := make([]byte, length)
	for i := range result {
		// 使用随机UUID的字节作为随机源
		randByte := byte(uuid.New().String()[i%36]) % byte(len(charset))
		result[i] = charset[randByte]
	}
	return string(result)
}
