# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project state

RaCash is a top-up website for digital products (game currency, vouchers, pulsa, data, PLN, e-wallet) that is multi-provider and multi-payment-gateway. The repository is still **spec-only**: everything lives in `docs/`, there is no application code yet, and there are no build, lint, or test commands. All docs are written in Indonesian; keep edits in Indonesian.

The planned code layout (`topup-backend/` in Go, `topup-frontend/` in Nuxt 4.6.0) is described in `docs/ARCHITECTURE.md` §2.3 and §3.2. Follow it when scaffolding begins.

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

## Backend architecture rules (for when code exists)

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
