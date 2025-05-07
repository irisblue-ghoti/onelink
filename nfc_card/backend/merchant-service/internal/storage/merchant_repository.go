package storage

import (
	"database/sql"
	"errors"
	"merchant-service/internal/domain/entities"
	"merchant-service/internal/domain/repositories"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresMerchantRepository PostgreSQL商户仓库实现
type PostgresMerchantRepository struct {
	DB *sqlx.DB
}

// 确保PostgresMerchantRepository实现了MerchantRepository接口
var _ repositories.MerchantRepository = (*PostgresMerchantRepository)(nil)

// NewPostgresMerchantRepository 创建PostgreSQL商户仓库
func NewPostgresMerchantRepository(db *sqlx.DB) *PostgresMerchantRepository {
	return &PostgresMerchantRepository{
		DB: db,
	}
}

// Create 创建商户
func (r *PostgresMerchantRepository) Create(merchant entities.Merchant) (entities.Merchant, error) {
	query := `
		INSERT INTO merchants (
			id, name, description, email, phone, logo_url, website, address, 
			status, plan_id, api_key, created_at, updated_at
		) VALUES (
			:id, :name, :description, :email, :phone, :logo_url, :website, :address, 
			:status, :plan_id, :api_key, :created_at, :updated_at
		) RETURNING *
	`

	rows, err := r.DB.NamedQuery(query, merchant)
	if err != nil {
		return entities.Merchant{}, err
	}
	defer rows.Close()

	if rows.Next() {
		var result entities.Merchant
		if err := rows.StructScan(&result); err != nil {
			return entities.Merchant{}, err
		}
		return result, nil
	}

	return entities.Merchant{}, errors.New("创建商户失败")
}

// FindByID 通过ID查找商户
func (r *PostgresMerchantRepository) FindByID(id string) (entities.Merchant, error) {
	var merchant entities.Merchant

	query := "SELECT * FROM merchants WHERE id = $1"
	if err := r.DB.Get(&merchant, query, id); err != nil {
		if err == sql.ErrNoRows {
			return entities.Merchant{}, errors.New("商户不存在")
		}
		return entities.Merchant{}, err
	}

	return merchant, nil
}

// FindAll 查找所有商户（分页）
func (r *PostgresMerchantRepository) FindAll(params entities.PaginationParams) ([]entities.Merchant, int, error) {
	var merchants []entities.Merchant
	var totalItems int

	// 确保有默认的分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}

	offset := (params.Page - 1) * params.Limit

	// 获取总记录数
	countQuery := "SELECT COUNT(*) FROM merchants"
	if err := r.DB.Get(&totalItems, countQuery); err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	query := `
		SELECT * FROM merchants
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	if err := r.DB.Select(&merchants, query, params.Limit, offset); err != nil {
		return nil, 0, err
	}

	return merchants, totalItems, nil
}

// Update 更新商户
func (r *PostgresMerchantRepository) Update(merchant entities.Merchant) (entities.Merchant, error) {
	query := `
		UPDATE merchants SET
			name = :name,
			description = :description,
			email = :email,
			phone = :phone,
			logo_url = :logo_url,
			website = :website,
			address = :address,
			status = :status,
			plan_id = :plan_id,
			updated_at = :updated_at
		WHERE id = :id
		RETURNING *
	`

	rows, err := r.DB.NamedQuery(query, merchant)
	if err != nil {
		return entities.Merchant{}, err
	}
	defer rows.Close()

	if rows.Next() {
		var result entities.Merchant
		if err := rows.StructScan(&result); err != nil {
			return entities.Merchant{}, err
		}
		return result, nil
	}

	return entities.Merchant{}, errors.New("更新商户失败")
}

// Delete 删除商户
func (r *PostgresMerchantRepository) Delete(id string) error {
	query := "DELETE FROM merchants WHERE id = $1"
	result, err := r.DB.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("商户不存在")
	}

	return nil
}

// UpdateApiKey 更新商户API密钥
func (r *PostgresMerchantRepository) UpdateApiKey(id string, apiKey string) error {
	query := "UPDATE merchants SET api_key = $1, updated_at = NOW() WHERE id = $2"
	result, err := r.DB.Exec(query, apiKey, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("商户不存在")
	}

	return nil
}

// CountAll 获取商户总数
func (r *PostgresMerchantRepository) CountAll() (int, error) {
	var count int
	if err := r.DB.Get(&count, "SELECT COUNT(*) FROM merchants"); err != nil {
		return 0, err
	}
	return count, nil
}

// FindByEmail 通过邮箱查找商户
func (r *PostgresMerchantRepository) FindByEmail(email string) (entities.Merchant, error) {
	var merchant entities.Merchant
	query := "SELECT * FROM merchants WHERE email = $1"
	if err := r.DB.Get(&merchant, query, email); err != nil {
		if err == sql.ErrNoRows {
			return entities.Merchant{}, errors.New("商户不存在")
		}
		return entities.Merchant{}, err
	}
	return merchant, nil
}

// FindByPlanID 通过套餐ID查找商户
func (r *PostgresMerchantRepository) FindByPlanID(planID uuid.UUID) ([]entities.Merchant, error) {
	var merchants []entities.Merchant
	query := "SELECT * FROM merchants WHERE plan_id = $1"
	if err := r.DB.Select(&merchants, query, planID); err != nil {
		return nil, err
	}
	return merchants, nil
}
