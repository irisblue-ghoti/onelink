-- 权限相关表

-- 角色表
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 权限表
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(resource, action)
);

-- 角色权限关联表
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

-- 用户角色关联表
CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- 部门表
CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    parent_id UUID REFERENCES departments(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 用户部门关联表
CREATE TABLE IF NOT EXISTS user_departments (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    is_manager BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, department_id)
);

-- 修改用户表，增加角色相关字段
ALTER TABLE users DROP COLUMN IF EXISTS role;
    
-- 创建索引
CREATE INDEX idx_roles_name ON roles(name);
CREATE INDEX idx_permissions_resource_action ON permissions(resource, action);
CREATE INDEX idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX idx_departments_merchant_id ON departments(merchant_id);
CREATE INDEX idx_departments_parent_id ON departments(parent_id);
CREATE INDEX idx_user_departments_user_id ON user_departments(user_id);
CREATE INDEX idx_user_departments_department_id ON user_departments(department_id);

-- 默认角色数据
INSERT INTO roles (id, name, description, created_at, updated_at) VALUES 
  (uuid_generate_v4(), 'admin', '系统管理员，拥有所有权限', NOW(), NOW()),
  (uuid_generate_v4(), 'merchant_admin', '商户管理员，可管理自己商户的所有内容', NOW(), NOW()),
  (uuid_generate_v4(), 'merchant_user', '商户普通用户，有有限的权限', NOW(), NOW()),
  (uuid_generate_v4(), 'department_manager', '部门经理，可管理自己部门的内容', NOW(), NOW()),
  (uuid_generate_v4(), 'operator', '运营人员', NOW(), NOW());

-- 默认权限数据
INSERT INTO permissions (id, name, description, resource, action, created_at, updated_at) VALUES 
  -- 商户权限
  (uuid_generate_v4(), 'merchants:list', '查看商户列表', 'merchants', 'list', NOW(), NOW()),
  (uuid_generate_v4(), 'merchants:create', '创建商户', 'merchants', 'create', NOW(), NOW()),
  (uuid_generate_v4(), 'merchants:read', '查看商户详情', 'merchants', 'read', NOW(), NOW()),
  (uuid_generate_v4(), 'merchants:update', '更新商户信息', 'merchants', 'update', NOW(), NOW()),
  (uuid_generate_v4(), 'merchants:delete', '删除商户', 'merchants', 'delete', NOW(), NOW()),
  (uuid_generate_v4(), 'merchants:manage_api_key', '管理商户API密钥', 'merchants', 'manage_api_key', NOW(), NOW()),
  
  -- 用户权限
  (uuid_generate_v4(), 'users:list', '查看用户列表', 'users', 'list', NOW(), NOW()),
  (uuid_generate_v4(), 'users:create', '创建用户', 'users', 'create', NOW(), NOW()),
  (uuid_generate_v4(), 'users:read', '查看用户详情', 'users', 'read', NOW(), NOW()),
  (uuid_generate_v4(), 'users:update', '更新用户信息', 'users', 'update', NOW(), NOW()),
  (uuid_generate_v4(), 'users:delete', '删除用户', 'users', 'delete', NOW(), NOW()),
  (uuid_generate_v4(), 'users:change_password', '修改用户密码', 'users', 'change_password', NOW(), NOW()),
  
  -- 部门权限
  (uuid_generate_v4(), 'departments:list', '查看部门列表', 'departments', 'list', NOW(), NOW()),
  (uuid_generate_v4(), 'departments:create', '创建部门', 'departments', 'create', NOW(), NOW()),
  (uuid_generate_v4(), 'departments:read', '查看部门详情', 'departments', 'read', NOW(), NOW()),
  (uuid_generate_v4(), 'departments:update', '更新部门信息', 'departments', 'update', NOW(), NOW()),
  (uuid_generate_v4(), 'departments:delete', '删除部门', 'departments', 'delete', NOW(), NOW()),
  
  -- 角色权限
  (uuid_generate_v4(), 'roles:list', '查看角色列表', 'roles', 'list', NOW(), NOW()),
  (uuid_generate_v4(), 'roles:create', '创建角色', 'roles', 'create', NOW(), NOW()),
  (uuid_generate_v4(), 'roles:read', '查看角色详情', 'roles', 'read', NOW(), NOW()),
  (uuid_generate_v4(), 'roles:update', '更新角色信息', 'roles', 'update', NOW(), NOW()),
  (uuid_generate_v4(), 'roles:delete', '删除角色', 'roles', 'delete', NOW(), NOW()),
  (uuid_generate_v4(), 'roles:assign', '分配角色', 'roles', 'assign', NOW(), NOW());

-- 为管理员角色分配所有权限
INSERT INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW()
FROM roles r, permissions p
WHERE r.name = 'admin';

-- 为商户管理员分配相关权限
WITH merchant_admin_role AS (
  SELECT id FROM roles WHERE name = 'merchant_admin'
),
merchant_permissions AS (
  SELECT id FROM permissions WHERE resource IN ('merchants', 'users', 'departments') AND action != 'delete'
)
INSERT INTO role_permissions (role_id, permission_id, created_at)
SELECT mr.id, mp.id, NOW()
FROM merchant_admin_role mr, merchant_permissions mp; 