package services

import (
	"log"

	"distribution-service/internal/adapters/douyin"
	"distribution-service/internal/adapters/kuaishou"
	"distribution-service/internal/adapters/wechat"
	"distribution-service/internal/adapters/xiaohongshu"
	"distribution-service/internal/storage"
)

// DistributionService 分发服务
type DistributionService struct {
	repos              *storage.Repositories
	douyinAdapter      *douyin.Adapter
	kuaishouAdapter    *kuaishou.Adapter
	wechatAdapter      *wechat.Adapter
	xiaohongshuAdapter *xiaohongshu.Adapter
	logger             *log.Logger
}

// NewDistributionService 创建分发服务
func NewDistributionService(
	repos *storage.Repositories,
	douyinAdapter *douyin.Adapter,
	kuaishouAdapter *kuaishou.Adapter,
	wechatAdapter *wechat.Adapter,
	xiaohongshuAdapter *xiaohongshu.Adapter,
	logger *log.Logger,
) *DistributionService {
	return &DistributionService{
		repos:              repos,
		douyinAdapter:      douyinAdapter,
		kuaishouAdapter:    kuaishouAdapter,
		wechatAdapter:      wechatAdapter,
		xiaohongshuAdapter: xiaohongshuAdapter,
		logger:             logger,
	}
}
