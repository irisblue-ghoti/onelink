package storage

import (
	"database/sql"
	"errors"
	"merchant-service/internal/domain/entities"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository 用户存储库
type UserRepository struct {
	DB *sqlx.DB
}

// NewUserRepository 创建用户存储库
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

// FindByEmail 通过邮箱查找用户
func (r *UserRepository) FindByEmail(email string) (entities.User, error) {
	var user entities.User

	query := "SELECT * FROM users WHERE email = $1"
	if err := r.DB.Get(&user, query, email); err != nil {
		if err == sql.ErrNoRows {
			return entities.User{}, errors.New("用户不存在")
		}
		return entities.User{}, err
	}

	return user, nil
}

// FindByID 通过ID查找用户
func (r *UserRepository) FindByID(id string) (entities.User, error) {
	var user entities.User

	query := "SELECT * FROM users WHERE id = $1"
	if err := r.DB.Get(&user, query, id); err != nil {
		if err == sql.ErrNoRows {
			return entities.User{}, errors.New("用户不存在")
		}
		return entities.User{}, err
	}

	return user, nil
}

// Create 创建新用户
func (r *UserRepository) Create(user entities.User) (entities.User, error) {
	// 检查邮箱是否已存在
	exists, err := r.EmailExists(user.Email)
	if err != nil {
		return entities.User{}, err
	}
	if exists {
		return entities.User{}, errors.New("该邮箱已被注册")
	}

	// 不使用NamedQuery，转而使用更直接的方式
	query := `
		INSERT INTO users (
			id, merchant_id, name, email, password_hash, role, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING *
	`

	var result entities.User
	err = r.DB.QueryRowx(
		query,
		user.ID,
		user.MerchantID,
		user.Name,
		user.Email,
		user.Password, // 密码字段，将被映射到password_hash列
		user.Role,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
	).StructScan(&result)

	if err != nil {
		return entities.User{}, errors.New("创建用户失败: " + err.Error())
	}

	return result, nil
}

// EmailExists 检查邮箱是否已存在
func (r *UserRepository) EmailExists(email string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM users WHERE email = $1"
	if err := r.DB.Get(&count, query, email); err != nil {
		return false, err
	}
	return count > 0, nil
}

// VerifyPassword 验证密码
func (r *UserRepository) VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
