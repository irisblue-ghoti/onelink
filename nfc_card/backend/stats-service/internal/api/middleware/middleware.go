package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// TenantMiddleware 租户中间件
func TenantMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取租户ID
		tenantID := c.GetHeader("X-Tenant-ID")

		// 如果没有提供租户ID，则尝试从查询参数获取
		if tenantID == "" {
			tenantID = c.Query("tenantId")
		}

		// 如果没有提供租户ID，则尝试从Bearer Token解析
		if tenantID == "" {
			// 从Authorization头部获取token
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")

				// 解析token，使用配置中的密钥
				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					// 使用配置中的JWT密钥
					return []byte(jwtSecret), nil
				})

				if err == nil && token.Valid {
					// 从token claims中获取租户ID
					if claims, ok := token.Claims.(jwt.MapClaims); ok {
						if tenantIDClaim, exists := claims["tenant_id"]; exists {
							if tenantIDStr, ok := tenantIDClaim.(string); ok {
								tenantID = tenantIDStr
							}
						}
					}
				}
			}
		}

		// 如果仍然没有租户ID，返回错误
		if tenantID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供租户ID"})
			c.Abort()
			return
		}

		// 将租户ID设置到上下文中
		c.Set("tenantID", tenantID)
		c.Next()
	}
}

// TenantAuthMiddleware 租户认证中间件
func TenantAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户的租户ID
		userTenantID, exists := c.Get("tenantID")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "无访问权限"})
			c.Abort()
			return
		}

		// 获取请求中的租户ID参数
		requestTenantID := c.Param("tenantId")
		if requestTenantID == "" {
			requestTenantID = c.Query("tenantId")
		}

		// 如果请求中没有指定租户ID，则使用用户的租户ID
		if requestTenantID == "" {
			c.Set("tenantID", userTenantID)
			c.Request.Header.Set("X-Tenant-ID", userTenantID.(string))
			c.Next()
			return
		}

		// 检查用户是否为管理员
		rolesInterface, exists := c.Get("roles")
		if exists {
			roles, ok := rolesInterface.([]string)
			if ok {
				for _, role := range roles {
					if role == "admin" {
						// 管理员可以访问任何租户的资源
						c.Set("tenantID", requestTenantID)
						c.Request.Header.Set("X-Tenant-ID", requestTenantID)
						c.Next()
						return
					}
				}
			}
		}

		// 非管理员只能访问自己租户的资源
		if userTenantID.(string) != requestTenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权访问其他租户资源"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Tenant-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
