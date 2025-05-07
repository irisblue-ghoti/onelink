package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"distribution-service/internal/config"
	"distribution-service/internal/domain/entities"
	"distribution-service/internal/domain/repositories"
	"distribution-service/internal/services"
	"distribution-service/internal/storage"
)

// PublishHandler 发布处理器
type PublishHandler struct {
	publishService *services.PublishService
}

// NewPublishHandler 创建发布处理器
func NewPublishHandler(
	jobRepo repositories.JobRepository,
	videoRepo repositories.VideoRepository,
	cfg *config.Config,
	kafkaProducer services.KafkaProducer,
	storageService storage.StorageService,
) *PublishHandler {
	return &PublishHandler{
		publishService: services.NewPublishService(jobRepo, videoRepo, cfg, kafkaProducer, storageService),
	}
}

// CreateJobRequest 创建分发任务请求
type CreateJobRequest struct {
	VideoID   string `json:"videoId" binding:"required,uuid"`
	NfcCardID string `json:"nfcCardId" binding:"required,uuid"`
	Channel   string `json:"channel" binding:"required,oneof=douyin kuaishou xiaohongshu wechat"`
}

// CreateJob 创建分发任务
func (h *PublishHandler) CreateJob(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 将字符串ID转换为UUID
	videoID, err := uuid.Parse(req.VideoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
		return
	}

	nfcCardID, err := uuid.Parse(req.NfcCardID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的NFC卡ID"})
		return
	}

	// 模拟租户ID
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// 创建任务
	job := entities.NewPublishJob(tenantID, videoID, nfcCardID, req.Channel)

	// 保存任务
	err = h.publishService.CreateJob(c.Request.Context(), job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, job)
}

// ListJobs 获取分发任务列表
func (h *PublishHandler) ListJobs(c *gin.Context) {
	// 解析查询参数
	status := c.Query("status")
	videoID := c.Query("videoId")
	nfcCardID := c.Query("nfcCardId")
	channel := c.Query("channel")

	// 模拟租户ID
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// 查询任务
	jobs, err := h.publishService.ListJobs(c.Request.Context(), tenantID, status, videoID, nfcCardID, channel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": jobs,
		"meta": gin.H{
			"total": len(jobs),
		},
	})
}

// GetJob 获取单个分发任务
func (h *PublishHandler) GetJob(c *gin.Context) {
	id := c.Param("id")
	jobID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	// 模拟租户ID
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// 查询任务
	job, err := h.publishService.GetJob(c.Request.Context(), tenantID, jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// GetPublishStatus 获取平台发布状态
func (h *PublishHandler) GetPublishStatus(c *gin.Context) {
	channel := c.Param("channel")
	platformID := c.Param("platform_id")

	// 检查参数
	if channel == "" || platformID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数"})
		return
	}

	// 查询平台状态
	status, err := h.publishService.GetPlatformStatus(c.Request.Context(), channel, platformID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GenerateShareLinkRequest 生成分享链接请求
type GenerateShareLinkRequest struct {
	PlatformID  string                 `json:"platformId" binding:"required"`
	ExtraParams map[string]interface{} `json:"extraParams"`
}

// GenerateShareLink 生成平台视频分享链接
func (h *PublishHandler) GenerateShareLink(c *gin.Context) {
	channel := c.Param("channel")
	if channel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少渠道参数"})
		return
	}

	var req GenerateShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用服务生成分享链接
	shareURL, err := h.publishService.GenerateShareLink(c.Request.Context(), channel, req.PlatformID, req.ExtraParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"shareUrl": shareURL,
	})
}

// GenerateJSConfigRequest 生成JSSDK配置请求
type GenerateJSConfigRequest struct {
	URL string `json:"url" binding:"required"`
}

// GenerateJSConfig 生成JSSDK配置
func (h *PublishHandler) GenerateJSConfig(c *gin.Context) {
	channel := c.Param("channel")
	if channel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少渠道参数"})
		return
	}

	var req GenerateJSConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用服务生成JSSDK配置
	config, err := h.publishService.GenerateJSConfig(c.Request.Context(), channel, req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetDetailedStats 获取平台视频详细统计数据
func (h *PublishHandler) GetDetailedStats(c *gin.Context) {
	channel := c.Param("channel")
	platformID := c.Param("platform_id")

	// 检查参数
	if channel == "" || platformID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数"})
		return
	}

	// 查询统计数据
	stats, err := h.publishService.GetDetailedStats(c.Request.Context(), channel, platformID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
