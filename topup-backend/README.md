# topup-backend

Backend RaCash: Go 1.25, GoFiber v2, GORM (PostgreSQL), Redis, dan Asynq. Arsitekturnya modular monolith. Desain lengkap ada di [`docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md), aturan wajibnya di [`docs/RULES.md`](../docs/RULES.md).

## Menjalankan

```bash
cp .env.example .env          # isi DB_HOST, DB_USER, DB_PASSWORD, DB_NAME, REDIS_URL
make keygen                   # salin 2 baris output (ENCRYPTION_KEYS, ENCRYPTION_ACTIVE_KEY_ID) ke .env
make migrate-up               # go run ./cmd/migrate up (migrasi + seed)
make run-api                  # :8080 (publik), :9090 (/metrics, internal)
make run-worker               # Asynq worker + scheduler + outbox relay
```

| Perintah | Keterangan |
|---|---|
| `go run ./cmd/migrate up` | Terapkan semua migrasi, lalu jalankan seed |
| `go run ./cmd/migrate seed` | Jalankan seed saja |
| `go run ./cmd/migrate down N` / `down all` | Batalkan N migrasi terakhir, atau semuanya |
| `go run ./cmd/migrate version` | Versi skema saat ini |
| `go run ./cmd/migrate force V` | Pulihkan state dirty |
| `go run ./cmd/migrate create <modul>_<deskripsi>` | Buat file migrasi bernomor berikutnya |
| `make keygen` / `go run ./cmd/keygen -id k1` | Buat kunci enkripsi AES-256 baru |
| `make keygen-rotate ID=k2` | Tambah kunci baru ke `ENCRYPTION_KEYS` di `.env` dan jadikan aktif (kunci lama tetap ada agar data lama bisa dibuka) |

Koneksi database memakai variabel terpisah: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, dan `DB_SSLMODE`. Password dengan karakter khusus tidak perlu di-encode.

Di `APP_ENV=local`, `migrate up`, API, dan worker otomatis membuat `DB_NAME` bila belum ada. Pengecekannya lewat database `DB_MAINTENANCE_NAME` (default `postgres`), dan `DB_USER` butuh hak `CREATEDB`. Di staging/produksi fitur ini mati, kecuali `DB_AUTO_CREATE=true`.

## Test

```bash
go test ./...                                   # unit + cek arsitektur
go test ./test/arch/...                         # hanya cek arsitektur
go test -run TestPercentOf ./internal/platform/money   # satu test
TEST_DB_HOST=localhost TEST_DB_USER=racash TEST_DB_PASSWORD=racash TEST_DB_NAME=racash_test \
  go test -count=1 ./test/integration/...
```

Integration test butuh PostgreSQL yang sudah dimigrasi dan di-seed, dan memakai variabel `TEST_DB_*` (sengaja terpisah dari `DB_*`, agar database development tidak terisi data test). Tanpa `TEST_DB_NAME`, test tersebut di-skip. Selain `test/integration`, setiap modul punya `internal/repository/model_test.go` yang memastikan model GORM sama dengan tabelnya (kolom dan nullability); test ini juga butuh `TEST_DB_*`.

## Skema & seed

- Satu migrasi `NNNN_<modul>_init` per modul (0002–0018), urut sesuai FK. FK yang menunjuk tabel modul yang dibuat belakangan (mis. `payments.order_id`) ditambahkan oleh migrasi modul tujuan.
- `balance_ledger` dan `audit_logs` append-only: `REVOKE UPDATE, DELETE, TRUNCATE ... FROM PUBLIC` ditambah trigger penolak, karena `REVOKE` tidak berlaku bagi pemilik tabel.
- Seed berasal dari kode dan idempoten: tier (`user/internal/domain`), katalog permission dan role sistem dengan matriks awal PRD §7.5 (`rbac/internal/domain`), dan kategori dasar (`catalog/internal/domain`). Seed hanya menambah yang belum ada. Pengecualiannya, deskripsi dan flag sensitif permission selalu mengikuti kode, dan `super_admin` selalu dilengkapi seluruh katalog. Matriks role lain hanya diterapkan saat role pertama kali dibuat.

## Struktur

```
cmd/{api,worker,migrate}   entrypoint
internal/app               kontrak Module, registry, bootstrap, server HTTP, worker
internal/platform          shared kernel tanpa logika bisnis
internal/modules/<modul>   modul domain; saat ini baru model GORM + seed (issue #4)
migrations/                SQL bernomor golang-migrate: NNNN_<modul>_<deskripsi>
test/arch                  cek arah dependensi, larangan import internal, GORM wajib
test/integration           test dengan PostgreSQL sungguhan
test/dbtest                helper test DB: koneksi TEST_DB_* dan cek model ↔ tabel
```

## Menambah modul

1. Buat `internal/modules/<nama>/`. Kontrak publik ada di root, implementasi di `internal/{domain,repository,service,handler}`.
2. Implementasikan `app.Module`; embed `app.Base` untuk hook yang tidak dipakai. Implementasikan juga `app.Scheduled` bila modul punya job cron.
3. Daftarkan modul di `internal/app/bootstrap.go` sesuai urutan dependensi.
4. Tambahkan modul ke `test/arch/rules.go` (`Allowed`) **dan** ke tabel ARCHITECTURE §2.4.
5. Repository memakai `uow.DBProvider`; service membungkus use case dengan `uow.Tx.Do` dan menerbitkan event lewat `event.Publisher` di dalam transaksi yang sama.

## Catatan platform

- **Error:** kembalikan `*errs.Error`. Error lain dirender sebagai `INTERNAL_ERROR` tanpa membocorkan pesan.
- **Idempotency:** pasang `mw.Idempotency(true)` di endpoint 🔁. Middleware auth mengisi `c.Locals("principal")` agar scope key per pemanggil.
- **Rate limit:** batas publik 120/menit per IP sudah terpasang di `/api/v1` (kecuali webhook). Batas khusus memakai `mw.RateLimitIP(...)`.
- **`/metrics`:** hanya tersedia di `METRICS_ADDR`, tidak di port publik.
