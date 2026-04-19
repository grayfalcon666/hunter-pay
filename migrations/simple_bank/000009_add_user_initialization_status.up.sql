-- 添加用户初始化状态字段
ALTER TABLE users ADD COLUMN status VARCHAR(50) DEFAULT 'REGISTERING';
ALTER TABLE users ADD COLUMN initialization_request_id VARCHAR(255);
ALTER TABLE users ADD COLUMN initialized_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN failed_reason TEXT;

-- 创建索引以优化状态查询
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_initialized_at ON users(initialized_at);

-- 创建用户初始化事件表（用于幂等性）
CREATE TABLE user_initialization_events (
    request_id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

-- 创建索引
CREATE INDEX idx_init_events_request_id ON user_initialization_events(request_id);
CREATE INDEX idx_init_events_username ON user_initialization_events(username);
CREATE INDEX idx_init_events_processed ON user_initialization_events(processed_at);