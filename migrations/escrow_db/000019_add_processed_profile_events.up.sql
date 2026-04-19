CREATE TABLE IF NOT EXISTS processed_profile_events (
    request_id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    bounty_id BIGINT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_processed_events_username ON processed_profile_events(username);
