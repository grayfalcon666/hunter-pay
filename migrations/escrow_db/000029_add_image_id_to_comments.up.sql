-- 为 comments 表添加 image_id 字段，关联评论图片
ALTER TABLE comments ADD COLUMN IF NOT EXISTS image_id BIGINT;

-- 图片随评论删除而清理（级联删除暂时用应用层处理）
CREATE INDEX IF NOT EXISTS idx_comments_image_id ON comments(image_id);
