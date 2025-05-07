-- 011_fix_alignment_issues.sql
-- 修复之前对齐过程中发现的问题

-- 确保用户表存在
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    merchant_id UUID REFERENCES merchants(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 重新尝试添加nfc_cards的user_id外键引用
ALTER TABLE nfc_cards DROP COLUMN IF EXISTS user_id;
ALTER TABLE nfc_cards ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id);

-- 确保NFC卡状态字段存在
ALTER TABLE nfc_cards ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'new';

-- 确保商户的approval_status字段存在
ALTER TABLE merchants ALTER COLUMN approval_status SET DEFAULT 'pending';

-- 重新创建商户审核相关表
CREATE TABLE IF NOT EXISTS merchant_approval_status (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    status VARCHAR(50) NOT NULL,
    reviewer_id UUID REFERENCES users(id),
    comments TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS merchant_approval_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    status_from VARCHAR(50) NOT NULL,
    status_to VARCHAR(50) NOT NULL,
    reviewer_id UUID REFERENCES users(id),
    comments TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 重新创建索引
CREATE INDEX IF NOT EXISTS idx_nfc_cards_user_id ON nfc_cards(user_id);
CREATE INDEX IF NOT EXISTS idx_nfc_cards_status ON nfc_cards(status);
CREATE INDEX IF NOT EXISTS idx_merchant_approval_status_merchant_id ON merchant_approval_status(merchant_id);
CREATE INDEX IF NOT EXISTS idx_merchant_approval_status_reviewer_id ON merchant_approval_status(reviewer_id);
CREATE INDEX IF NOT EXISTS idx_merchant_approval_history_merchant_id ON merchant_approval_history(merchant_id);
CREATE INDEX IF NOT EXISTS idx_merchant_approval_history_reviewer_id ON merchant_approval_history(reviewer_id);

-- 确保videos的transcode_status字段有正确的默认值
ALTER TABLE videos ALTER COLUMN transcode_status SET DEFAULT 'pending';

-- 确保publish_jobs的retry相关字段存在且有正确的默认值
ALTER TABLE publish_jobs ALTER COLUMN retry_count SET DEFAULT 0;
ALTER TABLE publish_jobs ALTER COLUMN max_retries SET DEFAULT 3; 