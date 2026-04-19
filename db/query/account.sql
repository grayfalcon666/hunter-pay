-- name: CreateAccount :one
INSERT INTO accounts (
  owner,
  balance,
  frozen_balance,
  currency
) VALUES (
  $1, $2, 0, $3
)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1;

-- name: ListAccounts :many
SELECT * FROM accounts
WHERE owner = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UpdateAccount :one
UPDATE accounts
SET balance = $2
WHERE id = $1
RETURNING *;

-- name: AddAccountBalance :one
UPDATE accounts
SET balance = balance + sqlc.arg(amount)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: FreezeAccount :one
-- Freeze amount: deduct from balance and add to frozen_balance
-- Fails if balance < amount (returns no rows)
UPDATE accounts
SET balance = balance - $2,
    frozen_balance = frozen_balance + $2
WHERE id = $1 AND balance >= $2
RETURNING *;

-- name: UnfreezeAccount :one
-- Unfreeze amount: return from frozen_balance back to balance
-- Fails if frozen_balance < amount (returns no rows)
UPDATE accounts
SET frozen_balance = frozen_balance - $2,
    balance = balance + $2
WHERE id = $1 AND frozen_balance >= $2
RETURNING *;

-- name: SpendFrozenBalance :one
-- Used for BOUNTY_PAYOUT: deduct from frozen_balance only
-- (balance was already deducted during Freeze)
-- Fails if frozen_balance < amount (returns no rows)
UPDATE accounts
SET frozen_balance = frozen_balance - $2
WHERE id = $1 AND frozen_balance >= $2
RETURNING *;

-- name: SpendFrozenForWithdrawal :one
-- Permanently deduct from frozen_balance for withdrawal (WITHDRAWAL_OUT)
-- Fails if frozen_balance < amount (returns no rows)
UPDATE accounts
SET frozen_balance = frozen_balance - $2
WHERE id = $1 AND frozen_balance >= $2
RETURNING *;

-- name: DeleteAccount :exec
DELETE FROM accounts
WHERE id = $1;
