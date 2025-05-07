package storage

import (
	"fmt"
	"merchant-service/internal/config"
	"merchant-service/internal/domain/repositories"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Repositories 所有仓库的集合
type Repositories struct {
	db                 *sqlx.DB
	MerchantRepository repositories.MerchantRepository
	UserRepository     *UserRepository
}

// NewDBConnection 创建数据库连接
func NewDBConnection(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	return sqlx.Connect("postgres", psqlInfo)
}

// NewRepositories 创建存储库集合
func NewRepositories(db *sqlx.DB) *Repositories {
	return &Repositories{
		db:                 db,
		MerchantRepository: NewPostgresMerchantRepository(db),
		UserRepository:     NewUserRepository(db),
	}
}

// Close 关闭数据库连接
func (r *Repositories) Close() error {
	return r.db.Close()
}

// MerchantRepository 商户存储库
type MerchantRepository struct {
	DB *sqlx.DB
}

// NewMerchantRepository 创建商户存储库
func NewMerchantRepository(db *sqlx.DB) *MerchantRepository {
	return &MerchantRepository{
		DB: db,
	}
}
