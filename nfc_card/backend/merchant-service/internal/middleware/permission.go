package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PermissionMiddleware 基于资源和操作的权限检查中间件
func PermissionMiddleware(resource string, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID和角色
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证的用户"})
			c.Abort()
			return
		}

		// 获取用户角色
		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "无访问权限"})
			c.Abort()
			return
		}

		rolesList, ok := roles.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "角色格式错误"})
			c.Abort()
			return
		}

		// 检查是否为管理员
		for _, role := range rolesList {
			if role == "admin" {
				// 管理员拥有所有权限
				c.Next()
				return
			}
		}

		// TODO: 从数据库检查用户是否具有指定资源和操作的权限
		// 这里应该调用权限服务来检查用户是否具有对应权限
		hasPermission := checkUserPermission(userID.(string), resource, action)
		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "无操作权限"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkUserPermission 检查用户是否具有指定权限
func checkUserPermission(userID string, resource string, action string) bool {
	// TODO: 从数据库中查询用户角色和权限
	// 1. 获取用户的所有角色
	// 2. 获取这些角色拥有的所有权限
	// 3. 检查是否包含指定的资源和操作
	return true // 临时返回true，实际应该查询数据库
}

// ResourceOwnershipMiddleware 资源所有权检查中间件
func ResourceOwnershipMiddleware(resourceType string, paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户商户ID
		merchantID, exists := c.Get("merchantID")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "无访问权限"})
			c.Abort()
			return
		}

		// 获取资源ID
		resourceID := c.Param(paramName)
		if resourceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "未提供资源ID"})
			c.Abort()
			return
		}

		// 检查用户是否为管理员
		roles, exists := c.Get("roles")
		if exists {
			rolesList, ok := roles.([]string)
			if ok {
				for _, role := range rolesList {
					if role == "admin" {
						// 管理员可以访问任何资源
						c.Next()
						return
					}
				}
			}
		}

		// TODO: 检查资源是否属于用户的商户
		// 这里应该调用相应的服务来检查资源所有权
		isOwner := checkResourceOwnership(resourceType, resourceID, merchantID.(string))
		if !isOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此资源"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkResourceOwnership 检查资源所有权
func checkResourceOwnership(resourceType string, resourceID string, merchantID string) bool {
	// TODO: 根据资源类型从数据库中查询资源所有者
	// 例如，对于商户，检查resourceID是否等于merchantID
	// 对于其他资源，查询资源表中的merchant_id是否等于用户的merchantID
	return true // 临时返回true，实际应该查询数据库
}
