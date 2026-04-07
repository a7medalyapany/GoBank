-- name: CreateEntry :one
INSERT INTO entries (account_id, amount) 
VALUES ($1, $2)
RETURNING *;

-- name: CreateTransferEntry :one
INSERT INTO entries (account_id, amount, transfer_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetEntry :one
SELECT * FROM entries 
WHERE id = $1;

-- name: ListEntries :many
SELECT * FROM entries
WHERE account_id = $1
ORDER BY id
LIMIT $2 OFFSET $3;

-- name: ListActivityEntries :many
SELECT
  e.id,
  e.account_id,
  e.amount,
  a.currency,
  e.created_at,
  e.transfer_id,
  ca.id AS counterpart_account_id,
  ca.owner AS counterpart_owner,
  ca.currency AS counterpart_currency
FROM entries e
JOIN accounts a ON a.id = e.account_id
LEFT JOIN transfers t ON t.id = e.transfer_id
LEFT JOIN accounts ca ON ca.id = CASE
  WHEN t.from_account_id = e.account_id THEN t.to_account_id
  WHEN t.to_account_id = e.account_id THEN t.from_account_id
  ELSE NULL
END
WHERE a.owner = sqlc.arg(owner)
ORDER BY e.created_at DESC, e.id DESC
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);
