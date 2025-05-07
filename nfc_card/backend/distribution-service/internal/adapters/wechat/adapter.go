package wechat

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"distribution-service/internal/config"
	"distribution-service/internal/domain/entities"
)

const (
	// 微信公众平台API基础URL
	wechatAPIBaseURL = "https://api.weixin.qq.com"

	// 接口端点
	accessTokenEndpoint   = "/cgi-bin/token"
	uploadVideoEndpoint   = "/cgi-bin/material/add_material"
	publishEndpoint       = "/cgi-bin/draft/add"
	publishStatusEndpoint = "/cgi-bin/draft/get"
	jsapiTicketEndpoint   = "/cgi-bin/ticket/getticket"
)

// WechatClient 微信API客户端
type WechatClient struct {
	appID       string
	appSecret   string
	token       string
	accessToken string
	jsapiTicket string
	httpClient  *http.Client
}

// NewWechatClient 创建微信客户端
func NewWechatClient(appID, appSecret, token string) *WechatClient {
	return &WechatClient{
		appID:      appID,
		appSecret:  appSecret,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAccessToken 获取访问令牌
func (c *WechatClient) GetAccessToken() (string, error) {
	// 如果令牌有效，直接返回
	if c.accessToken != "" {
		return c.accessToken, nil
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s?grant_type=client_credential&appid=%s&secret=%s",
		wechatAPIBaseURL, accessTokenEndpoint, c.appID, c.appSecret)

	// 发送请求
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("获取访问令牌失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析访问令牌响应失败: %w", err)
	}

	// 检查响应
	if result.ErrCode != 0 || result.AccessToken == "" {
		return "", fmt.Errorf("获取访问令牌失败: %s", result.ErrMsg)
	}

	// 保存访问令牌
	c.accessToken = result.AccessToken

	return c.accessToken, nil
}

// GetJSAPITicket 获取JSAPI票据
func (c *WechatClient) GetJSAPITicket() (string, error) {
	// 如果票据有效，直接返回
	if c.jsapiTicket != "" {
		return c.jsapiTicket, nil
	}

	// 获取访问令牌
	accessToken, err := c.GetAccessToken()
	if err != nil {
		return "", fmt.Errorf("获取访问令牌失败: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s&type=jsapi",
		wechatAPIBaseURL, jsapiTicketEndpoint, accessToken)

	// 发送请求
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("获取JSAPI票据失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Ticket    string `json:"ticket"`
		ExpiresIn int    `json:"expires_in"`
		ErrCode   int    `json:"errcode"`
		ErrMsg    string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析JSAPI票据响应失败: %w", err)
	}

	// 检查响应
	if result.ErrCode != 0 || result.Ticket == "" {
		return "", fmt.Errorf("获取JSAPI票据失败: %s", result.ErrMsg)
	}

	// 保存JSAPI票据
	c.jsapiTicket = result.Ticket

	return c.jsapiTicket, nil
}

// WechatAdapter 微信适配器
type WechatAdapter struct {
	client      *WechatClient
	tempDir     string
	callbackURL string
}

// NewWechatAdapter 创建微信适配器
func NewWechatAdapter(config config.WechatConfig, tempDir string) *WechatAdapter {
	client := NewWechatClient(config.AppID, config.AppSecret, config.Token)
	return &WechatAdapter{
		client:      client,
		tempDir:     tempDir,
		callbackURL: config.CallbackURL,
	}
}

// UploadVideo 上传视频到微信
func (a *WechatAdapter) UploadVideo(ctx context.Context, video *entities.Video, job *entities.PublishJob) error {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return fmt.Errorf("获取微信访问令牌失败: %w", err)
	}

	// 下载视频到临时目录
	videoPath, err := a.downloadVideo(video.StoragePath)
	if err != nil {
		return fmt.Errorf("下载视频失败: %w", err)
	}
	defer os.Remove(videoPath) // 确保临时文件被删除

	// 上传视频素材
	materialResult, err := a.uploadVideoMaterial(accessToken, videoPath, video.Title, video.Description)
	if err != nil {
		return fmt.Errorf("上传视频素材失败: %w", err)
	}

	// 创建草稿
	draftResult, err := a.createDraft(accessToken, materialResult.MediaID, video.Title, video.Description)
	if err != nil {
		return fmt.Errorf("创建草稿失败: %w", err)
	}

	// 更新任务状态
	job.Status = "completed"
	job.CompletedAt = time.Now()
	job.UpdatedAt = time.Now()
	job.Result = map[string]interface{}{
		"platformId":   draftResult.MediaID,
		"thumbnailUrl": materialResult.URL,
	}

	return nil
}

// GetPublishStatus 获取平台发布状态
func (a *WechatAdapter) GetPublishStatus(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 获取访问令牌
	accessToken, err := a.client.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("获取微信访问令牌失败: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s&media_id=%s",
		wechatAPIBaseURL, publishStatusEndpoint, accessToken, platformID)

	// 发送请求
	resp, err := a.client.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取发布状态失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		ErrCode  int    `json:"errcode"`
		ErrMsg   string `json:"errmsg"`
		NewsItem []struct {
			Title        string `json:"title"`
			ThumbMediaID string `json:"thumb_media_id"`
			Author       string `json:"author"`
			Digest       string `json:"digest"`
			Content      string `json:"content"`
			URL          string `json:"url"`
			ContentURL   string `json:"content_url"`
		} `json:"news_item"`
		CreateTime int64 `json:"create_time"`
		UpdateTime int64 `json:"update_time"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析发布状态响应失败: %w", err)
	}

	// 检查响应
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("获取发布状态失败: %s", result.ErrMsg)
	}

	// 如果没有内容项，返回空状态
	if len(result.NewsItem) == 0 {
		return map[string]interface{}{
			"platformId": platformID,
			"createTime": time.Unix(result.CreateTime, 0),
			"updateTime": time.Unix(result.UpdateTime, 0),
			"status":     "unknown",
		}, nil
	}

	// 返回状态
	return map[string]interface{}{
		"platformId": platformID,
		"title":      result.NewsItem[0].Title,
		"author":     result.NewsItem[0].Author,
		"digest":     result.NewsItem[0].Digest,
		"url":        result.NewsItem[0].URL,
		"contentUrl": result.NewsItem[0].ContentURL,
		"createTime": time.Unix(result.CreateTime, 0),
		"updateTime": time.Unix(result.UpdateTime, 0),
		"status":     "draft",
	}, nil
}

// GenerateJSConfig 生成JSSDK配置
func (a *WechatAdapter) GenerateJSConfig(ctx context.Context, url string) (map[string]interface{}, error) {
	// 获取JSAPI票据
	jsapiTicket, err := a.client.GetJSAPITicket()
	if err != nil {
		return nil, fmt.Errorf("获取JSAPI票据失败: %w", err)
	}

	// 生成随机字符串
	nonceStr := uuid.New().String()
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// 按照微信JSSDK规则生成签名
	// 参数按照字典序排序
	params := map[string]string{
		"jsapi_ticket": jsapiTicket,
		"noncestr":     nonceStr,
		"timestamp":    timestamp,
		"url":          url,
	}

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 拼接字符串
	var str string
	for _, k := range keys {
		str += k + "=" + params[k] + "&"
	}
	str = str[:len(str)-1] // 去掉最后的&

	// 计算SHA1签名
	h := sha1.New()
	h.Write([]byte(str))
	signature := fmt.Sprintf("%x", h.Sum(nil))

	// 返回配置
	return map[string]interface{}{
		"appId":     a.client.appID,
		"timestamp": timestamp,
		"nonceStr":  nonceStr,
		"signature": signature,
	}, nil
}

// GenerateShareLink 生成分享链接
func (a *WechatAdapter) GenerateShareLink(ctx context.Context, platformID string, extraParams map[string]interface{}) (string, error) {
	// 尝试先从平台获取状态信息，检查是否已有分享链接
	status, err := a.GetPublishStatus(ctx, platformID)
	if err == nil && status["url"] != "" {
		// 如果已经有分享链接，直接返回
		return status["url"].(string), nil
	}

	// 如果没有分享链接，返回错误
	return "", fmt.Errorf("无法获取分享链接，请确保内容已发布")
}

// 下载视频到临时目录
func (a *WechatAdapter) downloadVideo(storagePath string) (string, error) {
	// 检查storagePath是否已经是本地临时文件
	if strings.HasPrefix(filepath.Base(storagePath), "s3_") ||
		strings.HasPrefix(filepath.Base(storagePath), "url_") {
		// 已经是临时文件，直接返回
		log.Printf("使用已下载的临时文件: %s", storagePath)
		return storagePath, nil
	}

	// 创建临时文件
	tempID := uuid.New().String()
	tempFile := filepath.Join(a.tempDir, fmt.Sprintf("wechat_%s.mp4", tempID))

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

// 上传视频素材
func (a *WechatAdapter) uploadVideoMaterial(accessToken, videoPath, title, introduction string) (*struct {
	Type      string `json:"type"`
	MediaID   string `json:"media_id"`
	URL       string `json:"url"`
	CreatedAt int64  `json:"created_at"`
}, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s&type=video",
		wechatAPIBaseURL, uploadVideoEndpoint, accessToken)

	// 打开文件
	file, err := os.Open(videoPath)
	if err != nil {
		return nil, fmt.Errorf("打开视频文件失败: %w", err)
	}
	defer file.Close()

	// 创建multipart请求
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件部分
	part, err := writer.CreateFormFile("media", filepath.Base(videoPath))
	if err != nil {
		return nil, fmt.Errorf("创建表单文件失败: %w", err)
	}

	// 复制文件内容
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 添加描述信息
	description := struct {
		Title        string `json:"title"`
		Introduction string `json:"introduction"`
	}{
		Title:        title,
		Introduction: introduction,
	}

	descBytes, err := json.Marshal(description)
	if err != nil {
		return nil, fmt.Errorf("序列化描述信息失败: %w", err)
	}

	err = writer.WriteField("description", string(descBytes))
	if err != nil {
		return nil, fmt.Errorf("写入描述字段失败: %w", err)
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
		Type      string `json:"type"`
		MediaID   string `json:"media_id"`
		URL       string `json:"url"`
		CreatedAt int64  `json:"created_at"`
		ErrCode   int    `json:"errcode"`
		ErrMsg    string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析上传响应失败: %w", err)
	}

	// 检查响应
	if result.ErrCode != 0 && result.MediaID == "" {
		return nil, fmt.Errorf("上传视频素材失败: %s", result.ErrMsg)
	}

	return &struct {
		Type      string `json:"type"`
		MediaID   string `json:"media_id"`
		URL       string `json:"url"`
		CreatedAt int64  `json:"created_at"`
	}{
		Type:      result.Type,
		MediaID:   result.MediaID,
		URL:       result.URL,
		CreatedAt: result.CreatedAt,
	}, nil
}

// 创建草稿
func (a *WechatAdapter) createDraft(accessToken, mediaID, title, content string) (*struct {
	MediaID string `json:"media_id"`
}, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s",
		wechatAPIBaseURL, publishEndpoint, accessToken)

	// 构建请求体
	requestBody := struct {
		Articles []struct {
			Title              string `json:"title"`
			ThumbMediaID       string `json:"thumb_media_id"`
			Author             string `json:"author"`
			Digest             string `json:"digest"`
			Content            string `json:"content"`
			ContentSourceURL   string `json:"content_source_url"`
			NeedOpenComment    int    `json:"need_open_comment"`
			OnlyFansCanComment int    `json:"only_fans_can_comment"`
		} `json:"articles"`
	}{
		Articles: []struct {
			Title              string `json:"title"`
			ThumbMediaID       string `json:"thumb_media_id"`
			Author             string `json:"author"`
			Digest             string `json:"digest"`
			Content            string `json:"content"`
			ContentSourceURL   string `json:"content_source_url"`
			NeedOpenComment    int    `json:"need_open_comment"`
			OnlyFansCanComment int    `json:"only_fans_can_comment"`
		}{
			{
				Title:              title,
				ThumbMediaID:       mediaID, // 使用上传的视频作为缩略图
				Author:             "NFC碰一碰",
				Digest:             content[:Min(len(content), 120)], // 摘要最多120字
				Content:            fmt.Sprintf("<p>%s</p><p><video src=\"%s\"></video></p>", content, mediaID),
				ContentSourceURL:   "",
				NeedOpenComment:    1,
				OnlyFansCanComment: 0,
			},
		},
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化创建草稿请求体失败: %w", err)
	}

	// 发送请求
	resp, err := a.client.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("发送创建草稿请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		MediaID string `json:"media_id"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析创建草稿响应失败: %w", err)
	}

	// 检查响应
	if result.ErrCode != 0 || result.MediaID == "" {
		return nil, fmt.Errorf("创建草稿失败: %s", result.ErrMsg)
	}

	return &struct {
		MediaID string `json:"media_id"`
	}{
		MediaID: result.MediaID,
	}, nil
}

// 签名校验
func (a *WechatAdapter) CheckSignature(signature, timestamp, nonce string) bool {
	// 按字典序排序
	params := []string{a.client.token, timestamp, nonce}
	sort.Strings(params)

	// 拼接成一个字符串
	str := strings.Join(params, "")

	// SHA1加密
	h := sha1.New()
	h.Write([]byte(str))
	sha1Sum := fmt.Sprintf("%x", h.Sum(nil))

	// 比较签名
	return sha1Sum == signature
}

// Min 返回两个整数中的较小值
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Adapter 微信平台适配器
type Adapter struct {
	config config.WechatConfig
	logger *log.Logger
}

// NewAdapter 创建微信适配器
func NewAdapter(config config.WechatConfig, logger *log.Logger) *Adapter {
	return &Adapter{
		config: config,
		logger: logger,
	}
}

// PublishVideo 发布视频到微信
func (a *Adapter) PublishVideo(ctx context.Context, video *entities.Video, job *entities.PublishJob) error {
	a.logger.Printf("开始发布视频到微信公众号: %s, 标题: %s", video.ID, video.Title)

	// 0. 检查参数
	if video.StoragePath == "" {
		return fmt.Errorf("视频存储路径为空")
	}

	// 1. 获取视频文件
	videoPath, err := a.downloadVideoToTemp(video.StoragePath)
	if err != nil {
		return fmt.Errorf("下载视频失败: %w", err)
	}
	defer os.Remove(videoPath) // 确保临时文件被删除

	// 2. 获取访问令牌
	accessToken, err := a.getAccessToken()
	if err != nil {
		return fmt.Errorf("获取微信访问令牌失败: %w", err)
	}

	// 3. 上传视频素材
	mediaID, err := a.uploadVideoMaterial(accessToken, videoPath, video.Title)
	if err != nil {
		job.Status = "failed"
		job.ErrorMsg = fmt.Sprintf("上传视频素材失败: %v", err)
		job.UpdatedAt = time.Now()
		return fmt.Errorf("上传视频素材失败: %w", err)
	}

	// 4. 发布视频
	// 在微信中，我们通常通过创建图文消息来发布视频
	publishResult, err := a.createVideoNews(accessToken, mediaID, video.Title, video.Description, "")
	if err != nil {
		job.Status = "failed"
		job.ErrorMsg = fmt.Sprintf("创建图文消息失败: %v", err)
		job.UpdatedAt = time.Now()
		return fmt.Errorf("创建图文消息失败: %w", err)
	}

	// 5. 更新任务状态
	job.Status = "completed"
	job.CompletedAt = time.Now()
	job.UpdatedAt = time.Now()

	if job.Result == nil {
		job.Result = make(map[string]interface{})
	}
	job.Result["mediaId"] = mediaID
	job.Result["msgId"] = publishResult.MsgID
	job.Result["msgDataId"] = publishResult.MsgDataID

	a.logger.Printf("成功发布视频到微信, mediaId: %s, msgId: %s",
		mediaID, publishResult.MsgID)

	return nil
}

// 下载视频到临时目录
func (a *Adapter) downloadVideoToTemp(storagePath string) (string, error) {
	// 创建临时文件
	videoID := uuid.New().String()
	tempFilePath := filepath.Join(os.TempDir(), videoID+".mp4")

	// 简单实现 - 复制文件
	src, err := os.Open(storagePath)
	if err != nil {
		return "", fmt.Errorf("打开源视频文件失败: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("复制文件内容失败: %w", err)
	}

	return tempFilePath, nil
}

// 获取访问令牌
func (a *Adapter) getAccessToken() (string, error) {
	// 构建URL
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		a.config.AppID, a.config.AppSecret)

	// 发送请求
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("请求访问令牌失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析访问令牌响应失败: %w", err)
	}

	// 检查错误
	if result.ErrCode != 0 {
		return "", fmt.Errorf("获取访问令牌失败: %s", result.ErrMsg)
	}

	return result.AccessToken, nil
}

// 上传视频素材
func (a *Adapter) uploadVideoMaterial(accessToken, videoPath, title string) (string, error) {
	// 构建URL
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/material/add_material?access_token=%s&type=video",
		accessToken)

	// 创建multipart请求
	file, err := os.Open(videoPath)
	if err != nil {
		return "", fmt.Errorf("打开视频文件失败: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加视频文件
	part, err := writer.CreateFormFile("media", filepath.Base(videoPath))
	if err != nil {
		return "", fmt.Errorf("创建表单文件失败: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 添加描述信息
	description := fmt.Sprintf(`{"title":"%s","introduction":"视频介绍"}`, title)
	if err := writer.WriteField("description", description); err != nil {
		return "", fmt.Errorf("写入描述信息失败: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("关闭multipart writer失败: %w", err)
	}

	// 创建并发送请求
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 300 * time.Second} // 设置较长的超时时间
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		MediaID string `json:"media_id"`
		URL     string `json:"url"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查错误
	if result.ErrCode != 0 {
		return "", fmt.Errorf("上传视频素材失败: %s", result.ErrMsg)
	}

	return result.MediaID, nil
}

// 创建图文消息
func (a *Adapter) createVideoNews(accessToken, mediaID, title, description, thumbURL string) (*struct {
	MsgID     string `json:"msg_id"`
	MsgDataID string `json:"msg_data_id"`
}, error) {
	// 构建URL
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/draft/add?access_token=%s", accessToken)

	// 如果没有提供缩略图，使用默认图片
	if thumbURL == "" {
		thumbURL = "https://mmbiz.qpic.cn/mmbiz_jpg/UicQ7HgWiaUb3r8tYuL9dBOGXzDZqZvPtj2qtH1W5dPVn1cE6OgbU5FkpjXnNoJ0UqFWE3wDxcg2qTAUb4K4gNcw/0"
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"articles": []map[string]interface{}{
			{
				"title": title,
				"content": fmt.Sprintf(
					`<p style="text-align: center; margin-bottom: 20px;">%s</p><p><mp-video type="simple" vid="%s" width="100%%"></mp-video></p>`,
					description, mediaID,
				),
				"thumb_media_id":        mediaID, // 实际应用中应该上传专门的缩略图
				"digest":                description,
				"author":                "",
				"only_fans_can_comment": 0,
				"need_open_comment":     1,
			},
		},
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 发送请求
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		MediaID   string `json:"media_id"`
		MsgDataID string `json:"msg_data_id"`
		ErrCode   int    `json:"errcode"`
		ErrMsg    string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查错误
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("创建图文消息失败: %s", result.ErrMsg)
	}

	return &struct {
		MsgID     string `json:"msg_id"`
		MsgDataID string `json:"msg_data_id"`
	}{
		MsgID:     result.MediaID,
		MsgDataID: result.MsgDataID,
	}, nil
}

// GetPublishStatus 获取平台发布状态
func (a *Adapter) GetPublishStatus(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 创建微信适配器实例
	wechatAdapter := NewWechatAdapter(a.config, os.TempDir())

	// 直接调用WechatAdapter的GetPublishStatus方法
	return wechatAdapter.GetPublishStatus(ctx, platformID)
}

// GenerateShareLink 生成分享链接
func (a *Adapter) GenerateShareLink(ctx context.Context, platformID string, extraParams map[string]interface{}) (string, error) {
	// 创建微信适配器实例
	wechatAdapter := NewWechatAdapter(a.config, os.TempDir())

	// 直接调用WechatAdapter的GenerateShareLink方法
	return wechatAdapter.GenerateShareLink(ctx, platformID, extraParams)
}

// GenerateJSConfig 生成JS SDK配置
func (a *Adapter) GenerateJSConfig(ctx context.Context, url string) (map[string]interface{}, error) {
	// 创建微信适配器实例
	wechatAdapter := NewWechatAdapter(a.config, os.TempDir())

	// 直接调用WechatAdapter的GenerateJSConfig方法
	return wechatAdapter.GenerateJSConfig(ctx, url)
}

// GetDetailedStats 获取详细统计数据
func (a *Adapter) GetDetailedStats(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 创建微信适配器实例
	wechatAdapter := NewWechatAdapter(a.config, os.TempDir())

	// 获取平台发布状态
	status, err := wechatAdapter.GetPublishStatus(ctx, platformID)
	if err != nil {
		return nil, err
	}

	// 微信公众平台目前没有提供详细的统计API，只能返回基本信息
	// 添加最后更新时间
	if status != nil {
		status["lastUpdated"] = time.Now()
		status["viewCount"] = 0
		status["likeCount"] = 0
		status["commentCount"] = 0
		status["shareCount"] = 0
		status["totalEngagement"] = 0
	}

	return status, nil
}
