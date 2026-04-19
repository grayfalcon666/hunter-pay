CREATE TABLE IF NOT EXISTS task_records (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL,
    bounty_id BIGINT NOT NULL,
    amount BIGINT NOT NULL,
    outcome INTEGER NOT NULL,
    outcome_detail VARCHAR(100),
    employer_rating INTEGER DEFAULT 3,
    hunter_rating INTEGER DEFAULT 3,
    deadline_before TIMESTAMPTZ,
    deadline_after TIMESTAMPTZ,
    extend_count INTEGER DEFAULT 0,
    rating_finalized BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_records_username ON task_records(username);
CREATE INDEX IF NOT EXISTS idx_task_records_role_username ON task_records(role, username);
CREATE INDEX IF NOT EXISTS idx_task_records_bounty ON task_records(bounty_id);
CREATE INDEX IF NOT EXISTS idx_task_records_created ON task_records(created_at DESC);
