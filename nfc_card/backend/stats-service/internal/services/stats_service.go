package services

import (
	"context"
	"log"
	"stats-service/internal/adapters"
	"stats-service/internal/domain/entities"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// StatsRepo 统计数据仓库接口
type StatsRepo interface {
	GetPlatformStats(ctx context.Context, tenantID, videoID uuid.UUID, platform, platformID string) (*entities.PlatformStats, error)
	SavePlatformStats(ctx context.Context, stats *entities.PlatformStats) error
	GetVideoPlatforms(ctx context.Context, tenantID, videoID uuid.UUID) ([]struct {
		Platform   string    `db:"platform"`
		PlatformID uuid.UUID `db:"platform_id"`
	}, error)
	GetDailyStats(ctx context.Context, tenantID, videoID uuid.UUID, platform string, startDate, endDate time.Time) ([]*entities.DailyStats, error)
	DailyStatsExists(ctx context.Context, tenantID, videoID uuid.UUID, date time.Time) (bool, error)
	SaveDailyStats(ctx context.Context, stats *entities.DailyStats) error
	GetAllPlatformStats(ctx context.Context) ([]*entities.PlatformStats, error)
	GetPlatformsToRefresh(ctx context.Context, lastUpdateTime time.Time) ([]*entities.PlatformStats, error)
}

// StatsService 统计服务
type StatsService struct {
	repo     StatsRepo
	logger   *log.Logger
	adapters map[string]adapters.PlatformAdapter
}

// NewStatsService 创建统计服务
func NewStatsService(repo StatsRepo, logger *log.Logger) *StatsService {
	return &StatsService{
		repo:     repo,
		logger:   logger,
		adapters: make(map[string]adapters.PlatformAdapter),
	}
}

// RegisterAdapter 注册平台适配器
func (s *StatsService) RegisterAdapter(adapter adapters.PlatformAdapter) {
	s.adapters[adapter.GetPlatformName()] = adapter
}

// GetPlatformStats 获取平台统计数据
func (s *StatsService) GetPlatformStats(tenantID, videoID, platform, platformID string) (*entities.PlatformStats, error) {
	s.logger.Printf("获取平台统计数据: 租户=%s, 视频=%s, 平台=%s, 平台ID=%s",
		tenantID, videoID, platform, platformID)

	// 转换UUID字符串为UUID类型
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "无效的租户ID")
	}

	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		return nil, errors.Wrap(err, "无效的视频ID")
	}

	// 1. 先从数据库获取最新缓存的统计数据
	stats, err := s.repo.GetPlatformStats(context.Background(), tenantUUID, videoUUID, platform, platformID)
	if err != nil {
		return nil, errors.Wrap(err, "从数据库获取统计数据失败")
	}

	// 2. 如果数据存在且更新时间在24小时内，直接返回
	if stats != nil && time.Since(stats.LastUpdatedAt) < 24*time.Hour {
		return stats, nil
	}

	// 3. 如果数据不存在或已过期，尝试从平台获取最新数据
	adapter, ok := s.adapters[platform]
	if !ok {
		return nil, errors.Errorf("不支持的平台: %s", platform)
	}

	newStats, err := adapter.CollectStats(context.Background(), platformID)
	if err != nil {
		// 获取失败但有缓存数据，返回缓存数据
		if stats != nil {
			s.logger.Printf("获取平台最新统计数据失败，返回缓存数据: %v", err)
			return stats, nil
		}
		return nil, errors.Wrap(err, "获取平台统计数据失败")
	}

	// 4. 更新数据库
	newStats.TenantID = tenantUUID
	newStats.VideoID = videoUUID
	newStats.Platform = platform

	// 尝试解析平台ID为UUID
	platformUUID, err := uuid.Parse(platformID)
	if err != nil {
		// 如果无法解析，则生成一个基于字符串的UUID
		platformUUID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(platformID))
	}
	newStats.PlatformID = platformUUID

	if err := s.repo.SavePlatformStats(context.Background(), newStats); err != nil {
		s.logger.Printf("保存平台统计数据失败: %v", err)
	}

	return newStats, nil
}

// GetVideoStats 获取视频所有平台的统计数据
func (s *StatsService) GetVideoStats(tenantID, videoID string) ([]*entities.PlatformStats, error) {
	s.logger.Printf("获取视频所有平台统计数据: 租户=%s, 视频=%s", tenantID, videoID)

	// 转换UUID字符串为UUID类型
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "无效的租户ID")
	}

	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		return nil, errors.Wrap(err, "无效的视频ID")
	}

	// 获取视频的所有平台发布记录
	platforms, err := s.repo.GetVideoPlatforms(context.Background(), tenantUUID, videoUUID)
	if err != nil {
		return nil, errors.Wrap(err, "获取视频平台发布记录失败")
	}

	if len(platforms) == 0 {
		return []*entities.PlatformStats{}, nil
	}

	// 获取所有平台的统计数据
	allStats := make([]*entities.PlatformStats, 0, len(platforms))
	for _, p := range platforms {
		stats, err := s.GetPlatformStats(tenantID, videoID, p.Platform, p.PlatformID.String())
		if err != nil {
			s.logger.Printf("获取平台[%s]统计数据失败: %v", p.Platform, err)
			continue
		}
		allStats = append(allStats, stats)
	}

	return allStats, nil
}

// GetDailyStats 获取每日统计数据
func (s *StatsService) GetDailyStats(params entities.StatsQueryParams) ([]*entities.DailyStats, int, error) {
	s.logger.Printf("获取每日统计数据: 租户=%s, 视频=%s, 平台=%s, 开始=%s, 结束=%s",
		params.TenantID, params.VideoID, params.Platform, params.StartDate.Format("2006-01-02"), params.EndDate.Format("2006-01-02"))

	// 转换UUID字符串为UUID类型
	tenantUUID, err := uuid.Parse(params.TenantID)
	if err != nil {
		return nil, 0, errors.Wrap(err, "无效的租户ID")
	}

	videoUUID, err := uuid.Parse(params.VideoID)
	if err != nil {
		return nil, 0, errors.Wrap(err, "无效的视频ID")
	}

	// 从数据库获取每日统计数据
	dailyStats, err := s.repo.GetDailyStats(
		context.Background(),
		tenantUUID,
		videoUUID,
		params.Platform,
		params.StartDate,
		params.EndDate,
	)

	if err != nil {
		return nil, 0, errors.Wrap(err, "获取每日统计数据失败")
	}

	// 手动分页处理
	total := len(dailyStats)

	// 如果没有设置分页参数，返回全部
	if params.Page <= 0 || params.PageSize <= 0 {
		return dailyStats, total, nil
	}

	// 计算开始和结束索引
	startIndex := (params.Page - 1) * params.PageSize
	endIndex := startIndex + params.PageSize

	// 检查索引范围
	if startIndex >= total {
		return []*entities.DailyStats{}, total, nil
	}

	if endIndex > total {
		endIndex = total
	}

	return dailyStats[startIndex:endIndex], total, nil
}

// GetStatsTimeRange 获取指定时间范围内的统计数据
func (s *StatsService) GetStatsTimeRange(tenantID, videoID string, startDate, endDate time.Time, platform string) (map[string]interface{}, error) {
	s.logger.Printf("获取时间范围内统计数据: 租户=%s, 视频=%s, 平台=%s, 开始=%s, 结束=%s",
		tenantID, videoID, platform, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// 获取指定时间范围内的每日统计数据
	params := entities.StatsQueryParams{
		TenantID:  tenantID,
		VideoID:   videoID,
		Platform:  platform,
		StartDate: startDate,
		EndDate:   endDate,
	}

	dailyStats, _, err := s.GetDailyStats(params)
	if err != nil {
		return nil, errors.Wrap(err, "获取时间范围内统计数据失败")
	}

	// 统计数据汇总
	result := make(map[string]interface{})

	// 每日数据序列
	dates := make([]string, 0, len(dailyStats))
	views := make([]int64, 0, len(dailyStats))
	likes := make([]int64, 0, len(dailyStats))
	comments := make([]int64, 0, len(dailyStats))
	shares := make([]int64, 0, len(dailyStats))

	// 累计值
	var totalViews int64 = 0
	var totalLikes int64 = 0
	var totalComments int64 = 0
	var totalShares int64 = 0

	for _, stat := range dailyStats {
		dates = append(dates, stat.Date.Format("2006-01-02"))
		views = append(views, stat.ViewCount)
		likes = append(likes, stat.LikeCount)
		comments = append(comments, stat.CommentCount)
		shares = append(shares, stat.ShareCount)

		totalViews += stat.ViewCount
		totalLikes += stat.LikeCount
		totalComments += stat.CommentCount
		totalShares += stat.ShareCount
	}

	// 返回数据构造
	result["dates"] = dates
	result["views"] = views
	result["likes"] = likes
	result["comments"] = comments
	result["shares"] = shares

	result["total_views"] = totalViews
	result["total_likes"] = totalLikes
	result["total_comments"] = totalComments
	result["total_shares"] = totalShares

	return result, nil
}

// GetStatsOverview 获取统计数据概览
func (s *StatsService) GetStatsOverview(tenantID, videoID string) (map[string]interface{}, error) {
	s.logger.Printf("获取统计数据概览: 租户=%s, 视频=%s", tenantID, videoID)

	// 1. 获取视频所有平台的统计数据
	allStats, err := s.GetVideoStats(tenantID, videoID)
	if err != nil {
		return nil, errors.Wrap(err, "获取视频所有平台统计数据失败")
	}

	// 2. 计算总量和平台分布
	overview := map[string]interface{}{
		"total_views":    int64(0),
		"total_likes":    int64(0),
		"total_comments": int64(0),
		"total_shares":   int64(0),
		"platforms":      make([]map[string]interface{}, 0, len(allStats)),
	}

	for _, stats := range allStats {
		// 累加总量
		overview["total_views"] = overview["total_views"].(int64) + stats.ViewCount
		overview["total_likes"] = overview["total_likes"].(int64) + stats.LikeCount
		overview["total_comments"] = overview["total_comments"].(int64) + stats.CommentCount
		overview["total_shares"] = overview["total_shares"].(int64) + stats.ShareCount

		// 添加平台数据
		platformData := map[string]interface{}{
			"platform":     stats.Platform,
			"platform_id":  stats.PlatformID.String(),
			"views":        stats.ViewCount,
			"likes":        stats.LikeCount,
			"comments":     stats.CommentCount,
			"shares":       stats.ShareCount,
			"last_updated": stats.LastUpdatedAt.Format(time.RFC3339),
		}
		overview["platforms"] = append(overview["platforms"].([]map[string]interface{}), platformData)
	}

	// 3. 获取近7天数据趋势
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		s.logger.Printf("加载时区失败，使用默认UTC时区: %v", err)
		location = time.UTC
	}

	// 使用本地时区的当前时间
	now := time.Now().In(location)
	// 往前推7天，保持在同一时区
	startDate := now.AddDate(0, 0, -7)

	// 确保日期范围在当天的0点
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, location)
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, location)

	timeRange, err := s.GetStatsTimeRange(tenantID, videoID, startDate, endDate, "")
	if err != nil {
		s.logger.Printf("获取近7天数据趋势失败: %v", err)
	} else {
		overview["trends"] = timeRange
	}

	// 4. 添加数据分析结果
	analysisResult := s.analyzeStats(allStats)
	if analysisResult != nil {
		overview["analysis"] = analysisResult
	}

	// 5. 添加增长率统计
	if len(allStats) > 0 && timeRange != nil {
		growthRates := s.calculateGrowthRates(timeRange)
		if len(growthRates) > 0 {
			overview["growth_rates"] = growthRates
		}
	}

	// 6. 添加平台分布百分比
	if len(allStats) > 0 {
		platformDistribution := s.calculatePlatformDistribution(allStats)
		if len(platformDistribution) > 0 {
			overview["platform_distribution"] = platformDistribution
		}
	}

	return overview, nil
}

// analyzeStats 分析统计数据，提供数据洞察
func (s *StatsService) analyzeStats(stats []*entities.PlatformStats) map[string]interface{} {
	if len(stats) == 0 {
		return nil
	}

	// 初始化分析结果
	analysis := map[string]interface{}{}

	// 计算各平台占比
	platformShares := make(map[string]float64)
	var totalViews int64 = 0

	// 找出表现最好的平台
	var bestPlatform string
	var bestPlatformViews int64 = 0

	// 找出表现最差的平台
	var worstPlatform string
	var worstPlatformViews int64 = 0

	// 初始化最差平台为第一个平台
	if len(stats) > 0 {
		worstPlatform = stats[0].Platform
		worstPlatformViews = stats[0].ViewCount
	}

	for _, stat := range stats {
		totalViews += stat.ViewCount

		if stat.ViewCount > bestPlatformViews {
			bestPlatformViews = stat.ViewCount
			bestPlatform = stat.Platform
		}

		if stat.ViewCount < worstPlatformViews {
			worstPlatformViews = stat.ViewCount
			worstPlatform = stat.Platform
		}
	}

	// 计算占比
	for _, stat := range stats {
		if totalViews > 0 {
			platformShares[stat.Platform] = float64(stat.ViewCount) / float64(totalViews) * 100
		} else {
			platformShares[stat.Platform] = 0
		}
	}

	analysis["platform_shares"] = platformShares
	if bestPlatform != "" {
		analysis["best_platform"] = bestPlatform
		analysis["best_platform_views"] = bestPlatformViews
	}

	if worstPlatform != "" && len(stats) > 1 {
		analysis["worst_platform"] = worstPlatform
		analysis["worst_platform_views"] = worstPlatformViews
	}

	// 计算平均每个平台的数据
	if len(stats) > 0 {
		analysis["avg_views_per_platform"] = totalViews / int64(len(stats))
	}

	return analysis
}

// calculateGrowthRates 计算增长率
func (s *StatsService) calculateGrowthRates(timeRangeData map[string]interface{}) map[string]float64 {
	growthRates := make(map[string]float64)

	// 检查是否有足够的数据计算增长率
	dates, ok := timeRangeData["dates"].([]string)
	if !ok || len(dates) < 2 {
		return growthRates
	}

	// 获取最近一天和最早一天的数据
	views, ok := timeRangeData["views"].([]int64)
	if ok && len(views) >= 2 {
		firstDayViews := views[0]
		lastDayViews := views[len(views)-1]

		if firstDayViews > 0 {
			growthRate := (float64(lastDayViews) - float64(firstDayViews)) / float64(firstDayViews) * 100
			growthRates["views"] = growthRate
		}
	}

	likes, ok := timeRangeData["likes"].([]int64)
	if ok && len(likes) >= 2 {
		firstDayLikes := likes[0]
		lastDayLikes := likes[len(likes)-1]

		if firstDayLikes > 0 {
			growthRate := (float64(lastDayLikes) - float64(firstDayLikes)) / float64(firstDayLikes) * 100
			growthRates["likes"] = growthRate
		}
	}

	comments, ok := timeRangeData["comments"].([]int64)
	if ok && len(comments) >= 2 {
		firstDayComments := comments[0]
		lastDayComments := comments[len(comments)-1]

		if firstDayComments > 0 {
			growthRate := (float64(lastDayComments) - float64(firstDayComments)) / float64(firstDayComments) * 100
			growthRates["comments"] = growthRate
		}
	}

	shares, ok := timeRangeData["shares"].([]int64)
	if ok && len(shares) >= 2 {
		firstDayShares := shares[0]
		lastDayShares := shares[len(shares)-1]

		if firstDayShares > 0 {
			growthRate := (float64(lastDayShares) - float64(firstDayShares)) / float64(firstDayShares) * 100
			growthRates["shares"] = growthRate
		}
	}

	return growthRates
}

// calculatePlatformDistribution 计算平台分布
func (s *StatsService) calculatePlatformDistribution(stats []*entities.PlatformStats) map[string]map[string]float64 {
	distribution := make(map[string]map[string]float64)

	// 初始化各指标的总和
	totalViews := int64(0)
	totalLikes := int64(0)
	totalComments := int64(0)
	totalShares := int64(0)

	// 计算各项总和
	for _, stat := range stats {
		totalViews += stat.ViewCount
		totalLikes += stat.LikeCount
		totalComments += stat.CommentCount
		totalShares += stat.ShareCount
	}

	// 计算各平台占比
	viewsDistribution := make(map[string]float64)
	likesDistribution := make(map[string]float64)
	commentsDistribution := make(map[string]float64)
	sharesDistribution := make(map[string]float64)

	for _, stat := range stats {
		if totalViews > 0 {
			viewsDistribution[stat.Platform] = float64(stat.ViewCount) / float64(totalViews) * 100
		}
		if totalLikes > 0 {
			likesDistribution[stat.Platform] = float64(stat.LikeCount) / float64(totalLikes) * 100
		}
		if totalComments > 0 {
			commentsDistribution[stat.Platform] = float64(stat.CommentCount) / float64(totalComments) * 100
		}
		if totalShares > 0 {
			sharesDistribution[stat.Platform] = float64(stat.ShareCount) / float64(totalShares) * 100
		}
	}

	distribution["views"] = viewsDistribution
	distribution["likes"] = likesDistribution
	distribution["comments"] = commentsDistribution
	distribution["shares"] = sharesDistribution

	return distribution
}

// RefreshPlatformStats 刷新平台统计数据
func (s *StatsService) RefreshPlatformStats(tenantID, videoID, platform, platformID string) (*entities.PlatformStats, error) {
	s.logger.Printf("刷新平台统计数据: 租户=%s, 视频=%s, 平台=%s, 平台ID=%s",
		tenantID, videoID, platform, platformID)

	// 1. 获取平台适配器
	adapter, ok := s.adapters[platform]
	if !ok {
		return nil, errors.Errorf("不支持的平台: %s", platform)
	}

	// 2. 从平台获取最新数据
	newStats, err := adapter.CollectStats(context.Background(), platformID)
	if err != nil {
		return nil, errors.Wrap(err, "获取平台统计数据失败")
	}

	// 3. 更新数据库
	// 转换UUID字符串为UUID类型
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "无效的租户ID")
	}

	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		return nil, errors.Wrap(err, "无效的视频ID")
	}

	newStats.TenantID = tenantUUID
	newStats.VideoID = videoUUID
	newStats.Platform = platform

	// 尝试解析平台ID为UUID
	platformUUID, err := uuid.Parse(platformID)
	if err != nil {
		// 如果无法解析，则生成一个基于字符串的UUID
		platformUUID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(platformID))
	}
	newStats.PlatformID = platformUUID

	if err := s.repo.SavePlatformStats(context.Background(), newStats); err != nil {
		return nil, errors.Wrap(err, "保存平台统计数据失败")
	}

	return newStats, nil
}

// GetAdapter 获取指定平台的适配器
func (s *StatsService) GetAdapter(platform string) (adapters.PlatformAdapter, bool) {
	adapter, ok := s.adapters[platform]
	return adapter, ok
}
