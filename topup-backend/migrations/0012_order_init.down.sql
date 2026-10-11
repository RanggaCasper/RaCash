DROP TABLE IF EXISTS order_status_histories;
ALTER TABLE promo_usages DROP CONSTRAINT IF EXISTS promo_usages_order_id_fkey;
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_order_id_fkey;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS refund_destination;
DROP TYPE IF EXISTS order_source;
DROP TYPE IF EXISTS order_status;
