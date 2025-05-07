package repositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL驱动

	"distribution-service/internal/config"
	"distribution-service/internal/domain/entities"
)

// VideoRepository 视频仓库接口
type VideoRepository interface {
	// FindByID 根据ID查找视频
	FindByID(ctx context.Context, tenantID, videoID uuid.UUID) (*entities.Video, error)
}

// PostgresVideoRepository PostgreSQL视频仓库实现
type PostgresVideoRepository struct {
	db *sqlx.DB
}

// NewVideoRepository 创建视频仓库
func NewVideoRepository(dbConfig config.DatabaseConfig) VideoRepository {
	// 复用连接字符串
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.SSLMode)

	// 连接数据库
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		panic(fmt.Sprintf("连接数据库失败: %v", err))
	}

	return &PostgresVideoRepository{
		db: db,
	}
}

// FindByID 根据ID查找视频
func (r *PostgresVideoRepository) FindByID(ctx context.Context, tenantID, videoID uuid.UUID) (*entities.Video, error) {
	// 构建SQL语句
	query := `
		SELECT * FROM videos
		WHERE id = $1 AND tenant_id = $2
	`

	// 执行SQL
	var video entities.Video
	err := r.db.GetContext(ctx, &video, query, videoID, tenantID)
	if err != nil {
		return nil, err
	}

	return &video, nil
}
