package entities

import (
	"time"
)

// StatsQueryParams 统计数据查询参数
type StatsQueryParams struct {
	TenantID  string    `json:"tenantId"`
	VideoID   string    `json:"videoId"`
	NfcCardID string    `json:"nfcCardId"`
	Platform  string    `json:"platform"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
	Page      int       `json:"page"`
	PageSize  int       `json:"pageSize"`
}
