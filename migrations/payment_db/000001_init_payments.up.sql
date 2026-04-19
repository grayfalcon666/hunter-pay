CREATE TABLE payments (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL, 
    amount BIGINT NOT NULL,        
    currency VARCHAR(10) NOT NULL DEFAULT 'CNY',
    out_trade_no VARCHAR(100) UNIQUE NOT NULL, -- 唯一系统订单号
    alipay_trade_no VARCHAR(100) DEFAULT '',   -- 支付宝返回的流水号
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- PENDING, SUCCESS, FAILED
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_username ON payments (username);
CREATE INDEX idx_payments_status ON payments (status);