-- Backfill existing bounty-scoped chat messages into the unified all_messages table
-- Only insert messages that haven't already been backfilled
INSERT INTO all_messages (message_type, bounty_chat_id, sender_username, content, is_read, created_at)
SELECT 'bounty', chat_id, sender_username, content, false, created_at
FROM chat_messages
WHERE NOT EXISTS (
    SELECT 1 FROM all_messages m
    WHERE m.bounty_chat_id = chat_messages.chat_id
      AND m.sender_username = chat_messages.sender_username
      AND m.content = chat_messages.content
      AND m.created_at = chat_messages.created_at
);
