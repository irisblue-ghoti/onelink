package repositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"distribution-service/internal/config"
	"distribution-service/internal/domain/entities"
)

// JobRepository 任务仓库
type JobRepository interface {
	// Create 创建任务
	Create(ctx context.Context, job *entities.PublishJob) error

	// Update 更新任务
	Update(ctx context.Context, job *entities.PublishJob) error

	// FindByID 根据ID查找任务
	FindByID(ctx context.Context, tenantID, jobID uuid.UUID) (*entities.PublishJob, error)

	// Find 查找任务列表
	Find(ctx context.Context, tenantID uuid.UUID, status, videoID, nfcCardID, channel string) ([]*entities.PublishJob, error)
}

// PostgresJobRepository PostgreSQL任务仓库实现
type PostgresJobRepository struct {
	db *sqlx.DB
}

// NewJobRepository 创建任务仓库
func NewJobRepository(dbConfig config.DatabaseConfig) JobRepository {
	// 构建数据库连接字符串
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.SSLMode)

	// 连接数据库
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		// 在实际应用中，应该优雅处理错误
		panic(fmt.Sprintf("连接数据库失败: %v", err))
	}

	return &PostgresJobRepository{
		db: db,
	}
}

// Create 创建任务
func (r *PostgresJobRepository) Create(ctx context.Context, job *entities.PublishJob) error {
	// 构建SQL语句
	query := `
		INSERT INTO publish_jobs (
			id, tenant_id, video_id, nfc_card_id, channel, status, 
			result, error_msg, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :video_id, :nfc_card_id, :channel, :status, 
			:result, :error_msg, :created_at, :updated_at
		)
	`

	// 执行SQL
	_, err := r.db.NamedExecContext(ctx, query, job)
	return err
}

// Update 更新任务
func (r *PostgresJobRepository) Update(ctx context.Context, job *entities.PublishJob) error {
	// 构建SQL语句
	query := `
		UPDATE publish_jobs SET
			status = :status,
			result = :result,
			error_msg = :error_msg,
			updated_at = :updated_at,
			completed_at = :completed_at
		WHERE id = :id AND tenant_id = :tenant_id
	`

	// 执行SQL
	_, err := r.db.NamedExecContext(ctx, query, job)
	return err
}

// FindByID 根据ID查找任务
func (r *PostgresJobRepository) FindByID(ctx context.Context, tenantID, jobID uuid.UUID) (*entities.PublishJob, error) {
	// 构建SQL语句
	query := `
		SELECT * FROM publish_jobs
		WHERE id = $1 AND tenant_id = $2
	`

	// 执行SQL
	var job entities.PublishJob
	err := r.db.GetContext(ctx, &job, query, jobID, tenantID)
	if err != nil {
		return nil, err
	}

	return &job, nil
}

// Find 查找任务列表
func (r *PostgresJobRepository) Find(ctx context.Context, tenantID uuid.UUID, status, videoID, nfcCardID, channel string) ([]*entities.PublishJob, error) {
	// 构建基础SQL
	query := `
		SELECT * FROM publish_jobs
		WHERE tenant_id = $1
	`

	// 构建参数
	args := []interface{}{tenantID}
	argIndex := 2

	// 添加条件
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if videoID != "" {
		query += fmt.Sprintf(" AND video_id = $%d", argIndex)
		videoUUID, err := uuid.Parse(videoID)
		if err != nil {
			return nil, fmt.Errorf("无效的视频ID: %w", err)
		}
		args = append(args, videoUUID)
		argIndex++
	}

	if nfcCardID != "" {
		query += fmt.Sprintf(" AND nfc_card_id = $%d", argIndex)
		nfcCardUUID, err := uuid.Parse(nfcCardID)
		if err != nil {
			return nil, fmt.Errorf("无效的NFC卡ID: %w", err)
		}
		args = append(args, nfcCardUUID)
		argIndex++
	}

	if channel != "" {
		query += fmt.Sprintf(" AND channel = $%d", argIndex)
		args = append(args, channel)
	}

	// 添加排序
	query += " ORDER BY created_at DESC"

	// 执行SQL
	var jobs []*entities.PublishJob
	err := r.db.SelectContext(ctx, &jobs, query, args...)
	if err != nil {
		return nil, err
	}

	return jobs, nil
}
