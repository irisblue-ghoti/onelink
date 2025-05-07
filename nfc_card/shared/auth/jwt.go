package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// 定义JWT服务结构
type JWTService struct {
	secretKey   string
	expiryHours int
}

// 创建新的JWT服务
func NewJWTService(secretKey string, expiryHours int) *JWTService {
	if expiryHours <= 0 {
		expiryHours = 24 // 默认24小时过期
	}
	return &JWTService{
		secretKey:   secretKey,
		expiryHours: expiryHours,
	}
}

// Claims JWT令牌声明结构
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	TenantID string   `json:"tenant_id"` // 商户/租户ID
	Roles    []string `json:"roles"`     // 用户角色列表
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT令牌
func (s *JWTService) GenerateToken(userID, username, tenantID string, roles []string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		TenantID: tenantID,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(s.expiryHours))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

// ValidateToken 验证JWT令牌
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("无效的签名方法")
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效令牌")
}

// HasRole 检查用户是否拥有特定角色
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole 检查用户是否拥有任意一个指定角色
func (c *Claims) HasAnyRole(roles ...string) bool {
	for _, requiredRole := range roles {
		if c.HasRole(requiredRole) {
			return true
		}
	}
	return false
}

// HasAllRoles 检查用户是否拥有所有指定角色
func (c *Claims) HasAllRoles(roles ...string) bool {
	for _, requiredRole := range roles {
		if !c.HasRole(requiredRole) {
			return false
		}
	}
	return true
}
