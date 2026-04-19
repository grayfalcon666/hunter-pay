CREATE TABLE bounties (
    id BIGSERIAL PRIMARY KEY,
    employer_username VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    reward_amount BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bounties_employer_username ON bounties (employer_username);
CREATE INDEX idx_bounties_status ON bounties (status);
