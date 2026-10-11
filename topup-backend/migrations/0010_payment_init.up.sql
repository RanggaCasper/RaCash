-- 0010_payment_init: PG, metode bayar, routing PG, charge (database.dbml § MODUL: payment)
-- FK payments.deposit_id → deposits ditambahkan di 0011_wallet_init,
-- payments.order_id → orders di 0012_order_init.

CREATE TYPE payment_method_group AS ENUM ('QRIS', 'VIRTUAL_ACCOUNT', 'EWALLET', 'RETAIL', 'BALANCE');
CREATE TYPE fee_bearer AS ENUM ('CUSTOMER', 'MERCHANT', 'SPLIT');
CREATE TYPE payment_status AS ENUM ('PENDING', 'PAID', 'EXPIRED', 'FAILED', 'REFUNDED');
CREATE TYPE payment_purpose AS ENUM ('ORDER', 'DEPOSIT');

CREATE TABLE payment_gateways (
    id                    uuid PRIMARY KEY,
    code                  varchar(50)   NOT NULL,
    name                  varchar(100)  NOT NULL,
    mode                  vendor_mode   NOT NULL DEFAULT 'sandbox',
    credentials_encrypted bytea,
    credentials_key_id    varchar(32),
    is_active             boolean       NOT NULL DEFAULT false,
    circuit_state         circuit_state NOT NULL DEFAULT 'CLOSED',
    config                jsonb         NOT NULL DEFAULT '{}',
    created_at            timestamptz   NOT NULL DEFAULT now(),
    updated_at            timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT payment_gateways_code_uq UNIQUE (code)
);

CREATE TABLE payment_methods (
    id                 uuid PRIMARY KEY,
    code               varchar(50)          NOT NULL,
    name               varchar(100)         NOT NULL,
    "group"            payment_method_group NOT NULL,
    icon_media_id      uuid REFERENCES media (id),
    fee_flat           bigint               NOT NULL DEFAULT 0,
    fee_percent        numeric(6,3)         NOT NULL DEFAULT 0,
    fee_bearer         fee_bearer           NOT NULL DEFAULT 'CUSTOMER',
    fee_customer_share numeric(5,2),
    min_amount         bigint               NOT NULL DEFAULT 0,
    max_amount         bigint,
    expiry_minutes     integer              NOT NULL DEFAULT 60,
    instructions       jsonb                NOT NULL DEFAULT '[]',
    sort_order         integer              NOT NULL DEFAULT 0,
    is_active          boolean              NOT NULL DEFAULT true,
    created_at         timestamptz          NOT NULL DEFAULT now(),
    updated_at         timestamptz          NOT NULL DEFAULT now(),
    CONSTRAINT payment_methods_code_uq UNIQUE (code),
    CONSTRAINT payment_methods_fee_customer_share_chk CHECK (fee_customer_share BETWEEN 0 AND 100),
    -- fee_customer_share wajib bila SPLIT.
    CONSTRAINT payment_methods_split_share_chk CHECK (fee_bearer <> 'SPLIT' OR fee_customer_share IS NOT NULL),
    CONSTRAINT payment_methods_expiry_minutes_chk CHECK (expiry_minutes > 0)
);

CREATE TABLE payment_method_gateway_map (
    payment_method_id   uuid        NOT NULL REFERENCES payment_methods (id),
    gateway_id          uuid        NOT NULL REFERENCES payment_gateways (id),
    gateway_method_code varchar(64) NOT NULL,
    priority            integer     NOT NULL DEFAULT 1,
    is_active           boolean     NOT NULL DEFAULT true,
    PRIMARY KEY (payment_method_id, gateway_id)
);

CREATE TABLE payments (
    id                uuid PRIMARY KEY,
    purpose           payment_purpose NOT NULL,
    order_id          uuid,
    deposit_id        uuid,
    payment_method_id uuid            NOT NULL REFERENCES payment_methods (id),
    gateway_id        uuid REFERENCES payment_gateways (id),
    external_id       varchar(128),
    amount            bigint          NOT NULL,
    fee               bigint          NOT NULL DEFAULT 0,
    status            payment_status  NOT NULL DEFAULT 'PENDING',
    instructions      jsonb,
    expires_at        timestamptz,
    paid_at           timestamptz,
    refunded_at       timestamptz,
    raw_response      jsonb,
    created_at        timestamptz     NOT NULL DEFAULT now(),
    updated_at        timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT payments_gateway_external_uq UNIQUE (gateway_id, external_id),
    -- Tepat satu referensi terisi, sesuai purpose.
    CONSTRAINT payments_reference_chk CHECK (
        (purpose = 'ORDER' AND order_id IS NOT NULL AND deposit_id IS NULL)
        OR (purpose = 'DEPOSIT' AND deposit_id IS NOT NULL AND order_id IS NULL)
    )
);
CREATE INDEX payments_order_id_idx ON payments (order_id);
CREATE INDEX payments_deposit_id_idx ON payments (deposit_id);
CREATE INDEX payments_status_expires_at_idx ON payments (status, expires_at);
