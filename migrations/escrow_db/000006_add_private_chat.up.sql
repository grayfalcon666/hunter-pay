-- Private conversations (user-to-user, not bounty-scoped)
CREATE TABLE IF NOT EXISTS private_conversations (
    id BIGSERIAL PRIMARY KEY,
    user1_username VARCHAR(255) NOT NULL,
    user2_username VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user1_username, user2_username)
);

CREATE INDEX IF NOT EXISTS idx_private_conv_user1 ON private_conversations(user1_username);
CREATE INDEX IF NOT EXISTS idx_private_conv_user2 ON private_conversations(user2_username);

-- Unified messages table for both bounty-scoped and private chats
CREATE TABLE IF NOT EXISTS all_messages (
    id BIGSERIAL PRIMARY KEY,
    message_type VARCHAR(20) NOT NULL,
    conversation_id BIGINT,
    bounty_chat_id  BIGINT,
    sender_username VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_all_messages_private_conv
        FOREIGN KEY (conversation_id) REFERENCES private_conversations(id) ON DELETE CASCADE,
    CONSTRAINT fk_all_messages_bounty_chat
        FOREIGN KEY (bounty_chat_id) REFERENCES chats(id) ON DELETE CASCADE,
    CONSTRAINT chk_exactly_one_conversation
        CHECK (
            (message_type = 'private' AND conversation_id IS NOT NULL AND bounty_chat_id IS NULL) OR
            (message_type = 'bounty'  AND conversation_id IS NULL AND bounty_chat_id IS NOT NULL)
        )
);

CREATE INDEX IF NOT EXISTS idx_all_messages_conversation ON all_messages(conversation_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_all_messages_bounty_chat  ON all_messages(bounty_chat_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_all_messages_sender ON all_messages(sender_username);

-- Per-user unread counts for private conversations
CREATE TABLE IF NOT EXISTS conversation_unread_counts (
    username VARCHAR(255) NOT NULL,
    conversation_id BIGINT NOT NULL,
    unread_count INT NOT NULL DEFAULT 0,
    PRIMARY KEY (username, conversation_id),
    CONSTRAINT fk_unread_conv
        FOREIGN KEY (conversation_id) REFERENCES private_conversations(id) ON DELETE CASCADE
);
