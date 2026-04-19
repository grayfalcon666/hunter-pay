ALTER TABLE bounty_applications DROP CONSTRAINT IF EXISTS fk_bounty_applications_bounty; --数据库不允许直接删除一个被其他表关联依赖的表
DROP TABLE IF EXISTS bounty_applications;
