-- 0017_audit_init: log audit append-only (database.dbml § MODUL: audit)

CREATE TABLE audit_logs (
    id          uuid PRIMARY KEY,
    actor_id    uuid REFERENCES users (id),
    actor_roles varchar(64)[],
    action      varchar(64)  NOT NULL,
    entity      varchar(64)  NOT NULL,
    entity_id   varchar(64),
    reason      varchar(500),
    before      jsonb,
    after       jsonb,
    ip          inet,
    user_agent  varchar(512),
    request_id  varchar(64),
    created_at  timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_actor_created_at_idx ON audit_logs (actor_id, created_at);
CREATE INDEX audit_logs_entity_idx ON audit_logs (entity, entity_id);
CREATE INDEX audit_logs_action_created_at_idx ON audit_logs (action, created_at);
CREATE INDEX audit_logs_created_at_idx ON audit_logs (created_at);

-- Append-only (RULES §3.4). REVOKE berlaku untuk role aplikasi yang bukan
-- pemilik tabel; trigger menjaga juga pemilik/superuser.
REVOKE UPDATE, DELETE, TRUNCATE ON audit_logs FROM PUBLIC;

CREATE FUNCTION audit_logs_reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'tabel % bersifat append-only: % ditolak', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END;
$$;

CREATE TRIGGER audit_logs_append_only
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_reject_change();
CREATE TRIGGER audit_logs_no_truncate
    BEFORE TRUNCATE ON audit_logs
    FOR EACH STATEMENT EXECUTE FUNCTION audit_logs_reject_change();
