-- 添加用户资料初始化相关字段
ALTER TABLE user_profiles ADD COLUMN initialization_request_id VARCHAR(255);
ALTER TABLE user_profiles ADD COLUMN created_via_event BOOLEAN DEFAULT false;

-- 创建索引以优化查询
CREATE INDEX IF NOT EXISTS idx_profiles_init_request ON user_profiles(initialization_request_id);
CREATE INDEX IF NOT EXISTS idx_profiles_created_via_event ON user_profiles(created_via_event);
