-- Add frozen_balance column to accounts for internal bounty freezing
ALTER TABLE accounts ADD COLUMN frozen_balance bigint NOT NULL DEFAULT 0;
ALTER TABLE accounts ADD CONSTRAINT check_balance_gte_zero CHECK (balance >= 0);

-- System account for DEPOSIT/WITHDRAWAL/BOUNTY_PAYOUT transfers (must come after frozen_balance)
INSERT INTO users (username, hashed_password, full_name, email, password_changed_at, is_email_verified)
VALUES ('SYSTEM', '', 'System Account', 'system@internal', now(), true)
ON CONFLICT (username) DO NOTHING;

INSERT INTO accounts (id, owner, balance, frozen_balance, currency)
VALUES (999999, 'SYSTEM', 0, 0, 'CNY')
ON CONFLICT (id) DO NOTHING;
