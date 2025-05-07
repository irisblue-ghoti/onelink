package services

import (
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"time"

	"content-service/internal/config"
	"content-service/internal/domain/entities"
	"content-service/internal/messaging"
	"content-service/internal/storage"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// 定义服务错误类型和代码
const (
	// 错误类型
	ErrTypeDatabase     = "database_error"
	ErrTypeStorage      = "storage_error"
	ErrTypeValidation   = "validation_error"
	ErrTypeNotFound     = "not_found_error"
	ErrTypeTranscode    = "transcode_error"
	ErrTypeUnauthorized = "unauthorized_error"

	// 错误代码
	ErrCodeDBConnection     = "db_connection_failed"
	ErrCodeDBQuery          = "db_query_failed"
	ErrCodeFileUpload       = "file_upload_failed"
	ErrCodeFileDownload     = "file_download_failed"
	ErrCodeFileDelete       = "file_delete_failed"
	ErrCodeInvalidInput     = "invalid_input"
	ErrCodeResourceExists   = "resource_already_exists"
	ErrCodeResourceNotFound = "resource_not_found"
	ErrCodeTranscodeFailed  = "transcode_failed"
	ErrCodeUnauthorized     = "unauthorized_access"
)

// ServiceError 服务错误结构
type ServiceError struct {
	Type    string // 错误类型
	Code    string // 错误代码
	Message string // 错误消息
	Err     error  // 原始错误
}

// Error 实现error接口
func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s - %s", e.Type, e.Code, e.Err.Error())
	}
	return fmt.Sprintf("%s: %s - %s", e.Type, e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *ServiceError) Unwrap() error {
	return e.Err
}

// Is 用于错误比较
func (e *ServiceError) Is(target error) bool {
	t, ok := target.(*ServiceError)
	if !ok {
		return false
	}
	return e.Type == t.Type && e.Code == t.Code
}

// VideoService 视频服务
type VideoService struct {
	db               *sqlx.DB
	storageService   *storage.StorageService
	transcodeService *TranscodeService
	cfg              *config.Config
}

// NewVideoService 创建新的视频服务
func NewVideoService(cfg *config.Config, storageService *storage.StorageService, kafkaProducer *messaging.KafkaProducer) *VideoService {
	// 尝试连接数据库
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)
	db, err := sqlx.Connect("postgres", psqlInfo)
	if err != nil {
		// 结构化错误处理，但此处仍然需要panic因为这是服务初始化阶段
		dbErr := &ServiceError{
			Type:    ErrTypeDatabase,
			Code:    ErrCodeDBConnection,
			Message: "无法连接到数据库",
			Err:     err,
		}
		panic(dbErr)
	}

	// 创建转码服务
	transcodeService, err := NewTranscodeService(db, storageService, cfg, kafkaProducer)
	if err != nil {
		// 结构化错误处理
		tsErr := &ServiceError{
			Type:    ErrTypeTranscode,
			Code:    "transcode_service_init_failed",
			Message: "无法初始化转码服务",
			Err:     err,
		}
		panic(tsErr)
	}

	return &VideoService{
		db:               db,
		storageService:   storageService,
		transcodeService: transcodeService,
		cfg:              cfg,
	}
}

// Create 上传并创建新视频
func (s *VideoService) Create(tenantID string, file *multipart.FileHeader, dto entities.CreateVideoDTO) (entities.Video, error) {
	// 参数验证
	if file == nil {
		return entities.Video{}, &ServiceError{
			Type:    ErrTypeValidation,
			Code:    ErrCodeInvalidInput,
			Message: "文件不能为空",
		}
	}

	if dto.Title == "" {
		return entities.Video{}, &ServiceError{
			Type:    ErrTypeValidation,
			Code:    ErrCodeInvalidInput,
			Message: "标题不能为空",
		}
	}

	// 生成唯一ID和文件Key
	videoID := uuid.New().String()
	fileExt := filepath.Ext(file.Filename)
	fileKey := fmt.Sprintf("%s/%s%s", tenantID, videoID, fileExt)

	// 上传文件到存储服务
	if err := s.storageService.UploadFile(file, fileKey); err != nil {
		return entities.Video{}, &ServiceError{
			Type:    ErrTypeStorage,
			Code:    ErrCodeFileUpload,
			Message: "上传文件失败",
			Err:     err,
		}
	}

	// 将字符串转换为UUID
	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		// 删除已上传的文件
		_ = s.storageService.DeleteFile(fileKey)
		return entities.Video{}, &ServiceError{
			Type:    ErrTypeValidation,
			Code:    ErrCodeInvalidInput,
			Message: "无效的视频ID格式",
			Err:     err,
		}
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		// 删除已上传的文件
		_ = s.storageService.DeleteFile(fileKey)
		return entities.Video{}, &ServiceError{
			Type:    ErrTypeValidation,
			Code:    ErrCodeInvalidInput,
			Message: "无效的租户ID格式",
			Err:     err,
		}
	}

	// 创建视频记录
	video := entities.Video{
		ID:              videoUUID,
		TenantID:        tenantUUID,
		Title:           dto.Title,
		Description:     dto.Description,
		FileName:        file.Filename,
		FileKey:         fileKey,
		FileType:        file.Header.Get("Content-Type"),
		Size:            file.Size,
		Duration:        0, // 转码后更新
		Width:           0, // 转码后更新
		Height:          0, // 转码后更新
		IsTranscoded:    false,
		TranscodeStatus: entities.TranscodeStatusPending,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// 保存到数据库
	query := `
		INSERT INTO videos (
			id, tenant_id, title, description, file_name, file_key, file_type, 
			size, duration, width, height, is_transcoded, transcode_status, 
			created_at, updated_at
		) VALUES (
			:id, :tenant_id, :title, :description, :file_name, :file_key, :file_type, 
			:size, :duration, :width, :height, :is_transcoded, :transcode_status, 
			:created_at, :updated_at
		) RETURNING *
	`

	rows, err := s.db.NamedQuery(query, video)
	if err != nil {
		// 删除已上传的文件
		_ = s.storageService.DeleteFile(fileKey)
		return entities.Video{}, &ServiceError{
			Type:    ErrTypeDatabase,
			Code:    ErrCodeDBQuery,
			Message: "保存视频信息失败",
			Err:     err,
		}
	}
	defer rows.Close()

	if rows.Next() {
		var result entities.Video
		if err := rows.StructScan(&result); err != nil {
			return entities.Video{}, &ServiceError{
				Type:    ErrTypeDatabase,
				Code:    ErrCodeDBQuery,
				Message: "读取视频信息失败",
				Err:     err,
			}
		}

		// 创建成功后，启动视频转码过程
		go func() {
			if err := s.transcodeService.TranscodeVideo(result.ID.String(), result.TenantID.String()); err != nil {
				fmt.Printf("启动视频转码失败: %v\n", err)
			}
		}()

		return result, nil
	}

	return entities.Video{}, &ServiceError{
		Type:    ErrTypeDatabase,
		Code:    ErrCodeDBQuery,
		Message: "创建视频失败",
	}
}

// FindAll 获取租户所有视频
func (s *VideoService) FindAll(tenantID string, page int, limit int) ([]entities.Video, error) {
	var videos []entities.Video

	// 计算偏移量
	offset := (page - 1) * limit

	query := `
		SELECT * FROM videos
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := s.db.Select(&videos, query, tenantID, limit, offset); err != nil {
		return nil, fmt.Errorf("获取视频列表失败: %w", err)
	}

	return videos, nil
}

// CountVideos 获取租户的视频总数
func (s *VideoService) CountVideos(tenantID string) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM videos
		WHERE tenant_id = $1
	`
	if err := s.db.Get(&count, query, tenantID); err != nil {
		return 0, fmt.Errorf("获取视频总数失败: %w", err)
	}

	return count, nil
}

// FindOne 获取单个视频
func (s *VideoService) FindOne(id string, tenantID string) (entities.Video, error) {
	var video entities.Video

	query := "SELECT * FROM videos WHERE id = $1 AND tenant_id = $2"
	if err := s.db.Get(&video, query, id, tenantID); err != nil {
		return entities.Video{}, fmt.Errorf("获取视频信息失败: %w", err)
	}

	return video, nil
}

// Remove 删除视频
func (s *VideoService) Remove(id string, tenantID string) error {
	// 先获取视频信息
	video, err := s.FindOne(id, tenantID)
	if err != nil {
		return err
	}

	// 从数据库删除记录
	query := "DELETE FROM videos WHERE id = $1 AND tenant_id = $2"
	result, err := s.db.Exec(query, id, tenantID)
	if err != nil {
		return fmt.Errorf("删除视频记录失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("视频不存在")
	}

	// 从存储中删除文件
	if err := s.storageService.DeleteFile(video.FileKey); err != nil {
		return fmt.Errorf("删除视频文件失败: %w", err)
	}

	return nil
}

// GetVideoURL 获取视频访问URL
func (s *VideoService) GetVideoURL(video entities.Video) string {
	url, err := s.storageService.GetFileURL(video.FileKey)
	if err != nil {
		// 如果获取URL失败，返回一个固定的错误URL
		return ""
	}
	return url
}

// GetFileURL 获取指定文件的访问URL
func (s *VideoService) GetFileURL(fileKey string) string {
	url, err := s.storageService.GetFileURL(fileKey)
	if err != nil {
		return ""
	}
	return url
}

// StartTranscode 手动开始视频转码
func (s *VideoService) StartTranscode(videoID, tenantID string) error {
	// 查询视频信息
	_, err := s.FindOne(videoID, tenantID)
	if err != nil {
		return fmt.Errorf("获取视频信息失败: %w", err)
	}

	// 开始转码
	return s.transcodeService.TranscodeVideo(videoID, tenantID)
}
