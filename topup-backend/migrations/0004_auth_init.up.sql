-- 0004_auth_init: tabel modul auth (database.dbml § MODUL: auth)

CREATE TABLE sessions (
    id                  uuid PRIMARY KEY,
    user_id             uuid         NOT NULL REFERENCES users (id),
    refresh_token_hash  varchar(128) NOT NULL,
    previous_token_hash varchar(128),
    user_agent          varchar(512),
    ip                  inet,
    expires_at          timestamptz  NOT NULL,
    last_used_at        timestamptz,
    revoked_at          timestamptz,
    revoked_reason      varchar(64),
    created_at          timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT sessions_refresh_token_hash_uq UNIQUE (refresh_token_hash)
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE email_verifications (
    id         uuid PRIMARY KEY,
    user_id    uuid         NOT NULL REFERENCES users (id),
    token_hash varchar(128) NOT NULL,
    expires_at timestamptz  NOT NULL,
    used_at    timestamptz,
    created_at timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT email_verifications_token_hash_uq UNIQUE (token_hash)
);

CREATE TABLE password_resets (
    id         uuid PRIMARY KEY,
    user_id    uuid         NOT NULL REFERENCES users (id),
    token_hash varchar(128) NOT NULL,
    expires_at timestamptz  NOT NULL,
    used_at    timestamptz,
    ip         inet,
    created_at timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT password_resets_token_hash_uq UNIQUE (token_hash)
);

CREATE TABLE user_totp (
    user_id          uuid PRIMARY KEY REFERENCES users (id),
    secret_encrypted bytea       NOT NULL,
    secret_key_id    varchar(32) NOT NULL,
    confirmed_at     timestamptz,
    last_used_step   bigint,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE user_recovery_codes (
    id         uuid PRIMARY KEY,
    user_id    uuid         NOT NULL REFERENCES users (id),
    code_hash  varchar(128) NOT NULL,
    used_at    timestamptz,
    created_at timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX user_recovery_codes_user_id_idx ON user_recovery_codes (user_id);
