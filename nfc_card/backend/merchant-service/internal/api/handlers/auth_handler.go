package handlers

import (
	"errors"
	"merchant-service/internal/auth"
	"merchant-service/internal/domain/entities"
	"merchant-service/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler 处理认证相关的API请求
type AuthHandler struct {
	authService     *auth.JWTService
	userService     *services.UserService
	merchantService *services.MerchantService
}

// NewAuthHandler 创建新的认证处理器
func NewAuthHandler(authService *auth.JWTService, userService *services.UserService, merchantService *services.MerchantService) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		userService:     userService,
		merchantService: merchantService,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string        `json:"token"`
	User  entities.User `json:"user"`
}

// RegisterRequest 注册请求
// 注册请求结构体
// 只需要 name、email、password
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	User  entities.User `json:"user"`
	Token string        `json:"token"`
}

// Login 处理用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user entities.User
	var err error

	// 尝试使用用户服务进行认证
	user, err = h.authenticateUser(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 生成JWT令牌
	token, err := h.authService.GenerateToken(user.ID, user.MerchantID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User:  user,
	})
}

// GetCurrentUser 获取当前用户信息
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	// 尝试从用户服务获取用户信息
	user, err := h.getUserInfo(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// Register 处理用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查邮箱是否已存在
	if _, err := h.userService.FindByEmail(req.Email); err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱已被注册"})
		return
	}

	// 哈希密码
	hashedPassword, err := h.userService.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 生成新的商户UUID
	merchantID := uuid.New().String()

	// 创建用户（注册用户默认角色为 user，商户ID使用新生成的UUID）
	user, err := h.userService.Create(entities.CreateUserDTO{
		MerchantID: merchantID, // 使用新生成的UUID
		Name:       req.Name,
		Email:      req.Email,
		Password:   hashedPassword,
		Role:       "user",
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户创建失败", "details": err.Error()})
		return
	}

	// 生成JWT令牌
	token, err := h.authService.GenerateToken(user.ID, user.MerchantID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, RegisterResponse{
		User:  user,
		Token: token,
	})
}

// 用户认证
func (h *AuthHandler) authenticateUser(email, password string) (entities.User, error) {
	// 使用真实数据库中的用户
	user, err := h.userService.Authenticate(email, password)
	if err != nil {
		return entities.User{}, errors.New("邮箱或密码不正确")
	}
	return user, nil
}

// 获取用户信息
func (h *AuthHandler) getUserInfo(userID string) (entities.User, error) {
	// 从数据库获取用户信息
	user, err := h.userService.FindByID(userID)
	if err != nil {
		return entities.User{}, errors.New("用户不存在或数据库查询错误")
	}
	return user, nil
}
