-- name: CreateIdempotencyKey :exec
INSERT INTO idempotency_keys (
  key
) VALUES (
  $1
)
ON CONFLICT (key) DO NOTHING;

-- name: TryAcquireIdempotencyKey :one
-- Returns the key if newly inserted, or nothing if already exists.
-- Caller can distinguish via sql.ErrNoRows to know it was a duplicate.
INSERT INTO idempotency_keys (
  key
) VALUES (
  $1
)
ON CONFLICT (key) DO NOTHING
RETURNING key;