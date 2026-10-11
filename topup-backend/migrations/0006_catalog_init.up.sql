-- 0006_catalog_init: kategori, brand, produk, banner (database.dbml § MODUL: catalog)
-- Kategori dasar di-seed dari kode (cmd/migrate).

-- Tipe field brands.input_schema; divalidasi aplikasi karena disimpan di jsonb.
CREATE TYPE input_field_type AS ENUM ('text', 'number', 'phone', 'select');

CREATE TABLE categories (
    id            uuid PRIMARY KEY,
    slug          varchar(80)  NOT NULL,
    name          varchar(100) NOT NULL,
    icon_media_id uuid REFERENCES media (id),
    sort_order    integer      NOT NULL DEFAULT 0,
    is_active     boolean      NOT NULL DEFAULT true,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    updated_at    timestamptz  NOT NULL DEFAULT now(),
    deleted_at    timestamptz,
    CONSTRAINT categories_slug_uq UNIQUE (slug)
);

CREATE TABLE brands (
    id                     uuid PRIMARY KEY,
    category_id            uuid         NOT NULL REFERENCES categories (id),
    slug                   varchar(100) NOT NULL,
    name                   varchar(120) NOT NULL,
    icon_media_id          uuid REFERENCES media (id),
    banner_media_id        uuid REFERENCES media (id),
    description            text,
    input_schema           jsonb        NOT NULL DEFAULT '[]',
    supports_account_check boolean      NOT NULL DEFAULT false,
    is_popular             boolean      NOT NULL DEFAULT false,
    sort_order             integer      NOT NULL DEFAULT 0,
    is_active              boolean      NOT NULL DEFAULT true,
    seo_title              varchar(160),
    seo_description        varchar(320),
    created_at             timestamptz  NOT NULL DEFAULT now(),
    updated_at             timestamptz  NOT NULL DEFAULT now(),
    deleted_at             timestamptz,
    CONSTRAINT brands_slug_uq UNIQUE (slug)
);
CREATE INDEX brands_category_id_sort_order_idx ON brands (category_id, sort_order);

CREATE TABLE products (
    id             uuid PRIMARY KEY,
    brand_id       uuid         NOT NULL REFERENCES brands (id),
    sku            varchar(64)  NOT NULL,
    name           varchar(150) NOT NULL,
    description    text,
    badge          varchar(32),
    fixed_price    bigint,
    original_price bigint,
    sort_order     integer      NOT NULL DEFAULT 0,
    is_active      boolean      NOT NULL DEFAULT true,
    is_available   boolean      NOT NULL DEFAULT true,
    created_at     timestamptz  NOT NULL DEFAULT now(),
    updated_at     timestamptz  NOT NULL DEFAULT now(),
    deleted_at     timestamptz,
    CONSTRAINT products_sku_uq UNIQUE (sku)
);
CREATE INDEX products_brand_id_sort_order_idx ON products (brand_id, sort_order);

CREATE TABLE banners (
    id             uuid PRIMARY KEY,
    title          varchar(150) NOT NULL,
    image_media_id uuid         NOT NULL REFERENCES media (id),
    link_url       varchar(512),
    sort_order     integer      NOT NULL DEFAULT 0,
    starts_at      timestamptz,
    ends_at        timestamptz,
    is_active      boolean      NOT NULL DEFAULT true,
    created_at     timestamptz  NOT NULL DEFAULT now(),
    updated_at     timestamptz  NOT NULL DEFAULT now()
);
