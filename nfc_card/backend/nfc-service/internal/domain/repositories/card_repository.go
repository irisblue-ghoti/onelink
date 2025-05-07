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
	ErrCardNotFound     = errors.New("NFC卡片未找到")
	ErrUIDAlreadyExists = errors.New("UID已存在")
)

// NfcCardRepository 实现NFC卡片的数据库访问
type NfcCardRepository struct {
	db *sqlx.DB
}

// NewNfcCardRepository 创建NFC卡片仓库的实例
func NewNfcCardRepository(db *sqlx.DB) *NfcCardRepository {
	return &NfcCardRepository{
		db: db,
	}
}

// Create 创建新的NFC卡片
func (r *NfcCardRepository) Create(ctx context.Context, card *entities.CreateNfcCardDTO) (*entities.NfcCard, error) {
	// 检查UID是否已存在
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM nfc_cards WHERE uid = $1", card.UID)
	if err != nil {
		return nil, fmt.Errorf("检查UID是否存在失败: %w", err)
	}

	if count > 0 {
		return nil, ErrUIDAlreadyExists
	}

	// 创建新卡片
	newCard := &entities.NfcCard{
		ID:             uuid.New(),
		MerchantID:     card.MerchantID,
		UID:            card.UID,
		Name:           card.Name,
		Description:    card.Description,
		DefaultVideoID: card.DefaultVideoID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	query := `
		INSERT INTO nfc_cards (id, merchant_id, uid, name, description, default_video_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		newCard.ID,
		newCard.MerchantID,
		newCard.UID,
		newCard.Name,
		newCard.Description,
		newCard.DefaultVideoID,
		newCard.CreatedAt,
		newCard.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("创建NFC卡片失败: %w", err)
	}

	return newCard, nil
}

// FindByID 通过ID查找NFC卡片
func (r *NfcCardRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.NfcCard, error) {
	var card entities.NfcCard
	query := `
		SELECT id, merchant_id, uid, name, description, default_video_id, activated_at, created_at, updated_at
		FROM nfc_cards
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &card, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCardNotFound
		}
		return nil, fmt.Errorf("查找NFC卡片失败: %w", err)
	}

	return &card, nil
}

// FindByUID 通过UID查找NFC卡片
func (r *NfcCardRepository) FindByUID(ctx context.Context, uid string) (*entities.NfcCard, error) {
	var card entities.NfcCard
	query := `
		SELECT id, merchant_id, uid, name, description, default_video_id, activated_at, created_at, updated_at
		FROM nfc_cards
		WHERE uid = $1
	`

	err := r.db.GetContext(ctx, &card, query, uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCardNotFound
		}
		return nil, fmt.Errorf("查找NFC卡片失败: %w", err)
	}

	return &card, nil
}

// FindByMerchantID 查找商户的所有NFC卡片
func (r *NfcCardRepository) FindByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]*entities.NfcCard, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// 获取总数
	var total int
	countQuery := "SELECT COUNT(*) FROM nfc_cards WHERE merchant_id = $1"
	err := r.db.GetContext(ctx, &total, countQuery, merchantID)
	if err != nil {
		return nil, 0, fmt.Errorf("获取NFC卡片总数失败: %w", err)
	}

	// 获取分页数据
	query := `
		SELECT id, merchant_id, uid, name, description, default_video_id, activated_at, created_at, updated_at
		FROM nfc_cards
		WHERE merchant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var cards []*entities.NfcCard
	err = r.db.SelectContext(ctx, &cards, query, merchantID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("获取NFC卡片列表失败: %w", err)
	}

	return cards, total, nil
}

// Update 更新NFC卡片
func (r *NfcCardRepository) Update(ctx context.Context, id uuid.UUID, card *entities.UpdateNfcCardDTO) (*entities.NfcCard, error) {
	// 先检查卡片是否存在
	existingCard, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 更新需要修改的字段
	if card.Name != "" {
		existingCard.Name = card.Name
	}
	if card.Description != "" {
		existingCard.Description = card.Description
	}
	if card.DefaultVideoID != nil {
		existingCard.DefaultVideoID = card.DefaultVideoID
	}
	if card.ActivatedAt != nil {
		existingCard.ActivatedAt = card.ActivatedAt
	}
	existingCard.UpdatedAt = time.Now()

	query := `
		UPDATE nfc_cards
		SET name = $1, description = $2, default_video_id = $3, activated_at = $4, updated_at = $5
		WHERE id = $6
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		existingCard.Name,
		existingCard.Description,
		existingCard.DefaultVideoID,
		existingCard.ActivatedAt,
		existingCard.UpdatedAt,
		id,
	)

	if err != nil {
		return nil, fmt.Errorf("更新NFC卡片失败: %w", err)
	}

	return existingCard, nil
}

// Delete 删除NFC卡片
func (r *NfcCardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM nfc_cards WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("删除NFC卡片失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return ErrCardNotFound
	}

	return nil
}

// Activate 激活NFC卡片
func (r *NfcCardRepository) Activate(ctx context.Context, uid string) (*entities.NfcCard, error) {
	// 查找卡片
	card, err := r.FindByUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	// 设置激活时间
	now := time.Now()
	card.ActivatedAt = &now
	card.UpdatedAt = now

	query := `
		UPDATE nfc_cards
		SET activated_at = $1, updated_at = $2
		WHERE id = $3
	`

	_, err = r.db.ExecContext(ctx, query, card.ActivatedAt, card.UpdatedAt, card.ID)
	if err != nil {
		return nil, fmt.Errorf("激活NFC卡片失败: %w", err)
	}

	return card, nil
}

// BindToUser 将卡片绑定到用户
func (r *NfcCardRepository) BindToUser(ctx context.Context, cardID, userID uuid.UUID) (*entities.NfcCard, error) {
	// 查找卡片
	card, err := r.FindByID(ctx, cardID)
	if err != nil {
		return nil, err
	}

	// 设置绑定信息
	now := time.Now()
	card.UserID = &userID
	card.BoundAt = &now
	card.Status = entities.CardStatusBound
	card.UpdatedAt = now

	query := `
		UPDATE nfc_cards
		SET user_id = $1, bound_at = $2, status = $3, updated_at = $4
		WHERE id = $5
	`

	_, err = r.db.ExecContext(ctx, query, card.UserID, card.BoundAt, card.Status, card.UpdatedAt, card.ID)
	if err != nil {
		return nil, fmt.Errorf("绑定NFC卡片到用户失败: %w", err)
	}

	return card, nil
}

// UnbindFromUser 解除卡片与用户的绑定
func (r *NfcCardRepository) UnbindFromUser(ctx context.Context, cardID uuid.UUID) (*entities.NfcCard, error) {
	// 查找卡片
	card, err := r.FindByID(ctx, cardID)
	if err != nil {
		return nil, err
	}

	// 确保卡片确实已绑定
	if card.UserID == nil || card.Status != entities.CardStatusBound {
		return nil, fmt.Errorf("卡片没有绑定用户")
	}

	// 解除绑定
	now := time.Now()
	card.UserID = nil
	card.Status = entities.CardStatusActivated // 恢复到激活状态
	card.UpdatedAt = now

	query := `
		UPDATE nfc_cards
		SET user_id = NULL, status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err = r.db.ExecContext(ctx, query, card.Status, card.UpdatedAt, card.ID)
	if err != nil {
		return nil, fmt.Errorf("解除NFC卡片与用户的绑定失败: %w", err)
	}

	return card, nil
}

// Deactivate 停用NFC卡片
func (r *NfcCardRepository) Deactivate(ctx context.Context, id uuid.UUID) (*entities.NfcCard, error) {
	// 查找卡片
	card, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 设置停用信息
	now := time.Now()
	card.DeactivatedAt = &now
	card.Status = entities.CardStatusDeactivated
	card.UpdatedAt = now

	query := `
		UPDATE nfc_cards
		SET deactivated_at = $1, status = $2, updated_at = $3
		WHERE id = $4
	`

	_, err = r.db.ExecContext(ctx, query, card.DeactivatedAt, card.Status, card.UpdatedAt, card.ID)
	if err != nil {
		return nil, fmt.Errorf("停用NFC卡片失败: %w", err)
	}

	return card, nil
}

// Reactivate 重新激活已停用的NFC卡片
func (r *NfcCardRepository) Reactivate(ctx context.Context, id uuid.UUID) (*entities.NfcCard, error) {
	// 查找卡片
	card, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 确保卡片确实已停用
	if card.Status != entities.CardStatusDeactivated {
		return nil, fmt.Errorf("卡片不处于停用状态")
	}

	// 重新激活
	now := time.Now()
	card.DeactivatedAt = nil
	card.Status = entities.CardStatusActivated
	card.UpdatedAt = now

	query := `
		UPDATE nfc_cards
		SET deactivated_at = NULL, status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err = r.db.ExecContext(ctx, query, card.Status, card.UpdatedAt, card.ID)
	if err != nil {
		return nil, fmt.Errorf("重新激活NFC卡片失败: %w", err)
	}

	return card, nil
}

// UpdateStatus 更新卡片状态
func (r *NfcCardRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.CardStatus) (*entities.NfcCard, error) {
	// 查找卡片
	card, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 更新状态
	now := time.Now()
	card.Status = status
	card.UpdatedAt = now

	query := `
		UPDATE nfc_cards
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err = r.db.ExecContext(ctx, query, card.Status, card.UpdatedAt, card.ID)
	if err != nil {
		return nil, fmt.Errorf("更新NFC卡片状态失败: %w", err)
	}

	return card, nil
}

// FindByUserID 查找用户的所有已绑定NFC卡片
func (r *NfcCardRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.NfcCard, error) {
	query := `
		SELECT id, merchant_id, uid, name, description, default_video_id, status, user_id, 
		activated_at, bound_at, deactivated_at, expires_at, created_at, updated_at
		FROM nfc_cards
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	var cards []*entities.NfcCard
	err := r.db.SelectContext(ctx, &cards, query, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户的NFC卡片列表失败: %w", err)
	}

	return cards, nil
}
