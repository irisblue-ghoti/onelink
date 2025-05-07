package api

import (
	"merchant-service/internal/api/handlers"
	"merchant-service/internal/auth"
	"merchant-service/internal/config"
	"merchant-service/internal/middleware"
	"merchant-service/internal/services"

	"github.com/gin-gonic/gin"
)

// NewRouter 创建API路由
func NewRouter(cfg *config.Config, merchantService *services.MerchantService, userService *services.UserService, authService *auth.JWTService) *gin.Engine {
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

	// 初始化handlers
	authHandler := handlers.NewAuthHandler(authService, userService, merchantService)
	merchantsHandler := handlers.NewMerchantsHandler(merchantService)

	// API路由组
	apiV1 := router.Group("/api/v1")
	{
		// 认证路由 - 无需认证
		auth := apiV1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)

			// 需要认证的路由
			authProtected := auth.Group("")
			authProtected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
			{
				authProtected.GET("/me", authHandler.GetCurrentUser)
			}
		}

		// 商户路由 - 需要认证和权限
		merchants := apiV1.Group("/merchants")
		merchants.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			// 获取商户列表 - 需要 merchants:list 权限
			merchants.GET("", middleware.PermissionMiddleware("merchants", "list"), merchantsHandler.FindAll)

			// 创建商户 - 需要 merchants:create 权限
			merchants.POST("", middleware.PermissionMiddleware("merchants", "create"), merchantsHandler.Create)

			// 获取单个商户 - 需要 merchants:read 权限和资源所有权
			merchants.GET("/:id", middleware.PermissionMiddleware("merchants", "read"),
				middleware.ResourceOwnershipMiddleware("merchant", "id"), merchantsHandler.FindOne)

			// 更新商户 - 需要 merchants:update 权限和资源所有权
			merchants.PUT("/:id", middleware.PermissionMiddleware("merchants", "update"),
				middleware.ResourceOwnershipMiddleware("merchant", "id"), merchantsHandler.Update)

			// 删除商户 - 需要 merchants:delete 权限
			merchants.DELETE("/:id", middleware.PermissionMiddleware("merchants", "delete"), merchantsHandler.Remove)

			// 重新生成API密钥 - 需要 merchants:manage_api_key 权限和资源所有权
			merchants.POST("/:id/api-key", middleware.PermissionMiddleware("merchants", "manage_api_key"),
				middleware.ResourceOwnershipMiddleware("merchant", "id"), merchantsHandler.RegenerateApiKey)

			// 商户审核 - 需要 merchants:update 权限
			merchants.PUT("/:id/approval", middleware.PermissionMiddleware("merchants", "update"), merchantsHandler.UpdateApproval)
		}

		// 用户路由 - 需要认证和权限
		users := apiV1.Group("/users")
		users.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			// TODO: 实现用户相关API路由
		}

		// 角色和权限路由 - 需要认证和权限
		roles := apiV1.Group("/roles")
		roles.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			// TODO: 实现角色和权限相关API路由
		}

		// 部门路由 - 需要认证和权限
		departments := apiV1.Group("/departments")
		departments.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			// TODO: 实现部门相关API路由
		}

		// 商户统计路由 - 需要认证和权限
		statistics := apiV1.Group("/statistics")
		statistics.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			// TODO: 实现统计相关API路由
		}

		// 测试路由
		apiV1.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "测试成功",
			})
		})
	}

	return router
}
