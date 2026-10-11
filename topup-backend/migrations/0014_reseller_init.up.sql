-- 0014_reseller_init: API key, pengaturan H2H, callback (database.dbml § MODUL: reseller)

CREATE TYPE delivery_status AS ENUM ('QUEUED', 'SENT', 'FAILED');

CREATE TABLE api_keys (
    id               uuid PRIMARY KEY,
    user_id          uuid         NOT NULL REFERENCES users (id),
    name             varchar(100),
    key_prefix       varchar(16)  NOT NULL,
    key_hash         varchar(128) NOT NULL,
    secret_encrypted bytea        NOT NULL,
    secret_key_id    varchar(32)  NOT NULL,
    last_used_at     timestamptz,
    last_used_ip     inet,
    revoked_at       timestamptz,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT api_keys_key_hash_uq UNIQUE (key_hash)
);
CREATE INDEX api_keys_user_id_idx ON api_keys (user_id);

CREATE TABLE reseller_settings (
    user_id                    uuid PRIMARY KEY REFERENCES users (id),
    callback_url               varchar(512),
    callback_secret_encrypted  bytea,
    callback_secret_key_id     varchar(32),
    callback_secret_rotated_at timestamptz,
    ip_whitelist               cidr[]      NOT NULL DEFAULT '{}',
    rate_limit_per_minute      integer     NOT NULL DEFAULT 600,
    is_h2h_enabled             boolean     NOT NULL DEFAULT false,
    updated_at                 timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE reseller_callbacks (
    id               uuid PRIMARY KEY,
    user_id          uuid            NOT NULL REFERENCES users (id),
    order_id         uuid            NOT NULL REFERENCES orders (id),
    event            varchar(32)     NOT NULL,
    event_id         varchar(64)     NOT NULL,
    url              varchar(512)    NOT NULL,
    payload          jsonb           NOT NULL,
    status           delivery_status NOT NULL DEFAULT 'QUEUED',
    attempts         smallint        NOT NULL DEFAULT 0,
    last_http_status smallint,
    last_error       varchar(500),
    next_retry_at    timestamptz,
    delivered_at     timestamptz,
    created_at       timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT reseller_callbacks_event_id_uq UNIQUE (event_id)
);
CREATE INDEX reseller_callbacks_user_created_at_idx ON reseller_callbacks (user_id, created_at);
CREATE INDEX reseller_callbacks_status_next_retry_idx ON reseller_callbacks (status, next_retry_at);
