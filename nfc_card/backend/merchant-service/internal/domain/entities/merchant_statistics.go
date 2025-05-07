package entities

import (
	"time"

	"github.com/google/uuid"
)

// MerchantStatistics 商户统计实体
type MerchantStatistics struct {
	ID               uuid.UUID `json:"id" db:"id"`
	MerchantID       uuid.UUID `json:"merchantId" db:"merchant_id"`
	TotalUsers       int       `json:"totalUsers" db:"total_users"`
	TotalNfcCards    int       `json:"totalNfcCards" db:"total_nfc_cards"`
	TotalVideos      int       `json:"totalVideos" db:"total_videos"`
	TotalViews       int       `json:"totalViews" db:"total_views"`
	TotalPublishJobs int       `json:"totalPublishJobs" db:"total_publish_jobs"`
	Date             time.Time `json:"date" db:"date"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time `json:"updatedAt" db:"updated_at"`
}

// ReportType 报表类型
type ReportType string

const (
	// ReportTypeDaily 日报
	ReportTypeDaily ReportType = "daily"
	// ReportTypeWeekly 周报
	ReportTypeWeekly ReportType = "weekly"
	// ReportTypeMonthly 月报
	ReportTypeMonthly ReportType = "monthly"
	// ReportTypeQuarterly 季报
	ReportTypeQuarterly ReportType = "quarterly"
	// ReportTypeYearly 年报
	ReportTypeYearly ReportType = "yearly"
)

// MerchantReport 商户报表实体
type MerchantReport struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	MerchantID  uuid.UUID      `json:"merchantId" db:"merchant_id"`
	ReportType  ReportType     `json:"reportType" db:"report_type"`
	Period      string         `json:"period" db:"period"`
	Data        map[string]any `json:"data" db:"data"`
	GeneratedAt time.Time      `json:"generatedAt" db:"generated_at"`
	CreatedAt   time.Time      `json:"createdAt" db:"created_at"`
}

// GenerateReportDTO 生成报表DTO
type GenerateReportDTO struct {
	MerchantID uuid.UUID  `json:"merchantId" binding:"required"`
	ReportType ReportType `json:"reportType" binding:"required"`
	StartDate  time.Time  `json:"startDate" binding:"required"`
	EndDate    time.Time  `json:"endDate" binding:"required"`
}

// StatisticsQueryParams 统计查询参数
type StatisticsQueryParams struct {
	MerchantID uuid.UUID `form:"merchantId" binding:"required"`
	StartDate  time.Time `form:"startDate" binding:"required"`
	EndDate    time.Time `form:"endDate" binding:"required"`
}
