-- 0005_media_init: pustaka media terpusat (database.dbml § MODUL: media)

CREATE TABLE media (
    id              uuid PRIMARY KEY,
    storage_key     varchar(512) NOT NULL,
    url             varchar(512) NOT NULL,
    original_name   varchar(255),
    mime_type       varchar(64)  NOT NULL,
    size_bytes      integer      NOT NULL,
    width           integer,
    height          integer,
    alt_text        varchar(255),
    checksum_sha256 varchar(64)  NOT NULL,
    uploaded_by     uuid REFERENCES users (id),
    created_at      timestamptz  NOT NULL DEFAULT now(),
    deleted_at      timestamptz,
    CONSTRAINT media_storage_key_uq UNIQUE (storage_key)
);
CREATE INDEX media_checksum_sha256_idx ON media (checksum_sha256);
CREATE INDEX media_created_at_idx ON media (created_at);
