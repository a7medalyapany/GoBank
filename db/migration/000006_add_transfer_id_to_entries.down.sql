DROP INDEX IF EXISTS entries_account_id_created_at_idx;
DROP INDEX IF EXISTS entries_transfer_id_idx;

ALTER TABLE entries
DROP CONSTRAINT IF EXISTS entries_transfer_id_fkey;

ALTER TABLE entries
DROP COLUMN IF EXISTS transfer_id;
