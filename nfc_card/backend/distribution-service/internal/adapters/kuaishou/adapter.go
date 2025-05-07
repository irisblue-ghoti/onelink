package kuaishou

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"distribution-service/internal/config"
	"distribution-service/internal/domain/entities"
)

const (
	// 快手开放平台API基础URL
	kuaishouAPIBaseURL = "https://open.kuaishou.com"

	// 视频上传和发布端点
	uploadInitEndpoint   = "/openapi/photo/upload"
	uploadStatusEndpoint = "/openapi/photo/upload/status"
	publishEndpoint      = "/openapi/photo/publish"

	// 视频状态查询端点
	videoStatusEndpoint = "/openapi/photo/info"
)

// KuaishouClient 快手API客户端
type KuaishouClient struct {
	appID       string
	appSecret   string
	accessToken string
	httpClient  *http.Client
}

// NewKuaishouClient 创建快手客户端
func NewKuaishouClient(appID, appSecret string) *KuaishouClient {
	return &KuaishouClient{
		appID:      appID,
		appSecret:  appSecret,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAccessToken 获取访问令牌
func (c *KuaishouClient) GetAccessToken() (string, error) {
	// 如果令牌有效，直接返回
	if c.accessToken != "" {
		return c.accessToken, nil
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/oauth2/access_token?app_id=%s&app_secret=%s&grant_type=client_credentials",
		kuaishouAPIBaseURL, c.appID, c.appSecret)

	// 发送请求
	resp, err := c.httpClient.Post(url, "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("获取访问令牌失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Result      int    `json:"result"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Message     string `json:"error_msg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析访问令牌响应失败: %w", err)
	}

	// 检查响应
	if result.Result != 1 || result.AccessToken == "" {
		return "", fmt.Errorf("获取访问令牌失败: %s", result.Message)
	}

	// 保存访问令牌
	c.accessToken = result.AccessToken

	return c.accessToken, nil
}

// KuaishouAdapter 快手适配器
type KuaishouAdapter struct {
	client      *KuaishouClient
	tempDir     string
	callbackURL string
}

// NewKuaishouAdapter 创建快手适配器
func NewKuaishouAdapter(config config.KuaishouConfig, tempDir string) *KuaishouAdapter {
	client := NewKuaishouClient(config.AppID, config.AppSecret)
	return &KuaishouAdapter{
		client:      client,
		tempDir:     tempDir,
		callbackURL: config.CallbackURL,
	}
}

// UploadVideo 上传视频到快手
func (a *KuaishouAdapter) UploadVideo(ctx context.Context, video *entities.Video, job *entities.PublishJob) error {
	log.Printf("开始上传视频到快手: %s", video.Title)

	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		log.Printf("获取快手访问令牌失败: %v", err)
		return fmt.Errorf("获取快手访问令牌失败: %w", err)
	}

	// 更新任务状态
	job.Status = "downloading"
	job.UpdatedAt = time.Now()
	log.Printf("正在下载视频文件: %s", video.StoragePath)

	// 下载视频到临时目录
	videoPath, err := a.downloadVideo(video.StoragePath)
	if err != nil {
		log.Printf("下载视频失败: %v", err)
		return fmt.Errorf("下载视频失败: %w", err)
	}
	defer func() {
		log.Printf("清理临时文件: %s", videoPath)
		os.Remove(videoPath) // 确保临时文件被删除
	}()

	// 更新任务状态
	job.Status = "uploading"
	job.UpdatedAt = time.Now()
	log.Printf("开始上传视频到快手")

	// 上传视频
	uploadResult, err := a.uploadVideo(accessToken, videoPath)
	if err != nil {
		log.Printf("上传视频失败: %v", err)
		return fmt.Errorf("上传视频失败: %w", err)
	}

	log.Printf("视频上传初始化成功，upload_id: %s", uploadResult.UploadID)

	// 更新任务状态
	job.Status = "processing"
	job.UpdatedAt = time.Now()
	log.Printf("视频处理中...")

	// 轮询上传状态
	maxRetries := 30 // 最多等待5分钟
	retryInterval := 10 * time.Second

	for i := 0; i < maxRetries; i++ {
		uploadStatus, err := a.checkUploadStatus(accessToken, uploadResult.UploadID)
		if err != nil {
			log.Printf("检查上传状态失败: %v", err)
			return fmt.Errorf("检查上传状态失败: %w", err)
		}

		log.Printf("上传状态: %s", uploadStatus.Status)

		if uploadStatus.Status == "SUCCESS" {
			// 发布视频
			log.Printf("上传完成，开始发布视频")
			publishResult, err := a.publishVideo(accessToken, uploadResult.UploadID, video.Title, video.Description)
			if err != nil {
				log.Printf("发布视频失败: %v", err)
				return fmt.Errorf("发布视频失败: %w", err)
			}

			log.Printf("视频发布成功，photo_id: %s, 分享链接: %s", publishResult.PhotoID, publishResult.ShareURL)

			// 更新任务状态
			job.Status = "completed"
			job.CompletedAt = time.Now()
			job.UpdatedAt = time.Now()
			job.Result = map[string]interface{}{
				"platformId": publishResult.PhotoID,
				"url":        publishResult.ShareURL,
			}

			return nil
		} else if uploadStatus.Status == "FAIL" {
			log.Printf("上传视频失败: %s", uploadStatus.Message)
			return fmt.Errorf("上传视频失败: %s", uploadStatus.Message)
		}

		// 等待后重试
		log.Printf("视频处理中，%d秒后重试...", retryInterval/time.Second)
		time.Sleep(retryInterval)
	}

	log.Printf("上传超时")
	return fmt.Errorf("上传超时")
}

// GetPublishStatus 获取平台发布状态
func (a *KuaishouAdapter) GetPublishStatus(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("获取快手访问令牌失败: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s&photo_id=%s",
		kuaishouAPIBaseURL, videoStatusEndpoint, accessToken, platformID)

	// 发送请求
	resp, err := a.client.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取视频状态失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Result  int    `json:"result"`
		Message string `json:"error_msg"`
		Data    struct {
			PhotoID    string `json:"photo_id"`
			Title      string `json:"caption"`
			CreateTime int64  `json:"create_time"`
			Status     string `json:"status"`
			ShareURL   string `json:"share_url"`
			Metrics    struct {
				LikeCount    int `json:"like_count"`
				CommentCount int `json:"comment_count"`
				ViewCount    int `json:"view_count"`
				ShareCount   int `json:"share_count"`
			} `json:"metrics"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析视频状态响应失败: %w", err)
	}

	// 检查响应
	if result.Result != 1 {
		return nil, fmt.Errorf("获取视频状态失败: %s", result.Message)
	}

	// 返回状态
	return map[string]interface{}{
		"platformId":   result.Data.PhotoID,
		"title":        result.Data.Title,
		"createTime":   time.Unix(result.Data.CreateTime, 0),
		"status":       result.Data.Status,
		"shareUrl":     result.Data.ShareURL,
		"likeCount":    result.Data.Metrics.LikeCount,
		"commentCount": result.Data.Metrics.CommentCount,
		"viewCount":    result.Data.Metrics.ViewCount,
		"shareCount":   result.Data.Metrics.ShareCount,
	}, nil
}

// 下载视频到临时目录
func (a *KuaishouAdapter) downloadVideo(storagePath string) (string, error) {
	// 检查storagePath是否已经是本地临时文件
	if strings.HasPrefix(filepath.Base(storagePath), "s3_") ||
		strings.HasPrefix(filepath.Base(storagePath), "url_") {
		// 已经是临时文件，直接返回
		log.Printf("使用已下载的临时文件: %s", storagePath)
		return storagePath, nil
	}

	// 创建临时文件
	tempID := uuid.New().String()
	tempFile := filepath.Join(a.tempDir, fmt.Sprintf("kuaishou_%s.mp4", tempID))

	// 如果storagePath是HTTP(S)链接，则直接下载
	if strings.HasPrefix(storagePath, "http://") || strings.HasPrefix(storagePath, "https://") {
		// 创建HTTP客户端
		client := &http.Client{Timeout: 5 * time.Minute}

		// 创建请求
		req, err := http.NewRequest("GET", storagePath, nil)
		if err != nil {
			return "", fmt.Errorf("创建下载请求失败: %w", err)
		}

		// 发送请求
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("下载视频失败: %w", err)
		}
		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("下载视频失败: HTTP状态码 %d", resp.StatusCode)
		}

		// 创建目标文件
		file, err := os.Create(tempFile)
		if err != nil {
			return "", fmt.Errorf("创建临时文件失败: %w", err)
		}
		defer file.Close()

		// 将响应内容复制到文件
		_, err = io.Copy(file, resp.Body)
		if err != nil {
			// 删除临时文件
			os.Remove(tempFile)
			return "", fmt.Errorf("保存视频内容失败: %w", err)
		}

		log.Printf("从URL下载视频到临时文件: %s", tempFile)
		return tempFile, nil
	}

	// 否则，尝试从本地文件系统复制
	if _, err := os.Stat(storagePath); err == nil {
		// 复制文件
		src, err := os.Open(storagePath)
		if err != nil {
			return "", fmt.Errorf("打开源文件失败: %w", err)
		}
		defer src.Close()

		dst, err := os.Create(tempFile)
		if err != nil {
			return "", fmt.Errorf("创建临时文件失败: %w", err)
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		if err != nil {
			// 删除临时文件
			os.Remove(tempFile)
			return "", fmt.Errorf("复制文件内容失败: %w", err)
		}

		log.Printf("从本地路径复制视频到临时文件: %s", tempFile)
		return tempFile, nil
	}

	// 如果走到这里，表示无法处理这个路径
	return "", fmt.Errorf("无法处理的存储路径: %s", storagePath)
}

// 上传视频
func (a *KuaishouAdapter) uploadVideo(accessToken, videoPath string) (*struct {
	UploadID string `json:"upload_id"`
}, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s",
		kuaishouAPIBaseURL, uploadInitEndpoint, accessToken)

	// 打开文件
	file, err := os.Open(videoPath)
	if err != nil {
		return nil, fmt.Errorf("打开视频文件失败: %w", err)
	}
	defer file.Close()

	// 创建multipart请求
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加回调URL（可选）
	if a.callbackURL != "" {
		_ = writer.WriteField("callback_url", a.callbackURL)
	}

	// 添加文件部分
	part, err := writer.CreateFormFile("file", filepath.Base(videoPath))
	if err != nil {
		return nil, fmt.Errorf("创建表单文件失败: %w", err)
	}

	// 复制文件内容
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭multipart writer失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("创建上传请求失败: %w", err)
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	resp, err := a.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Result   int    `json:"result"`
		Message  string `json:"error_msg"`
		UploadID string `json:"upload_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析上传响应失败: %w", err)
	}

	// 检查响应
	if result.Result != 1 || result.UploadID == "" {
		return nil, fmt.Errorf("上传视频失败: %s", result.Message)
	}

	return &struct {
		UploadID string `json:"upload_id"`
	}{
		UploadID: result.UploadID,
	}, nil
}

// 检查上传状态
func (a *KuaishouAdapter) checkUploadStatus(accessToken, uploadID string) (*struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s&upload_id=%s",
		kuaishouAPIBaseURL, uploadStatusEndpoint, accessToken, uploadID)

	// 发送请求
	resp, err := a.client.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("检查上传状态失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Result  int    `json:"result"`
		Message string `json:"error_msg"`
		Status  string `json:"status"` // SUCCESS, PROCESSING, FAIL
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析上传状态响应失败: %w", err)
	}

	return &struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Status:  result.Status,
		Message: result.Message,
	}, nil
}

// 发布视频
func (a *KuaishouAdapter) publishVideo(accessToken, uploadID, title, description string) (*struct {
	PhotoID  string `json:"photo_id"`
	ShareURL string `json:"share_url"`
}, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s",
		kuaishouAPIBaseURL, publishEndpoint, accessToken)

	// 构建请求体
	requestBody := struct {
		UploadID    string `json:"upload_id"`
		Caption     string `json:"caption"`
		Description string `json:"description,omitempty"`
	}{
		UploadID:    uploadID,
		Caption:     title,
		Description: description,
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化发布视频请求体失败: %w", err)
	}

	// 发送请求
	resp, err := a.client.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("发送发布视频请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Result   int    `json:"result"`
		Message  string `json:"error_msg"`
		PhotoID  string `json:"photo_id"`
		ShareURL string `json:"share_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析发布视频响应失败: %w", err)
	}

	// 检查响应
	if result.Result != 1 || result.PhotoID == "" {
		return nil, fmt.Errorf("发布视频失败: %s", result.Message)
	}

	return &struct {
		PhotoID  string `json:"photo_id"`
		ShareURL string `json:"share_url"`
	}{
		PhotoID:  result.PhotoID,
		ShareURL: result.ShareURL,
	}, nil
}

// GenerateShareLink 生成分享链接
func (a *KuaishouAdapter) GenerateShareLink(ctx context.Context, platformID string, extraParams map[string]interface{}) (string, error) {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return "", fmt.Errorf("获取快手访问令牌失败: %w", err)
	}

	// 尝试先从平台获取状态信息，检查是否已有分享链接
	status, err := a.GetPublishStatus(ctx, platformID)
	if err == nil && status["shareUrl"] != "" {
		// 如果已经有分享链接，直接返回
		return status["shareUrl"].(string), nil
	}

	// 快手API不提供专门的分享链接生成接口，分享链接在发布视频时会自动生成
	// 如果需要重新获取分享链接，可以通过视频状态接口获取
	url := fmt.Sprintf("%s%s?access_token=%s&photo_id=%s",
		kuaishouAPIBaseURL, videoStatusEndpoint, accessToken, platformID)

	// 发送请求
	resp, err := a.client.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("获取视频状态失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Result  int    `json:"result"`
		Message string `json:"error_msg"`
		Data    struct {
			PhotoID    string `json:"photo_id"`
			Title      string `json:"caption"`
			CreateTime int64  `json:"create_time"`
			Status     string `json:"status"`
			ShareURL   string `json:"share_url"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析视频状态响应失败: %w", err)
	}

	// 检查响应
	if result.Result != 1 {
		return "", fmt.Errorf("获取视频状态失败: %s", result.Message)
	}

	// 返回分享链接
	if result.Data.ShareURL == "" {
		return "", fmt.Errorf("视频分享链接不可用")
	}

	return result.Data.ShareURL, nil
}

// GetDetailedStats 获取详细统计数据
func (a *KuaishouAdapter) GetDetailedStats(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("获取快手访问令牌失败: %w", err)
	}

	// 构建请求URL - 使用视频统计接口
	url := fmt.Sprintf("%s/openapi/photo/stat?access_token=%s&photo_id=%s",
		kuaishouAPIBaseURL, accessToken, platformID)

	// 发送请求
	resp, err := a.client.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取视频统计数据失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Result  int    `json:"result"`
		Message string `json:"error_msg"`
		Data    struct {
			PhotoID    string `json:"photo_id"`
			Title      string `json:"caption"`
			CreateTime int64  `json:"create_time"`
			Status     string `json:"status"`
			ShareURL   string `json:"share_url"`
			Metrics    struct {
				LikeCount    int64 `json:"like_count"`
				CommentCount int64 `json:"comment_count"`
				ViewCount    int64 `json:"view_count"`
				ShareCount   int64 `json:"share_count"`
				CollectCount int64 `json:"collect_count"`
			} `json:"metrics"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析视频统计响应失败: %w", err)
	}

	// 检查响应
	if result.Result != 1 {
		return nil, fmt.Errorf("获取视频统计数据失败: %s", result.Message)
	}

	// 创建详细统计数据
	detailedStats := map[string]interface{}{
		"platformId":   result.Data.PhotoID,
		"title":        result.Data.Title,
		"createTime":   time.Unix(result.Data.CreateTime, 0),
		"status":       result.Data.Status,
		"shareUrl":     result.Data.ShareURL,
		"likeCount":    result.Data.Metrics.LikeCount,
		"commentCount": result.Data.Metrics.CommentCount,
		"viewCount":    result.Data.Metrics.ViewCount,
		"shareCount":   result.Data.Metrics.ShareCount,
		"collectCount": result.Data.Metrics.CollectCount,
		"totalEngagement": result.Data.Metrics.LikeCount +
			result.Data.Metrics.CommentCount +
			result.Data.Metrics.ViewCount +
			result.Data.Metrics.ShareCount +
			result.Data.Metrics.CollectCount,
		"lastUpdated": time.Now(),
	}

	return detailedStats, nil
}

// Adapter 快手平台适配器
type Adapter struct {
	config config.KuaishouConfig
}

// NewAdapter 创建快手适配器
func NewAdapter(config config.KuaishouConfig) *Adapter {
	return &Adapter{
		config: config,
	}
}

// PublishVideo 发布视频到快手
func (a *Adapter) PublishVideo(ctx context.Context, video *entities.Video, job *entities.PublishJob) error {
	// 创建快手适配器实例
	kuaishouAdapter := NewKuaishouAdapter(a.config, os.TempDir())

	// 调用KuaishouAdapter的上传视频方法
	err := kuaishouAdapter.UploadVideo(ctx, video, job)
	if err != nil {
		job.Status = "failed"
		job.ErrorMsg = fmt.Sprintf("发布视频到快手失败: %v", err)
		job.UpdatedAt = time.Now()
		return err
	}

	// 尝试获取更详细的视频状态
	if job.Status == "completed" && job.Result != nil {
		if platformID, ok := job.Result["platformId"].(string); ok && platformID != "" {
			// 获取发布状态
			status, statusErr := kuaishouAdapter.GetPublishStatus(ctx, platformID)
			if statusErr == nil {
				// 更新结果信息
				for k, v := range status {
					job.Result[k] = v
				}
			}

			// 尝试获取更详细的统计数据
			stats, statsErr := kuaishouAdapter.GetDetailedStats(ctx, platformID)
			if statsErr == nil {
				// 更新结果信息中的统计数据
				for k, v := range stats {
					if k != "platformId" && k != "title" { // 避免覆盖基本信息
						job.Result[k] = v
					}
				}
			}

			// 确保有分享链接
			if shareURL, ok := job.Result["shareUrl"].(string); !ok || shareURL == "" {
				if shareLink, linkErr := kuaishouAdapter.GenerateShareLink(ctx, platformID, job.Params); linkErr == nil {
					job.Result["shareUrl"] = shareLink
				}
			}
		}
	}

	return nil
}

// GetPublishStatus 获取平台发布状态
func (a *Adapter) GetPublishStatus(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 创建快手适配器实例
	kuaishouAdapter := NewKuaishouAdapter(a.config, os.TempDir())

	// 直接调用KuaishouAdapter的GetPublishStatus方法
	return kuaishouAdapter.GetPublishStatus(ctx, platformID)
}

// GenerateShareLink 生成分享链接
func (a *Adapter) GenerateShareLink(ctx context.Context, platformID string, extraParams map[string]interface{}) (string, error) {
	// 创建快手适配器实例
	kuaishouAdapter := NewKuaishouAdapter(a.config, os.TempDir())

	// 直接调用KuaishouAdapter的GenerateShareLink方法
	return kuaishouAdapter.GenerateShareLink(ctx, platformID, extraParams)
}

// GenerateJSConfig 生成JS SDK配置
func (a *Adapter) GenerateJSConfig(ctx context.Context, url string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("快手平台不支持JSSDK配置生成")
}

// GetDetailedStats 获取详细统计数据
func (a *Adapter) GetDetailedStats(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 创建快手适配器实例
	kuaishouAdapter := NewKuaishouAdapter(a.config, os.TempDir())

	// 直接调用KuaishouAdapter的GetDetailedStats方法
	return kuaishouAdapter.GetDetailedStats(ctx, platformID)
}
