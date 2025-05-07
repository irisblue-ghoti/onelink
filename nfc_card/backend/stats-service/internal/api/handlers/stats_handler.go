package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"stats-service/internal/domain/entities"
	"stats-service/internal/services"
)

// StatsHandler 处理统计数据相关的API请求
type StatsHandler struct {
	statsService *services.StatsService
}

// NewStatsHandler 创建一个新的统计数据处理器
func NewStatsHandler(statsService *services.StatsService) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
	}
}

// GetPlatformStats 获取特定平台的统计数据
// GET /api/v1/stats/platform/:videoId/:platform/:platformId
func (h *StatsHandler) GetPlatformStats(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取参数
	videoID := c.Param("videoId")
	platform := c.Param("platform")
	platformID := c.Param("platformId")

	// 检查参数
	if videoID == "" || platform == "" || platformID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数"})
		return
	}

	// 获取统计数据
	stats, err := h.statsService.GetPlatformStats(tenantIDStr, videoID, platform, platformID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetVideoStats 获取视频在所有平台的统计数据
// GET /api/v1/stats/video/:videoId
func (h *StatsHandler) GetVideoStats(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取视频ID
	videoID := c.Param("videoId")
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少视频ID参数"})
		return
	}

	// 获取统计数据
	stats, err := h.statsService.GetVideoStats(tenantIDStr, videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetDailyStats 获取每日统计数据
// GET /api/v1/stats/daily
func (h *StatsHandler) GetDailyStats(c *gin.Context) {
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

	// 获取分页参数
	page := 1
	pageSize := 20

	if pageParam := c.Query("page"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	if pageSizeParam := c.Query("pageSize"); pageSizeParam != "" {
		if parsedPageSize, err := strconv.Atoi(pageSizeParam); err == nil && parsedPageSize > 0 {
			pageSize = parsedPageSize
		}
	}

	// 处理日期参数和时区
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载时区失败"})
		return
	}

	// 设置默认日期范围为最近30天
	now := time.Now().In(location)
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location).AddDate(0, 0, -30)
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, location)

	// 解析日期参数
	if startDateStr != "" {
		parsedStartDate, err := time.ParseInLocation("2006-01-02", startDateStr, location)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "开始日期格式无效，应为YYYY-MM-DD"})
			return
		}
		// 确保时间是当天的0点
		startDate = time.Date(parsedStartDate.Year(), parsedStartDate.Month(), parsedStartDate.Day(), 0, 0, 0, 0, location)
	}

	if endDateStr != "" {
		parsedEndDate, err := time.ParseInLocation("2006-01-02", endDateStr, location)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "结束日期格式无效，应为YYYY-MM-DD"})
			return
		}
		// 确保时间是当天的23:59:59
		endDate = time.Date(parsedEndDate.Year(), parsedEndDate.Month(), parsedEndDate.Day(), 23, 59, 59, 0, location)
	}

	// 构建查询参数
	params := entities.StatsQueryParams{
		TenantID:  tenantIDStr,
		VideoID:   videoID,
		Platform:  platform,
		StartDate: startDate,
		EndDate:   endDate,
		Page:      page,
		PageSize:  pageSize,
	}

	// 获取统计数据
	stats, total, err := h.statsService.GetDailyStats(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 计算总页数
	totalPages := (total + pageSize - 1) / pageSize

	// 返回分页响应
	c.JSON(http.StatusOK, gin.H{
		"data": stats,
		"meta": gin.H{
			"currentPage":  page,
			"itemsPerPage": pageSize,
			"totalItems":   total,
			"totalPages":   totalPages,
		},
	})
}

// GetStatsTimeRange 获取指定时间范围内的统计数据
// GET /api/v1/stats/time-range/:videoId
func (h *StatsHandler) GetStatsTimeRange(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取视频ID
	videoID := c.Param("videoId")
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少视频ID参数"})
		return
	}

	// 获取查询参数
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")
	platform := c.Query("platform")

	// 检查日期参数
	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "开始日期和结束日期参数为必填"})
		return
	}

	// 处理时区
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载时区失败"})
		return
	}

	// 解析日期，使用上海时区
	startDate, err := time.ParseInLocation("2006-01-02", startDateStr, location)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "开始日期格式无效，应为YYYY-MM-DD"})
		return
	}

	endDate, err := time.ParseInLocation("2006-01-02", endDateStr, location)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "结束日期格式无效，应为YYYY-MM-DD"})
		return
	}

	// 设置时间为当天的起始和结束
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, location)
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, location)

	// 获取统计数据
	stats, err := h.statsService.GetStatsTimeRange(tenantIDStr, videoID, startDate, endDate, platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetStatsOverview 获取统计数据概览
// GET /api/v1/stats/overview/:videoId
func (h *StatsHandler) GetStatsOverview(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取视频ID
	videoID := c.Param("videoId")
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少视频ID参数"})
		return
	}

	// 获取统计数据概览
	overview, err := h.statsService.GetStatsOverview(tenantIDStr, videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, overview)
}

// RefreshPlatformStats 手动刷新平台统计数据
// POST /api/v1/stats/refresh/:videoId/:platform/:platformId
func (h *StatsHandler) RefreshPlatformStats(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取参数
	videoID := c.Param("videoId")
	platform := c.Param("platform")
	platformID := c.Param("platformId")

	// 检查参数
	if videoID == "" || platform == "" || platformID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数"})
		return
	}

	// 刷新统计数据
	stats, err := h.statsService.RefreshPlatformStats(tenantIDStr, videoID, platform, platformID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
