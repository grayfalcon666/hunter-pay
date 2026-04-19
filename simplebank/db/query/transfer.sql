-- name: CreateTransfer :one
INSERT INTO transfers (
  from_account_id,
  to_account_id,
  amount,
  trade_type,
  trade_id,
  description
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetTransfer :one
SELECT * FROM transfers
WHERE id = $1 LIMIT 1;

-- name: ListTransfers :many
SELECT * FROM transfers
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: ListTransfersByFromAccount :many
SELECT * FROM transfers
WHERE from_account_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: ListTransfersByToAccount :many
SELECT * FROM transfers
WHERE to_account_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: ListAccountTransfers :many
WITH account_tx AS (
    SELECT transfers.id, transfers.from_account_id, transfers.to_account_id, transfers.amount, transfers.trade_type, transfers.trade_id, transfers.description, transfers.created_at
    FROM transfers
    WHERE transfers.from_account_id = $1
    UNION ALL
    SELECT transfers.id, transfers.from_account_id, transfers.to_account_id, transfers.amount, transfers.trade_type, transfers.trade_id, transfers.description, transfers.created_at
    FROM transfers
    WHERE transfers.to_account_id = $1 AND transfers.from_account_id != transfers.to_account_id
)
SELECT * FROM account_tx
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAccountTransfers :one
WITH account_tx AS (
    SELECT 1 FROM transfers WHERE transfers.from_account_id = $1
    UNION ALL
    SELECT 1 FROM transfers WHERE transfers.to_account_id = $1 AND transfers.from_account_id != transfers.to_account_id
)
SELECT COUNT(*) FROM account_tx;

-- name: UpdateTransfer :one
UPDATE transfers
SET amount = $2
WHERE id = $1
RETURNING *;

-- name: DeleteTransfer :exec
DELETE FROM transfers
WHERE id = $1;
