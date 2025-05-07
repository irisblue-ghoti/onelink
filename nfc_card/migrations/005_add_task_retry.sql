-- 更新发布任务表，添加重试相关字段
ALTER TABLE publish_jobs ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE publish_jobs ADD COLUMN IF NOT EXISTS max_retries INTEGER NOT NULL DEFAULT 3;
ALTER TABLE publish_jobs ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE publish_jobs ADD COLUMN IF NOT EXISTS last_error TEXT;

-- 添加任务执行历史表，用于追踪任务状态变更
CREATE TABLE IF NOT EXISTS task_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL, -- 关联的任务ID
    task_type VARCHAR(50) NOT NULL, -- 任务类型
    status VARCHAR(50) NOT NULL, -- 任务状态
    retry_count INTEGER NOT NULL DEFAULT 0, -- 重试次数
    error_message TEXT, -- 错误信息
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 添加死信队列表，用于记录最终失败的任务
CREATE TABLE IF NOT EXISTS dead_letter_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    original_task_id UUID NOT NULL, -- 原始任务ID
    task_type VARCHAR(50) NOT NULL, -- 任务类型
    payload JSONB NOT NULL, -- 任务数据
    error_message TEXT NOT NULL, -- 失败原因
    retry_count INTEGER NOT NULL, -- 已重试次数
    max_retries INTEGER NOT NULL, -- 最大重试次数
    failed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    service_name VARCHAR(100) NOT NULL -- 来源服务
);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_task_history_task_id ON task_history(task_id);
CREATE INDEX IF NOT EXISTS idx_task_history_status ON task_history(status);
CREATE INDEX IF NOT EXISTS idx_dead_letter_original_task_id ON dead_letter_tasks(original_task_id);
CREATE INDEX IF NOT EXISTS idx_dead_letter_task_type ON dead_letter_tasks(task_type);
CREATE INDEX IF NOT EXISTS idx_publish_jobs_next_retry ON publish_jobs(next_retry_at) WHERE next_retry_at IS NOT NULL; 