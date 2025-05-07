-- 创建应用RLS所需的函数
CREATE OR REPLACE FUNCTION auth.current_tenant_id()
RETURNS UUID AS $$
BEGIN
  RETURN current_setting('app.current_tenant', TRUE)::UUID;
EXCEPTION
  WHEN OTHERS THEN
    RETURN NULL;
END
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 为所有表添加tenant_id列并设置RLS策略的函数
CREATE OR REPLACE FUNCTION auth.create_tenant_schema_for_table(
  table_name text,
  schema_name text DEFAULT 'public'
)
RETURNS void AS $$
BEGIN
  -- 启用RLS
  EXECUTE format(
    'ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY',
    schema_name,
    table_name
  );
  
  -- 创建RLS策略
  EXECUTE format(
    'DROP POLICY IF EXISTS tenant_isolation_policy ON %I.%I',
    schema_name,
    table_name
  );
  
  EXECUTE format(
    'CREATE POLICY tenant_isolation_policy ON %I.%I
     USING (merchant_id = auth.current_tenant_id())
     WITH CHECK (merchant_id = auth.current_tenant_id())',
    schema_name,
    table_name
  );
END;
$$ LANGUAGE plpgsql;

-- 对需要RLS的表应用策略
SELECT auth.create_tenant_schema_for_table('videos');
SELECT auth.create_tenant_schema_for_table('nfc_cards');
SELECT auth.create_tenant_schema_for_table('publish_jobs');
SELECT auth.create_tenant_schema_for_table('channel_accounts');
SELECT auth.create_tenant_schema_for_table('stats');
SELECT auth.create_tenant_schema_for_table('short_links');
SELECT auth.create_tenant_schema_for_table('billing_records');

-- 创建管理员角色可以绕过RLS的策略
ALTER TABLE videos FORCE ROW LEVEL SECURITY;
ALTER TABLE nfc_cards FORCE ROW LEVEL SECURITY;
ALTER TABLE publish_jobs FORCE ROW LEVEL SECURITY;
ALTER TABLE channel_accounts FORCE ROW LEVEL SECURITY;
ALTER TABLE stats FORCE ROW LEVEL SECURITY;
ALTER TABLE short_links FORCE ROW LEVEL SECURITY;
ALTER TABLE billing_records FORCE ROW LEVEL SECURITY;

-- 为管理员用户添加绕过RLS的能力
CREATE POLICY admin_policy ON videos TO admin USING (true);
CREATE POLICY admin_policy ON nfc_cards TO admin USING (true);
CREATE POLICY admin_policy ON publish_jobs TO admin USING (true);
CREATE POLICY admin_policy ON channel_accounts TO admin USING (true);
CREATE POLICY admin_policy ON stats TO admin USING (true);
CREATE POLICY admin_policy ON short_links TO admin USING (true);
CREATE POLICY admin_policy ON billing_records TO admin USING (true); 