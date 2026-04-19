-- name: CreateLedgerEntry :one
INSERT INTO ledger_entries (
    account_id,
    bounty_id,
    event_type,
    amount,
    counterparty,
    description
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListLedgerEntriesByAccount :many
SELECT * FROM ledger_entries
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: ListLedgerEntriesByAccountAndType :many
SELECT * FROM ledger_entries
WHERE account_id = $1 AND event_type = $2
ORDER BY created_at DESC
LIMIT $3
OFFSET $4;

-- name: ListLedgerEntriesByBounty :many
SELECT * FROM ledger_entries
WHERE bounty_id = $1
ORDER BY created_at DESC;
