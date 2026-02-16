-- Convert bigint balances (cents) back to decimal (2 decimal places)
ALTER TABLE accounts 
  ALTER COLUMN balance TYPE DECIMAL(20,2)
  USING (balance::DECIMAL / 100);

ALTER TABLE entries 
  ALTER COLUMN amount TYPE DECIMAL(20,2)
  USING (amount::DECIMAL / 100);

ALTER TABLE transfers 
  ALTER COLUMN amount TYPE DECIMAL(20,2)
  USING (amount::DECIMAL / 100);

-- Restore comments
COMMENT ON COLUMN accounts.balance IS 'Balance in major currency unit (2 decimal places)';
COMMENT ON COLUMN entries.amount IS 'Amount in major currency unit (can be +ve or -ve)';
COMMENT ON COLUMN transfers.amount IS 'Amount in major currency unit (must be +ve)';
