-- 给申请表加上猎人的收款账户 ID
ALTER TABLE bounty_applications
ADD COLUMN hunter_account_id BIGINT NOT NULL DEFAULT 0;
