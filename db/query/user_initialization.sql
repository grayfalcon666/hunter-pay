-- name: CreateUserAccount :one
INSERT INTO accounts (owner, balance, currency, created_at)
VALUES (@owner, 0, @currency, NOW())
RETURNING id;

-- name: UpdateUserStatus :exec
UPDATE users
SET status = @status::VARCHAR,
    initialization_request_id = @request_id::VARCHAR,
    initialized_at = CASE WHEN @status = 'INITIALIZED' THEN NOW() ELSE initialized_at END,
    failed_reason = @failed_reason::TEXT
WHERE username = @username;

-- name: CheckEventProcessed :one
SELECT EXISTS (
    SELECT 1 FROM user_initialization_events
    WHERE request_id = @request_id
) as processed;

-- name: MarkEventProcessed :exec
INSERT INTO user_initialization_events (request_id, username, event_type, payload, created_at)
VALUES (@request_id, @username, @event_type, @payload::jsonb, NOW())
ON CONFLICT (request_id) DO NOTHING;
