-- 0008_pricing_init: aturan margin (database.dbml § MODUL: pricing)

CREATE TYPE pricing_scope AS ENUM ('global', 'category', 'brand', 'product');
-- amount_type dipakai bersama oleh pricing_rules dan promos (0009).
CREATE TYPE amount_type AS ENUM ('FLAT', 'PERCENT');

CREATE TABLE pricing_rules (
    id           uuid PRIMARY KEY,
    scope        pricing_scope NOT NULL,
    scope_id     uuid,
    tier_id      uuid REFERENCES tiers (id),
    margin_type  amount_type   NOT NULL,
    margin_value numeric(12,3) NOT NULL,
    min_margin   bigint        NOT NULL DEFAULT 0,
    is_active    boolean       NOT NULL DEFAULT true,
    created_by   uuid REFERENCES users (id),
    created_at   timestamptz   NOT NULL DEFAULT now(),
    updated_at   timestamptz   NOT NULL DEFAULT now(),
    -- Satu aturan per scope+tier; NULL (global / semua tier) dianggap sama.
    CONSTRAINT pricing_rules_scope_tier_uq UNIQUE NULLS NOT DISTINCT (scope, scope_id, tier_id),
    -- scope_id polimorfik (tanpa FK): null hanya untuk scope global.
    CONSTRAINT pricing_rules_scope_id_chk CHECK ((scope = 'global') = (scope_id IS NULL))
);
