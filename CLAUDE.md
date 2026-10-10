# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project state

RaCash is a top-up website for digital products (game currency, vouchers, pulsa, data, PLN, e-wallet) that is multi-provider and multi-payment-gateway. Specs live in `docs/` and are written in Indonesian; keep doc edits in Indonesian. Work is tracked as GitHub issues grouped by milestone M0–M4 (PRD §11).

- `topup-backend/` (Go 1.25, Fiber v2, GORM, Redis, Asynq): the platform kernel and app wiring exist, but no domain modules yet. Its README covers setup and module conventions.
- `topup-frontend/` (Nuxt 4.6.0): not started. The planned layout is in ARCHITECTURE §3.2.

## Commands (run in `topup-backend/`)

```bash
go build ./... && go vet ./...
go test ./...                                          # unit + architecture check
go test -run TestPercentOf ./internal/platform/money   # single test
go test ./test/arch/...                                # module-boundary rules only
TEST_DB_HOST=... TEST_DB_USER=... TEST_DB_NAME=... go test -count=1 ./test/integration/...  # migrated Postgres; skipped without TEST_DB_NAME
go run ./cmd/migrate up | down N | version | force V | create <modul>_<deskripsi>
make keygen | make keygen-rotate ID=k2   # generate ENCRYPTION_KEYS (rotation keeps old keys)
go run ./cmd/api      # :8080 public, :9090 /metrics (internal only)
go run ./cmd/worker   # Asynq server + scheduler + outbox relay
```

The API and worker need these variables (see `.env.example`):
- Database: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`. These are separate variables, not a single URL; `config.Database.URL()` builds the escaped DSN.
- `REDIS_URL`, `ENCRYPTION_KEYS`, and `ENCRYPTION_ACTIVE_KEY_ID`.

Integration tests read the same fields with a `TEST_` prefix.

In `APP_ENV=local`, `migrate up`, the API, and the worker create `DB_NAME` if it is missing (`database.EnsureExists`, which goes through `DB_MAINTENANCE_NAME`). Elsewhere this only happens with `DB_AUTO_CREATE=true`. CI is `.github/workflows/backend.yml`: tidy check, vet, migrate up/down/up, and `go test -race` against Postgres 17.

## Documents and how they relate

The workflow is contract-first: **DBML → API contract → `openapi.yaml` → implementation**.

| File | Role |
|---|---|
| `docs/PRD.md` | Requirements (feature IDs like `PRV-03`, `RBAC-08`), state machine, RBAC matrix, release plan. §13 is the decision log: settled questions are struck through with **diputuskan/direvisi**. |
| `docs/ARCHITECTURE.md` | Modular monolith design, module→table ownership, module dependency direction, event catalog, worker jobs. |
| `docs/database.dbml` | Schema. It is the **source of truth** for tables and columns; PRD §6 is only a summary. |
| `docs/API_CONTRACT.md` | REST contract (envelope, error codes, endpoints, permissions per route). It is the source for `docs/openapi.yaml`. |
| `docs/openapi.yaml` | OpenAPI 3.1 derived from the contract. It uses vendor extensions `x-permission`, `x-sensitive` (step-up), and `x-idempotent`. Update it in the same change as API_CONTRACT and keep it passing `openapi-spec-validator`. |
| `docs/RULES.md` | Binding project rules (WAJIB/DILARANG): GORM, module boundaries, migrations, API, frontend, security, testing. Read it before writing code. |

Each doc has a version in its header, and the docs cite each other's versions. When you change a doc meaningfully, bump its version, update the version references in the other three, and add a row to the API_CONTRACT changelog when the contract changes. A decision usually touches several files: for example, adding a column means updating the DBML, the PRD §6 summary, the API payloads, and the ARCHITECTURE module table. After editing, grep across `docs/` for stale terms.

## Decisions that are easy to get wrong

- **MVP vendors:** Digiflazz is the only provider and Xendit is the only PG. Others (VIP Reseller, Midtrans, …) are post-MVP candidates. Use Digiflazz and Xendit in examples. The router and fallback are still built.
- **Fallback:** a provider is never retried for the same order. `product_routing.max_attempts` counts *distinct* providers, so with a single provider a failure goes straight to `FAILED`. The `priority` strategy uses the global `providers.priority`; there is no per-product priority.
- **Expiry:** `payment_methods.expiry_minutes` is the only TTL source, and the PG charge expires at the same moment. `EXPIRED` means payment failed: no late-payment handling. A `PAID` webhook after expiry raises an admin alert and is never fulfilled. The same applies to deposits.
- **Order state machine** (PRD §4.2): automatic transitions only move forward. Manual admin transitions (retry/switch from `FAILED`, mark) are listed explicitly and must match the table in API_CONTRACT §8.6.
- **Refunds:** members are refunded automatically to their balance. Guests (no balance) get a notification telling them to contact CS (`settings.cs_contact`); Finance then refunds outside the system and records it with `destination: MANUAL`.
- **Pricing resolution:** product → brand → category → global. The global rule applies only when the category has no rule.
- **Role vs tier:** role controls access and tier controls price. Assigning the `reseller` role sets tier `RESELLER_SILVER`; revoking it sets tier `MEMBER`. `RESELLER_*` tiers require the reseller role.
- **RBAC:**
  - Deny by default; permissions are `resource:action`, seeded from code and never created through the UI.
  - The JWT carries `perm_version`, not permissions.
  - Sensitive permissions (★) need a TOTP step-up token. 2FA is mandatory for internal roles from M0.
  - Anti-escalation rules are in PRD §7.6.
- **Media:** images are stored once in the `media` table (S3-compatible storage) and referenced through `*_media_id`. API responses return resolved `*_url` fields.
- **WhatsApp:** sent through the third-party gateway Kirimdev, behind a notification adapter. It is used for customer notifications, OTP, and admin alerts. There is no Telegram and no Baileys.
- **Money and time:** money is an integer number of rupiah (`bigint`). Time is stored as `timestamptz` in UTC and displayed in WIB. IDs are UUID v7. Invoice numbers use the format `RC-YYYYMMDD-XXXXXX`.

## Backend architecture rules

- **GORM is mandatory** for all data access (ARCHITECTURE §2.7):
  - No `database/sql`, `sqlx`, or `pgx` in module code.
  - Row locks use `clause.Locking`; optimistic locking checks `RowsAffected`.
  - `db.Raw`/`db.Exec` only as a parameterized last resort inside a repository.
  - The schema is still managed by numbered SQL migrations (golang-migrate). `AutoMigrate` is banned outside local.
- **Modular monolith:** each module owns its tables (ARCHITECTURE §2.1). Other modules go through the owner's public package; Go `internal/` blocks direct imports.
- **No GORM `Preload`/`Joins` across modules.** Dependency direction follows the table in §2.4; when a reverse dependency is needed, use a domain event through the transactional outbox.
- **Event consumers must be idempotent,** keyed on `event_id`.
- **The `report` module** may read across modules only through read-only `report_*` SQL views.
- **Vendors:** every provider/PG is an adapter behind an interface (§2.8), plus a `fake` adapter for local runs and E2E tests.
- **Webhooks:** stored in `webhook_events`, which is unique per `source + event_id`. Signatures are verified by the adapter. State changes and the outbox write happen in one transaction.
- **`balance_ledger` is append-only.** `users.balance` is updated in the same transaction with `SELECT ... FOR UPDATE`.

## Platform gotchas (`topup-backend/internal/platform`)

- **Adding a module:** add it to `Allowed` in `test/arch/rules.go` and to ARCHITECTURE §2.4, or the architecture test fails. Register it in `internal/app/bootstrap.go` in dependency order.
- **Transactions:** repositories take `uow.DBProvider` and call `DB(ctx)`. Services wrap a use case in `uow.Tx.Do`. `event.Outbox.Publish` returns `ErrNotInTx` outside a transaction, by design.
- **Errors:** handlers return `*errs.Error`, and any other error is rendered as `INTERNAL_ERROR` with the message hidden. `httpx.AccessLog` renders errors itself so that metrics see the final status. Keep the middleware order in `internal/app/http.go`: request id → metrics → access log → recover.
- **Idempotency:** `idempotency_keys.response_body` stores the body as a JSON *string* inside `jsonb`. `jsonb` reorders keys, and replays must be byte-identical. Only responses below 400 are stored; errors release the key.
- **Event delivery:** the outbox relay enqueues one Asynq task per subscriber (`event:<type>:<subscriber>`) with TaskID `<event_id>:<subscriber>` for dedupe.
