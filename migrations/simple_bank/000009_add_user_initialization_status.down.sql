-- 删除用户初始化事件表
DROP TABLE IF EXISTS user_initialization_events;

-- 删除索引
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_initialized_at;

-- 删除用户表中的初始化状态字段
ALTER TABLE users DROP COLUMN IF EXISTS status;
ALTER TABLE users DROP COLUMN IF EXISTS initialization_request_id;
ALTER TABLE users DROP COLUMN IF EXISTS initialized_at;
ALTER TABLE users DROP COLUMN IF EXISTS failed_reason;

-- 删除事件表索引
DROP INDEX IF EXISTS idx_init_events_request_id;
DROP INDEX IF EXISTS idx_init_events_username;
DROP INDEX IF EXISTS idx_init_events_processed;