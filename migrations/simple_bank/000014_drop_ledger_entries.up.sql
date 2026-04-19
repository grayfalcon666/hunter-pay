-- Drop ledger_entries table after all code that writes to it has been decommissioned
-- WARNING: This must be executed AFTER Phase 1-5 code changes are deployed and verified
DROP TABLE IF EXISTS ledger_entries;
