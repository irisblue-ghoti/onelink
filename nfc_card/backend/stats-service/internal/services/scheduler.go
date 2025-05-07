package services

import (
	"context"
	"log"
	"stats-service/internal/adapters"
	"stats-service/internal/domain/entities"
	"sync"
	"time"

	"github.com/google/uuid"
)

// StatsScheduler 统计数据调度器
type StatsScheduler struct {
	statsService  *StatsService
	repo          StatsRepo
	logger        *log.Logger
	refreshPeriod time.Duration
	stop          chan struct{}
	wg            sync.WaitGroup
}

// NewStatsScheduler 创建新的统计数据调度器
func NewStatsScheduler(
	statsService *StatsService,
	repo StatsRepo,
	logger *log.Logger,
) *StatsScheduler {
	return &StatsScheduler{
		statsService:  statsService,
		repo:          repo,
		logger:        logger,
		refreshPeriod: 6 * time.Hour, // 默认6小时刷新一次
		stop:          make(chan struct{}),
	}
}

// SetRefreshPeriod 设置刷新周期
func (s *StatsScheduler) SetRefreshPeriod(period time.Duration) {
	s.refreshPeriod = period
}

// Start 启动调度器
func (s *StatsScheduler) Start() {
	s.wg.Add(1)
	go s.scheduledRefresh()
	s.logger.Printf("统计数据调度器已启动，刷新周期: %v", s.refreshPeriod)
}

// Stop 停止调度器
func (s *StatsScheduler) Stop() {
	close(s.stop)
	s.wg.Wait()
	s.logger.Printf("统计数据调度器已停止")
}

// scheduledRefresh 定时刷新统计数据
func (s *StatsScheduler) scheduledRefresh() {
	defer s.wg.Done()

	// 立即执行一次
	s.refreshAll()

	ticker := time.NewTicker(s.refreshPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.refreshAll()
		case <-s.stop:
			return
		}
	}
}

// refreshAll 刷新所有统计数据
func (s *StatsScheduler) refreshAll() {
	s.logger.Printf("开始刷新所有统计数据...")

	// 获取需要刷新的平台发布数据
	// 查询超过24小时未更新的统计数据
	lastUpdateTime := time.Now().Add(-24 * time.Hour)
	platforms, err := s.repo.GetPlatformsToRefresh(context.Background(), lastUpdateTime)
	if err != nil {
		s.logger.Printf("获取需要刷新的平台数据失败: %v", err)
		return
	}

	s.logger.Printf("找到%d个需要刷新的平台数据", len(platforms))

	// 按平台分组
	platformGroups := make(map[string][]string)
	for _, p := range platforms {
		if _, ok := platformGroups[p.Platform]; !ok {
			platformGroups[p.Platform] = []string{}
		}
		platformGroups[p.Platform] = append(platformGroups[p.Platform], p.PlatformID.String())
	}

	// 更新每个平台的数据
	for platform, ids := range platformGroups {
		adapter, ok := s.statsService.GetAdapter(platform)
		if !ok {
			s.logger.Printf("平台 %s 没有适配器，跳过", platform)
			continue
		}

		// 分批处理，避免一次性请求过多
		batchSize := 10
		for i := 0; i < len(ids); i += batchSize {
			end := i + batchSize
			if end > len(ids) {
				end = len(ids)
			}
			batch := ids[i:end]

			s.logger.Printf("刷新平台 %s 的统计数据，批次: %d-%d", platform, i, end)
			s.refreshBatch(adapter, batch)
		}
	}

	// 生成/更新每日统计数据
	s.generateDailyStats()

	s.logger.Printf("所有统计数据刷新完成")
}

// refreshBatch 刷新一批统计数据
func (s *StatsScheduler) refreshBatch(adapter adapters.PlatformAdapter, platformIDs []string) {
	// 使用适配器批量获取数据
	stats, err := adapter.CollectBatchStats(context.Background(), platformIDs)
	if err != nil {
		s.logger.Printf("批量获取平台 %s 统计数据失败: %v", adapter.GetPlatformName(), err)
		return
	}

	// 保存到数据库
	for _, stat := range stats {
		if err := s.repo.SavePlatformStats(context.Background(), stat); err != nil {
			s.logger.Printf("保存平台统计数据失败: %v", err)
		}
	}
}

// generateDailyStats 生成每日统计数据
func (s *StatsScheduler) generateDailyStats() {
	s.logger.Printf("开始生成每日统计数据...")

	// 获取当前日期（使用UTC时间，避免时区问题）
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		s.logger.Printf("加载时区失败，使用默认UTC时区: %v", err)
		location = time.UTC
	}

	// 设置为当地时间的前一天
	now := time.Now().In(location)
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, location)

	// 获取所有平台最新数据
	platforms, err := s.repo.GetAllPlatformStats(context.Background())
	if err != nil {
		s.logger.Printf("获取平台统计数据失败: %v", err)
		return
	}

	// 按照租户和视频ID分组
	videoGroups := make(map[uuid.UUID]map[uuid.UUID][]*entities.PlatformStats)
	for _, p := range platforms {
		if _, ok := videoGroups[p.TenantID]; !ok {
			videoGroups[p.TenantID] = make(map[uuid.UUID][]*entities.PlatformStats)
		}
		if _, ok := videoGroups[p.TenantID][p.VideoID]; !ok {
			videoGroups[p.TenantID][p.VideoID] = []*entities.PlatformStats{}
		}
		videoGroups[p.TenantID][p.VideoID] = append(videoGroups[p.TenantID][p.VideoID], p)
	}

	// 为每个视频生成每日统计数据
	for tenantID, videos := range videoGroups {
		for videoID, stats := range videos {
			// 检查昨天的数据是否已存在
			exists, err := s.repo.DailyStatsExists(context.Background(), tenantID, videoID, yesterday)
			if err != nil {
				s.logger.Printf("检查每日统计数据是否存在失败: %v", err)
				continue
			}

			if exists {
				s.logger.Printf("租户 %s 视频 %s 的昨天统计数据已存在，跳过", tenantID, videoID)
				continue
			}

			// 计算每个平台的每日统计数据
			platformDailyStats := make(map[string]*entities.DailyStats)
			for _, stat := range stats {
				if _, ok := platformDailyStats[stat.Platform]; !ok {
					platformDailyStats[stat.Platform] = &entities.DailyStats{
						ID:        uuid.New(),
						TenantID:  tenantID,
						VideoID:   videoID,
						NfcCardID: stat.NfcCardID,
						Platform:  stat.Platform,
						Date:      yesterday,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
				}

				// 累加数据
				platformDailyStats[stat.Platform].ViewCount += stat.ViewCount
				platformDailyStats[stat.Platform].LikeCount += stat.LikeCount
				platformDailyStats[stat.Platform].CommentCount += stat.CommentCount
				platformDailyStats[stat.Platform].ShareCount += stat.ShareCount
				platformDailyStats[stat.Platform].CollectCount += stat.CollectCount
			}

			// 保存每日统计数据
			for _, dailyStat := range platformDailyStats {
				if err := s.repo.SaveDailyStats(context.Background(), dailyStat); err != nil {
					s.logger.Printf("保存每日统计数据失败: %v", err)
				}
			}

			s.logger.Printf("已为租户 %s 视频 %s 生成 %d 个平台的每日统计数据",
				tenantID, videoID, len(platformDailyStats))
		}
	}

	s.logger.Printf("每日统计数据生成完成")
}
