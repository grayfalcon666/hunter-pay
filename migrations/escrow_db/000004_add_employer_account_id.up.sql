-- 添加雇主扣款账户字段
ALTER TABLE bounties
ADD COLUMN employer_account_id BIGINT NOT NULL DEFAULT 0;
