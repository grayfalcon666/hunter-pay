CREATE TABLE bounty_applications (
    id BIGSERIAL PRIMARY KEY,
    bounty_id BIGINT NOT NULL,
    hunter_username VARCHAR(255) NOT NULL, -- 类型改为 VARCHAR
    status VARCHAR(50) NOT NULL DEFAULT 'APPLIED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE bounty_applications ADD CONSTRAINT fk_bounty_applications_bounty
    FOREIGN KEY (bounty_id) REFERENCES bounties (id) ON DELETE CASCADE;

CREATE UNIQUE INDEX idx_unique_bounty_hunter ON bounty_applications (bounty_id, hunter_username);
CREATE INDEX idx_applications_hunter_username ON bounty_applications (hunter_username);
