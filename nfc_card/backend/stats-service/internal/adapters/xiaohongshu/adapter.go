package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"stats-service/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

// 小红书API接口地址
const (
	XHSBaseURL         = "https://open.xiaohongshu.com/api/v1"
	XHSNoteDetailPath  = "/note/detail"
	XHSNoteMetricsPath = "/note/metrics"
)

// XiaohongshuAdapter 小红书平台适配器
type XiaohongshuAdapter struct {
	appId     string
	appSecret string
	client    *http.Client
}

// NewXiaohongshuAdapter 创建新的小红书适配器
func NewXiaohongshuAdapter(appId, appSecret string) *XiaohongshuAdapter {
	return &XiaohongshuAdapter{
		appId:     appId,
		appSecret: appSecret,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// GetPlatformName 获取平台名称
func (a *XiaohongshuAdapter) GetPlatformName() string {
	return "xiaohongshu"
}

// CollectStats 收集平台统计数据
func (a *XiaohongshuAdapter) CollectStats(ctx context.Context, platformID string) (*entities.PlatformStats, error) {
	// 获取笔记详情
	noteDetail, err := a.getNoteDetail(ctx, platformID)
	if err != nil {
		return nil, fmt.Errorf("获取小红书笔记详情失败: %w", err)
	}

	// 获取笔记统计数据
	noteMetrics, err := a.getNoteMetrics(ctx, platformID)
	if err != nil {
		return nil, fmt.Errorf("获取小红书笔记统计数据失败: %w", err)
	}

	// 合并数据并返回
	stats := &entities.PlatformStats{
		ID:           uuid.New(),
		Platform:     a.GetPlatformName(),
		PlatformID:   uuid.MustParse(platformID),
		ViewCount:    noteMetrics.ViewCount,
		LikeCount:    noteMetrics.LikeCount,
		CommentCount: noteMetrics.CommentCount,
		ShareCount:   noteMetrics.ShareCount,
		CollectCount: noteMetrics.CollectCount,
		RawData: map[string]interface{}{
			"noteId":     noteDetail.ID,
			"title":      noteDetail.Title,
			"createTime": noteDetail.CreateTime,
			"status":     noteDetail.Status,
			"shareUrl":   noteDetail.ShareURL,
			"metrics":    noteMetrics,
		},
		LastUpdatedAt: time.Now(),
		CreatedAt:     time.Now(),
	}

	return stats, nil
}

// CollectBatchStats 批量收集统计数据
func (a *XiaohongshuAdapter) CollectBatchStats(ctx context.Context, platformIDs []string) ([]*entities.PlatformStats, error) {
	statsList := make([]*entities.PlatformStats, 0, len(platformIDs))

	for _, platformID := range platformIDs {
		stats, err := a.CollectStats(ctx, platformID)
		if err != nil {
			fmt.Printf("获取小红书笔记(%s)数据失败: %v\n", platformID, err)
			continue
		}

		statsList = append(statsList, stats)

		// 避免API限流，每次请求间隔100毫秒
		time.Sleep(100 * time.Millisecond)
	}

	if len(statsList) == 0 {
		return nil, fmt.Errorf("所有小红书笔记数据获取失败")
	}

	return statsList, nil
}

// NoteDetail 笔记详情
type NoteDetail struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	CreateTime time.Time `json:"createTime"`
	Status     string    `json:"status"`
	ShareURL   string    `json:"shareUrl"`
}

// NoteMetrics 笔记统计数据
type NoteMetrics struct {
	ViewCount    int64 `json:"viewCount"`
	LikeCount    int64 `json:"likeCount"`
	CommentCount int64 `json:"commentCount"`
	ShareCount   int64 `json:"shareCount"`
	CollectCount int64 `json:"collectCount"`
}

// getNoteDetail 获取笔记详情
func (a *XiaohongshuAdapter) getNoteDetail(ctx context.Context, noteID string) (*NoteDetail, error) {
	// 构建请求URL
	reqURL, err := url.Parse(XHSBaseURL + XHSNoteDetailPath)
	if err != nil {
		return nil, err
	}

	// 设置查询参数
	query := reqURL.Query()
	query.Set("noteId", noteID)
	reqURL.RawQuery = query.Encode()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	token, err := a.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败: %s, 状态码: %d", string(body), resp.StatusCode)
	}

	// 解析响应
	var result struct {
		Success bool       `json:"success"`
		Data    NoteDetail `json:"data"`
		Message string     `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("API请求成功但操作失败: %s", result.Message)
	}

	return &result.Data, nil
}

// getNoteMetrics 获取笔记统计数据
func (a *XiaohongshuAdapter) getNoteMetrics(ctx context.Context, noteID string) (*NoteMetrics, error) {
	// 构建请求URL
	reqURL, err := url.Parse(XHSBaseURL + XHSNoteMetricsPath)
	if err != nil {
		return nil, err
	}

	// 设置查询参数
	query := reqURL.Query()
	query.Set("noteId", noteID)
	reqURL.RawQuery = query.Encode()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	token, err := a.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败: %s, 状态码: %d", string(body), resp.StatusCode)
	}

	// 解析响应
	var result struct {
		Success bool        `json:"success"`
		Data    NoteMetrics `json:"data"`
		Message string      `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("API请求成功但操作失败: %s", result.Message)
	}

	return &result.Data, nil
}

// getAccessToken 获取访问令牌
func (a *XiaohongshuAdapter) getAccessToken(ctx context.Context) (string, error) {
	// 注意：这是简化版的实现，实际应用中需要实现完整的OAuth2流程
	// 并且应该缓存令牌，避免频繁请求

	// 模拟获取令牌的逻辑
	// 实际实现应当根据小红书API文档进行
	return "simulated_access_token", nil
}
