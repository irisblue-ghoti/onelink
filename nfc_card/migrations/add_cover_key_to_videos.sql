-- 为videos表添加cover_key字段
ALTER TABLE videos ADD COLUMN IF NOT EXISTS cover_key TEXT;

-- 添加注释
COMMENT ON COLUMN videos.cover_key IS '视频封面文件在存储服务中的路径';

-- 更新触发器以更新updated_at字段
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 确保videos表有更新触发器
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger 
        WHERE tgname = 'update_videos_modtime'
    ) THEN
        CREATE TRIGGER update_videos_modtime
        BEFORE UPDATE ON videos
        FOR EACH ROW
        EXECUTE FUNCTION update_modified_column();
    END IF;
END $$; 