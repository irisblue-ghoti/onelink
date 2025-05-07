package storage

import (
	"context"
	"fmt"
	"stats-service/internal/domain/entities"
	"stats-service/internal/domain/repositories"
	"stats-service/internal/services"
	"time"

	"github.com/google/uuid"
)

// RepoAdapter 仓库适配器，使Repositories实现StatsRepo接口
type RepoAdapter struct {
	repos *Repositories
	repo  *repositories.StatsRepository
}

// NewRepoAdapter 创建新的仓库适配器
func NewRepoAdapter(repos *Repositories, repo *repositories.StatsRepository) services.StatsRepo {
	return &RepoAdapter{
		repos: repos,
		repo:  repo,
	}
}

// GetPlatformStats 获取平台统计数据
func (a *RepoAdapter) GetPlatformStats(ctx context.Context, tenantID, videoID uuid.UUID, platform, platformID string) (*entities.PlatformStats, error) {
	return a.repos.GetPlatformStats(ctx, tenantID, videoID, platform, platformID)
}

// SavePlatformStats 保存平台统计数据
func (a *RepoAdapter) SavePlatformStats(ctx context.Context, stats *entities.PlatformStats) error {
	return a.repos.SavePlatformStats(ctx, stats)
}

// GetVideoPlatforms 获取视频的所有平台发布记录
func (a *RepoAdapter) GetVideoPlatforms(ctx context.Context, tenantID, videoID uuid.UUID) ([]struct {
	Platform   string    `db:"platform"`
	PlatformID uuid.UUID `db:"platform_id"`
}, error) {
	if a.repo != nil {
		return a.repo.GetVideoPlatforms(ctx, tenantID, videoID)
	}
	return nil, fmt.Errorf("仓库未初始化")
}

// GetDailyStats 获取每日统计数据
func (a *RepoAdapter) GetDailyStats(ctx context.Context, tenantID, videoID uuid.UUID, platform string, startDate, endDate time.Time) ([]*entities.DailyStats, error) {
	return a.repos.GetDailyStats(ctx, tenantID, videoID, platform, startDate, endDate)
}

// DailyStatsExists 检查每日统计数据是否存在
func (a *RepoAdapter) DailyStatsExists(ctx context.Context, tenantID, videoID uuid.UUID, date time.Time) (bool, error) {
	if a.repo != nil {
		return a.repo.DailyStatsExists(ctx, tenantID, videoID, date)
	}
	return false, fmt.Errorf("仓库未初始化")
}

// SaveDailyStats 保存每日统计数据
func (a *RepoAdapter) SaveDailyStats(ctx context.Context, stats *entities.DailyStats) error {
	return a.repos.SaveDailyStats(ctx, stats)
}

// GetAllPlatformStats 获取所有平台统计数据
func (a *RepoAdapter) GetAllPlatformStats(ctx context.Context) ([]*entities.PlatformStats, error) {
	if a.repo != nil {
		return a.repo.GetAllPlatformStats(ctx)
	}
	return nil, fmt.Errorf("仓库未初始化")
}

// GetPlatformsToRefresh 获取需要刷新的平台
func (a *RepoAdapter) GetPlatformsToRefresh(ctx context.Context, lastUpdateTime time.Time) ([]*entities.PlatformStats, error) {
	if a.repo != nil {
		return a.repo.GetPlatformsToRefresh(ctx, lastUpdateTime)
	}
	return nil, fmt.Errorf("仓库未初始化")
}
