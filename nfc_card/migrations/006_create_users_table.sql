-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    merchant_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_merchant_id ON users(merchant_id);

-- 添加示例用户 (密码为bcrypt编码的 'password')
INSERT INTO users (id, merchant_id, name, email, password, role, status, created_at, updated_at)
VALUES 
    ('1', '1', '管理员', 'admin@example.com', '$2a$10$JqgbLBYYjE/oYrXTKL.xNe3TeEGlUL2TqVKJvP/JfBKIjvVAzH6ji', 'admin', 'active', NOW(), NOW()),
    ('2', '2', '商户用户', 'merchant@example.com', '$2a$10$JqgbLBYYjE/oYrXTKL.xNe3TeEGlUL2TqVKJvP/JfBKIjvVAzH6ji', 'merchant', 'active', NOW(), NOW());

-- 添加外键约束
ALTER TABLE users
    ADD CONSTRAINT fk_users_merchant
    FOREIGN KEY (merchant_id)
    REFERENCES merchants(id)
    ON DELETE CASCADE;

-- 添加行级安全策略
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- 创建策略
CREATE POLICY user_tenant_isolation_policy ON users
    USING (merchant_id = current_setting('app.current_tenant_id')::VARCHAR);

-- 为超级用户创建策略
CREATE POLICY user_superuser_policy ON users
    USING (current_setting('app.is_superuser')::BOOLEAN = true);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP POLICY IF EXISTS user_tenant_isolation_policy ON users;
DROP POLICY IF EXISTS user_superuser_policy ON users;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd 