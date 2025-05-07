-- 商户审核相关表

-- 更新商户表，增加审核状态字段
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS approval_status VARCHAR(50) NOT NULL DEFAULT 'pending';

-- 商户审核状态表
CREATE TABLE IF NOT EXISTS merchant_approval_status (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    status VARCHAR(50) NOT NULL,
    reviewer_id UUID REFERENCES users(id),
    comments TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 商户审核历史表
CREATE TABLE IF NOT EXISTS merchant_approval_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    status_from VARCHAR(50) NOT NULL,
    status_to VARCHAR(50) NOT NULL,
    reviewer_id UUID REFERENCES users(id),
    comments TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX idx_merchant_approval_status_merchant_id ON merchant_approval_status(merchant_id);
CREATE INDEX idx_merchant_approval_status_reviewer_id ON merchant_approval_status(reviewer_id);
CREATE INDEX idx_merchant_approval_history_merchant_id ON merchant_approval_history(merchant_id);
CREATE INDEX idx_merchant_approval_history_reviewer_id ON merchant_approval_history(reviewer_id); 