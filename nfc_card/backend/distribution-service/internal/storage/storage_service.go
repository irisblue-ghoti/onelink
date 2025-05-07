package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
)

// S3Config S3配置
type S3Config struct {
	AccessKey       string
	SecretKey       string
	Region          string
	Endpoint        string
	Bucket          string
	ForcePathStyle  bool
	PresignedURLTTL int
	TempDirectory   string
	ContentAPIURL   string
	ContentAPIToken string
}

// StorageService 存储服务接口
type StorageService interface {
	// DownloadFile 从S3下载文件到临时目录
	DownloadFile(ctx context.Context, storagePath string) (string, error)

	// GetSignedURL 获取带签名的URL
	GetSignedURL(storagePath string, ttl time.Duration) (string, error)

	// CleanupTempFiles 清理临时文件
	CleanupTempFiles() error
}

// S3StorageService S3存储服务实现
type S3StorageService struct {
	config    S3Config
	s3Client  *s3.S3
	s3Session *session.Session
	tempDir   string
}

// NewS3StorageService 创建S3存储服务
func NewS3StorageService(config S3Config) (*S3StorageService, error) {
	// 创建AWS会话
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		Endpoint:         aws.String(config.Endpoint),
		Region:           aws.String(config.Region),
		S3ForcePathStyle: aws.Bool(config.ForcePathStyle),
	}

	s3Session, err := session.NewSession(s3Config)
	if err != nil {
		return nil, fmt.Errorf("创建S3会话失败: %w", err)
	}

	// 创建S3客户端
	s3Client := s3.New(s3Session)

	// 创建临时目录
	tempDir := config.TempDirectory
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	// 确保临时目录存在
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	return &S3StorageService{
		config:    config,
		s3Client:  s3Client,
		s3Session: s3Session,
		tempDir:   tempDir,
	}, nil
}

// DownloadFile 从S3下载文件到临时目录
func (s *S3StorageService) DownloadFile(ctx context.Context, storagePath string) (string, error) {
	// 检查是否是URL或S3路径
	if storagePath == "" {
		return "", fmt.Errorf("存储路径不能为空")
	}

	// 如果是HTTP/HTTPS URL，则通过HTTP下载
	if isURL(storagePath) {
		return s.downloadFromURL(ctx, storagePath)
	}

	// 否则，从S3下载
	return s.downloadFromS3(ctx, storagePath)
}

// downloadFromS3 从S3下载文件
func (s *S3StorageService) downloadFromS3(ctx context.Context, s3Key string) (string, error) {
	// 创建临时文件
	tempID := uuid.New().String()
	tempFile := filepath.Join(s.tempDir, fmt.Sprintf("s3_%s%s", tempID, filepath.Ext(s3Key)))

	// 创建文件
	file, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer file.Close()

	// 创建下载器
	downloader := s3manager.NewDownloader(s.s3Session)

	// 下载文件
	_, err = downloader.DownloadWithContext(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		// 删除临时文件
		_ = os.Remove(tempFile)
		return "", fmt.Errorf("从S3下载文件失败: %w", err)
	}

	return tempFile, nil
}

// downloadFromURL 从URL下载文件
func (s *S3StorageService) downloadFromURL(ctx context.Context, url string) (string, error) {
	// 创建HTTP请求
	req, err := createHTTPRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 发送请求
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP请求失败: %s", resp.Status)
	}

	// 创建临时文件
	tempID := uuid.New().String()
	tempFile := filepath.Join(s.tempDir, fmt.Sprintf("url_%s%s", tempID, getExtFromURL(url)))

	// 创建文件
	file, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer file.Close()

	// 将响应内容复制到文件
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		// 删除临时文件
		_ = os.Remove(tempFile)
		return "", fmt.Errorf("复制文件内容失败: %w", err)
	}

	return tempFile, nil
}

// GetSignedURL 获取带签名的URL
func (s *S3StorageService) GetSignedURL(storagePath string, ttl time.Duration) (string, error) {
	// 如果已经是URL，直接返回
	if isURL(storagePath) {
		return storagePath, nil
	}

	// 设置请求
	req, _ := s.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(storagePath),
	})

	// 生成预签名URL
	url, err := req.Presign(ttl)
	if err != nil {
		return "", fmt.Errorf("生成预签名URL失败: %w", err)
	}

	return url, nil
}

// CleanupTempFiles 清理临时文件
func (s *S3StorageService) CleanupTempFiles() error {
	// 遍历临时目录
	err := filepath.Walk(s.tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录和非临时文件
		if info.IsDir() || !isTempFile(info.Name()) {
			return nil
		}

		// 检查文件是否过期（超过24小时）
		if time.Since(info.ModTime()) > 24*time.Hour {
			return os.Remove(path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("清理临时文件失败: %w", err)
	}

	return nil
}

// 辅助函数

// isURL 检查字符串是否是URL
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// getExtFromURL 从URL获取文件扩展名
func getExtFromURL(url string) string {
	ext := filepath.Ext(url)
	if ext == "" {
		return ".mp4" // 默认扩展名
	}
	return ext
}

// isTempFile 检查文件名是否是临时文件
func isTempFile(name string) bool {
	return strings.HasPrefix(name, "s3_") || strings.HasPrefix(name, "url_")
}

// createHTTPRequestWithContext 创建带上下文的HTTP请求
func createHTTPRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	return req, nil
}
