CREATE TABLE IF NOT EXISTS fulfillment_outbox (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL,
    bounty_id BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 5,
    last_error TEXT,
    request_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fulfillment_outbox_status ON fulfillment_outbox(status);
CREATE INDEX IF NOT EXISTS idx_fulfillment_outbox_request_id ON fulfillment_outbox(request_id);
