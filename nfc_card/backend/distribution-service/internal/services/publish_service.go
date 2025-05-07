package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"distribution-service/internal/adapters/douyin"
	"distribution-service/internal/adapters/kuaishou"
	"distribution-service/internal/adapters/wechat"
	"distribution-service/internal/adapters/xiaohongshu"
	"distribution-service/internal/config"
	"distribution-service/internal/domain/entities"
	"distribution-service/internal/domain/repositories"
	"distribution-service/internal/storage"
)

// KafkaProducer Kafka生产者接口
type KafkaProducer interface {
	// SendMessage 发送消息到指定主题
	SendMessage(topic string, messageType string, data interface{}) error
}

// PlatformAdapter 平台适配器接口
type PlatformAdapter interface {
	// UploadVideo 上传视频到平台
	UploadVideo(ctx context.Context, video *entities.Video, job *entities.PublishJob) error

	// GetPublishStatus 获取平台发布状态
	GetPublishStatus(ctx context.Context, platformID string) (map[string]interface{}, error)
}

// EnhancedPlatformAdapter 增强平台适配器接口，提供更多功能
type EnhancedPlatformAdapter interface {
	PlatformAdapter

	// GenerateShareLink 生成分享链接
	GenerateShareLink(ctx context.Context, platformID string, extraParams map[string]interface{}) (string, error)

	// GenerateJSConfig 生成JSSDK配置
	GenerateJSConfig(ctx context.Context, url string) (map[string]interface{}, error)

	// GetDetailedStats 获取详细统计数据
	GetDetailedStats(ctx context.Context, platformID string) (map[string]interface{}, error)
}

// PublishService 发布服务
type PublishService struct {
	jobRepository      repositories.JobRepository
	videoRepository    repositories.VideoRepository
	douyinAdapter      PlatformAdapter
	kuaishouAdapter    PlatformAdapter
	xiaohongshuAdapter PlatformAdapter
	wechatAdapter      PlatformAdapter
	kafkaProducer      KafkaProducer
	storageService     storage.StorageService
}

// NewPublishService 创建发布服务
func NewPublishService(
	jobRepo repositories.JobRepository,
	videoRepo repositories.VideoRepository,
	config *config.Config,
	kafkaProducer KafkaProducer,
	storageService storage.StorageService,
) *PublishService {
	// 创建适配器
	douyinAdapter := douyin.NewDouyinAdapter(config.Adapters.Douyin, config.Adapters.TempDir)
	kuaishouAdapter := kuaishou.NewKuaishouAdapter(config.Adapters.Kuaishou, config.Adapters.TempDir)
	xiaohongshuAdapter := xiaohongshu.NewXiaohongshuAdapter(config.Adapters.Xiaohongshu, config.Adapters.TempDir)
	wechatAdapter := wechat.NewWechatAdapter(config.Adapters.Wechat, config.Adapters.TempDir)

	return &PublishService{
		jobRepository:      jobRepo,
		videoRepository:    videoRepo,
		douyinAdapter:      douyinAdapter,
		kuaishouAdapter:    kuaishouAdapter,
		xiaohongshuAdapter: xiaohongshuAdapter,
		wechatAdapter:      wechatAdapter,
		kafkaProducer:      kafkaProducer,
		storageService:     storageService,
	}
}

// SetKafkaProducer 设置Kafka生产者
func (s *PublishService) SetKafkaProducer(producer KafkaProducer) {
	s.kafkaProducer = producer
}

// CreateJob 创建分发任务
func (s *PublishService) CreateJob(ctx context.Context, job *entities.PublishJob) error {
	// 保存任务
	if err := s.jobRepository.Create(ctx, job); err != nil {
		return fmt.Errorf("保存任务失败: %w", err)
	}

	// 发送任务创建事件
	if s.kafkaProducer != nil {
		// 添加重试相关字段
		jobData := map[string]interface{}{
			"id":         job.ID.String(),
			"tenantId":   job.TenantID.String(),
			"videoId":    job.VideoID.String(),
			"nfcCardId":  job.NfcCardID.String(),
			"channel":    job.Channel,
			"status":     job.Status,
			"params":     job.Params,
			"retryCount": 0,
			"maxRetries": 3,
			"createdAt":  job.CreatedAt,
			"updatedAt":  job.UpdatedAt,
		}

		if err := s.kafkaProducer.SendMessage("publish-events", "publish_job.created", jobData); err != nil {
			log.Printf("发送任务创建事件失败: %v", err)
		}
	}

	// 异步处理任务
	go s.processJob(job)

	return nil
}

// ListJobs 获取分发任务列表
func (s *PublishService) ListJobs(ctx context.Context, tenantID uuid.UUID, status, videoID, nfcCardID, channel string) ([]*entities.PublishJob, error) {
	return s.jobRepository.Find(ctx, tenantID, status, videoID, nfcCardID, channel)
}

// GetJob 获取单个分发任务
func (s *PublishService) GetJob(ctx context.Context, tenantID, jobID uuid.UUID) (*entities.PublishJob, error) {
	return s.jobRepository.FindByID(ctx, tenantID, jobID)
}

// GetPlatformStatus 获取平台发布状态
func (s *PublishService) GetPlatformStatus(ctx context.Context, channel, platformID string) (map[string]interface{}, error) {
	var adapter PlatformAdapter

	// 根据渠道选择适配器
	switch channel {
	case "douyin":
		adapter = s.douyinAdapter
	case "kuaishou":
		adapter = s.kuaishouAdapter
	case "xiaohongshu":
		adapter = s.xiaohongshuAdapter
	case "wechat":
		adapter = s.wechatAdapter
	default:
		return nil, fmt.Errorf("不支持的渠道: %s", channel)
	}

	// 查询平台状态
	return adapter.GetPublishStatus(ctx, platformID)
}

// GenerateShareLink 生成分享链接
func (s *PublishService) GenerateShareLink(ctx context.Context, channel, platformID string, extraParams map[string]interface{}) (string, error) {
	// 检查是否支持增强功能
	var enhancedAdapter EnhancedPlatformAdapter
	var ok bool

	switch channel {
	case "douyin":
		enhancedAdapter, ok = s.douyinAdapter.(EnhancedPlatformAdapter)
	case "kuaishou":
		enhancedAdapter, ok = s.kuaishouAdapter.(EnhancedPlatformAdapter)
	case "xiaohongshu":
		enhancedAdapter, ok = s.xiaohongshuAdapter.(EnhancedPlatformAdapter)
	case "wechat":
		enhancedAdapter, ok = s.wechatAdapter.(EnhancedPlatformAdapter)
	default:
		ok = false
	}

	if !ok {
		return "", fmt.Errorf("渠道 %s 不支持生成分享链接", channel)
	}

	return enhancedAdapter.GenerateShareLink(ctx, platformID, extraParams)
}

// GenerateJSConfig 生成JSSDK配置
func (s *PublishService) GenerateJSConfig(ctx context.Context, channel, url string) (map[string]interface{}, error) {
	// 检查是否支持增强功能
	var enhancedAdapter EnhancedPlatformAdapter
	var ok bool

	switch channel {
	case "xiaohongshu":
		enhancedAdapter, ok = s.xiaohongshuAdapter.(EnhancedPlatformAdapter)
	case "wechat":
		enhancedAdapter, ok = s.wechatAdapter.(EnhancedPlatformAdapter)
	default:
		ok = false
	}

	if !ok {
		return nil, fmt.Errorf("渠道 %s 不支持JSSDK配置", channel)
	}

	return enhancedAdapter.GenerateJSConfig(ctx, url)
}

// GetDetailedStats 获取详细统计数据
func (s *PublishService) GetDetailedStats(ctx context.Context, channel, platformID string) (map[string]interface{}, error) {
	// 检查是否支持增强功能
	var enhancedAdapter EnhancedPlatformAdapter
	var ok bool

	switch channel {
	case "kuaishou":
		enhancedAdapter, ok = s.kuaishouAdapter.(EnhancedPlatformAdapter)
	default:
		ok = false
	}

	if !ok {
		return nil, fmt.Errorf("渠道 %s 不支持获取详细统计数据", channel)
	}

	return enhancedAdapter.GetDetailedStats(ctx, platformID)
}

// UpdateJobStatus 更新任务状态
func (s *PublishService) UpdateJobStatus(ctx context.Context, job *entities.PublishJob, status, errorMsg string) error {
	job.Status = status
	job.ErrorMsg = errorMsg
	job.UpdatedAt = time.Now()

	// 如果任务完成，设置完成时间
	if status == "completed" || status == "failed" {
		job.CompletedAt = time.Now()
	}

	// 更新任务状态
	if err := s.jobRepository.Update(ctx, job); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 发送任务更新事件
	if s.kafkaProducer != nil {
		// 添加重试相关字段
		jobData := map[string]interface{}{
			"id":          job.ID.String(),
			"tenantId":    job.TenantID.String(),
			"videoId":     job.VideoID.String(),
			"nfcCardId":   job.NfcCardID.String(),
			"channel":     job.Channel,
			"status":      status,
			"errorMsg":    errorMsg,
			"result":      job.Result,
			"updatedAt":   job.UpdatedAt,
			"completedAt": job.CompletedAt,
		}

		if err := s.kafkaProducer.SendMessage("publish-events", "publish_job.updated", jobData); err != nil {
			log.Printf("发送任务更新事件失败: %v", err)
		}

		// 如果任务完成或失败，还要发送任务完成事件
		if status == "completed" || status == "failed" {
			if err := s.kafkaProducer.SendMessage("publish-events", "publish_job.completed", jobData); err != nil {
				log.Printf("发送任务完成事件失败: %v", err)
			}
		}
	}

	return nil
}

// processJob 处理发布任务
func (s *PublishService) processJob(job *entities.PublishJob) {
	ctx := context.Background()

	// 更新任务状态为处理中
	if err := s.UpdateJobStatus(ctx, job, "processing", ""); err != nil {
		log.Printf("更新任务状态失败: %v", err)
		return
	}

	// 查询视频信息
	video, err := s.videoRepository.FindByID(ctx, job.TenantID, job.VideoID)
	if err != nil {
		log.Printf("查询视频信息失败: %v", err)
		s.UpdateJobStatus(ctx, job, "failed", fmt.Sprintf("查询视频信息失败: %v", err))
		return
	}

	// 如果存储服务可用，下载视频到临时目录
	if s.storageService != nil && video.StoragePath != "" {
		tempFilePath, err := s.storageService.DownloadFile(ctx, video.StoragePath)
		if err != nil {
			log.Printf("下载视频失败: %v", err)
			s.UpdateJobStatus(ctx, job, "failed", fmt.Sprintf("下载视频失败: %v", err))
			return
		}

		// 更新视频的临时存储路径
		video.StoragePath = tempFilePath

		// 任务完成后清理临时文件
		defer func() {
			if err := s.storageService.CleanupTempFiles(); err != nil {
				log.Printf("清理临时文件失败: %v", err)
			}
		}()
	}

	// 根据渠道选择适配器
	var adapter PlatformAdapter
	switch job.Channel {
	case "douyin":
		adapter = s.douyinAdapter
	case "kuaishou":
		adapter = s.kuaishouAdapter
	case "xiaohongshu":
		adapter = s.xiaohongshuAdapter
	case "wechat":
		adapter = s.wechatAdapter
	default:
		s.UpdateJobStatus(ctx, job, "failed", fmt.Sprintf("不支持的渠道: %s", job.Channel))
		return
	}

	// 上传视频到平台
	err = adapter.UploadVideo(ctx, video, job)
	if err != nil {
		log.Printf("上传视频到%s失败: %v", job.Channel, err)
		s.UpdateJobStatus(ctx, job, "failed", fmt.Sprintf("上传视频失败: %v", err))
		return
	}

	// 更新任务状态为已完成
	s.UpdateJobStatus(ctx, job, "completed", "")
}

// GetVideo 获取视频信息
func (s *PublishService) GetVideo(ctx context.Context, tenantID, videoID uuid.UUID) (*entities.Video, error) {
	return s.videoRepository.FindByID(ctx, tenantID, videoID)
}

// PublishToDouyin 发布到抖音
func (s *PublishService) PublishToDouyin(ctx context.Context, job *entities.PublishJob, video *entities.Video) error {
	if s.douyinAdapter == nil {
		return fmt.Errorf("抖音适配器未配置")
	}

	// 上传视频到平台
	return s.douyinAdapter.UploadVideo(ctx, video, job)
}

// PublishToKuaishou 发布到快手
func (s *PublishService) PublishToKuaishou(ctx context.Context, job *entities.PublishJob, video *entities.Video) error {
	if s.kuaishouAdapter == nil {
		return fmt.Errorf("快手适配器未配置")
	}

	// 上传视频到平台
	return s.kuaishouAdapter.UploadVideo(ctx, video, job)
}

// PublishToXiaohongshu 发布到小红书
func (s *PublishService) PublishToXiaohongshu(ctx context.Context, job *entities.PublishJob, video *entities.Video) error {
	if s.xiaohongshuAdapter == nil {
		return fmt.Errorf("小红书适配器未配置")
	}

	// 上传视频到平台
	return s.xiaohongshuAdapter.UploadVideo(ctx, video, job)
}

// PublishToWechat 发布到微信
func (s *PublishService) PublishToWechat(ctx context.Context, job *entities.PublishJob, video *entities.Video) error {
	if s.wechatAdapter == nil {
		return fmt.Errorf("微信适配器未配置")
	}

	// 上传视频到平台
	return s.wechatAdapter.UploadVideo(ctx, video, job)
}
