package handlers

import (
	"math"
	"net/http"

	"merchant-service/internal/domain/entities"
	"merchant-service/internal/services"

	"github.com/gin-gonic/gin"
)

// MerchantsHandler 处理商户相关的API请求
type MerchantsHandler struct {
	merchantService *services.MerchantService
}

// NewMerchantsHandler 创建新的商户处理器
func NewMerchantsHandler(merchantService *services.MerchantService) *MerchantsHandler {
	return &MerchantsHandler{
		merchantService: merchantService,
	}
}

// Create 创建商户
func (h *MerchantsHandler) Create(c *gin.Context) {
	var dto entities.CreateMerchantDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	merchant, err := h.merchantService.Create(dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, merchant)
}

// FindAll 获取所有商户
func (h *MerchantsHandler) FindAll(c *gin.Context) {
	var params entities.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	merchants, totalItems, err := h.merchantService.FindAll(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": merchants,
		"meta": entities.PaginationMeta{
			CurrentPage:  params.Page,
			ItemsPerPage: params.Limit,
			TotalItems:   totalItems,
			TotalPages:   int(math.Ceil(float64(totalItems) / float64(params.Limit))),
		},
	})
}

// FindOne 获取单个商户
func (h *MerchantsHandler) FindOne(c *gin.Context) {
	id := c.Param("id")
	merchant, err := h.merchantService.FindOne(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, merchant)
}

// Update 更新商户
func (h *MerchantsHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var dto entities.UpdateMerchantDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	merchant, err := h.merchantService.Update(id, dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, merchant)
}

// Remove 删除商户
func (h *MerchantsHandler) Remove(c *gin.Context) {
	id := c.Param("id")
	if err := h.merchantService.Remove(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RegenerateApiKey 重新生成API Key
func (h *MerchantsHandler) RegenerateApiKey(c *gin.Context) {
	id := c.Param("id")
	apiKey, err := h.merchantService.RegenerateApiKey(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"apiKey": apiKey})
}

// UpdateApproval 更新商户审核状态
func (h *MerchantsHandler) UpdateApproval(c *gin.Context) {
	id := c.Param("id")
	var dto entities.UpdateMerchantApprovalDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取审核人信息
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证信息"})
		return
	}

	// 调用服务方法更新商户审核状态
	merchant, err := h.merchantService.UpdateApproval(id, dto, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, merchant)
}
