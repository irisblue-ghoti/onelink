package adapters

import (
	"context"

	"stats-service/internal/domain/entities"
)

// PlatformAdapter 平台适配器接口
type PlatformAdapter interface {
	// GetPlatformName 获取平台名称
	GetPlatformName() string

	// CollectStats 收集平台统计数据
	CollectStats(ctx context.Context, platformID string) (*entities.PlatformStats, error)

	// CollectBatchStats 批量收集平台统计数据
	CollectBatchStats(ctx context.Context, platformIDs []string) ([]*entities.PlatformStats, error)
}
