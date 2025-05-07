package handlers

import (
	"net/http"
	"strconv"

	"nfc-service/internal/domain/entities"
	"nfc-service/internal/services/cards"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CardHandler 处理NFC卡片相关的API请求
type CardHandler struct {
	service cards.Service
}

// NewCardHandler 创建卡片处理程序
func NewCardHandler(service cards.Service) *CardHandler {
	return &CardHandler{
		service: service,
	}
}

// RegisterRoutes 注册路由
func (h *CardHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		nfcCards := api.Group("/nfc-cards")
		{
			nfcCards.POST("", h.CreateCard)
			nfcCards.GET("", h.GetCards)
			nfcCards.GET("/:id", h.GetCardByID)
			nfcCards.PUT("/:id", h.UpdateCard)
			nfcCards.DELETE("/:id", h.DeleteCard)
			nfcCards.POST("/activate", h.ActivateCard)
		}
	}
}

// CreateCard 创建NFC卡片
func (h *CardHandler) CreateCard(c *gin.Context) {
	var dto entities.CreateNfcCardDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	card, err := h.service.Create(c.Request.Context(), &dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, card)
}

// GetCards 获取所有NFC卡片
func (h *CardHandler) GetCards(c *gin.Context) {
	// 获取商户ID
	merchantIDStr := c.Query("merchantId")
	if merchantIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户ID是必需的"})
		return
	}

	merchantID, err := uuid.Parse(merchantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户ID格式无效"})
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	cards, total, err := h.service.GetByMerchantID(c.Request.Context(), merchantID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": cards,
		"meta": gin.H{
			"currentPage":  page,
			"itemsPerPage": pageSize,
			"totalItems":   total,
			"totalPages":   (total + pageSize - 1) / pageSize,
		},
	})
}

// GetCardByID 通过ID获取NFC卡片
func (h *CardHandler) GetCardByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "卡片ID格式无效"})
		return
	}

	card, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

// UpdateCard 更新NFC卡片
func (h *CardHandler) UpdateCard(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "卡片ID格式无效"})
		return
	}

	var dto entities.UpdateNfcCardDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	card, err := h.service.Update(c.Request.Context(), id, &dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

// DeleteCard 删除NFC卡片
func (h *CardHandler) DeleteCard(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "卡片ID格式无效"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "卡片删除成功"})
}

// ActivateCard 激活NFC卡片
func (h *CardHandler) ActivateCard(c *gin.Context) {
	var dto entities.ActivateNfcCardDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	card, err := h.service.Activate(c.Request.Context(), dto.UID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}
