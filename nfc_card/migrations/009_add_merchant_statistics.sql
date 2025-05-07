-- 商户统计相关表

-- 商户统计表
CREATE TABLE IF NOT EXISTS merchant_statistics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    total_users INTEGER NOT NULL DEFAULT 0,
    total_nfc_cards INTEGER NOT NULL DEFAULT 0,
    total_videos INTEGER NOT NULL DEFAULT 0,
    total_views INTEGER NOT NULL DEFAULT 0,
    total_publish_jobs INTEGER NOT NULL DEFAULT 0,
    date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(merchant_id, date)
);

-- 商户报表表
CREATE TABLE IF NOT EXISTS merchant_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    report_type VARCHAR(50) NOT NULL,
    period VARCHAR(50) NOT NULL,
    data JSONB NOT NULL,
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX idx_merchant_statistics_merchant_id ON merchant_statistics(merchant_id);
CREATE INDEX idx_merchant_statistics_date ON merchant_statistics(date);
CREATE INDEX idx_merchant_reports_merchant_id ON merchant_reports(merchant_id);
CREATE INDEX idx_merchant_reports_report_type ON merchant_reports(report_type);
CREATE INDEX idx_merchant_reports_period ON merchant_reports(period); 