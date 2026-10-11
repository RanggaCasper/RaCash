DROP TABLE IF EXISTS balance_ledger;
DROP FUNCTION IF EXISTS balance_ledger_reject_change();
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_deposit_id_fkey;
DROP TABLE IF EXISTS deposits;
DROP TYPE IF EXISTS ledger_type;
DROP TYPE IF EXISTS deposit_status;
