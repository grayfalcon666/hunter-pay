-- Add business semantics columns to transfers table
ALTER TABLE transfers ADD COLUMN trade_type varchar(50) NOT NULL DEFAULT 'TRANSFER';
ALTER TABLE transfers ADD COLUMN trade_id bigint;
ALTER TABLE transfers ADD COLUMN description text;

-- Indexes for bidirectional pagination and business type queries
CREATE INDEX IF NOT EXISTS idx_transfers_from_created ON transfers (from_account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_to_created ON transfers (to_account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_trade ON transfers (trade_type, trade_id);

-- Set explicit TRANSFER type for existing records
UPDATE transfers SET trade_type = 'TRANSFER' WHERE trade_type IS NULL;
