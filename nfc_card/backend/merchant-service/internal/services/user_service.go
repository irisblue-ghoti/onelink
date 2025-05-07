package services

import (
	"errors"
	"log"
	"merchant-service/internal/domain/entities"
	"merchant-service/internal/storage"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserService 用户服务
type UserService struct {
	repo   *storage.UserRepository
	logger *log.Logger
}

// NewUserService 创建用户服务
func NewUserService(repo *storage.UserRepository, logger *log.Logger) *UserService {
	return &UserService{
		repo:   repo,
		logger: logger,
	}
}

// Create 创建用户
func (s *UserService) Create(dto entities.CreateUserDTO) (entities.User, error) {
	// 注意：User结构体的Password字段映射到数据库的password_hash字段
	// 在实体定义中是 `db:"password_hash"`
	user := entities.User{
		ID:         uuid.New().String(),
		MerchantID: dto.MerchantID,
		Name:       dto.Name,
		Email:      dto.Email,
		Password:   dto.Password, // 这个值会映射到数据库的password_hash字段
		Role:       dto.Role,
		Status:     "active",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return s.repo.Create(user)
}

// FindByEmail 通过邮箱查找用户
func (s *UserService) FindByEmail(email string) (entities.User, error) {
	return s.repo.FindByEmail(email)
}

// FindByID 通过ID查找用户
func (s *UserService) FindByID(id string) (entities.User, error) {
	return s.repo.FindByID(id)
}

// Authenticate 用户认证
func (s *UserService) Authenticate(email, password string) (entities.User, error) {
	user, err := s.FindByEmail(email)
	if err != nil {
		return entities.User{}, errors.New("邮箱或密码不正确")
	}

	// 验证密码
	if !s.VerifyPassword(user.Password, password) {
		return entities.User{}, errors.New("邮箱或密码不正确")
	}

	// 检查用户状态
	if user.Status != "active" {
		return entities.User{}, errors.New("账户未激活")
	}

	return user, nil
}

// VerifyPassword 验证密码
func (s *UserService) VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// HashPassword 哈希密码
func (s *UserService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
