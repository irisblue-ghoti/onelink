package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// 用户角色常量
const (
	RoleAdmin    = "admin"
	RoleMerchant = "merchant"
	RoleUser     = "user"
)

// Claims JWT令牌声明结构
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	TenantID string   `json:"tenant_id"` // 商户/租户ID
	Roles    []string `json:"roles"`     // 用户角色列表
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
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// 验证签名算法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.NewValidationError("无效的签名方法", jwt.ValidationErrorSignatureInvalid)
			}
			return []byte(secretKey), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token无效或已过期: " + err.Error()})
			c.Abort()
			return
		}

		// 验证Claims
		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			// 设置用户信息到上下文
			c.Set("userID", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("tenantID", claims.TenantID)
			c.Set("roles", claims.Roles)

			// 设置租户ID到请求头，以便于微服务间传递
			c.Request.Header.Set("X-Tenant-ID", claims.TenantID)

			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户信息"})
			c.Abort()
		}
	}
}

// RoleMiddleware 角色验证中间件
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
