CREATE TABLE IF NOT EXISTS ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id BIGINT NOT NULL REFERENCES accounts(id),
    bounty_id BIGINT,
    event_type TEXT NOT NULL,
    amount BIGINT NOT NULL,
    counterparty TEXT,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ledger_entries_account_id ON ledger_entries(account_id);
CREATE INDEX idx_ledger_entries_bounty_id ON ledger_entries(bounty_id);
CREATE INDEX idx_ledger_entries_created_at ON ledger_entries(created_at DESC);
