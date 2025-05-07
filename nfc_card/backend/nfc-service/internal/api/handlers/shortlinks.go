package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"nfc-service/internal/domain/entities"
	"nfc-service/internal/services/shortlinks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ShortLinkHandler 处理短链接相关的HTTP请求
type ShortLinkHandler struct {
	service shortlinks.Service
	baseURL string
}

// NewShortLinkHandler 创建新的短链接处理程序
func NewShortLinkHandler(service shortlinks.Service, baseURL string) *ShortLinkHandler {
	return &ShortLinkHandler{
		service: service,
		baseURL: baseURL,
	}
}

// RegisterRoutes 注册路由
func (h *ShortLinkHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		shortlinks := api.Group("/shortlinks")
		{
			shortlinks.POST("", h.CreateShortLink)
			shortlinks.GET("/:id", h.GetShortLinkByID)
			shortlinks.GET("/slug/:slug", h.GetShortLinkBySlug)
			shortlinks.GET("/merchant/:merchantID", h.GetShortLinksByMerchantID)
			shortlinks.GET("/card/:cardID", h.GetShortLinksByNfcCardID)
			shortlinks.PUT("/:id", h.UpdateShortLink)
			shortlinks.DELETE("/:id", h.DeleteShortLink)
		}

		// 重定向路由，用于短链接访问
		router.GET("/:slug", h.RedirectToTarget)
	}
}

// CreateShortLink 创建短链接
func (h *ShortLinkHandler) CreateShortLink(c *gin.Context) {
	var dto entities.CreateShortLinkDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shortLink, err := h.service.Create(c.Request.Context(), &dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构建完整URL
	fullURL := h.service.GetFullURL(h.baseURL, shortLink.Slug)

	response := map[string]interface{}{
		"shortlink": shortLink,
		"full_url":  fullURL,
	}

	c.JSON(http.StatusCreated, response)
}

// GetShortLinkByID 通过ID获取短链接
func (h *ShortLinkHandler) GetShortLinkByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID格式"})
		return
	}

	shortLink, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if shortLink == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "短链接不存在"})
		return
	}

	// 构建完整URL
	fullURL := h.service.GetFullURL(h.baseURL, shortLink.Slug)

	response := map[string]interface{}{
		"shortlink": shortLink,
		"full_url":  fullURL,
	}

	c.JSON(http.StatusOK, response)
}

// GetShortLinkBySlug 通过Slug获取短链接
func (h *ShortLinkHandler) GetShortLinkBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slug不能为空"})
		return
	}

	shortLink, err := h.service.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if shortLink == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "短链接不存在"})
		return
	}

	// 构建完整URL
	fullURL := h.service.GetFullURL(h.baseURL, shortLink.Slug)

	response := map[string]interface{}{
		"shortlink": shortLink,
		"full_url":  fullURL,
	}

	c.JSON(http.StatusOK, response)
}

// GetShortLinksByMerchantID 获取商户的所有短链接
func (h *ShortLinkHandler) GetShortLinksByMerchantID(c *gin.Context) {
	merchantIDStr := c.Param("merchantID")
	merchantID, err := uuid.Parse(merchantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商户ID格式"})
		return
	}

	page := 1
	pageSize := 10

	if pageStr := c.Query("page"); pageStr != "" {
		json.Unmarshal([]byte(pageStr), &page)
	}

	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		json.Unmarshal([]byte(pageSizeStr), &pageSize)
	}

	shortLinks, total, err := h.service.GetByMerchantID(c.Request.Context(), merchantID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": shortLinks,
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
			"total":     total,
		},
	})
}

// GetShortLinksByNfcCardID 获取NFC卡片的所有短链接
func (h *ShortLinkHandler) GetShortLinksByNfcCardID(c *gin.Context) {
	cardIDStr := c.Param("cardID")
	cardID, err := uuid.Parse(cardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的卡片ID格式"})
		return
	}

	shortLinks, err := h.service.GetByNfcCardID(c.Request.Context(), cardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": shortLinks})
}

// UpdateShortLink 更新短链接
func (h *ShortLinkHandler) UpdateShortLink(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID格式"})
		return
	}

	var dto entities.UpdateShortLinkDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shortLink, err := h.service.Update(c.Request.Context(), id, &dto)
	if err != nil {
		if errors.Is(err, errors.New("短链接不存在")) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构建完整URL
	fullURL := h.service.GetFullURL(h.baseURL, shortLink.Slug)

	response := map[string]interface{}{
		"shortlink": shortLink,
		"full_url":  fullURL,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteShortLink 删除短链接
func (h *ShortLinkHandler) DeleteShortLink(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID格式"})
		return
	}

	err = h.service.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "短链接已成功删除"})
}

// RedirectToTarget 重定向到目标URL
func (h *ShortLinkHandler) RedirectToTarget(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slug不能为空"})
		return
	}

	// 获取短链接
	shortLink, err := h.service.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if shortLink == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "短链接不存在"})
		return
	}

	// 异步增加点击次数
	go func() {
		ctx := c.Request.Context()
		h.service.IncrementClicks(ctx, slug)
	}()

	// 重定向到目标URL
	c.Redirect(http.StatusTemporaryRedirect, shortLink.TargetURL)
}
