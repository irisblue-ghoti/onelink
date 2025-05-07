package api

import (
	"nfc-service/internal/api/handlers"
	"nfc-service/internal/api/middleware"
	"nfc-service/internal/config"
	"nfc-service/internal/services/cards"
	"nfc-service/internal/services/shortlinks"

	"github.com/gin-gonic/gin"
)

// NewRouter 创建并配置API路由器
func NewRouter(cfg *config.Config, cardService cards.Service, shortlinkService *shortlinks.ShortlinkService) *gin.Engine {
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
	cardHandler := handlers.NewCardHandler(cardService)
	shortLinkHandler := handlers.NewShortLinkHandler(shortlinkService, cfg.ShortLink.BaseURL)

	// API路由组 - 公共路由
	apiV1 := router.Group("/api/v1")
	{
		// 重定向路由，用于短链接访问（无需认证）
		apiV1.GET("/r/:slug", shortLinkHandler.RedirectToTarget)

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
		// NFC卡片路由
		nfcCards := protectedAPI.Group("/nfc-cards")
		nfcCards.Use(middleware.TenantAuthMiddleware(cfg.JWT.Secret))
		{
			nfcCards.POST("", cardHandler.CreateCard)
			nfcCards.GET("", cardHandler.GetCards)
			nfcCards.GET("/:id", cardHandler.GetCardByID)
			nfcCards.PUT("/:id", cardHandler.UpdateCard)
			nfcCards.DELETE("/:id", cardHandler.DeleteCard)
			nfcCards.POST("/activate", cardHandler.ActivateCard)
		}

		// 短链接路由
		shortlinks := protectedAPI.Group("/shortlinks")
		shortlinks.Use(middleware.TenantAuthMiddleware(cfg.JWT.Secret))
		{
			shortlinks.POST("", shortLinkHandler.CreateShortLink)
			shortlinks.GET("/:id", shortLinkHandler.GetShortLinkByID)
			shortlinks.GET("/slug/:slug", shortLinkHandler.GetShortLinkBySlug)
			shortlinks.GET("/merchant/:merchantID", shortLinkHandler.GetShortLinksByMerchantID)
			shortlinks.GET("/card/:cardID", shortLinkHandler.GetShortLinksByNfcCardID)
			shortlinks.PUT("/:id", shortLinkHandler.UpdateShortLink)
			shortlinks.DELETE("/:id", shortLinkHandler.DeleteShortLink)
		}
	}

	return router
}
