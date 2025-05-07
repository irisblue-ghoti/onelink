-- 创建nacos_config数据库
CREATE DATABASE nacos_config;

\c nacos_config;

CREATE TABLE IF NOT EXISTS "config_info" (
  "id" bigserial NOT NULL PRIMARY KEY,
  "data_id" varchar(255) NOT NULL,
  "group_id" varchar(255) DEFAULT NULL,
  "content" text NOT NULL,
  "md5" varchar(32) DEFAULT NULL,
  "gmt_create" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "gmt_modified" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "src_user" text,
  "src_ip" varchar(50) DEFAULT NULL,
  "app_name" varchar(128) DEFAULT NULL,
  "tenant_id" varchar(128) DEFAULT '',
  "c_desc" varchar(256) DEFAULT NULL,
  "c_use" varchar(64) DEFAULT NULL,
  "effect" varchar(64) DEFAULT NULL,
  "type" varchar(64) DEFAULT NULL,
  "c_schema" text,
  "encrypted_data_key" text NOT NULL DEFAULT ''
);

CREATE INDEX idx_data_id ON config_info (data_id);
CREATE INDEX idx_group_id ON config_info (group_id);
CREATE INDEX idx_tenant_id ON config_info (tenant_id);

CREATE TABLE IF NOT EXISTS "config_info_aggr" (
  "id" bigserial NOT NULL PRIMARY KEY,
  "data_id" varchar(255) NOT NULL,
  "group_id" varchar(255) NOT NULL,
  "datum_id" varchar(255) NOT NULL,
  "content" text NOT NULL,
  "gmt_modified" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "app_name" varchar(128) DEFAULT NULL,
  "tenant_id" varchar(128) DEFAULT ''
);

CREATE INDEX idx_config_info_aggr_1 ON config_info_aggr (data_id, group_id, tenant_id);
CREATE INDEX idx_config_info_aggr_2 ON config_info_aggr (datum_id);

CREATE TABLE IF NOT EXISTS "config_info_beta" (
  "id" bigserial NOT NULL PRIMARY KEY,
  "data_id" varchar(255) NOT NULL,
  "group_id" varchar(128) NOT NULL,
  "app_name" varchar(128) DEFAULT NULL,
  "content" text NOT NULL,
  "beta_ips" varchar(1024) DEFAULT NULL,
  "md5" varchar(32) DEFAULT NULL,
  "gmt_create" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "gmt_modified" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "src_user" text,
  "src_ip" varchar(50) DEFAULT NULL,
  "tenant_id" varchar(128) DEFAULT ''
);

CREATE INDEX idx_config_info_beta_1 ON config_info_beta (data_id, group_id, tenant_id);

CREATE TABLE IF NOT EXISTS "config_info_tag" (
  "id" bigserial NOT NULL PRIMARY KEY,
  "data_id" varchar(255) NOT NULL,
  "group_id" varchar(128) NOT NULL,
  "tenant_id" varchar(128) DEFAULT '',
  "tag_id" varchar(128) NOT NULL,
  "app_name" varchar(128) DEFAULT NULL,
  "content" text NOT NULL,
  "md5" varchar(32) DEFAULT NULL,
  "gmt_create" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "gmt_modified" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "src_user" text,
  "src_ip" varchar(50) DEFAULT NULL
);

CREATE INDEX idx_config_info_tag_1 ON config_info_tag (data_id, group_id, tenant_id);
CREATE INDEX idx_config_info_tag_2 ON config_info_tag (tag_id);

CREATE TABLE IF NOT EXISTS "config_tags_relation" (
  "id" bigserial NOT NULL PRIMARY KEY,
  "tag_name" varchar(128) NOT NULL,
  "tag_type" varchar(64) DEFAULT NULL,
  "data_id" varchar(255) NOT NULL,
  "group_id" varchar(128) NOT NULL,
  "tenant_id" varchar(128) DEFAULT '',
  "nid" bigserial NOT NULL
);

CREATE INDEX idx_config_tags_relation_1 ON config_tags_relation (id, tag_name, tag_type);
CREATE INDEX idx_config_tags_relation_2 ON config_tags_relation (tenant_id);

CREATE TABLE IF NOT EXISTS "group_capacity" (
  "id" bigserial NOT NULL PRIMARY KEY,
  "group_id" varchar(128) NOT NULL DEFAULT '' UNIQUE,
  "quota" int DEFAULT '0',
  "usage" int DEFAULT '0',
  "max_size" int DEFAULT '0',
  "max_aggr_count" int DEFAULT '0',
  "max_aggr_size" int DEFAULT '0',
  "max_history_count" int DEFAULT '0',
  "gmt_create" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "gmt_modified" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX uk_group_id ON group_capacity (group_id);

CREATE TABLE IF NOT EXISTS "his_config_info" (
  "id" bigserial NOT NULL,
  "nid" bigserial NOT NULL PRIMARY KEY,
  "data_id" varchar(255) NOT NULL,
  "group_id" varchar(128) NOT NULL,
  "app_name" varchar(128) DEFAULT NULL,
  "content" text NOT NULL,
  "md5" varchar(32) DEFAULT NULL,
  "gmt_create" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "gmt_modified" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "src_user" text,
  "src_ip" varchar(50) DEFAULT NULL,
  "op_type" char(10) DEFAULT NULL,
  "tenant_id" varchar(128) DEFAULT ''
);

CREATE INDEX idx_his_config_info ON his_config_info (data_id, group_id, tenant_id);

CREATE TABLE IF NOT EXISTS "tenant_capacity" (
  "id" bigserial NOT NULL PRIMARY KEY,
  "tenant_id" varchar(128) NOT NULL DEFAULT '' UNIQUE,
  "quota" int DEFAULT '0',
  "usage" int DEFAULT '0',
  "max_size" int DEFAULT '0',
  "max_aggr_count" int DEFAULT '0',
  "max_aggr_size" int DEFAULT '0',
  "max_history_count" int DEFAULT '0',
  "gmt_create" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "gmt_modified" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX uk_tenant_id ON tenant_capacity (tenant_id);

CREATE TABLE IF NOT EXISTS "tenant_info" (
  "id" bigserial NOT NULL PRIMARY KEY,
  "kp" varchar(128) NOT NULL,
  "tenant_id" varchar(128) DEFAULT '',
  "tenant_name" varchar(128) DEFAULT '',
  "tenant_desc" varchar(256) DEFAULT NULL,
  "create_source" varchar(32) DEFAULT NULL,
  "gmt_create" int NOT NULL,
  "gmt_modified" int NOT NULL
);

CREATE INDEX idx_tenant_info_kp ON tenant_info (kp);
CREATE UNIQUE INDEX uk_tenant_info_kp_tenant_id ON tenant_info (kp, tenant_id);

CREATE TABLE IF NOT EXISTS "users" (
  "username" varchar(50) NOT NULL PRIMARY KEY,
  "password" varchar(500) NOT NULL,
  "enabled" boolean NOT NULL
);

CREATE TABLE IF NOT EXISTS "roles" (
  "username" varchar(50) NOT NULL,
  "role" varchar(50) NOT NULL,
  CONSTRAINT uk_username_role UNIQUE (username, role)
);

CREATE TABLE IF NOT EXISTS "permissions" (
  "role" varchar(50) NOT NULL,
  "resource" varchar(512) NOT NULL,
  "action" varchar(8) NOT NULL,
  CONSTRAINT uk_role_permission UNIQUE (role, resource, action)
);

INSERT INTO users (username, password, enabled) VALUES ('nacos', '$2a$10$EuWPZHzz32dJN7jexM34MOeYirDdFAZm2kuWj7VEOJhhZkDrxfvUu', TRUE);
INSERT INTO roles (username, role) VALUES ('nacos', 'ROLE_ADMIN'); 