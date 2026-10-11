-- 0007_provider_init: supplier, routing, mapping SKU, riwayat harga (database.dbml § MODUL: provider)

-- vendor_mode & circuit_state dipakai juga oleh payment_gateways (0010).
CREATE TYPE vendor_mode AS ENUM ('sandbox', 'production');
CREATE TYPE circuit_state AS ENUM ('CLOSED', 'OPEN', 'HALF_OPEN');
CREATE TYPE routing_strategy AS ENUM ('cheapest', 'priority', 'weighted');

CREATE TABLE providers (
    id                    uuid PRIMARY KEY,
    code                  varchar(50)   NOT NULL,
    name                  varchar(100)  NOT NULL,
    mode                  vendor_mode   NOT NULL DEFAULT 'sandbox',
    credentials_encrypted bytea,
    credentials_key_id    varchar(32),
    is_active             boolean       NOT NULL DEFAULT false,
    priority              integer       NOT NULL DEFAULT 100,
    balance               bigint,
    balance_min_alert     bigint        NOT NULL DEFAULT 0,
    balance_checked_at    timestamptz,
    circuit_state         circuit_state NOT NULL DEFAULT 'CLOSED',
    circuit_opened_at     timestamptz,
    last_price_sync_at    timestamptz,
    config                jsonb         NOT NULL DEFAULT '{}',
    created_at            timestamptz   NOT NULL DEFAULT now(),
    updated_at            timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT providers_code_uq UNIQUE (code)
);

CREATE TABLE product_routing (
    product_id   uuid PRIMARY KEY REFERENCES products (id),
    strategy     routing_strategy NOT NULL DEFAULT 'cheapest',
    max_attempts smallint         NOT NULL DEFAULT 3,
    updated_by   uuid REFERENCES users (id),
    updated_at   timestamptz      NOT NULL DEFAULT now(),
    CONSTRAINT product_routing_max_attempts_chk CHECK (max_attempts >= 1)
);

CREATE TABLE product_provider_map (
    id             uuid PRIMARY KEY,
    product_id     uuid         NOT NULL REFERENCES products (id),
    provider_id    uuid         NOT NULL REFERENCES providers (id),
    provider_sku   varchar(100) NOT NULL,
    cost_price     bigint,
    is_available   boolean      NOT NULL DEFAULT true,
    is_active      boolean      NOT NULL DEFAULT true,
    weight         integer,
    last_synced_at timestamptz,
    created_at     timestamptz  NOT NULL DEFAULT now(),
    updated_at     timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT product_provider_map_product_provider_uq UNIQUE (product_id, provider_id)
);
CREATE INDEX product_provider_map_provider_sku_idx ON product_provider_map (provider_id, provider_sku);

CREATE TABLE provider_price_histories (
    id                      uuid PRIMARY KEY,
    product_provider_map_id uuid        NOT NULL REFERENCES product_provider_map (id),
    old_cost_price          bigint,
    new_cost_price          bigint      NOT NULL,
    changed_at              timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX provider_price_histories_map_changed_at_idx ON provider_price_histories (product_provider_map_id, changed_at);
