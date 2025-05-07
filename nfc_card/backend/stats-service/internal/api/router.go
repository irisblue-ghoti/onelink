package api

import (
	"stats-service/internal/api/handlers"
	"stats-service/internal/api/middleware"
	"stats-service/internal/config"
	"stats-service/internal/services"

	"github.com/gin-gonic/gin"
)

// NewRouter 创建API路由
func NewRouter(cfg *config.Config, statsService *services.StatsService) *gin.Engine {
	router := gin.Default()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())

	// 健康检查路由
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// 初始化处理程序
	statsHandler := handlers.NewStatsHandler(statsService)
	exportHandler := handlers.NewExportHandler(statsService)

	// API路由组 - 公共路由
	apiV1 := router.Group("/api/v1")
	{
		// 测试路由
		apiV1.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "测试成功",
			})
		})
	}

	// API路由组 - 受保护路由
	protectedAPI := router.Group("/api/v1")
	protectedAPI.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	{
		// 统计相关路由
		stats := protectedAPI.Group("/stats")
		stats.Use(middleware.TenantAuthMiddleware(cfg.JWT.Secret))
		{
			// 平台统计数据
			stats.GET("/platform/:videoId/:platform/:platformId", statsHandler.GetPlatformStats)

			// 视频总体统计数据
			stats.GET("/video/:videoId", statsHandler.GetVideoStats)

			// 视频统计数据概览
			stats.GET("/overview/:videoId", statsHandler.GetStatsOverview)

			// 手动刷新统计数据
			stats.POST("/refresh/:videoId/:platform/:platformId", statsHandler.RefreshPlatformStats)

			// 按时间段统计
			stats.GET("/daily/:videoId", statsHandler.GetDailyStats)

			// 按时间范围统计
			stats.GET("/time-range/:videoId", statsHandler.GetStatsTimeRange)

			// 导出每日统计数据
			stats.GET("/export/daily", exportHandler.ExportDailyStats)

			// 导出平台统计数据
			stats.GET("/export/platform", exportHandler.ExportPlatformStats)

			// 导出所有平台统计数据
			stats.GET("/export/all", exportHandler.ExportAllPlatformsData)

			// 导出统计数据趋势
			stats.GET("/export/trends", exportHandler.ExportStatsTrends)
		}
	}

	return router
}
