-- Threaded comments on bounty pages (楼中楼)
CREATE TABLE IF NOT EXISTS comments (
    id BIGSERIAL PRIMARY KEY,
    bounty_id BIGINT NOT NULL,
    parent_id BIGINT,
    author_username VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_comments_bounty
        FOREIGN KEY (bounty_id) REFERENCES bounties(id) ON DELETE CASCADE,
    CONSTRAINT fk_comments_parent
        FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_comments_bounty ON comments(bounty_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_comments_parent  ON comments(parent_id);
