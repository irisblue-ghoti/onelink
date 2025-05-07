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

				// 解析token，这里应该使用配置中的密钥
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
