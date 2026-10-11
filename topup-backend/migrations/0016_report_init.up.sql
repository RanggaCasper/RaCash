-- 0016_report_init: rekonsiliasi (database.dbml § MODUL: report)
-- View report_* dibuat bersama fitur laporan.

CREATE TYPE reconciliation_status AS ENUM ('RUNNING', 'COMPLETED', 'FAILED');
CREATE TYPE reconciliation_target AS ENUM ('PROVIDER', 'PAYMENT_GATEWAY');
CREATE TYPE reconciliation_item_type AS ENUM ('MISSING_INTERNAL', 'MISSING_EXTERNAL', 'AMOUNT_MISMATCH', 'STATUS_MISMATCH');

CREATE TABLE reconciliation_runs (
    id             uuid PRIMARY KEY,
    target_type    reconciliation_target NOT NULL,
    provider_id    uuid REFERENCES providers (id),
    gateway_id     uuid REFERENCES payment_gateways (id),
    period_date    date                  NOT NULL,
    status         reconciliation_status NOT NULL DEFAULT 'RUNNING',
    total_internal integer,
    total_external integer,
    mismatch_count integer,
    triggered_by   uuid REFERENCES users (id),
    started_at     timestamptz           NOT NULL DEFAULT now(),
    finished_at    timestamptz,
    error          varchar(500)
);
CREATE INDEX reconciliation_runs_target_period_idx ON reconciliation_runs (target_type, period_date);

CREATE TABLE reconciliation_items (
    id              uuid PRIMARY KEY,
    run_id          uuid                     NOT NULL REFERENCES reconciliation_runs (id),
    type            reconciliation_item_type NOT NULL,
    reference       varchar(128)             NOT NULL,
    order_id        uuid REFERENCES orders (id),
    internal_data   jsonb,
    external_data   jsonb,
    resolved_at     timestamptz,
    resolved_by     uuid REFERENCES users (id),
    resolution_note varchar(500)
);
CREATE INDEX reconciliation_items_run_id_idx ON reconciliation_items (run_id);
CREATE INDEX reconciliation_items_resolved_at_idx ON reconciliation_items (resolved_at);
