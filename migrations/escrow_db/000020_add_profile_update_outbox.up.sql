CREATE TABLE IF NOT EXISTS profile_update_outbox (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    bounty_id BIGINT NOT NULL,
    delta_completed INTEGER DEFAULT 0,
    delta_earnings BIGINT DEFAULT 0,
    delta_posted INTEGER DEFAULT 0,
    delta_completed_as_employer INTEGER DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 5,
    last_error TEXT,
    request_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_username ON profile_update_outbox(username);
CREATE INDEX IF NOT EXISTS idx_outbox_status ON profile_update_outbox(status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_outbox_request_id ON profile_update_outbox(request_id);
