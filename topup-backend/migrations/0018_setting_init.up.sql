-- 0018_setting_init: konfigurasi sistem key-value (database.dbml § MODUL: setting)

CREATE TABLE settings (
    key         varchar(100) PRIMARY KEY,
    value       jsonb        NOT NULL,
    is_secret   boolean      NOT NULL DEFAULT false,
    description varchar(255),
    updated_by  uuid REFERENCES users (id),
    updated_at  timestamptz  NOT NULL DEFAULT now()
);
