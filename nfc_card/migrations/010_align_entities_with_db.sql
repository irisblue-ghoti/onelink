-- 010_align_entities_with_db.sql
-- 对齐数据库表结构与后端实体

-- 1. 更新 nfc_cards 表，使其与 NfcCard 实体对齐
ALTER TABLE nfc_cards 
    ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'new',
    ADD COLUMN IF NOT EXISTS bound_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS deactivated_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE;

-- 2. 更新 videos 表，使其与 Video 实体对齐
ALTER TABLE videos
    ADD COLUMN IF NOT EXISTS file_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS file_key VARCHAR(255),
    ADD COLUMN IF NOT EXISTS cover_key VARCHAR(255),
    ADD COLUMN IF NOT EXISTS file_type VARCHAR(100),
    ADD COLUMN IF NOT EXISTS size BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS width INTEGER,
    ADD COLUMN IF NOT EXISTS height INTEGER,
    ADD COLUMN IF NOT EXISTS is_transcoded BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS transcode_status VARCHAR(50) DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS storage_path VARCHAR(255);

-- 由于 videos 表原来的 status 字段与 Video 实体的 transcodeStatus 不一致，需要进行迁移
-- 首先创建新字段
ALTER TABLE videos 
    ADD COLUMN IF NOT EXISTS transcode_status VARCHAR(50);

-- 根据现有 status 字段设置 transcode_status 字段值
UPDATE videos SET transcode_status = 
    CASE 
        WHEN status = 'draft' THEN 'pending'
        WHEN status = 'processing' THEN 'processing'
        WHEN status = 'published' THEN 'completed'
        WHEN status = 'failed' THEN 'failed'
        ELSE 'pending'
    END
WHERE transcode_status IS NULL;

-- 3. 更新 merchants 表，使其与 Merchant 实体对齐
ALTER TABLE merchants
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS email VARCHAR(255),
    ADD COLUMN IF NOT EXISTS phone VARCHAR(50),
    ADD COLUMN IF NOT EXISTS website VARCHAR(255),
    ADD COLUMN IF NOT EXISTS address TEXT,
    ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'active';

-- 4. 更新 short_links 表，确保与 ShortLink 实体对齐
ALTER TABLE short_links
    ADD COLUMN IF NOT EXISTS title VARCHAR(255),
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS expiry_date TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;

-- 5. 更新 publish_jobs 表，使其与 PublishJob 实体对齐
ALTER TABLE publish_jobs
    ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_retries INTEGER DEFAULT 3,
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS last_error TEXT;

-- 创建新的索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_nfc_cards_user_id ON nfc_cards(user_id);
CREATE INDEX IF NOT EXISTS idx_nfc_cards_status ON nfc_cards(status);
CREATE INDEX IF NOT EXISTS idx_videos_transcode_status ON videos(transcode_status);
CREATE INDEX IF NOT EXISTS idx_short_links_is_active ON short_links(is_active);
CREATE INDEX IF NOT EXISTS idx_merchants_email ON merchants(email);
CREATE INDEX IF NOT EXISTS idx_merchants_status ON merchants(status);
CREATE INDEX IF NOT EXISTS idx_publish_jobs_retry_count ON publish_jobs(retry_count);
CREATE INDEX IF NOT EXISTS idx_publish_jobs_next_retry_at ON publish_jobs(next_retry_at);

-- 更新 merchants 表的约束，确保 email 唯一（如果您希望强制唯一）
-- ALTER TABLE merchants ADD CONSTRAINT unique_merchant_email UNIQUE(email); 