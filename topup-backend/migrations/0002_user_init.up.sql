-- 0002_user_init: tabel modul user (database.dbml § MODUL: user)

CREATE TYPE user_status AS ENUM ('PENDING_VERIFICATION', 'ACTIVE', 'SUSPENDED');

CREATE TABLE tiers (
    id                      uuid PRIMARY KEY,
    code                    varchar(32) NOT NULL,
    name                    varchar(64) NOT NULL,
    rank                    smallint    NOT NULL DEFAULT 0,
    is_default              boolean     NOT NULL DEFAULT false,
    auto_upgrade_min_volume bigint,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tiers_code_uq UNIQUE (code)
);
-- Hanya satu tier default untuk member baru.
CREATE UNIQUE INDEX tiers_default_uq ON tiers (is_default) WHERE is_default;

CREATE TABLE users (
    id                 uuid PRIMARY KEY,
    name               varchar(120) NOT NULL,
    email              varchar(254) NOT NULL,
    phone              varchar(20),
    password_hash      varchar(255) NOT NULL,
    tier_id            uuid         NOT NULL REFERENCES tiers (id),
    status             user_status  NOT NULL DEFAULT 'PENDING_VERIFICATION',
    balance            bigint       NOT NULL DEFAULT 0,
    perm_version       integer      NOT NULL DEFAULT 1,
    two_factor_enabled boolean      NOT NULL DEFAULT false,
    email_verified_at  timestamptz,
    last_login_at      timestamptz,
    suspended_reason   varchar(255),
    created_at         timestamptz  NOT NULL DEFAULT now(),
    updated_at         timestamptz  NOT NULL DEFAULT now(),
    deleted_at         timestamptz,
    CONSTRAINT users_email_uq UNIQUE (email),
    CONSTRAINT users_email_lower_chk CHECK (email = lower(email)),
    CONSTRAINT users_balance_chk CHECK (balance >= 0)
);
CREATE INDEX users_phone_idx ON users (phone);
CREATE INDEX users_tier_id_idx ON users (tier_id);
CREATE INDEX users_status_idx ON users (status);
CREATE INDEX users_created_at_idx ON users (created_at);

CREATE TABLE staff_invitations (
    id               uuid PRIMARY KEY,
    email            varchar(254) NOT NULL,
    role_ids         uuid[]       NOT NULL,
    token_hash       varchar(128) NOT NULL,
    invited_by       uuid         NOT NULL REFERENCES users (id),
    accepted_user_id uuid REFERENCES users (id),
    expires_at       timestamptz  NOT NULL,
    accepted_at      timestamptz,
    revoked_at       timestamptz,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT staff_invitations_token_hash_uq UNIQUE (token_hash)
);
CREATE INDEX staff_invitations_email_idx ON staff_invitations (email);
