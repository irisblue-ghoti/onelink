package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"content-service/internal/domain/entities"
	"content-service/internal/services"

	"github.com/gin-gonic/gin"
)

// 允许的视频格式和大小限制
var (
	// 允许的视频MIME类型
	allowedVideoTypes = map[string]bool{
		"video/mp4":       true,
		"video/quicktime": true,
		"video/x-msvideo": true,
		"video/webm":      true,
	}

	// 允许的视频文件扩展名
	allowedVideoExtensions = map[string]bool{
		".mp4":  true,
		".mov":  true,
		".avi":  true,
		".webm": true,
	}

	// 默认上传大小限制为200MB
	maxUploadSize int64 = 200 * 1024 * 1024
)

// VideosHandler 处理视频相关API请求
type VideosHandler struct {
	contentService *services.ContentService
}

// NewVideosHandler 创建新的视频处理器
func NewVideosHandler(contentService *services.ContentService) *VideosHandler {
	return &VideosHandler{
		contentService: contentService,
	}
}

// validateVideoFile 验证视频文件格式和安全性
func (h *VideosHandler) validateVideoFile(file *multipart.FileHeader) error {
	// 检查文件大小
	if file.Size > maxUploadSize {
		return fmt.Errorf("文件过大，最大允许%dMB", maxUploadSize/(1024*1024))
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedVideoExtensions[ext] {
		return fmt.Errorf("不支持的文件格式，允许的格式: mp4, mov, avi, webm")
	}

	// 检查文件MIME类型
	contentType := file.Header.Get("Content-Type")
	if !allowedVideoTypes[contentType] {
		return fmt.Errorf("不支持的文件类型，允许的类型: video/mp4, video/quicktime, video/x-msvideo, video/webm")
	}

	// TODO: 实现病毒/恶意软件扫描
	// TODO: 实现内容安全检查（暴力、色情等）

	return nil
}

// Create 上传并创建视频
func (h *VideosHandler) Create(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 解析表单数据
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未上传文件或文件无效"})
		return
	}

	// 验证视频文件
	if err := h.validateVideoFile(file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("文件验证失败: %s", err.Error()),
			"code":  "invalid_file",
		})
		return
	}

	// 解析JSON数据
	var dto entities.CreateVideoDTO
	if err := c.ShouldBind(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("输入数据验证失败: %s", err.Error()),
			"code":  "invalid_input",
		})
		return
	}

	// 创建视频
	video, err := h.contentService.Create(tenantIDStr, file, dto)
	if err != nil {
		// 检查是否为服务错误
		serviceError, ok := err.(*services.ServiceError)
		if ok {
			// 返回结构化错误响应
			c.JSON(getStatusCodeForError(serviceError), gin.H{
				"error": serviceError.Message,
				"code":  serviceError.Code,
				"type":  serviceError.Type,
			})
		} else {
			// 未知错误
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
				"code":  "unknown_error",
			})
		}
		return
	}

	// 获取视频URL和封面URL
	videoURL := h.contentService.GetVideoURL(video)
	var coverURL string
	if video.CoverKey != "" {
		coverURL = h.contentService.GetFileURL(video.CoverKey)
	}

	// 返回响应
	c.JSON(http.StatusCreated, entities.VideoResponse{
		ID:           video.ID,
		Title:        video.Title,
		Description:  video.Description,
		URL:          videoURL,
		CoverURL:     coverURL,
		Duration:     video.Duration,
		Width:        video.Width,
		Height:       video.Height,
		Size:         video.Size,
		IsTranscoded: video.IsTranscoded,
		CreatedAt:    video.CreatedAt,
	})
}

// getStatusCodeForError 根据错误类型返回适当的HTTP状态码
func getStatusCodeForError(err *services.ServiceError) int {
	switch err.Type {
	case services.ErrTypeValidation:
		return http.StatusBadRequest
	case services.ErrTypeNotFound:
		return http.StatusNotFound
	case services.ErrTypeUnauthorized:
		return http.StatusForbidden
	case services.ErrTypeDatabase, services.ErrTypeStorage, services.ErrTypeTranscode:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// FindAll 获取所有视频
func (h *VideosHandler) FindAll(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取分页参数
	page := 1
	limit := 10

	if pageParam := c.Query("page"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// 获取视频总数
	totalVideos, err := h.contentService.CountVideos(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取视频总数失败"})
		return
	}

	// 获取视频列表
	videos, err := h.contentService.FindAll(tenantIDStr, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换为响应格式
	var responseItems []entities.VideoResponse
	for _, video := range videos {
		// 获取视频URL
		videoURL := h.contentService.GetVideoURL(video)

		// 获取封面URL
		var coverURL string
		if video.CoverKey != "" {
			coverURL = h.contentService.GetFileURL(video.CoverKey)
		}

		responseItems = append(responseItems, entities.VideoResponse{
			ID:           video.ID,
			Title:        video.Title,
			Description:  video.Description,
			URL:          videoURL,
			CoverURL:     coverURL,
			Duration:     video.Duration,
			Width:        video.Width,
			Height:       video.Height,
			Size:         video.Size,
			IsTranscoded: video.IsTranscoded,
			CreatedAt:    video.CreatedAt,
		})
	}

	// 计算总页数
	totalPages := (totalVideos + limit - 1) / limit

	// 返回带分页的响应
	c.JSON(http.StatusOK, gin.H{
		"data": responseItems,
		"meta": gin.H{
			"currentPage":  page,
			"itemsPerPage": limit,
			"totalItems":   totalVideos,
			"totalPages":   totalPages,
		},
	})
}

// FindOne 获取单个视频
func (h *VideosHandler) FindOne(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取视频ID
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未指定视频ID"})
		return
	}

	// 获取视频
	video, err := h.contentService.FindOne(id, tenantIDStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 获取视频URL和封面URL
	videoURL := h.contentService.GetVideoURL(video)
	var coverURL string
	if video.CoverKey != "" {
		coverURL = h.contentService.GetFileURL(video.CoverKey)
	}

	// 返回详细视频信息
	c.JSON(http.StatusOK, entities.DetailedVideoResponse{
		VideoResponse: entities.VideoResponse{
			ID:           video.ID,
			Title:        video.Title,
			Description:  video.Description,
			URL:          videoURL,
			CoverURL:     coverURL,
			Duration:     video.Duration,
			Width:        video.Width,
			Height:       video.Height,
			Size:         video.Size,
			IsTranscoded: video.IsTranscoded,
			CreatedAt:    video.CreatedAt,
		},
		TranscodeStatus: video.TranscodeStatus,
		UpdatedAt:       video.UpdatedAt,
	})
}

// Remove 删除视频
func (h *VideosHandler) Remove(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取视频ID
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未指定视频ID"})
		return
	}

	// 删除视频
	if err := h.contentService.Remove(id, tenantIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Transcode 开始视频转码
func (h *VideosHandler) Transcode(c *gin.Context) {
	// 获取租户ID
	tenantID, _ := c.Get("tenantID")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "租户ID格式错误"})
		return
	}

	// 获取视频ID
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未指定视频ID"})
		return
	}

	// 开始转码
	if err := h.contentService.StartTranscode(id, tenantIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "视频转码任务已启动"})
}
