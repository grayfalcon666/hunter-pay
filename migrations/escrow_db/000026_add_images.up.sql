-- 统一图片资源表
-- 存储相对路径，物理文件由业务层在删除时同步清理
CREATE TABLE IF NOT EXISTS images (
    id BIGSERIAL PRIMARY KEY,
    -- 业务关联：entity_type + entity_id 形成多态关联
    entity_type VARCHAR(50) NOT NULL,  -- 'avatar' | 'resume' | 'bounty' | 'comment'
    entity_id BIGINT NOT NULL,
    -- 相对路径，不含域名，格式如 "uploads/2026/04/abc123.jpg"
    relative_path VARCHAR(500) NOT NULL,
    -- 原始文件名（仅供展示用，不用于定位）
    original_name VARCHAR(255),
    -- 文件大小（字节）
    file_size BIGINT,
    -- MIME 类型
    mime_type VARCHAR(100),
    -- 上传时间
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 复合索引：快速查找某业务实体关联的所有图片
CREATE INDEX idx_images_entity ON images(entity_type, entity_id);
-- 单一索引：按 entity_id 快速筛选
CREATE INDEX idx_images_entity_id ON images(entity_id);

-- 注释说明
COMMENT ON TABLE images IS '统一图片资源表，仅存储相对路径';
COMMENT ON COLUMN images.entity_type IS '业务类型: avatar(头像), resume(简历), bounty(悬赏描述), comment(评论图片)';
COMMENT ON COLUMN images.relative_path IS '相对于上传根目录的路径，不含域名';
