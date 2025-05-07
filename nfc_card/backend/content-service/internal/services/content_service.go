package services

import (
	"errors"
	"log"
	"mime/multipart"

	"content-service/internal/config"
	"content-service/internal/domain/entities"
	"content-service/internal/messaging"
	"content-service/internal/storage"
)

// ContentService 内容服务
type ContentService struct {
	repos       *storage.Repositories
	kafkaClient *messaging.KafkaClient
	logger      *log.Logger
	// 添加VideoService作为内部实现
	videoService *VideoService
}

// NewContentService 创建内容服务实例
func NewContentService(repos *storage.Repositories, kafkaClient *messaging.KafkaClient, logger *log.Logger, cfg *config.Config) *ContentService {
	// 如果未提供配置，创建默认配置
	if cfg == nil {
		cfg = &config.Config{
			Database: config.DatabaseConfig{
				Host:     "postgres",
				Port:     "5432",
				User:     "postgres",
				Password: "postgres",
				DBName:   "nfc_card",
				SSLMode:  "disable",
			},
		}
	}

	// 获取kafka配置
	if kafkaClient != nil && cfg.Kafka.Topic == "" {
		// 设置默认主题
		cfg.Kafka.Topic = "content-events"
		logger.Println("已初始化Kafka配置")
	} else {
		logger.Println("警告: 未找到Kafka配置，部分功能可能不可用")
	}

	// 创建存储服务
	storageService := &storage.StorageService{}

	// 创建KafkaProducer
	kafkaProducer := &messaging.KafkaProducer{}

	// 创建VideoService
	videoService := NewVideoService(cfg, storageService, kafkaProducer)

	return &ContentService{
		repos:        repos,
		kafkaClient:  kafkaClient,
		logger:       logger,
		videoService: videoService,
	}
}

// 以下方法都是为了兼容VideoService接口

// Create 上传并创建新视频
func (s *ContentService) Create(tenantID string, file *multipart.FileHeader, dto entities.CreateVideoDTO) (entities.Video, error) {
	if s.videoService != nil {
		return s.videoService.Create(tenantID, file, dto)
	}
	return entities.Video{}, errors.New("视频服务未初始化")
}

// FindAll 获取租户所有视频
func (s *ContentService) FindAll(tenantID string, page int, limit int) ([]entities.Video, error) {
	if s.videoService != nil {
		return s.videoService.FindAll(tenantID, page, limit)
	}
	return nil, errors.New("视频服务未初始化")
}

// CountVideos 获取租户的视频总数
func (s *ContentService) CountVideos(tenantID string) (int, error) {
	if s.videoService != nil {
		return s.videoService.CountVideos(tenantID)
	}
	return 0, errors.New("视频服务未初始化")
}

// FindOne 获取单个视频
func (s *ContentService) FindOne(id string, tenantID string) (entities.Video, error) {
	if s.videoService != nil {
		return s.videoService.FindOne(id, tenantID)
	}
	return entities.Video{}, errors.New("视频服务未初始化")
}

// Remove 删除视频
func (s *ContentService) Remove(id string, tenantID string) error {
	if s.videoService != nil {
		return s.videoService.Remove(id, tenantID)
	}
	return errors.New("视频服务未初始化")
}

// GetVideoURL 获取视频访问URL
func (s *ContentService) GetVideoURL(video entities.Video) string {
	if s.videoService != nil {
		return s.videoService.GetVideoURL(video)
	}
	return ""
}

// GetFileURL 获取指定文件的访问URL
func (s *ContentService) GetFileURL(fileKey string) string {
	if s.videoService != nil {
		return s.videoService.GetFileURL(fileKey)
	}
	return ""
}

// StartTranscode 手动开始视频转码
func (s *ContentService) StartTranscode(videoID, tenantID string) error {
	if s.videoService != nil {
		return s.videoService.StartTranscode(videoID, tenantID)
	}
	return errors.New("视频服务未初始化")
}
