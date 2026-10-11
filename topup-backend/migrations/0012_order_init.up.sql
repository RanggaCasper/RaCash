-- 0012_order_init: order & riwayat status (database.dbml § MODUL: order)

CREATE TYPE order_status AS ENUM ('PENDING_PAYMENT', 'PAID', 'PROCESSING', 'SUCCESS', 'FAILED', 'REFUNDED', 'EXPIRED');
CREATE TYPE order_source AS ENUM ('WEB', 'H2H');
CREATE TYPE refund_destination AS ENUM ('BALANCE', 'SOURCE', 'MANUAL');

CREATE TABLE orders (
    id                      uuid PRIMARY KEY,
    invoice_no              varchar(32)  NOT NULL,
    source                  order_source NOT NULL DEFAULT 'WEB',
    user_id                 uuid REFERENCES users (id),
    reseller_ref_id         varchar(64),
    product_id              uuid         NOT NULL REFERENCES products (id),
    product_snapshot        jsonb        NOT NULL,
    target_data             jsonb        NOT NULL,
    target_display          varchar(120) NOT NULL,
    contact_email           varchar(254),
    contact_whatsapp        varchar(20),
    tier_id                 uuid REFERENCES tiers (id),
    payment_method_id       uuid         NOT NULL REFERENCES payment_methods (id),
    promo_id                uuid REFERENCES promos (id),
    price                   bigint       NOT NULL,
    estimated_cost          bigint       NOT NULL,
    actual_cost             bigint,
    discount                bigint       NOT NULL DEFAULT 0,
    fee                     bigint       NOT NULL DEFAULT 0,
    total                   bigint       NOT NULL,
    status                  order_status NOT NULL DEFAULT 'PENDING_PAYMENT',
    serial_number_encrypted bytea,
    serial_number_key_id    varchar(32),
    delivery_message        varchar(500),
    failure_reason          varchar(255),
    refund_destination      refund_destination,
    version                 integer      NOT NULL DEFAULT 1,
    expires_at              timestamptz  NOT NULL,
    paid_at                 timestamptz,
    completed_at            timestamptz,
    created_at              timestamptz  NOT NULL DEFAULT now(),
    updated_at              timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT orders_invoice_no_uq UNIQUE (invoice_no),
    CONSTRAINT orders_total_chk CHECK (total = price - discount + fee)
);
-- ref_id reseller unik per reseller, hanya untuk order H2H.
CREATE UNIQUE INDEX orders_reseller_ref_uq ON orders (user_id, reseller_ref_id) WHERE source = 'H2H';
CREATE INDEX orders_user_created_at_idx ON orders (user_id, created_at);
CREATE INDEX orders_status_expires_at_idx ON orders (status, expires_at);
CREATE INDEX orders_status_updated_at_idx ON orders (status, updated_at);
CREATE INDEX orders_product_id_idx ON orders (product_id);
CREATE INDEX orders_created_at_idx ON orders (created_at);
CREATE INDEX orders_contact_email_idx ON orders (contact_email);
CREATE INDEX orders_contact_whatsapp_idx ON orders (contact_whatsapp);

ALTER TABLE payments
    ADD CONSTRAINT payments_order_id_fkey FOREIGN KEY (order_id) REFERENCES orders (id);
ALTER TABLE promo_usages
    ADD CONSTRAINT promo_usages_order_id_fkey FOREIGN KEY (order_id) REFERENCES orders (id);

CREATE TABLE order_status_histories (
    id          uuid PRIMARY KEY,
    order_id    uuid         NOT NULL REFERENCES orders (id),
    from_status order_status,
    to_status   order_status NOT NULL,
    actor_type  varchar(16)  NOT NULL,
    actor_id    uuid REFERENCES users (id),
    reason      varchar(255),
    created_at  timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT order_status_histories_actor_type_chk
        CHECK (actor_type IN ('system', 'user', 'admin', 'provider', 'gateway'))
);
CREATE INDEX order_status_histories_order_created_at_idx ON order_status_histories (order_id, created_at);
