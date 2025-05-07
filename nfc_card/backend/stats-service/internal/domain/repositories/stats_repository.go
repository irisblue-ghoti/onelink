package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"stats-service/internal/config"
	"stats-service/internal/domain/entities"
)

// StatsRepository 统计数据仓库
type StatsRepository struct {
	db *sqlx.DB
}

// NewStatsRepository 创建统计数据仓库
func NewStatsRepository(cfg config.DatabaseConfig) (*StatsRepository, error) {
	db, err := sqlx.Connect("postgres", cfg.DatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	return &StatsRepository{
		db: db,
	}, nil
}

// SavePlatformStats 保存平台统计数据
func (r *StatsRepository) SavePlatformStats(ctx context.Context, stats *entities.PlatformStats) error {
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
func (r *StatsRepository) SaveDailyStats(ctx context.Context, stats *entities.DailyStats) error {
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

// GetLatestPlatformStats 获取最新平台统计数据
func (r *StatsRepository) GetLatestPlatformStats(ctx context.Context, tenantID, videoID uuid.UUID, platforms []string) ([]*entities.PlatformStats, error) {
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
func (r *StatsRepository) GetDailyStats(ctx context.Context, tenantID, videoID uuid.UUID, platform string, startDate, endDate time.Time) ([]*entities.DailyStats, error) {
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
func (r *StatsRepository) GetTotalStats(ctx context.Context, tenantID, videoID uuid.UUID) (map[string]int64, error) {
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

	var totalStats struct {
		TotalViews    int64 `db:"total_views"`
		TotalLikes    int64 `db:"total_likes"`
		TotalComments int64 `db:"total_comments"`
		TotalShares   int64 `db:"total_shares"`
		TotalCollects int64 `db:"total_collects"`
	}

	err := r.db.GetContext(ctx, &totalStats, query, tenantID, videoID)
	if err != nil {
		return nil, fmt.Errorf("查询总计统计数据失败: %w", err)
	}

	return map[string]int64{
		"views":    totalStats.TotalViews,
		"likes":    totalStats.TotalLikes,
		"comments": totalStats.TotalComments,
		"shares":   totalStats.TotalShares,
		"collects": totalStats.TotalCollects,
	}, nil
}

// GetVideoPlatforms 获取视频的所有平台发布记录
func (r *StatsRepository) GetVideoPlatforms(ctx context.Context, tenantID, videoID uuid.UUID) ([]struct {
	Platform   string    `db:"platform"`
	PlatformID uuid.UUID `db:"platform_id"`
}, error) {
	query := `
		SELECT DISTINCT platform, platform_id
		FROM platform_stats
		WHERE tenant_id = $1 AND video_id = $2
	`

	var results []struct {
		Platform   string    `db:"platform"`
		PlatformID uuid.UUID `db:"platform_id"`
	}

	err := r.db.SelectContext(ctx, &results, query, tenantID, videoID)
	if err != nil {
		return nil, fmt.Errorf("获取视频平台发布记录失败: %w", err)
	}

	return results, nil
}

// GetPlatformsToRefresh 获取需要刷新的平台统计数据
func (r *StatsRepository) GetPlatformsToRefresh(ctx context.Context, lastUpdateTime time.Time) ([]*entities.PlatformStats, error) {
	query := `
		SELECT 
			id, tenant_id, video_id, nfc_card_id, platform, platform_id,
			view_count, like_count, comment_count, share_count, collect_count,
			raw_data, last_updated_at, created_at
		FROM platform_stats
		WHERE last_updated_at < $1
		ORDER BY last_updated_at ASC
	`

	rows, err := r.db.QueryxContext(ctx, query, lastUpdateTime)
	if err != nil {
		return nil, fmt.Errorf("查询需要刷新的平台统计数据失败: %w", err)
	}
	defer rows.Close()

	var stats []*entities.PlatformStats
	for rows.Next() {
		var stat entities.PlatformStats
		var rawDataJSON string

		// 构造扫描目标
		dest := []interface{}{
			&stat.ID, &stat.TenantID, &stat.VideoID, &stat.NfcCardID, &stat.Platform, &stat.PlatformID,
			&stat.ViewCount, &stat.LikeCount, &stat.CommentCount, &stat.ShareCount, &stat.CollectCount,
			&rawDataJSON, &stat.LastUpdatedAt, &stat.CreatedAt,
		}

		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("扫描平台统计数据失败: %w", err)
		}

		// 解析JSON数据
		if rawDataJSON != "" {
			if err := json.Unmarshal([]byte(rawDataJSON), &stat.RawData); err != nil {
				return nil, fmt.Errorf("解析平台统计数据JSON失败: %w", err)
			}
		}

		stats = append(stats, &stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代平台统计数据失败: %w", err)
	}

	return stats, nil
}

// GetAllPlatformStats 获取所有平台统计数据
func (r *StatsRepository) GetAllPlatformStats(ctx context.Context) ([]*entities.PlatformStats, error) {
	query := `
		SELECT 
			id, tenant_id, video_id, nfc_card_id, platform, platform_id,
			view_count, like_count, comment_count, share_count, collect_count,
			raw_data, last_updated_at, created_at
		FROM platform_stats
		ORDER BY last_updated_at DESC
	`

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询所有平台统计数据失败: %w", err)
	}
	defer rows.Close()

	var stats []*entities.PlatformStats
	for rows.Next() {
		var stat entities.PlatformStats
		var rawDataJSON string

		// 构造扫描目标
		dest := []interface{}{
			&stat.ID, &stat.TenantID, &stat.VideoID, &stat.NfcCardID, &stat.Platform, &stat.PlatformID,
			&stat.ViewCount, &stat.LikeCount, &stat.CommentCount, &stat.ShareCount, &stat.CollectCount,
			&rawDataJSON, &stat.LastUpdatedAt, &stat.CreatedAt,
		}

		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("扫描平台统计数据失败: %w", err)
		}

		// 解析JSON数据
		if rawDataJSON != "" {
			if err := json.Unmarshal([]byte(rawDataJSON), &stat.RawData); err != nil {
				return nil, fmt.Errorf("解析平台统计数据JSON失败: %w", err)
			}
		}

		stats = append(stats, &stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代平台统计数据失败: %w", err)
	}

	return stats, nil
}

// DailyStatsExists 检查某一天的每日统计数据是否存在
func (r *StatsRepository) DailyStatsExists(ctx context.Context, tenantID, videoID uuid.UUID, date time.Time) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM daily_stats 
			WHERE tenant_id = $1 AND video_id = $2 AND date = $3
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, tenantID, videoID, date).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("检查每日统计数据是否存在失败: %w", err)
	}

	return exists, nil
}
