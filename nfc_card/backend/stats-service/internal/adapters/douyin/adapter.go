package douyin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"stats-service/internal/config"
	"stats-service/internal/domain/entities"
)

const (
	douyinAPIBaseURL    = "https://open.douyin.com"
	videoStatusEndpoint = "/video/data/"
	accessTokenEndpoint = "/oauth/access_token/"
	clientTokenEndpoint = "/oauth/client_token/"
	batchDataEndpoint   = "/video/data/batch/" // 批量获取视频数据端点
)

// DouyinAdapter 抖音适配器
type DouyinAdapter struct {
	config     config.DouyinConfig
	httpClient *http.Client
}

// NewDouyinAdapter 创建抖音适配器
func NewDouyinAdapter(cfg config.DouyinConfig) *DouyinAdapter {
	return &DouyinAdapter{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetPlatformName 获取平台名称
func (a *DouyinAdapter) GetPlatformName() string {
	return "douyin"
}

// CollectStats 收集平台统计数据
func (a *DouyinAdapter) CollectStats(ctx context.Context, platformID string) (*entities.PlatformStats, error) {
	// 获取访问令牌
	accessToken, err := a.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("获取抖音访问令牌失败: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s?access_token=%s&item_id=%s",
		douyinAPIBaseURL, videoStatusEndpoint, accessToken, platformID)

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
		Data struct {
			ItemID     string `json:"item_id"`
			Title      string `json:"title"`
			CreateTime int64  `json:"create_time"`
			IsReviewed bool   `json:"is_reviewed"`
			ShareURL   string `json:"share_url"`
			Statistics struct {
				CommentCount int64 `json:"comment_count"`
				DiggCount    int64 `json:"digg_count"`    // 点赞数
				PlayCount    int64 `json:"play_count"`    // 播放数
				ShareCount   int64 `json:"share_count"`   // 分享数
				ForwardCount int64 `json:"forward_count"` // 转发数
				CollectCount int64 `json:"collect_count"` // 收藏数
			} `json:"statistics"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析视频状态响应失败: %w", err)
	}

	// 检查响应
	if result.Data.ItemID == "" {
		return nil, fmt.Errorf("获取视频状态失败: %s", result.Message)
	}

	// 创建统计数据
	stats := &entities.PlatformStats{
		ID:           uuid.New(),
		Platform:     a.GetPlatformName(),
		PlatformID:   uuid.MustParse(platformID),
		ViewCount:    result.Data.Statistics.PlayCount,
		LikeCount:    result.Data.Statistics.DiggCount,
		CommentCount: result.Data.Statistics.CommentCount,
		ShareCount:   result.Data.Statistics.ShareCount + result.Data.Statistics.ForwardCount,
		CollectCount: result.Data.Statistics.CollectCount,
		RawData: map[string]interface{}{
			"itemId":     result.Data.ItemID,
			"title":      result.Data.Title,
			"createTime": time.Unix(result.Data.CreateTime, 0),
			"isReviewed": result.Data.IsReviewed,
			"shareUrl":   result.Data.ShareURL,
			"statistics": result.Data.Statistics,
		},
		LastUpdatedAt: time.Now(),
		CreatedAt:     time.Now(),
	}

	return stats, nil
}

// CollectBatchStats 批量收集平台统计数据
func (a *DouyinAdapter) CollectBatchStats(ctx context.Context, platformIDs []string) ([]*entities.PlatformStats, error) {
	statsList := make([]*entities.PlatformStats, 0, len(platformIDs))

	for _, platformID := range platformIDs {
		stats, err := a.CollectStats(ctx, platformID)
		if err != nil {
			fmt.Printf("获取抖音视频(%s)数据失败: %v\n", platformID, err)
			continue
		}

		statsList = append(statsList, stats)

		// 避免API限流，每次请求间隔100毫秒
		time.Sleep(100 * time.Millisecond)
	}

	if len(statsList) == 0 {
		return nil, fmt.Errorf("所有抖音视频数据获取失败")
	}

	return statsList, nil
}

// GetDetailedStats 获取详细统计数据
func (a *DouyinAdapter) GetDetailedStats(ctx context.Context, platformID string) (map[string]interface{}, error) {
	// 获取访问令牌
	accessToken, err := a.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("获取抖音访问令牌失败: %w", err)
	}

	// 构建请求URL - 使用基本视频状态端点获取详细数据
	url := fmt.Sprintf("%s%s?access_token=%s&item_id=%s",
		douyinAPIBaseURL, videoStatusEndpoint, accessToken, platformID)

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
		Data struct {
			ItemID     string `json:"item_id"`
			Title      string `json:"title"`
			CreateTime int64  `json:"create_time"`
			IsReviewed bool   `json:"is_reviewed"`
			ShareURL   string `json:"share_url"`
			Statistics struct {
				CommentCount int64 `json:"comment_count"`
				DiggCount    int64 `json:"digg_count"`    // 点赞数
				PlayCount    int64 `json:"play_count"`    // 播放数
				ShareCount   int64 `json:"share_count"`   // 分享数
				ForwardCount int64 `json:"forward_count"` // 转发数
				CollectCount int64 `json:"collect_count"` // 收藏数
			} `json:"statistics"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析视频状态响应失败: %w", err)
	}

	// 检查响应
	if result.Data.ItemID == "" {
		return nil, fmt.Errorf("获取视频状态失败: %s", result.Message)
	}

	// 创建详细统计数据
	detailedStats := map[string]interface{}{
		"platformId":   result.Data.ItemID,
		"title":        result.Data.Title,
		"createTime":   time.Unix(result.Data.CreateTime, 0),
		"isReviewed":   result.Data.IsReviewed,
		"shareUrl":     result.Data.ShareURL,
		"commentCount": result.Data.Statistics.CommentCount,
		"likeCount":    result.Data.Statistics.DiggCount,
		"playCount":    result.Data.Statistics.PlayCount,
		"shareCount":   result.Data.Statistics.ShareCount,
		"forwardCount": result.Data.Statistics.ForwardCount,
		"collectCount": result.Data.Statistics.CollectCount,
		"totalEngagement": result.Data.Statistics.CommentCount +
			result.Data.Statistics.DiggCount +
			result.Data.Statistics.ShareCount +
			result.Data.Statistics.ForwardCount +
			result.Data.Statistics.CollectCount,
		"lastUpdated": time.Now(),
	}

	return detailedStats, nil
}

// getAccessToken 获取抖音访问令牌
func (a *DouyinAdapter) getAccessToken() (string, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s%s?client_key=%s&client_secret=%s&grant_type=client_credential",
		douyinAPIBaseURL, clientTokenEndpoint, a.config.AppID, a.config.AppSecret)

	// 发送请求
	resp, err := a.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("获取客户端令牌失败: %w", err)
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
		return "", fmt.Errorf("解析令牌响应失败: %w", err)
	}

	// 检查响应
	if result.Data.AccessToken == "" {
		return "", fmt.Errorf("获取访问令牌失败: %s", result.Message)
	}

	return result.Data.AccessToken, nil
}
