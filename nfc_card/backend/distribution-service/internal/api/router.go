package api

import (
	"distribution-service/internal/api/handlers"
	"distribution-service/internal/api/middleware"
	"distribution-service/internal/config"
	"distribution-service/internal/domain/repositories"
	"distribution-service/internal/services"
	"distribution-service/internal/storage"

	"github.com/gin-gonic/gin"
)

// NewRouter 创建路由
func NewRouter(
	cfg *config.Config,
	jobRepo repositories.JobRepository,
	videoRepo repositories.VideoRepository,
	kafkaProducer services.KafkaProducer,
	storageService storage.StorageService,
) *gin.Engine {
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
	publishHandler := handlers.NewPublishHandler(
		jobRepo,
		videoRepo,
		cfg,
		kafkaProducer,
		storageService,
	)

	// API路由组 - 公共路由
	apiV1 := router.Group("/api/v1")
	{
		// 测试路由
		apiV1.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
	}

	// API路由组 - 受保护路由
	protectedAPI := router.Group("/api/v1")
	protectedAPI.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	{
		// 分发相关路由
		publish := protectedAPI.Group("/publish")
		publish.Use(middleware.TenantAuthMiddleware(cfg.JWT.Secret))
		{
			// 创建分发任务
			publish.POST("/jobs", publishHandler.CreateJob)

			// 获取分发任务列表
			publish.GET("/jobs", publishHandler.ListJobs)

			// 获取单个分发任务
			publish.GET("/jobs/:id", publishHandler.GetJob)

			// 获取平台发布状态
			publish.GET("/status/:channel/:platform_id", publishHandler.GetPublishStatus)

			// 生成分享链接
			publish.POST("/share/:channel", publishHandler.GenerateShareLink)

			// 生成JSSDK配置
			publish.POST("/jsconfig/:channel", publishHandler.GenerateJSConfig)

			// 获取详细统计数据
			publish.GET("/stats/:channel/:platform_id", publishHandler.GetDetailedStats)
		}
	}

	return router
}
