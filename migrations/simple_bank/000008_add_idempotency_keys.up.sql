CREATE TABLE idempotency_keys (
    key VARCHAR(255) PRIMARY KEY, -- 对应 out_trade_no
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);