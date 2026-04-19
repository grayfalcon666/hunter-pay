-- One chat row per bounty (created lazily when the first message is sent)
CREATE TABLE IF NOT EXISTS chats (
    id BIGSERIAL PRIMARY KEY,
    bounty_id BIGINT NOT NULL UNIQUE,
    employer_username VARCHAR(255) NOT NULL,
    hunter_username VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE chats ADD CONSTRAINT fk_chats_bounty
    FOREIGN KEY (bounty_id) REFERENCES bounties(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_chats_bounty_id ON chats(bounty_id);

-- Individual messages
CREATE TABLE IF NOT EXISTS chat_messages (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    sender_username VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE chat_messages ADD CONSTRAINT fk_chat_messages_chat
    FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_chat_messages_chat_id ON chat_messages(chat_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_created_at ON chat_messages(created_at);
