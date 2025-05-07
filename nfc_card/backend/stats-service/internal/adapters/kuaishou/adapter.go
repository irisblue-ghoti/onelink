package kuaishou

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"stats-service/internal/config"
	"stats-service/internal/domain/entities"
)

const (
	kuaishouAPIBaseURL  = "https://open.kuaishou.com"
	videoStatusEndpoint = "/openapi/photo/stat"
	accessTokenEndpoint = "/oauth2/access_token"
)

// KuaishouAdapter 快手适配器
type KuaishouAdapter struct {
	config     config.KuaishouConfig
	httpClient *http.Client
}

// NewKuaishouAdapter 创建快手适配器
func NewKuaishouAdapter(cfg config.KuaishouConfig) *KuaishouAdapter {
	return &KuaishouAdapter{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetPlatformName 获取平台名称
func (a *KuaishouAdapter) GetPlatformName() string {
	return "kuaishou"
}

// CollectStats 收集平台统计数据
func (a *KuaishouAdapter) CollectStats(ctx context.Context, platformID string) (*entities.PlatformStats, error) {
	// 获取访问令牌
	accessToken, err := a.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("获取快手访问令牌失败: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s&photo_id=%s",
		kuaishouAPIBaseURL, videoStatusEndpoint, accessToken, platformID)

	// 发送请求
	resp, err := a.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取视频状态失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %w", err)
	}

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

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析视频状态响应失败: %w", err)
	}

	// 检查响应
	if result.Result != 1 {
		return nil, fmt.Errorf("获取视频状态失败: %s", result.Message)
	}

	// 创建统计数据
	stats := &entities.PlatformStats{
		ID:           uuid.New(),
		Platform:     a.GetPlatformName(),
		PlatformID:   uuid.MustParse(platformID),
		ViewCount:    result.Data.Metrics.ViewCount,
		LikeCount:    result.Data.Metrics.LikeCount,
		CommentCount: result.Data.Metrics.CommentCount,
		ShareCount:   result.Data.Metrics.ShareCount,
		CollectCount: result.Data.Metrics.CollectCount,
		RawData: map[string]interface{}{
			"photoId":    result.Data.PhotoID,
			"title":      result.Data.Title,
			"createTime": time.Unix(result.Data.CreateTime, 0),
			"status":     result.Data.Status,
			"shareUrl":   result.Data.ShareURL,
			"metrics":    result.Data.Metrics,
		},
		LastUpdatedAt: time.Now(),
		CreatedAt:     time.Now(),
	}

	return stats, nil
}

// CollectBatchStats 批量收集平台统计数据
func (a *KuaishouAdapter) CollectBatchStats(ctx context.Context, platformIDs []string) ([]*entities.PlatformStats, error) {
	if len(platformIDs) == 0 {
		return nil, fmt.Errorf("平台ID列表不能为空")
	}

	// 获取访问令牌
	accessToken, err := a.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("获取快手访问令牌失败: %w", err)
	}

	// 由于快手API不支持批量获取视频统计数据，我们需要逐个查询
	statsList := make([]*entities.PlatformStats, 0, len(platformIDs))

	for _, platformID := range platformIDs {
		// 构建请求URL
		url := fmt.Sprintf("%s%s?access_token=%s&photo_id=%s",
			kuaishouAPIBaseURL, videoStatusEndpoint, accessToken, platformID)

		// 发送请求
		resp, err := a.httpClient.Get(url)
		if err != nil {
			// 记录错误但继续处理其他ID
			fmt.Printf("获取视频(%s)状态失败: %v\n", platformID, err)
			continue
		}

		// 读取响应内容
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() // 确保关闭响应体

		if err != nil {
			fmt.Printf("读取视频(%s)响应内容失败: %v\n", platformID, err)
			continue
		}

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

		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("解析视频(%s)状态响应失败: %v\n", platformID, err)
			continue
		}

		// 检查响应
		if result.Result != 1 {
			fmt.Printf("获取视频(%s)状态失败: %s\n", platformID, result.Message)
			continue
		}

		// 创建统计数据
		stats := &entities.PlatformStats{
			ID:           uuid.New(),
			Platform:     a.GetPlatformName(),
			PlatformID:   uuid.MustParse(platformID),
			ViewCount:    result.Data.Metrics.ViewCount,
			LikeCount:    result.Data.Metrics.LikeCount,
			CommentCount: result.Data.Metrics.CommentCount,
			ShareCount:   result.Data.Metrics.ShareCount,
			CollectCount: result.Data.Metrics.CollectCount,
			RawData: map[string]interface{}{
				"photoId":    result.Data.PhotoID,
				"title":      result.Data.Title,
				"createTime": time.Unix(result.Data.CreateTime, 0),
				"status":     result.Data.Status,
				"shareUrl":   result.Data.ShareURL,
				"metrics":    result.Data.Metrics,
			},
			LastUpdatedAt: time.Now(),
			CreatedAt:     time.Now(),
		}

		statsList = append(statsList, stats)

		// 避免API限流，每次请求间隔100毫秒
		time.Sleep(100 * time.Millisecond)
	}

	if len(statsList) == 0 {
		return nil, fmt.Errorf("所有视频数据获取失败")
	}

	return statsList, nil
}

// getAccessToken 获取快手访问令牌
func (a *KuaishouAdapter) getAccessToken() (string, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s", kuaishouAPIBaseURL, accessTokenEndpoint)

	// 构建请求参数
	data := fmt.Sprintf("app_id=%s&app_secret=%s&grant_type=client_credentials",
		a.config.AppID, a.config.AppSecret)

	// 发送请求
	resp, err := a.httpClient.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("获取客户端令牌失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Result       int    `json:"result"`
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析令牌响应失败: %w", err)
	}

	// 检查响应
	if result.Result != 1 || result.AccessToken == "" {
		return "", fmt.Errorf("获取访问令牌失败")
	}

	return result.AccessToken, nil
}
