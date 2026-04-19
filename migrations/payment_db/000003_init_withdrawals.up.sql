CREATE TABLE withdrawals (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    account_id BIGINT NOT NULL,
    amount BIGINT NOT NULL,         -- 提现金额
    alipay_account VARCHAR(255) NOT NULL, -- 用户填写的收款支付宝账号(邮箱或手机号)
    out_biz_no VARCHAR(100) UNIQUE NOT NULL, -- 提现单号
    pay_fund_order_id VARCHAR(100) DEFAULT '', -- 支付宝返回的打款流水号
    status VARCHAR(50) NOT NULL DEFAULT 'PROCESSING', -- PROCESSING, SUCCESS, FAILED
    error_msg TEXT DEFAULT '',      -- 如果失败，记录失败原因
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_withdrawals_username ON withdrawals (username);