-- 接单邀请表
CREATE TABLE invitations (
    id BIGSERIAL PRIMARY KEY,
    bounty_id BIGINT NOT NULL REFERENCES bounties(id) ON DELETE CASCADE,
    poster_username VARCHAR(255) NOT NULL,
    hunter_username VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'ACCEPTED', 'DECLINED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (bounty_id, hunter_username)
);

CREATE INDEX idx_invitations_hunter ON invitations(hunter_username);
CREATE INDEX idx_invitations_poster ON invitations(poster_username);
CREATE INDEX idx_invitations_status ON invitations(status);
CREATE INDEX idx_invitations_bounty ON invitations(bounty_id);
