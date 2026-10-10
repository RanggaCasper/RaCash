-- 0001_platform_init: tabel infrastruktur milik platform (database.dbml § PLATFORM)

CREATE TYPE webhook_source_type AS ENUM ('PAYMENT_GATEWAY', 'PROVIDER');
CREATE TYPE outbox_status AS ENUM ('PENDING', 'PUBLISHED', 'FAILED');

CREATE TABLE webhook_events (
    id              uuid PRIMARY KEY,
    source_type     webhook_source_type NOT NULL,
    source_code     varchar(50)  NOT NULL,
    event_id        varchar(191) NOT NULL,
    headers         jsonb,
    payload         jsonb        NOT NULL,
    ip              inet,
    signature_valid boolean      NOT NULL,
    processed_at    timestamptz,
    process_error   varchar(500),
    received_at     timestamptz  NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX webhook_events_source_event_uq ON webhook_events (source_type, source_code, event_id);
CREATE INDEX webhook_events_received_at_idx ON webhook_events (received_at);

CREATE TABLE outbox_events (
    id             uuid PRIMARY KEY,
    module         varchar(32)   NOT NULL,
    event_type     varchar(64)   NOT NULL,
    aggregate_type varchar(32)   NOT NULL,
    aggregate_id   uuid          NOT NULL,
    payload        jsonb         NOT NULL,
    status         outbox_status NOT NULL DEFAULT 'PENDING',
    attempts       smallint      NOT NULL DEFAULT 0,
    last_error     varchar(500),
    available_at   timestamptz   NOT NULL DEFAULT now(),
    published_at   timestamptz,
    created_at     timestamptz   NOT NULL DEFAULT now()
);
CREATE INDEX outbox_events_pending_idx ON outbox_events (status, available_at) WHERE status = 'PENDING';
CREATE INDEX outbox_events_aggregate_idx ON outbox_events (aggregate_type, aggregate_id);

CREATE TABLE idempotency_keys (
    scope           varchar(64)  NOT NULL,
    key             varchar(128) NOT NULL,
    request_hash    varchar(128) NOT NULL,
    response_status smallint,
    response_body   jsonb,
    locked_until    timestamptz,
    expires_at      timestamptz  NOT NULL,
    created_at      timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (scope, key)
);
CREATE INDEX idempotency_keys_expires_at_idx ON idempotency_keys (expires_at);
