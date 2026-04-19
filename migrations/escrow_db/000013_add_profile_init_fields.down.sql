-- 删除索引
DROP INDEX IF EXISTS idx_profiles_init_request;
DROP INDEX IF EXISTS idx_profiles_created_via_event;

-- 删除用户资料表中的初始化字段
ALTER TABLE user_profiles DROP COLUMN IF EXISTS initialization_request_id;
ALTER TABLE user_profiles DROP COLUMN IF EXISTS created_via_event;
