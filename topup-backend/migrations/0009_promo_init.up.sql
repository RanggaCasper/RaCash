-- 0009_promo_init: kode promo & pemakaiannya (database.dbml § MODUL: promo)
-- FK promo_usages.order_id → orders ditambahkan di 0012_order_init.

CREATE TABLE promos (
    id              uuid PRIMARY KEY,
    code            varchar(32)   NOT NULL,
    name            varchar(120)  NOT NULL,
    type            amount_type   NOT NULL,
    value           numeric(12,3) NOT NULL,
    max_discount    bigint,
    min_transaction bigint        NOT NULL DEFAULT 0,
    quota           integer,
    used_count      integer       NOT NULL DEFAULT 0,
    per_user_limit  integer,
    rules           jsonb         NOT NULL DEFAULT '{}',
    starts_at       timestamptz   NOT NULL,
    ends_at         timestamptz   NOT NULL,
    is_active       boolean       NOT NULL DEFAULT true,
    created_by      uuid REFERENCES users (id),
    created_at      timestamptz   NOT NULL DEFAULT now(),
    updated_at      timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT promos_code_uq UNIQUE (code),
    CONSTRAINT promos_code_upper_chk CHECK (code = upper(code))
);

CREATE TABLE promo_usages (
    id           uuid PRIMARY KEY,
    promo_id     uuid        NOT NULL REFERENCES promos (id),
    order_id     uuid        NOT NULL,
    user_id      uuid REFERENCES users (id),
    contact_hash varchar(128),
    discount     bigint      NOT NULL,
    released_at  timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT promo_usages_order_id_uq UNIQUE (order_id)
);
CREATE INDEX promo_usages_promo_user_idx ON promo_usages (promo_id, user_id);
CREATE INDEX promo_usages_promo_contact_idx ON promo_usages (promo_id, contact_hash);
