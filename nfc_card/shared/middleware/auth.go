package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nfc_card/shared/auth"
)

// 用户角色常量
const (
	RoleAdmin    = "admin"
	RoleMerchant = "merchant"
	RoleUser     = "user"
)

// AuthMiddleware 统一认证中间件
// 验证JWT令牌并将用户信息存储到上下文中
func AuthMiddleware(secretKey string) gin.HandlerFunc {
	jwtService := auth.NewJWTService(secretKey, 0) // 使用默认过期时间

	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证信息"})
			c.Abort()
			return
		}

		// 检查Token格式
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证格式无效"})
			c.Abort()
			return
		}

		// 验证Token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token无效或已过期: " + err.Error()})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("tenantID", claims.TenantID)
		c.Set("roles", claims.Roles)

		// 设置租户ID到请求头，以便于微服务间传递
		c.Request.Header.Set("X-Tenant-ID", claims.TenantID)

		c.Next()
	}
}

// RoleMiddleware 角色验证中间件
// 验证用户是否具有指定角色
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户角色
		rolesInterface, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "无访问权限"})
			c.Abort()
			return
		}

		roles, ok := rolesInterface.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "角色格式错误"})
			c.Abort()
			return
		}

		// 检查用户是否有允许的角色
		hasRole := false
		for _, role := range roles {
			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "无操作权限"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// TenantAuthMiddleware 租户认证中间件
// 验证用户是否有权访问特定租户的资源
func TenantAuthMiddleware() gin.HandlerFunc {
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
					if role == RoleAdmin {
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
