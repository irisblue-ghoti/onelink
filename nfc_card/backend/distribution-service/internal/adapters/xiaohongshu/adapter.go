package xiaohongshu

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
	// 小红书开放平台API基础URL
	xiaohongshuAPIBaseURL = "https://ark.xiaohongshu.com"

	// 视频上传和发布端点
	uploadInitEndpoint = "/api/oauth/ark/media/upload"
	publishEndpoint    = "/api/oauth/ark/content/create/video"

	// 视频状态查询端点
	videoStatusEndpoint = "/api/oauth/ark/content/video_status"

	// JSSDK配置端点
	jssdkConfigEndpoint = "/api/oauth/ark/jssdk/config"
)

// XiaohongshuClient 小红书API客户端
type XiaohongshuClient struct {
	appID       string
	appSecret   string
	accessToken string
	httpClient  *http.Client
}

// NewXiaohongshuClient 创建小红书客户端
func NewXiaohongshuClient(appID, appSecret string) *XiaohongshuClient {
	return &XiaohongshuClient{
		appID:      appID,
		appSecret:  appSecret,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAccessToken 获取访问令牌
func (c *XiaohongshuClient) GetAccessToken() (string, error) {
	// 如果令牌有效，直接返回
	if c.accessToken != "" {
		return c.accessToken, nil
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/api/oauth/token?app_id=%s&app_secret=%s&grant_type=client_credentials",
		xiaohongshuAPIBaseURL, c.appID, c.appSecret)

	// 发送请求
	resp, err := c.httpClient.Post(url, "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("获取访问令牌失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Code        int    `json:"code"`
		Success     bool   `json:"success"`
		Message     string `json:"message"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析访问令牌响应失败: %w", err)
	}

	// 检查响应
	if !result.Success || result.Code != 0 || result.AccessToken == "" {
		return "", fmt.Errorf("获取访问令牌失败: %s", result.Message)
	}

	// 保存访问令牌
	c.accessToken = result.AccessToken

	return c.accessToken, nil
}

// XiaohongshuAdapter 小红书适配器
type XiaohongshuAdapter struct {
	client      *XiaohongshuClient
	tempDir     string
	callbackURL string
}

// NewXiaohongshuAdapter 创建小红书适配器
func NewXiaohongshuAdapter(config config.XiaohongshuConfig, tempDir string) *XiaohongshuAdapter {
	client := NewXiaohongshuClient(config.AppID, config.AppSecret)
	return &XiaohongshuAdapter{
		client:      client,
		tempDir:     tempDir,
		callbackURL: config.CallbackURL,
	}
}

// UploadVideo 上传视频到小红书
func (a *XiaohongshuAdapter) UploadVideo(ctx context.Context, video *entities.Video, job *entities.PublishJob) error {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return fmt.Errorf("获取小红书访问令牌失败: %w", err)
	}

	// 下载视频到临时目录
	videoPath, err := a.downloadVideo(video.StoragePath)
	if err != nil {
		return fmt.Errorf("下载视频失败: %w", err)
	}
	defer os.Remove(videoPath) // 确保临时文件被删除

	// 上传视频
	uploadResult, err := a.uploadVideo(accessToken, videoPath)
	if err != nil {
		return fmt.Errorf("上传视频失败: %w", err)
	}

	// 发布视频
	publishResult, err := a.publishVideo(accessToken, uploadResult.FileID, video.Title, video.Description, video.CoverURL)
	if err != nil {
		return fmt.Errorf("发布视频失败: %w", err)
	}

	// 更新任务状态
	job.Status = "completed"
	job.CompletedAt = time.Now()
	job.UpdatedAt = time.Now()
	job.Result = map[string]interface{}{
		"platformId": publishResult.NoteID,
		"url":        publishResult.ShareURL,
	}

	return nil
}

// GetPublishStatus 获取平台发布状态
func (a *XiaohongshuAdapter) GetPublishStatus(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("获取小红书访问令牌失败: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s",
		xiaohongshuAPIBaseURL, videoStatusEndpoint, accessToken)

	// 构建请求体
	requestBody := struct {
		NoteID string `json:"note_id"`
	}{
		NoteID: platformID,
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化视频状态请求体失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建视频状态请求失败: %w", err)
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := a.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取视频状态失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Code    int    `json:"code"`
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			NoteID       string `json:"note_id"`
			Status       string `json:"status"` // PUBLISHED, REVIEWING, REJECTED
			RejectReason string `json:"reject_reason,omitempty"`
			Title        string `json:"title"`
			ShareURL     string `json:"share_url"`
			CreateTime   int64  `json:"create_time"`
			Stats        struct {
				LikeCount    int `json:"likes"`
				CommentCount int `json:"comments"`
				CollectCount int `json:"collects"`
				ViewCount    int `json:"views"`
			} `json:"stats"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析视频状态响应失败: %w", err)
	}

	// 检查响应
	if !result.Success || result.Code != 0 {
		return nil, fmt.Errorf("获取视频状态失败: %s", result.Message)
	}

	// 返回状态
	return map[string]interface{}{
		"platformId":   result.Data.NoteID,
		"title":        result.Data.Title,
		"status":       result.Data.Status,
		"rejectReason": result.Data.RejectReason,
		"createTime":   time.Unix(result.Data.CreateTime, 0),
		"shareUrl":     result.Data.ShareURL,
		"likeCount":    result.Data.Stats.LikeCount,
		"commentCount": result.Data.Stats.CommentCount,
		"collectCount": result.Data.Stats.CollectCount,
		"viewCount":    result.Data.Stats.ViewCount,
	}, nil
}

// GenerateJSConfig 生成JSSDK配置
func (a *XiaohongshuAdapter) GenerateJSConfig(ctx context.Context, url string) (map[string]interface{}, error) {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("获取小红书访问令牌失败: %w", err)
	}

	// 构建请求URL
	apiURL := fmt.Sprintf("%s%s?access_token=%s",
		xiaohongshuAPIBaseURL, jssdkConfigEndpoint, accessToken)

	// 生成时间戳和随机字符串
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonceStr := uuid.New().String()

	// 构建请求体
	requestBody := struct {
		URL       string `json:"url"`
		Timestamp string `json:"timestamp"`
		NonceStr  string `json:"nonce_str"`
	}{
		URL:       url,
		Timestamp: timestamp,
		NonceStr:  nonceStr,
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化JSSDK配置请求体失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建JSSDK配置请求失败: %w", err)
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := a.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取JSSDK配置失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Code    int    `json:"code"`
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			AppID     string `json:"app_id"`
			Timestamp string `json:"timestamp"`
			NonceStr  string `json:"nonce_str"`
			Signature string `json:"signature"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析JSSDK配置响应失败: %w", err)
	}

	// 检查响应
	if !result.Success || result.Code != 0 {
		return nil, fmt.Errorf("获取JSSDK配置失败: %s", result.Message)
	}

	// 返回配置
	return map[string]interface{}{
		"appId":     result.Data.AppID,
		"timestamp": result.Data.Timestamp,
		"nonceStr":  result.Data.NonceStr,
		"signature": result.Data.Signature,
	}, nil
}

// GenerateShareLink 生成分享链接
func (a *XiaohongshuAdapter) GenerateShareLink(ctx context.Context, platformID string, extraParams map[string]interface{}) (string, error) {
	// 尝试先从平台获取状态信息，检查是否已有分享链接
	status, err := a.GetPublishStatus(ctx, platformID)
	if err == nil && status["shareUrl"] != "" {
		// 如果已经有分享链接，直接返回
		return status["shareUrl"].(string), nil
	}

	// 如果没有分享链接，创建一个带有追踪参数的链接
	// 假设 status["shareUrl"] 是基础链接
	if status != nil && status["shareUrl"] != "" {
		baseURL := status["shareUrl"].(string)

		// 添加额外参数
		if len(extraParams) > 0 {
			separator := "?"
			if strings.Contains(baseURL, "?") {
				separator = "&"
			}

			for key, value := range extraParams {
				baseURL += fmt.Sprintf("%s%s=%v", separator, key, value)
				separator = "&"
			}
		}

		return baseURL, nil
	}

	return "", fmt.Errorf("无法获取分享链接，请确保内容已发布")
}

// 下载视频到临时目录
func (a *XiaohongshuAdapter) downloadVideo(storagePath string) (string, error) {
	// 检查storagePath是否已经是本地临时文件
	if strings.HasPrefix(filepath.Base(storagePath), "s3_") ||
		strings.HasPrefix(filepath.Base(storagePath), "url_") {
		// 已经是临时文件，直接返回
		log.Printf("使用已下载的临时文件: %s", storagePath)
		return storagePath, nil
	}

	// 创建临时文件
	tempID := uuid.New().String()
	tempFile := filepath.Join(a.tempDir, fmt.Sprintf("xiaohongshu_%s.mp4", tempID))

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
func (a *XiaohongshuAdapter) uploadVideo(accessToken, videoPath string) (*struct {
	FileID string `json:"file_id"`
}, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s",
		xiaohongshuAPIBaseURL, uploadInitEndpoint, accessToken)

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

	// 指定文件类型
	_ = writer.WriteField("file_type", "video")

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
		Code    int    `json:"code"`
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			FileID string `json:"file_id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析上传响应失败: %w", err)
	}

	// 检查响应
	if !result.Success || result.Code != 0 || result.Data.FileID == "" {
		return nil, fmt.Errorf("上传视频失败: %s", result.Message)
	}

	return &struct {
		FileID string `json:"file_id"`
	}{
		FileID: result.Data.FileID,
	}, nil
}

// 发布视频
func (a *XiaohongshuAdapter) publishVideo(accessToken, fileID, title, description, coverURL string) (*struct {
	NoteID   string `json:"note_id"`
	ShareURL string `json:"share_url"`
}, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s",
		xiaohongshuAPIBaseURL, publishEndpoint, accessToken)

	// 构建请求体
	requestBody := struct {
		FileID      string   `json:"file_id"`
		Title       string   `json:"title"`
		Desc        string   `json:"desc,omitempty"`
		CoverURL    string   `json:"cover_url,omitempty"`
		Tags        []string `json:"tags,omitempty"`
		PublishTime int64    `json:"publish_time,omitempty"` // 可选，发布时间戳
	}{
		FileID:   fileID,
		Title:    title,
		Desc:     description,
		CoverURL: coverURL,
		Tags:     []string{"视频", "NFC碰一碰"},
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化发布视频请求体失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建发布视频请求失败: %w", err)
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := a.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送发布视频请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Code    int    `json:"code"`
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			NoteID   string `json:"note_id"`
			ShareURL string `json:"share_url"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析发布视频响应失败: %w", err)
	}

	// 检查响应
	if !result.Success || result.Code != 0 || result.Data.NoteID == "" {
		return nil, fmt.Errorf("发布视频失败: %s", result.Message)
	}

	return &struct {
		NoteID   string `json:"note_id"`
		ShareURL string `json:"share_url"`
	}{
		NoteID:   result.Data.NoteID,
		ShareURL: result.Data.ShareURL,
	}, nil
}

// Adapter 小红书平台适配器
type Adapter struct {
	config config.XiaohongshuConfig
}

// NewAdapter 创建小红书适配器
func NewAdapter(config config.XiaohongshuConfig) *Adapter {
	return &Adapter{
		config: config,
	}
}

// PublishVideo 发布视频到小红书
func (a *Adapter) PublishVideo(ctx context.Context, video *entities.Video, job *entities.PublishJob) error {
	// 创建小红书适配器实例
	xiaohongshuAdapter := NewXiaohongshuAdapter(a.config, os.TempDir())

	// 更新任务状态
	job.Status = "processing"
	job.UpdatedAt = time.Now()

	// 调用XiaohongshuAdapter的上传视频方法
	err := xiaohongshuAdapter.UploadVideo(ctx, video, job)
	if err != nil {
		job.Status = "failed"
		job.ErrorMsg = fmt.Sprintf("发布视频到小红书失败: %v", err)
		job.UpdatedAt = time.Now()
		return err
	}

	// 尝试获取更详细的视频状态
	if job.Status == "completed" && job.Result != nil {
		if platformID, ok := job.Result["platformId"].(string); ok && platformID != "" {
			// 获取发布状态
			status, statusErr := xiaohongshuAdapter.GetPublishStatus(ctx, platformID)
			if statusErr == nil {
				// 更新结果信息
				for k, v := range status {
					job.Result[k] = v
				}
			}

			// 确保有分享链接
			if shareURL, ok := job.Result["shareUrl"].(string); !ok || shareURL == "" {
				if shareLink, linkErr := xiaohongshuAdapter.GenerateShareLink(ctx, platformID, job.Params); linkErr == nil {
					job.Result["shareUrl"] = shareLink
				}
			}
		}
	}

	return nil
}

// GetPublishStatus 获取平台发布状态
func (a *Adapter) GetPublishStatus(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 创建小红书适配器实例
	xiaohongshuAdapter := NewXiaohongshuAdapter(a.config, os.TempDir())

	// 直接调用XiaohongshuAdapter的GetPublishStatus方法
	return xiaohongshuAdapter.GetPublishStatus(ctx, platformID)
}

// GenerateShareLink 生成分享链接
func (a *Adapter) GenerateShareLink(ctx context.Context, platformID string, extraParams map[string]interface{}) (string, error) {
	// 创建小红书适配器实例
	xiaohongshuAdapter := NewXiaohongshuAdapter(a.config, os.TempDir())

	// 直接调用XiaohongshuAdapter的GenerateShareLink方法
	return xiaohongshuAdapter.GenerateShareLink(ctx, platformID, extraParams)
}

// GenerateJSConfig 生成JS SDK配置
func (a *Adapter) GenerateJSConfig(ctx context.Context, url string) (map[string]interface{}, error) {
	// 创建小红书适配器实例
	xiaohongshuAdapter := NewXiaohongshuAdapter(a.config, os.TempDir())

	// 直接调用XiaohongshuAdapter的GenerateJSConfig方法
	return xiaohongshuAdapter.GenerateJSConfig(ctx, url)
}

// GetDetailedStats 获取详细统计数据
func (a *Adapter) GetDetailedStats(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 创建小红书适配器实例
	xiaohongshuAdapter := NewXiaohongshuAdapter(a.config, os.TempDir())

	// 获取平台发布状态
	status, err := xiaohongshuAdapter.GetPublishStatus(ctx, platformID)
	if err != nil {
		return nil, err
	}

	// 小红书平台获取基本的统计信息已经在GetPublishStatus中
	// 添加总参与度计算
	if status != nil {
		// 计算总参与度
		var totalEngagement int64 = 0

		if likeCount, ok := status["likeCount"].(int); ok {
			totalEngagement += int64(likeCount)
		}

		if commentCount, ok := status["commentCount"].(int); ok {
			totalEngagement += int64(commentCount)
		}

		if collectCount, ok := status["collectCount"].(int); ok {
			totalEngagement += int64(collectCount)
		}

		if viewCount, ok := status["viewCount"].(int); ok {
			totalEngagement += int64(viewCount)
		}

		// 添加总参与度和最后更新时间
		status["totalEngagement"] = totalEngagement
		status["lastUpdated"] = time.Now()
	}

	return status, nil
}
