package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"stats-service/internal/config"
	"stats-service/internal/domain/entities"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// Repositories 所有仓库的集合
type Repositories struct {
	db        *sqlx.DB
	StatsRepo *StatsRepository
}

// StatsRepository 统计数据仓库
type StatsRepository struct {
	db *sqlx.DB
}

// NewDBConnection 创建数据库连接
func NewDBConnection(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	return sqlx.Connect("postgres", psqlInfo)
}

// NewRepositories 创建存储库集合
func NewRepositories(db *sqlx.DB) *Repositories {
	return &Repositories{
		db:        db,
		StatsRepo: &StatsRepository{db: db},
	}
}

// Close 关闭数据库连接
func (r *Repositories) Close() error {
	return r.db.Close()
}

// GetPlatformStats 获取平台统计数据
func (r *Repositories) GetPlatformStats(ctx context.Context, tenantID, videoID uuid.UUID, platform, platformID string) (*entities.PlatformStats, error) {
	query := `
		SELECT 
			id, tenant_id, video_id, nfc_card_id, platform, platform_id,
			view_count, like_count, comment_count, share_count, collect_count,
			raw_data, last_updated_at, created_at
		FROM platform_stats
		WHERE tenant_id = $1 AND video_id = $2 AND platform = $3 AND platform_id = $4
		LIMIT 1
	`

	platformUUID, err := uuid.Parse(platformID)
	if err != nil {
		// 如果无法解析为UUID，使用SHA1创建一个
		platformUUID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(platformID))
	}

	row := r.db.QueryRowxContext(ctx, query, tenantID, videoID, platform, platformUUID)

	var stat entities.PlatformStats
	var rawDataJSON []byte

	err = row.Scan(
		&stat.ID, &stat.TenantID, &stat.VideoID, &stat.NfcCardID, &stat.Platform, &stat.PlatformID,
		&stat.ViewCount, &stat.LikeCount, &stat.CommentCount, &stat.ShareCount, &stat.CollectCount,
		&rawDataJSON, &stat.LastUpdatedAt, &stat.CreatedAt,
	)

	if err != nil {
		// 如果没有记录，返回nil
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("获取平台统计数据失败: %w", err)
	}

	// 解析原始数据
	if len(rawDataJSON) > 0 {
		if err := json.Unmarshal(rawDataJSON, &stat.RawData); err != nil {
			return nil, fmt.Errorf("解析原始数据失败: %w", err)
		}
	}

	return &stat, nil
}

// GetLatestPlatformStats 获取最新平台统计数据
func (r *Repositories) GetLatestPlatformStats(ctx context.Context, tenantID, videoID uuid.UUID, platforms []string) ([]*entities.PlatformStats, error) {
	query := `
		SELECT 
			id, tenant_id, video_id, nfc_card_id, platform, platform_id,
			view_count, like_count, comment_count, share_count, collect_count,
			raw_data, last_updated_at, created_at
		FROM platform_stats
		WHERE tenant_id = $1 AND video_id = $2
	`

	args := []interface{}{tenantID, videoID}
	if len(platforms) > 0 {
		query += " AND platform = ANY($3)"
		args = append(args, pq.Array(platforms))
	}

	query += " ORDER BY last_updated_at DESC"

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询平台统计数据失败: %w", err)
	}
	defer rows.Close()

	var stats []*entities.PlatformStats
	for rows.Next() {
		var stat entities.PlatformStats
		var rawDataJSON []byte

		err := rows.Scan(
			&stat.ID, &stat.TenantID, &stat.VideoID, &stat.NfcCardID, &stat.Platform, &stat.PlatformID,
			&stat.ViewCount, &stat.LikeCount, &stat.CommentCount, &stat.ShareCount, &stat.CollectCount,
			&rawDataJSON, &stat.LastUpdatedAt, &stat.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描平台统计数据失败: %w", err)
		}

		// 解析原始数据
		if len(rawDataJSON) > 0 {
			if err := json.Unmarshal(rawDataJSON, &stat.RawData); err != nil {
				return nil, fmt.Errorf("解析原始数据失败: %w", err)
			}
		}

		stats = append(stats, &stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代平台统计数据失败: %w", err)
	}

	return stats, nil
}

// GetDailyStats 获取每日统计数据
func (r *Repositories) GetDailyStats(ctx context.Context, tenantID, videoID uuid.UUID, platform string, startDate, endDate time.Time) ([]*entities.DailyStats, error) {
	query := `
		SELECT 
			id, tenant_id, video_id, nfc_card_id, platform, date,
			view_count, like_count, comment_count, share_count, collect_count,
			created_at, updated_at
		FROM daily_stats
		WHERE tenant_id = $1 AND video_id = $2 AND platform = $3 AND date BETWEEN $4 AND $5
		ORDER BY date
	`

	rows, err := r.db.QueryxContext(ctx, query, tenantID, videoID, platform, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("查询每日统计数据失败: %w", err)
	}
	defer rows.Close()

	var stats []*entities.DailyStats
	for rows.Next() {
		var stat entities.DailyStats
		err := rows.StructScan(&stat)
		if err != nil {
			return nil, fmt.Errorf("扫描每日统计数据失败: %w", err)
		}
		stats = append(stats, &stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代每日统计数据失败: %w", err)
	}

	return stats, nil
}

// GetTotalStats 获取总计统计数据
func (r *Repositories) GetTotalStats(ctx context.Context, tenantID, videoID uuid.UUID) (map[string]int64, error) {
	query := `
		SELECT 
			COALESCE(SUM(view_count), 0) as total_views,
			COALESCE(SUM(like_count), 0) as total_likes,
			COALESCE(SUM(comment_count), 0) as total_comments,
			COALESCE(SUM(share_count), 0) as total_shares,
			COALESCE(SUM(collect_count), 0) as total_collects
		FROM platform_stats
		WHERE tenant_id = $1 AND video_id = $2
	`

	var results struct {
		TotalViews    int64 `db:"total_views"`
		TotalLikes    int64 `db:"total_likes"`
		TotalComments int64 `db:"total_comments"`
		TotalShares   int64 `db:"total_shares"`
		TotalCollects int64 `db:"total_collects"`
	}

	err := r.db.GetContext(ctx, &results, query, tenantID, videoID)
	if err != nil {
		return nil, fmt.Errorf("获取总计统计数据失败: %w", err)
	}

	return map[string]int64{
		"total_views":    results.TotalViews,
		"total_likes":    results.TotalLikes,
		"total_comments": results.TotalComments,
		"total_shares":   results.TotalShares,
		"total_collects": results.TotalCollects,
	}, nil
}

// SavePlatformStats 保存平台统计数据
func (r *Repositories) SavePlatformStats(ctx context.Context, stats *entities.PlatformStats) error {
	// 先检查是否存在相同记录
	query := `
		SELECT id FROM platform_stats
		WHERE tenant_id = $1 AND video_id = $2 AND nfc_card_id = $3 AND platform = $4 AND platform_id = $5
		LIMIT 1
	`
	var existingID string
	err := r.db.GetContext(ctx, &existingID, query,
		stats.TenantID, stats.VideoID, stats.NfcCardID, stats.Platform, stats.PlatformID)

	// 转换原始数据为JSON
	rawDataJSON, err := json.Marshal(stats.RawData)
	if err != nil {
		return fmt.Errorf("序列化原始数据失败: %w", err)
	}

	if existingID != "" {
		// 更新现有记录
		updateQuery := `
			UPDATE platform_stats SET
				view_count = $1,
				like_count = $2,
				comment_count = $3,
				share_count = $4,
				collect_count = $5,
				raw_data = $6,
				last_updated_at = $7
			WHERE id = $8
		`
		_, err = r.db.ExecContext(ctx, updateQuery,
			stats.ViewCount, stats.LikeCount, stats.CommentCount, stats.ShareCount, stats.CollectCount,
			rawDataJSON, time.Now(), existingID)
		if err != nil {
			return fmt.Errorf("更新平台统计数据失败: %w", err)
		}
	} else {
		// 插入新记录
		insertQuery := `
			INSERT INTO platform_stats (
				id, tenant_id, video_id, nfc_card_id, platform, platform_id,
				view_count, like_count, comment_count, share_count, collect_count,
				raw_data, last_updated_at, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
			)
		`
		_, err = r.db.ExecContext(ctx, insertQuery,
			stats.ID, stats.TenantID, stats.VideoID, stats.NfcCardID, stats.Platform, stats.PlatformID,
			stats.ViewCount, stats.LikeCount, stats.CommentCount, stats.ShareCount, stats.CollectCount,
			rawDataJSON, stats.LastUpdatedAt, stats.CreatedAt)
		if err != nil {
			return fmt.Errorf("插入平台统计数据失败: %w", err)
		}
	}

	return nil
}

// SaveDailyStats 保存每日统计数据
func (r *Repositories) SaveDailyStats(ctx context.Context, stats *entities.DailyStats) error {
	// 先检查是否存在相同记录
	query := `
		SELECT id FROM daily_stats
		WHERE tenant_id = $1 AND video_id = $2 AND nfc_card_id = $3 AND platform = $4 AND date = $5
		LIMIT 1
	`
	var existingID string
	err := r.db.GetContext(ctx, &existingID, query,
		stats.TenantID, stats.VideoID, stats.NfcCardID, stats.Platform, stats.Date)

	if existingID != "" {
		// 更新现有记录
		updateQuery := `
			UPDATE daily_stats SET
				view_count = $1,
				like_count = $2,
				comment_count = $3,
				share_count = $4,
				collect_count = $5,
				updated_at = $6
			WHERE id = $7
		`
		_, err = r.db.ExecContext(ctx, updateQuery,
			stats.ViewCount, stats.LikeCount, stats.CommentCount, stats.ShareCount, stats.CollectCount,
			time.Now(), existingID)
		if err != nil {
			return fmt.Errorf("更新每日统计数据失败: %w", err)
		}
	} else {
		// 插入新记录
		insertQuery := `
			INSERT INTO daily_stats (
				id, tenant_id, video_id, nfc_card_id, platform, date,
				view_count, like_count, comment_count, share_count, collect_count,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
			)
		`
		_, err = r.db.ExecContext(ctx, insertQuery,
			stats.ID, stats.TenantID, stats.VideoID, stats.NfcCardID, stats.Platform, stats.Date,
			stats.ViewCount, stats.LikeCount, stats.CommentCount, stats.ShareCount, stats.CollectCount,
			stats.CreatedAt, stats.UpdatedAt)
		if err != nil {
			return fmt.Errorf("插入每日统计数据失败: %w", err)
		}
	}

	return nil
}
