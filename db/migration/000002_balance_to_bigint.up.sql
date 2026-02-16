-- Convert existing decimal balances to bigint (cents)
-- 2 decimal places (cents)
ALTER TABLE accounts 
  ALTER COLUMN balance TYPE BIGINT 
  USING (balance * 100)::BIGINT;

ALTER TABLE entries 
  ALTER COLUMN amount TYPE BIGINT 
  USING (amount * 100)::BIGINT;

ALTER TABLE transfers 
  ALTER COLUMN amount TYPE BIGINT 
  USING (amount * 100)::BIGINT;

-- Update comments
COMMENT ON COLUMN accounts.balance IS 'Balance in cents (smallest currency unit)';
COMMENT ON COLUMN entries.amount IS 'Amount in cents (can be +ve or -ve)';
COMMENT ON COLUMN transfers.amount IS 'Amount in cents (must be +ve)';