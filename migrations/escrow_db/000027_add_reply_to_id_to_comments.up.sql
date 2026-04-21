-- 为 comments 表添加 reply_to_id 字段，实现两级楼中楼（扁平化嵌套评论）
-- parent_id: 指向根评论（父亲评论），NULL 表示自己是根评论
-- reply_to_id: 指向具体被回复的那条评论，NULL 表示直接回复父亲
ALTER TABLE comments ADD COLUMN IF NOT EXISTS reply_to_id BIGINT;

-- reply_to_id 也引用 comments(id)，允许级联删除
ALTER TABLE comments
    ADD CONSTRAINT fk_comments_reply_to
    FOREIGN KEY (reply_to_id) REFERENCES comments(id) ON DELETE CASCADE;

-- 索引：按 reply_to_id 快速查找某评论的所有回复
CREATE INDEX IF NOT EXISTS idx_comments_reply_to ON comments(reply_to_id);
-- 复合索引：按根评论 + 创建时间排序（楼中楼常用查询）
CREATE INDEX IF NOT EXISTS idx_comments_parent_created ON comments(parent_id, created_at ASC);

COMMENT ON COLUMN comments.reply_to_id IS '指向被回复的具体评论ID，NULL表示直接回复父亲评论';
