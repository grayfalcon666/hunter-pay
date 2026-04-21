-- 将 images.entity_id 从 BIGINT 改为 VARCHAR(100)
-- 原因：avatar/resume 用 username（字符串），bounty/comment 用数字字符串
ALTER TABLE images ALTER COLUMN entity_id TYPE VARCHAR(100);
