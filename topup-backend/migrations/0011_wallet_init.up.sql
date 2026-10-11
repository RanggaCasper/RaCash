-- 0011_wallet_init: deposit & ledger saldo (database.dbml § MODUL: wallet)

CREATE TYPE deposit_status AS ENUM ('PENDING', 'PAID', 'EXPIRED', 'FAILED');
CREATE TYPE ledger_type AS ENUM ('DEPOSIT', 'PURCHASE', 'REFUND', 'ADJUSTMENT');

CREATE TABLE deposits (
    id                uuid PRIMARY KEY,
    user_id           uuid           NOT NULL REFERENCES users (id),
    amount            bigint         NOT NULL,
    status            deposit_status NOT NULL DEFAULT 'PENDING',
    payment_method_id uuid           NOT NULL REFERENCES payment_methods (id),
    expires_at        timestamptz    NOT NULL,
    paid_at           timestamptz,
    created_at        timestamptz    NOT NULL DEFAULT now(),
    updated_at        timestamptz    NOT NULL DEFAULT now(),
    CONSTRAINT deposits_amount_chk CHECK (amount > 0)
);
CREATE INDEX deposits_user_created_at_idx ON deposits (user_id, created_at);
CREATE INDEX deposits_status_expires_at_idx ON deposits (status, expires_at);

ALTER TABLE payments
    ADD CONSTRAINT payments_deposit_id_fkey FOREIGN KEY (deposit_id) REFERENCES deposits (id);

CREATE TABLE balance_ledger (
    id             uuid PRIMARY KEY,
    user_id        uuid         NOT NULL REFERENCES users (id),
    type           ledger_type  NOT NULL,
    amount         bigint       NOT NULL,
    balance_before bigint       NOT NULL,
    balance_after  bigint       NOT NULL,
    reference_type varchar(32)  NOT NULL,
    reference_id   uuid         NOT NULL,
    description    varchar(255) NOT NULL,
    actor_id       uuid REFERENCES users (id),
    reason         varchar(255),
    created_at     timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT balance_ledger_balance_after_chk CHECK (balance_after >= 0),
    CONSTRAINT balance_ledger_arithmetic_chk CHECK (balance_after = balance_before + amount),
    -- Mencegah potong/refund ganda untuk referensi yang sama.
    CONSTRAINT balance_ledger_reference_uq UNIQUE (reference_type, reference_id, type)
);
CREATE INDEX balance_ledger_user_created_at_idx ON balance_ledger (user_id, created_at);

-- Append-only (RULES §3.4). REVOKE berlaku untuk role aplikasi yang bukan
-- pemilik tabel; trigger menjaga juga pemilik/superuser.
REVOKE UPDATE, DELETE, TRUNCATE ON balance_ledger FROM PUBLIC;

CREATE FUNCTION balance_ledger_reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'tabel % bersifat append-only: % ditolak', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END;
$$;

CREATE TRIGGER balance_ledger_append_only
    BEFORE UPDATE OR DELETE ON balance_ledger
    FOR EACH ROW EXECUTE FUNCTION balance_ledger_reject_change();
CREATE TRIGGER balance_ledger_no_truncate
    BEFORE TRUNCATE ON balance_ledger
    FOR EACH STATEMENT EXECUTE FUNCTION balance_ledger_reject_change();
