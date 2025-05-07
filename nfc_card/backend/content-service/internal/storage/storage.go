package storage

import (
	"fmt"

	"content-service/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Repositories 所有仓库的集合
type Repositories struct {
	db *sqlx.DB
	// 可以添加具体的仓库实例，如VideoRepo等
}

// NewDBConnection 创建数据库连接
func NewDBConnection(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	// 构建不包含敏感信息的连接字符串用于日志记录
	logSafeDSN := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.DBName, cfg.SSLMode)

	// 实际连接字符串包含密码
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	// 使用隐藏密码的DSN字符串进行日志输出
	fmt.Printf("正在连接数据库: %s\n", logSafeDSN)

	return sqlx.Connect("postgres", psqlInfo)
}

// NewRepositories 创建存储库集合
func NewRepositories(db *sqlx.DB) *Repositories {
	return &Repositories{
		db: db,
	}
}

// GetDB 返回数据库连接实例
func (r *Repositories) GetDB() *sqlx.DB {
	return r.db
}

// Close 关闭数据库连接
func (r *Repositories) Close() error {
	return r.db.Close()
}
