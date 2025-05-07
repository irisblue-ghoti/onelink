package storage

import (
	"fmt"

	"distribution-service/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Repositories 所有仓库的集合
type Repositories struct {
	db *sqlx.DB
	// 可以添加具体的仓库实例，如VideoRepo、JobRepo等
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
		db: db,
	}
}

// Close 关闭数据库连接
func (r *Repositories) Close() error {
	return r.db.Close()
}
