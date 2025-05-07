package messaging

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"nfc-service/internal/domain/entities"
	"nfc-service/internal/services/cards"
	"nfc-service/internal/services/shortlinks"

	"github.com/google/uuid"
)

// 消息类型常量
const (
	TypeCardCreated   = "card.created"
	TypeCardUpdated   = "card.updated"
	TypeCardActivated = "card.activated"
	TypeCardBound     = "card.bound"
	TypeCardUnbound   = "card.unbound"
)

// CardHandler 处理NFC卡相关消息
type CardHandler struct {
	cardService      cards.Service
	shortlinkService shortlinks.Service
	logger           *log.Logger
}

// NewCardHandler 创建卡片消息处理器
func NewCardHandler(
	cardService cards.Service,
	shortlinkService shortlinks.Service,
	logger *log.Logger,
) *CardHandler {
	return &CardHandler{
		cardService:      cardService,
		shortlinkService: shortlinkService,
		logger:           logger,
	}
}

// HandleMessage 处理接收到的消息
func (h *CardHandler) HandleMessage(topic, msgType string, data []byte) error {
	h.logger.Printf("处理卡片消息: topic=%s, type=%s", topic, msgType)

	switch msgType {
	case TypeCardCreated:
		return h.handleCardCreated(data)
	case TypeCardUpdated:
		return h.handleCardUpdated(data)
	case TypeCardActivated:
		return h.handleCardActivated(data)
	case TypeCardBound:
		return h.handleCardBound(data)
	case TypeCardUnbound:
		return h.handleCardUnbound(data)
	default:
		h.logger.Printf("未知的消息类型: %s", msgType)
		return nil
	}
}

// CardCreatedEvent 卡片创建事件数据
type CardCreatedEvent struct {
	ID         uuid.UUID `json:"id"`
	MerchantID uuid.UUID `json:"merchant_id"`
	UID        string    `json:"uid"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
}

// handleCardCreated 处理卡片创建事件
func (h *CardHandler) handleCardCreated(data []byte) error {
	var event CardCreatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	h.logger.Printf("处理卡片创建事件: ID=%v, UID=%v", event.ID, event.UID)

	// 创建默认的短链接
	defaultLink := &entities.CreateShortLinkDTO{
		TenantID:  event.MerchantID,
		NfcCardID: event.ID,
		TargetURL: "/nfc-landing/" + event.UID,
		Slug:      "", // 系统自动生成
	}

	_, err := h.shortlinkService.Create(context.Background(), defaultLink)
	if err != nil {
		h.logger.Printf("为新卡片创建默认短链接失败: %v", err)
		return err
	}

	h.logger.Printf("为新卡片 %s 创建了默认短链接", event.ID)
	return nil
}

// CardUpdatedEvent 卡片更新事件数据
type CardUpdatedEvent struct {
	ID         uuid.UUID              `json:"id"`
	MerchantID uuid.UUID              `json:"merchant_id"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Changes    map[string]interface{} `json:"changes"`
}

// handleCardUpdated 处理卡片更新事件
func (h *CardHandler) handleCardUpdated(data []byte) error {
	var event CardUpdatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	h.logger.Printf("处理卡片更新事件: ID=%v, 变更=%v", event.ID, event.Changes)

	// 如果默认视频ID发生变化，更新相关短链接
	if _, hasVideoChange := event.Changes["default_video_id"]; hasVideoChange {
		h.logger.Printf("卡片默认视频已更新，需要更新短链接")

		// 获取该卡片的所有短链接
		links, err := h.shortlinkService.GetByNfcCardID(context.Background(), event.ID)
		if err != nil {
			h.logger.Printf("获取卡片短链接失败: %v", err)
			return err
		}

		// 更新默认短链接的目标URL (此处仅作为示例，实际逻辑可能需要扩展ShortLink实体)
		if len(links) > 0 {
			for _, link := range links {
				// 这里我们假设第一个链接为默认链接进行更新
				// 实际项目中可能需要添加IsDefault标记到实体中
				updateDTO := &entities.UpdateShortLinkDTO{
					TargetURL: "/nfc-landing/" + event.ID.String(),
				}

				_, err := h.shortlinkService.Update(context.Background(), link.ID, updateDTO)
				if err != nil {
					h.logger.Printf("更新短链接失败: %v", err)
				} else {
					h.logger.Printf("已更新短链接 %s", link.ID)
					break // 只更新第一个链接
				}
			}
		}
	}

	return nil
}

// CardActivatedEvent 卡片激活事件数据
type CardActivatedEvent struct {
	ID          uuid.UUID `json:"id"`
	MerchantID  uuid.UUID `json:"merchant_id"`
	UID         string    `json:"uid"`
	ActivatedAt time.Time `json:"activated_at"`
}

// handleCardActivated 处理卡片激活事件
func (h *CardHandler) handleCardActivated(data []byte) error {
	var event CardActivatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	h.logger.Printf("处理卡片激活事件: ID=%v, UID=%v, 激活时间=%v",
		event.ID, event.UID, event.ActivatedAt)

	// 获取卡片信息
	card, err := h.cardService.GetByID(context.Background(), event.ID)
	if err != nil {
		h.logger.Printf("获取卡片信息失败: %v", err)
		return err
	}

	// 处理激活后的业务逻辑
	// 1. 检查是否已有短链接，如果没有则创建
	links, err := h.shortlinkService.GetByNfcCardID(context.Background(), event.ID)
	if err != nil {
		h.logger.Printf("获取卡片短链接失败: %v", err)
		return err
	}

	// 如果没有短链接，创建一个
	if len(links) == 0 {
		defaultLink := &entities.CreateShortLinkDTO{
			TenantID:  card.MerchantID,
			NfcCardID: card.ID,
			TargetURL: "/nfc-landing/" + card.UID,
			Slug:      "", // 系统自动生成
		}

		_, err := h.shortlinkService.Create(context.Background(), defaultLink)
		if err != nil {
			h.logger.Printf("为激活卡片创建默认短链接失败: %v", err)
			return err
		}

		h.logger.Printf("为激活卡片 %s 创建了默认短链接", card.ID)
	}

	// 2. 可以触发其他激活后的处理逻辑，例如通知商户等
	h.logger.Printf("NFC卡片 %s 激活流程处理完成", card.ID)

	return nil
}

// CardBindEvent 卡片绑定事件数据
type CardBindEvent struct {
	CardID  uuid.UUID `json:"card_id"`
	UserID  uuid.UUID `json:"user_id"`
	BoundAt time.Time `json:"bound_at"`
}

// handleCardBound 处理卡片绑定事件
func (h *CardHandler) handleCardBound(data []byte) error {
	var event CardBindEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	h.logger.Printf("处理卡片绑定事件: 卡片=%v, 用户=%v, 绑定时间=%v",
		event.CardID, event.UserID, event.BoundAt)

	// 获取卡片信息
	card, err := h.cardService.GetByID(context.Background(), event.CardID)
	if err != nil {
		h.logger.Printf("获取卡片信息失败: %v", err)
		return err
	}

	// 更新卡片状态或关联用户信息的逻辑（具体取决于业务需求）
	// 这里可能需要对卡片实体进行扩展，添加用户ID等字段

	h.logger.Printf("卡片 %s 与用户 %s 绑定处理完成", card.ID, event.UserID)

	return nil
}

// CardUnbindEvent 卡片解绑事件数据
type CardUnbindEvent struct {
	CardID    uuid.UUID `json:"card_id"`
	UserID    uuid.UUID `json:"user_id"`
	UnboundAt time.Time `json:"unbound_at"`
}

// handleCardUnbound 处理卡片解绑事件
func (h *CardHandler) handleCardUnbound(data []byte) error {
	var event CardUnbindEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	h.logger.Printf("处理卡片解绑事件: 卡片=%v, 用户=%v, 解绑时间=%v",
		event.CardID, event.UserID, event.UnboundAt)

	// 获取卡片信息
	card, err := h.cardService.GetByID(context.Background(), event.CardID)
	if err != nil {
		h.logger.Printf("获取卡片信息失败: %v", err)
		return err
	}

	// 处理卡片解绑逻辑
	// 同样，可能需要对卡片实体进行扩展，移除用户ID等字段

	h.logger.Printf("卡片 %s 与用户 %s 解绑处理完成", card.ID, event.UserID)

	return nil
}
