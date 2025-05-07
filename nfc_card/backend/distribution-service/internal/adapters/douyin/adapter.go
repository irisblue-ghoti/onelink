package douyin

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
	// 抖音开放平台API基础URL
	douyinAPIBaseURL = "https://open.douyin.com"

	// 视频上传和发布端点
	uploadInitEndpoint     = "/video/upload/init/"
	uploadPartEndpoint     = "/video/upload/part/"
	uploadCompleteEndpoint = "/video/upload/complete/"
	publishVideoEndpoint   = "/video/publish/"

	// 视频状态查询端点
	videoStatusEndpoint = "/video/data/"

	// 分享链接生成端点
	shareVideoEndpoint = "/share/video/create/"

	// 分片大小 - 5MB
	chunkSize = 5 * 1024 * 1024
)

// DouyinClient 抖音API客户端
type DouyinClient struct {
	clientKey    string
	clientSecret string
	accessToken  string
	httpClient   *http.Client
}

// NewDouyinClient 创建抖音客户端
func NewDouyinClient(clientKey, clientSecret string) *DouyinClient {
	return &DouyinClient{
		clientKey:    clientKey,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAccessToken 获取访问令牌
func (c *DouyinClient) GetAccessToken() (string, error) {
	// 如果令牌有效，直接返回
	if c.accessToken != "" {
		return c.accessToken, nil
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/oauth/client_token/?client_key=%s&client_secret=%s&grant_type=client_credential",
		douyinAPIBaseURL, c.clientKey, c.clientSecret)

	// 发送请求
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("获取访问令牌失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Data struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析访问令牌响应失败: %w", err)
	}

	// 检查响应
	if result.Data.AccessToken == "" {
		return "", fmt.Errorf("获取访问令牌失败: %s", result.Message)
	}

	// 保存访问令牌
	c.accessToken = result.Data.AccessToken

	return c.accessToken, nil
}

// DouyinAdapter 抖音适配器
type DouyinAdapter struct {
	client  *DouyinClient
	tempDir string
}

// NewDouyinAdapter 创建抖音适配器
func NewDouyinAdapter(config config.DouyinConfig, tempDir string) *DouyinAdapter {
	client := NewDouyinClient(config.ClientKey, config.ClientSecret)
	return &DouyinAdapter{
		client:  client,
		tempDir: tempDir,
	}
}

// UploadVideo 上传视频到抖音
func (a *DouyinAdapter) UploadVideo(ctx context.Context, video *entities.Video, job *entities.PublishJob) error {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return fmt.Errorf("获取抖音访问令牌失败: %w", err)
	}

	// 下载视频到临时目录
	videoPath, err := a.downloadVideo(video.StoragePath)
	if err != nil {
		return fmt.Errorf("下载视频失败: %w", err)
	}
	defer os.Remove(videoPath) // 确保临时文件被删除

	// 初始化上传
	uploadID, err := a.initUpload(accessToken)
	if err != nil {
		return fmt.Errorf("初始化上传失败: %w", err)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(videoPath)
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 计算分片数量
	fileSize := fileInfo.Size()
	chunks := int(fileSize / chunkSize)
	if fileSize%chunkSize != 0 {
		chunks++
	}

	// 打开文件
	file, err := os.Open(videoPath)
	if err != nil {
		return fmt.Errorf("打开视频文件失败: %w", err)
	}
	defer file.Close()

	// 上传分片
	for i := 0; i < chunks; i++ {
		// 更新任务状态
		job.Status = "processing"
		job.UpdatedAt = time.Now()

		// 读取分片数据
		buffer := make([]byte, chunkSize)
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("读取分片数据失败: %w", err)
		}

		// 上传分片
		err = a.uploadChunk(accessToken, uploadID, i+1, buffer[:bytesRead])
		if err != nil {
			return fmt.Errorf("上传第%d个分片失败: %w", i+1, err)
		}
	}

	// 完成上传
	uploadResult, err := a.completeUpload(accessToken, uploadID)
	if err != nil {
		return fmt.Errorf("完成上传失败: %w", err)
	}

	// 发布视频
	publishResult, err := a.publishVideo(accessToken, uploadResult.VideoID, video.Title, video.Description)
	if err != nil {
		return fmt.Errorf("发布视频失败: %w", err)
	}

	// 更新任务状态
	job.Status = "completed"
	job.CompletedAt = time.Now()
	job.UpdatedAt = time.Now()
	job.Result = map[string]interface{}{
		"platformId": publishResult.ItemID,
		"url":        publishResult.ShareURL,
	}

	return nil
}

// GetPublishStatus 获取平台发布状态
func (a *DouyinAdapter) GetPublishStatus(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("获取抖音访问令牌失败: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s&item_id=%s",
		douyinAPIBaseURL, videoStatusEndpoint, accessToken, platformID)

	// 发送请求
	resp, err := a.client.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取视频状态失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Data struct {
			ItemID     string `json:"item_id"`
			Title      string `json:"title"`
			CreateTime int64  `json:"create_time"`
			IsReviewed bool   `json:"is_reviewed"`
			ShareURL   string `json:"share_url"`
			Statistics struct {
				CommentCount int `json:"comment_count"`
				DiggCount    int `json:"digg_count"`
				PlayCount    int `json:"play_count"`
				ShareCount   int `json:"share_count"`
			} `json:"statistics"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析视频状态响应失败: %w", err)
	}

	// 检查响应
	if result.Data.ItemID == "" {
		return nil, fmt.Errorf("获取视频状态失败: %s", result.Message)
	}

	// 返回状态
	return map[string]interface{}{
		"platformId":   result.Data.ItemID,
		"title":        result.Data.Title,
		"createTime":   time.Unix(result.Data.CreateTime, 0),
		"isReviewed":   result.Data.IsReviewed,
		"shareUrl":     result.Data.ShareURL,
		"commentCount": result.Data.Statistics.CommentCount,
		"likeCount":    result.Data.Statistics.DiggCount,
		"playCount":    result.Data.Statistics.PlayCount,
		"shareCount":   result.Data.Statistics.ShareCount,
	}, nil
}

// 下载视频到临时目录
func (a *DouyinAdapter) downloadVideo(storagePath string) (string, error) {
	// 检查storagePath是否已经是本地临时文件
	if strings.HasPrefix(filepath.Base(storagePath), "s3_") ||
		strings.HasPrefix(filepath.Base(storagePath), "url_") {
		// 已经是临时文件，直接返回
		log.Printf("使用已下载的临时文件: %s", storagePath)
		return storagePath, nil
	}

	// 创建临时文件
	tempID := uuid.New().String()
	tempFile := filepath.Join(a.tempDir, fmt.Sprintf("douyin_%s.mp4", tempID))

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

// 初始化上传
func (a *DouyinAdapter) initUpload(accessToken string) (string, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s",
		douyinAPIBaseURL, uploadInitEndpoint, accessToken)

	// 发送请求
	resp, err := a.client.httpClient.Post(url, "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("初始化上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Data struct {
			UploadID string `json:"upload_id"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析初始化上传响应失败: %w", err)
	}

	// 检查响应
	if result.Data.UploadID == "" {
		return "", fmt.Errorf("初始化上传失败: %s", result.Message)
	}

	return result.Data.UploadID, nil
}

// 上传分片
func (a *DouyinAdapter) uploadChunk(accessToken, uploadID string, partNumber int, data []byte) error {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s&upload_id=%s&part_number=%d",
		douyinAPIBaseURL, uploadPartEndpoint, accessToken, uploadID, partNumber)

	// 创建multipart请求
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件部分
	part, err := writer.CreateFormFile("video", fmt.Sprintf("part_%d.mp4", partNumber))
	if err != nil {
		return fmt.Errorf("创建表单文件失败: %w", err)
	}

	// 写入分片数据
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("写入分片数据失败: %w", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("关闭multipart writer失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("创建上传分片请求失败: %w", err)
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	resp, err := a.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送上传分片请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("上传分片失败，HTTP状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var result struct {
		Data struct {
			PartNumber int `json:"part_number"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析上传分片响应失败: %w", err)
	}

	// 检查响应
	if result.Data.PartNumber != partNumber {
		return fmt.Errorf("上传分片失败: %s", result.Message)
	}

	return nil
}

// 完成上传
func (a *DouyinAdapter) completeUpload(accessToken, uploadID string) (*struct {
	VideoID string `json:"video_id"`
}, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s&upload_id=%s",
		douyinAPIBaseURL, uploadCompleteEndpoint, accessToken, uploadID)

	// 发送请求
	resp, err := a.client.httpClient.Post(url, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("完成上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Data struct {
			VideoID string `json:"video_id"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析完成上传响应失败: %w", err)
	}

	// 检查响应
	if result.Data.VideoID == "" {
		return nil, fmt.Errorf("完成上传失败: %s", result.Message)
	}

	return &struct {
		VideoID string `json:"video_id"`
	}{
		VideoID: result.Data.VideoID,
	}, nil
}

// 发布视频
func (a *DouyinAdapter) publishVideo(accessToken, videoID, title, description string) (*struct {
	ItemID   string `json:"item_id"`
	ShareURL string `json:"share_url"`
}, error) {
	// 使用无额外参数调用publishVideoWithParams
	return a.publishVideoWithParams(accessToken, videoID, title, description, nil)
}

// GenerateShareLink 生成分享链接
func (a *DouyinAdapter) GenerateShareLink(ctx context.Context, platformID string, extraParams map[string]interface{}) (string, error) {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return "", fmt.Errorf("获取抖音访问令牌失败: %w", err)
	}

	// 尝试先从平台获取状态信息，检查是否已有分享链接
	status, err := a.GetPublishStatus(ctx, platformID)
	if err == nil && status["shareUrl"] != "" {
		// 如果已经有分享链接，直接返回
		return status["shareUrl"].(string), nil
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s",
		douyinAPIBaseURL, shareVideoEndpoint, accessToken)

	// 构建请求体
	requestBody := map[string]interface{}{
		"item_id": platformID,
	}

	// 添加额外参数（如果有）
	for k, v := range extraParams {
		requestBody[k] = v
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("序列化生成分享链接请求体失败: %w", err)
	}

	// 发送请求
	resp, err := a.client.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("发送生成分享链接请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Data struct {
			ShareID  string `json:"share_id"`
			ShareURL string `json:"share_url"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析生成分享链接响应失败: %w", err)
	}

	// 检查响应
	if result.Data.ShareURL == "" {
		return "", fmt.Errorf("生成分享链接失败: %s", result.Message)
	}

	return result.Data.ShareURL, nil
}

// UploadVideoWithCover 上传视频和封面到抖音
func (a *DouyinAdapter) UploadVideoWithCover(ctx context.Context, video *entities.Video, coverPath string, job *entities.PublishJob) error {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return fmt.Errorf("获取抖音访问令牌失败: %w", err)
	}

	// 下载视频到临时目录
	videoPath, err := a.downloadVideo(video.StoragePath)
	if err != nil {
		return fmt.Errorf("下载视频失败: %w", err)
	}
	defer os.Remove(videoPath) // 确保临时文件被删除

	// 上传封面图片（如果提供）
	var coverURL string
	if coverPath != "" {
		coverURL, err = a.uploadCoverImage(accessToken, coverPath)
		if err != nil {
			log.Printf("警告：封面图片上传失败: %v", err)
			// 不中断主流程，继续上传视频
		}
	}

	// 初始化上传
	uploadID, err := a.initUpload(accessToken)
	if err != nil {
		return fmt.Errorf("初始化上传失败: %w", err)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(videoPath)
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 计算分片数量
	fileSize := fileInfo.Size()
	chunks := int(fileSize / chunkSize)
	if fileSize%chunkSize != 0 {
		chunks++
	}

	// 打开文件
	file, err := os.Open(videoPath)
	if err != nil {
		return fmt.Errorf("打开视频文件失败: %w", err)
	}
	defer file.Close()

	// 上传分片
	for i := 0; i < chunks; i++ {
		// 更新任务状态
		job.Status = "processing"
		job.UpdatedAt = time.Now()

		// 读取分片数据
		buffer := make([]byte, chunkSize)
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("读取分片数据失败: %w", err)
		}

		// 上传分片
		err = a.uploadChunk(accessToken, uploadID, i+1, buffer[:bytesRead])
		if err != nil {
			return fmt.Errorf("上传第%d个分片失败: %w", i+1, err)
		}
	}

	// 完成上传
	uploadResult, err := a.completeUpload(accessToken, uploadID)
	if err != nil {
		return fmt.Errorf("完成上传失败: %w", err)
	}

	// 如果有封面URL，添加到参数中
	if coverURL != "" {
		if job.Params == nil {
			job.Params = make(map[string]interface{})
		}
		job.Params["coverUrl"] = coverURL
	}

	// 发布视频
	publishResult, err := a.publishVideoWithParams(accessToken, uploadResult.VideoID, video.Title, video.Description, job.Params)
	if err != nil {
		return fmt.Errorf("发布视频失败: %w", err)
	}

	// 更新任务状态
	job.Status = "completed"
	job.CompletedAt = time.Now()
	job.UpdatedAt = time.Now()
	job.Result = map[string]interface{}{
		"platformId": publishResult.ItemID,
		"url":        publishResult.ShareURL,
	}

	return nil
}

// uploadCoverImage 上传封面图片
func (a *DouyinAdapter) uploadCoverImage(accessToken, imagePath string) (string, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s/image/upload/?access_token=%s",
		douyinAPIBaseURL, accessToken)

	// 打开文件
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("打开封面图片文件失败: %w", err)
	}
	defer file.Close()

	// 创建multipart请求
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件部分
	part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
	if err != nil {
		return "", fmt.Errorf("创建表单文件失败: %w", err)
	}

	// 复制文件内容
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("关闭multipart writer失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("创建上传封面请求失败: %w", err)
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	resp, err := a.client.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送上传封面请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Data struct {
			ImageID  string `json:"image_id"`
			ImageURL string `json:"image_url"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析上传封面响应失败: %w", err)
	}

	// 检查响应
	if result.Data.ImageURL == "" {
		return "", fmt.Errorf("上传封面失败: %s", result.Message)
	}

	return result.Data.ImageURL, nil
}

// publishVideoWithParams 使用额外参数发布视频
func (a *DouyinAdapter) publishVideoWithParams(accessToken, videoID, title, description string, params map[string]interface{}) (*struct {
	ItemID   string `json:"item_id"`
	ShareURL string `json:"share_url"`
}, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s",
		douyinAPIBaseURL, publishVideoEndpoint, accessToken)

	// 构建请求体
	requestBody := struct {
		VideoID     string   `json:"video_id"`
		Title       string   `json:"title"`
		Description string   `json:"description,omitempty"`
		CoverURL    string   `json:"cover_url,omitempty"`
		Tags        []string `json:"tags,omitempty"`
		AtUsers     []string `json:"at_users,omitempty"`
		MicroAppID  string   `json:"micro_app_id,omitempty"`
		MicroAppURL string   `json:"micro_app_url,omitempty"`
		POIInfo     *struct {
			POIID   string `json:"poi_id"`
			POIName string `json:"poi_name"`
		} `json:"poi_info,omitempty"`
	}{
		VideoID:     videoID,
		Title:       title,
		Description: description,
	}

	// 添加额外参数（如果有）
	if params != nil {
		if coverURL, ok := params["coverUrl"].(string); ok && coverURL != "" {
			requestBody.CoverURL = coverURL
		}

		if tags, ok := params["tags"].([]string); ok && len(tags) > 0 {
			requestBody.Tags = tags
		} else if tagsInterface, ok := params["tags"].([]interface{}); ok && len(tagsInterface) > 0 {
			// 处理可能的接口类型列表
			requestBody.Tags = make([]string, len(tagsInterface))
			for i, tag := range tagsInterface {
				if strTag, ok := tag.(string); ok {
					requestBody.Tags[i] = strTag
				}
			}
		}

		if atUsers, ok := params["atUsers"].([]string); ok && len(atUsers) > 0 {
			requestBody.AtUsers = atUsers
		}

		if microAppID, ok := params["microAppId"].(string); ok && microAppID != "" {
			requestBody.MicroAppID = microAppID
		}

		if microAppURL, ok := params["microAppUrl"].(string); ok && microAppURL != "" {
			requestBody.MicroAppURL = microAppURL
		}

		// 处理POI信息
		if poiID, ok := params["poiId"].(string); ok && poiID != "" {
			poiName, _ := params["poiName"].(string)
			requestBody.POIInfo = &struct {
				POIID   string `json:"poi_id"`
				POIName string `json:"poi_name"`
			}{
				POIID:   poiID,
				POIName: poiName,
			}
		}
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
		Data struct {
			ItemID   string `json:"item_id"`
			ShareURL string `json:"share_url"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析发布视频响应失败: %w", err)
	}

	// 检查响应
	if result.Data.ItemID == "" {
		return nil, fmt.Errorf("发布视频失败: %s", result.Message)
	}

	return &struct {
		ItemID   string `json:"item_id"`
		ShareURL string `json:"share_url"`
	}{
		ItemID:   result.Data.ItemID,
		ShareURL: result.Data.ShareURL,
	}, nil
}

// Adapter 抖音平台适配器
type Adapter struct {
	config  config.DouyinConfig
	client  *DouyinClient
	logger  *log.Logger
	tempDir string
}

// NewAdapter 创建抖音适配器
func NewAdapter(config config.DouyinConfig, logger *log.Logger) *Adapter {
	// 创建临时目录用于处理文件
	tempDir := filepath.Join(os.TempDir(), "douyin-uploads")
	os.MkdirAll(tempDir, 0755)

	return &Adapter{
		config:  config,
		client:  NewDouyinClient(config.ClientKey, config.ClientSecret),
		logger:  logger,
		tempDir: tempDir,
	}
}

// PublishVideo 发布视频到抖音
func (a *Adapter) PublishVideo(ctx context.Context, video *entities.Video, job *entities.PublishJob) error {
	a.logger.Printf("开始发布视频到抖音: %s, 标题: %s", video.ID, video.Title)

	// 创建抖音适配器实例
	douyinAdapter := NewDouyinAdapter(a.config, a.tempDir)

	// 1. 上传视频到抖音
	err := douyinAdapter.UploadVideo(ctx, video, job)
	if err != nil {
		job.Status = "failed"
		job.ErrorMsg = err.Error()
		job.UpdatedAt = time.Now()
		return fmt.Errorf("上传视频到抖音失败: %w", err)
	}

	a.logger.Printf("成功发布视频到抖音, 平台ID: %s", job.Result["platformId"])
	return nil
}

// GetPublishStatus 获取平台发布状态
func (a *Adapter) GetPublishStatus(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 创建抖音适配器实例
	douyinAdapter := NewDouyinAdapter(a.config, a.tempDir)

	// 直接调用DouyinAdapter的GetPublishStatus方法
	return douyinAdapter.GetPublishStatus(ctx, platformID)
}

// GenerateShareLink 生成分享链接
func (a *Adapter) GenerateShareLink(ctx context.Context, platformID string, extraParams map[string]interface{}) (string, error) {
	// 创建抖音适配器实例
	douyinAdapter := NewDouyinAdapter(a.config, a.tempDir)

	// 直接调用DouyinAdapter的GenerateShareLink方法
	return douyinAdapter.GenerateShareLink(ctx, platformID, extraParams)
}

// GenerateJSConfig 生成JS SDK配置
func (a *Adapter) GenerateJSConfig(ctx context.Context, url string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("抖音平台不支持JSSDK配置生成")
}

// GetDetailedStats 获取详细统计数据
func (a *Adapter) GetDetailedStats(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 创建抖音适配器实例
	douyinAdapter := NewDouyinAdapter(a.config, a.tempDir)

	// 获取发布状态即可作为详细统计数据
	return douyinAdapter.GetPublishStatus(ctx, platformID)
}
