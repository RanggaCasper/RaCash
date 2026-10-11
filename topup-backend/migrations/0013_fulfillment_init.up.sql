-- 0013_fulfillment_init: percobaan pembelian ke provider (database.dbml § MODUL: fulfillment)

CREATE TYPE attempt_status AS ENUM ('SUCCESS', 'PENDING', 'FAILED');

CREATE TABLE order_attempts (
    id                uuid PRIMARY KEY,
    order_id          uuid           NOT NULL REFERENCES orders (id),
    attempt_no        smallint       NOT NULL,
    provider_id       uuid           NOT NULL REFERENCES providers (id),
    provider_sku      varchar(100)   NOT NULL,
    ref_id            varchar(64)    NOT NULL,
    provider_trx_id   varchar(128),
    cost_price        bigint         NOT NULL,
    status            attempt_status NOT NULL DEFAULT 'PENDING',
    error_code        varchar(64),
    vendor_error_code varchar(64),
    vendor_message    varchar(500),
    is_retryable      boolean,
    request_payload   jsonb,
    response_payload  jsonb,
    triggered_by      uuid REFERENCES users (id),
    latency_ms        integer,
    created_at        timestamptz    NOT NULL DEFAULT now(),
    updated_at        timestamptz    NOT NULL DEFAULT now(),
    CONSTRAINT order_attempts_ref_id_uq UNIQUE (ref_id),
    CONSTRAINT order_attempts_order_attempt_no_uq UNIQUE (order_id, attempt_no)
);
CREATE INDEX order_attempts_provider_created_at_idx ON order_attempts (provider_id, created_at);
CREATE INDEX order_attempts_provider_trx_idx ON order_attempts (provider_id, provider_trx_id);
CREATE INDEX order_attempts_status_created_at_idx ON order_attempts (status, created_at);
