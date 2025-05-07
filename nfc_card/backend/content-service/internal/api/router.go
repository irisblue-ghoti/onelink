package api

import (
	"net/http"

	"content-service/internal/api/handlers"
	"content-service/internal/api/middleware"
	"content-service/internal/config"
	"content-service/internal/services"

	"github.com/gin-gonic/gin"
)

// NewRouter 创建并配置API路由
func NewRouter(cfg *config.Config, contentService *services.ContentService) http.Handler {
	router := gin.Default()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())

	// 健康检查路由
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 初始化处理程序
	videosHandler := handlers.NewVideosHandler(contentService)

	// API路由组 - 公共路由（无需认证）
	apiV1 := router.Group("/api/v1")
	{
		// 测试路由
		apiV1.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "测试成功",
			})
		})
	}

	// API路由组 - 受保护路由（需要认证）
	protectedAPI := router.Group("/api/v1")
	protectedAPI.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	{
		// 视频相关路由
		videos := protectedAPI.Group("/videos")
		videos.Use(middleware.TenantAuthMiddleware(cfg.JWT.Secret))
		{
			// 创建视频
			videos.POST("", videosHandler.Create)

			// 获取视频列表
			videos.GET("", videosHandler.FindAll)

			// 获取单个视频
			videos.GET("/:id", videosHandler.FindOne)

			// 删除视频
			videos.DELETE("/:id", videosHandler.Remove)

			// 转码视频
			videos.POST("/:id/transcode", videosHandler.Transcode)
		}
	}

	return router
}
