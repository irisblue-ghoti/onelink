package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"content-service/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// StorageService 提供对象存储功能
type StorageService struct {
	client     *minio.Client
	bucketName string
	cfg        *config.Config
}

// NewStorageService 创建新的存储服务
func NewStorageService(cfg *config.Config) (*StorageService, error) {
	// 创建MinIO客户端
	client, err := minio.New(cfg.Storage.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Storage.AccessKey, cfg.Storage.SecretKey, ""),
		Secure: cfg.Storage.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("创建MinIO客户端失败: %w", err)
	}

	// 检查存储桶是否存在，不存在则创建
	bucketExists, err := client.BucketExists(context.Background(), cfg.Storage.BucketName)
	if err != nil {
		return nil, fmt.Errorf("检查存储桶失败: %w", err)
	}

	if !bucketExists {
		err = client.MakeBucket(context.Background(), cfg.Storage.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("创建存储桶失败: %w", err)
		}
	}

	return &StorageService{
		client:     client,
		bucketName: cfg.Storage.BucketName,
		cfg:        cfg,
	}, nil
}

// UploadFile 上传文件到对象存储
func (s *StorageService) UploadFile(file *multipart.FileHeader, objectKey string) error {
	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	// 上传文件到MinIO
	_, err = s.client.PutObject(
		context.Background(),
		s.bucketName,
		objectKey,
		src,
		file.Size,
		minio.PutObjectOptions{ContentType: file.Header.Get("Content-Type")},
	)
	if err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}

	return nil
}

// GetFileURL 获取文件的访问URL
func (s *StorageService) GetFileURL(objectKey string) (string, error) {
	// 获取预签名URL
	url, err := s.client.PresignedGetObject(
		context.Background(),
		s.bucketName,
		objectKey,
		time.Hour*24, // URL有效期24小时
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("获取文件URL失败: %w", err)
	}

	return url.String(), nil
}

// DeleteFile 从对象存储中删除文件
func (s *StorageService) DeleteFile(objectKey string) error {
	err := s.client.RemoveObject(
		context.Background(),
		s.bucketName,
		objectKey,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}

	return nil
}

// GetObject 获取文件内容
func (s *StorageService) GetObject(objectKey string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(
		context.Background(),
		s.bucketName,
		objectKey,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("获取文件内容失败: %w", err)
	}

	return obj, nil
}
