package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// 用户角色
type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleMerchant UserRole = "merchant"
)

// 用户信息
type UserClaims struct {
	UserID     string   `json:"userId"`
	Username   string   `json:"username"`
	MerchantID string   `json:"merchantId,omitempty"`
	Roles      []string `json:"roles"`
	jwt.RegisteredClaims
}

// AuthMiddleware 认证中间件
func AuthMiddleware(secretKey string) gin.HandlerFunc {
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

		// 解析Token
		token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token无效或已过期"})
			c.Abort()
			return
		}

		// 验证Claims
		if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
			// 设置用户信息到上下文
			c.Set("userID", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("merchantID", claims.MerchantID)
			c.Set("roles", claims.Roles)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户信息"})
			c.Abort()
		}
	}
}

// RoleMiddleware 角色中间件
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

// TenantAuthMiddleware 租户认证中间件
func TenantAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户的租户ID
		userTenantID, exists := c.Get("merchantID")
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
					if role == string(RoleAdmin) {
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
