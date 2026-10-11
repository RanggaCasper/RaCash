-- 0015_notification_init: log pengiriman email & WhatsApp (database.dbml § MODUL: notification)
-- delivery_status dibuat di 0014_reseller_init.

CREATE TYPE notification_channel AS ENUM ('EMAIL', 'WHATSAPP');

CREATE TABLE notification_logs (
    id                  uuid PRIMARY KEY,
    channel             notification_channel NOT NULL,
    recipient           varchar(254)         NOT NULL,
    template            varchar(64)          NOT NULL,
    subject             varchar(255),
    payload             jsonb                NOT NULL,
    related_type        varchar(32),
    related_id          uuid,
    status              delivery_status      NOT NULL DEFAULT 'QUEUED',
    attempts            smallint             NOT NULL DEFAULT 0,
    provider_message_id varchar(128),
    last_error          varchar(500),
    sent_at             timestamptz,
    created_at          timestamptz          NOT NULL DEFAULT now()
);
CREATE INDEX notification_logs_related_idx ON notification_logs (related_type, related_id);
CREATE INDEX notification_logs_status_created_at_idx ON notification_logs (status, created_at);
