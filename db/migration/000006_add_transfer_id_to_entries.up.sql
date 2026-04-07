ALTER TABLE entries
ADD COLUMN transfer_id BIGINT NULL;

ALTER TABLE entries
ADD CONSTRAINT entries_transfer_id_fkey
FOREIGN KEY (transfer_id) REFERENCES transfers (id);

CREATE INDEX ON entries (transfer_id);
CREATE INDEX ON entries (account_id, created_at DESC);
