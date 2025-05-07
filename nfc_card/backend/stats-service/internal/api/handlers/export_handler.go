package handlers

import (
	"encoding/csv"
	"net/http"
	"stats-service/internal/domain/entities"
	"stats-service/internal/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ExportHandler 数据导出处理器
type ExportHandler struct {
	statsService *services.StatsService
}

// NewExportHandler 创建新的数据导出处理器
func NewExportHandler(statsService *services.StatsService) *ExportHandler {
	return &ExportHandler{
		statsService: statsService,
	}
}

// ExportDailyStats 导出每日统计数据为CSV
// GET /api/v1/stats/export/daily
func (h *ExportHandler) ExportDailyStats(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取查询参数
	videoID := c.Query("videoId")
	platform := c.Query("platform")
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	// 检查参数
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少视频ID参数"})
		return
	}

	// 使用上海时区
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载时区失败"})
		return
	}

	// 解析日期参数
	now := time.Now().In(location)
	startDate := now.AddDate(0, 0, -30) // 默认30天前
	endDate := now

	if startDateStr != "" {
		parsedStartDate, err := time.ParseInLocation("2006-01-02", startDateStr, location)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "开始日期格式无效，应为YYYY-MM-DD"})
			return
		}
		startDate = parsedStartDate
	}

	if endDateStr != "" {
		parsedEndDate, err := time.ParseInLocation("2006-01-02", endDateStr, location)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "结束日期格式无效，应为YYYY-MM-DD"})
			return
		}
		endDate = parsedEndDate
	}

	// 查询数据
	params := entities.StatsQueryParams{
		TenantID:  tenantIDStr,
		VideoID:   videoID,
		Platform:  platform,
		StartDate: startDate,
		EndDate:   endDate,
	}

	dailyStats, _, err := h.statsService.GetDailyStats(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 设置响应头
	filename := "daily_stats_" + videoID
	if platform != "" {
		filename += "_" + platform
	}
	filename += "_" + time.Now().Format("20060102") + ".csv"

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename="+filename)

	// 创建CSV写入器
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// 写入表头
	headers := []string{"日期", "平台", "播放量", "点赞数", "评论数", "分享数", "收藏数"}
	if err := writer.Write(headers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入CSV表头失败"})
		return
	}

	// 写入数据行
	for _, stat := range dailyStats {
		row := []string{
			stat.Date.Format("2006-01-02"),
			stat.Platform,
			strconv.FormatInt(stat.ViewCount, 10),
			strconv.FormatInt(stat.LikeCount, 10),
			strconv.FormatInt(stat.CommentCount, 10),
			strconv.FormatInt(stat.ShareCount, 10),
			strconv.FormatInt(stat.CollectCount, 10),
		}
		if err := writer.Write(row); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入CSV数据行失败"})
			return
		}
	}
}

// ExportPlatformStats 导出平台统计数据为CSV
// GET /api/v1/stats/export/platform
func (h *ExportHandler) ExportPlatformStats(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取查询参数
	videoID := c.Query("videoId")

	// 检查参数
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少视频ID参数"})
		return
	}

	// 查询数据
	platformStats, err := h.statsService.GetVideoStats(tenantIDStr, videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 设置响应头
	filename := "platform_stats_" + videoID + "_" + time.Now().Format("20060102") + ".csv"
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename="+filename)

	// 创建CSV写入器
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// 写入表头
	headers := []string{"平台", "平台ID", "播放量", "点赞数", "评论数", "分享数", "收藏数", "最后更新时间"}
	if err := writer.Write(headers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入CSV表头失败"})
		return
	}

	// 写入数据行
	for _, stat := range platformStats {
		row := []string{
			stat.Platform,
			stat.PlatformID.String(),
			strconv.FormatInt(stat.ViewCount, 10),
			strconv.FormatInt(stat.LikeCount, 10),
			strconv.FormatInt(stat.CommentCount, 10),
			strconv.FormatInt(stat.ShareCount, 10),
			strconv.FormatInt(stat.CollectCount, 10),
			stat.LastUpdatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(row); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入CSV数据行失败"})
			return
		}
	}
}

// ExportAllPlatformsData 导出所有平台统计数据
// GET /api/v1/stats/export/all
func (h *ExportHandler) ExportAllPlatformsData(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取查询参数
	videoID := c.Query("videoId")
	format := c.Query("format") // 支持csv, json

	// 检查参数
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少视频ID参数"})
		return
	}

	// 默认使用CSV格式
	if format == "" {
		format = "csv"
	}

	// 获取统计数据概览
	overview, err := h.statsService.GetStatsOverview(tenantIDStr, videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 设置文件名
	filename := "stats_overview_" + videoID + "_" + time.Now().Format("20060102")

	// 根据要求的格式导出数据
	switch format {
	case "json":
		// 设置响应头
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename="+filename+".json")

		// 直接输出JSON
		c.JSON(http.StatusOK, overview)

	case "csv":
		// 设置响应头
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename="+filename+".csv")

		// 创建CSV写入器
		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// 写入总量数据
		if err := writer.Write([]string{"指标", "总量"}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入CSV表头失败"})
			return
		}

		totalViews, _ := overview["total_views"].(int64)
		totalLikes, _ := overview["total_likes"].(int64)
		totalComments, _ := overview["total_comments"].(int64)
		totalShares, _ := overview["total_shares"].(int64)

		rows := [][]string{
			{"播放量", strconv.FormatInt(totalViews, 10)},
			{"点赞数", strconv.FormatInt(totalLikes, 10)},
			{"评论数", strconv.FormatInt(totalComments, 10)},
			{"分享数", strconv.FormatInt(totalShares, 10)},
		}

		for _, row := range rows {
			if err := writer.Write(row); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "写入CSV数据行失败"})
				return
			}
		}

		// 写入平台数据
		writer.Write([]string{}) // 空行
		writer.Write([]string{"平台数据"})

		// 平台数据表头
		platformHeaders := []string{"平台", "平台ID", "播放量", "点赞数", "评论数", "分享数", "最后更新时间"}
		if err := writer.Write(platformHeaders); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入CSV表头失败"})
			return
		}

		// 平台数据行
		platforms, _ := overview["platforms"].([]map[string]interface{})
		for _, platform := range platforms {
			platformRow := []string{
				platform["platform"].(string),
				platform["platform_id"].(string),
				strconv.FormatInt(platform["views"].(int64), 10),
				strconv.FormatInt(platform["likes"].(int64), 10),
				strconv.FormatInt(platform["comments"].(int64), 10),
				strconv.FormatInt(platform["shares"].(int64), 10),
				platform["last_updated"].(string),
			}

			if err := writer.Write(platformRow); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "写入CSV数据行失败"})
				return
			}
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的导出格式，仅支持csv和json"})
		return
	}
}

// ExportStatsTrends 导出统计数据趋势
// GET /api/v1/stats/export/trends
func (h *ExportHandler) ExportStatsTrends(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取查询参数
	videoID := c.Query("videoId")
	platform := c.Query("platform")
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")
	format := c.Query("format") // 支持csv, json

	// 检查参数
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少视频ID参数"})
		return
	}

	// 默认使用CSV格式
	if format == "" {
		format = "csv"
	}

	// 使用上海时区
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载时区失败"})
		return
	}

	// 解析日期参数
	now := time.Now().In(location)
	startDate := now.AddDate(0, 0, -30) // 默认30天前
	endDate := now

	if startDateStr != "" {
		parsedStartDate, err := time.ParseInLocation("2006-01-02", startDateStr, location)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "开始日期格式无效，应为YYYY-MM-DD"})
			return
		}
		startDate = parsedStartDate
	}

	if endDateStr != "" {
		parsedEndDate, err := time.ParseInLocation("2006-01-02", endDateStr, location)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "结束日期格式无效，应为YYYY-MM-DD"})
			return
		}
		endDate = parsedEndDate
	}

	// 获取统计数据趋势
	trends, err := h.statsService.GetStatsTimeRange(tenantIDStr, videoID, startDate, endDate, platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 设置文件名
	filename := "stats_trends_" + videoID
	if platform != "" {
		filename += "_" + platform
	}
	filename += "_" + time.Now().Format("20060102")

	// 根据要求的格式导出数据
	switch format {
	case "json":
		// 设置响应头
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename="+filename+".json")

		// 直接输出JSON
		c.JSON(http.StatusOK, trends)

	case "csv":
		// 设置响应头
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename="+filename+".csv")

		// 创建CSV写入器
		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// 写入表头
		headers := []string{"日期", "播放量", "点赞数", "评论数", "分享数"}
		if err := writer.Write(headers); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入CSV表头失败"})
			return
		}

		// 写入数据行
		dates, _ := trends["dates"].([]string)
		views, _ := trends["views"].([]int64)
		likes, _ := trends["likes"].([]int64)
		comments, _ := trends["comments"].([]int64)
		shares, _ := trends["shares"].([]int64)

		for i := 0; i < len(dates); i++ {
			row := []string{
				dates[i],
				strconv.FormatInt(views[i], 10),
				strconv.FormatInt(likes[i], 10),
				strconv.FormatInt(comments[i], 10),
				strconv.FormatInt(shares[i], 10),
			}

			if err := writer.Write(row); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "写入CSV数据行失败"})
				return
			}
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的导出格式，仅支持csv和json"})
		return
	}
}
