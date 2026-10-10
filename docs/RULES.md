# RULES — RaCash

| Item | Keterangan |
|---|---|
| Versi | 0.1 (Draft) |
| Tanggal | 10 Oktober 2026 |
| Berlaku untuk | Semua kode, migrasi, dokumen, dan review di proyek RaCash |
| Dokumen terkait | `PRD.md` (v0.5), `ARCHITECTURE.md` (v0.2), `API_CONTRACT.md` (v0.3), `database.dbml` (v0.2), `openapi.yaml` (0.3.0) |

Kata **WAJIB** / **DILARANG** bersifat mengikat; pelanggaran harus diperbaiki sebelum merge. **Sebaiknya** = default yang boleh disimpangi dengan alasan tertulis di PR. Aturan bertanda *(usulan)* belum diputuskan pemilik produk.

---

## 1. Dokumen & kontrak

1. Alur kerja **WAJIB**: `database.dbml` → `API_CONTRACT.md` → `openapi.yaml` → implementasi. Perubahan skema atau API dimulai dari dokumen, bukan dari kode.
2. Sumber kebenaran:
   - Skema database: `database.dbml`.
   - Perilaku API: `API_CONTRACT.md`. `openapi.yaml` adalah turunannya dan **WAJIB** diperbarui di PR yang sama.
   - Kebutuhan dan keputusan bisnis: `PRD.md` (log keputusan di §13).
3. Setiap perubahan dokumen yang berarti **WAJIB**:
   - menaikkan versi dokumen itu;
   - memperbarui rujukan versi di dokumen lain;
   - menambah baris changelog `API_CONTRACT.md` bila kontrak berubah.
4. `openapi.yaml` **WAJIB** lolos validasi OpenAPI 3.1 (`openapi-spec-validator` atau `redocly lint`) dan tidak boleh punya `$ref` putus.
5. Keputusan baru yang mengubah perilaku **WAJIB** dicatat di PRD §13. Keputusan tidak boleh hanya ada di chat atau di PR.
6. Bahasa dokumen: Indonesia. Identifier kode, nama tabel, enum, dan kode error: Inggris.

## 2. Konvensi umum

| Aspek | Aturan |
|---|---|
| Uang | `int64` / `BIGINT` rupiah bulat. **DILARANG** `float` untuk uang. Persentase: `numeric` di DB, `number` di JSON |
| Waktu | Disimpan `timestamptz` UTC; API RFC 3339 UTC; tampilan WIB (`Asia/Jakarta`). Ambil waktu lewat `platform/clock`, bukan `time.Now()` langsung, agar bisa diuji |
| ID | UUID v7 dibuat di aplikasi |
| Nomor invoice | `RC-YYYYMMDD-XXXXXX` |
| JSON | `snake_case`; field kosong dikirim `null` |
| Permission | `resource:action`, didefinisikan di kode dan di-seed. **DILARANG** dibuat lewat UI |
| Kode error | `UPPER_SNAKE_CASE`, hanya dari katalog `API_CONTRACT.md` §1.4. Kode baru **WAJIB** ditambahkan ke kontrak dulu |
| Event | `<aggregate>.<past_tense>`, mis. `order.succeeded`. Event baru **WAJIB** masuk katalog ARCHITECTURE §2.5 |

## 3. Backend (Go)

### 3.1 Batas modul
1. Satu modul = satu folder di `internal/modules/<nama>`, berisi kontrak publik di root dan implementasi di `internal/`.
2. Hanya modul pemilik yang boleh membaca/menulis tabelnya (ARCHITECTURE §2.1). Modul lain memakai kontrak publik pemilik.
3. Arah dependensi **WAJIB** mengikuti ARCHITECTURE §2.4. Kebutuhan arah sebaliknya diselesaikan dengan domain event lewat outbox. Pelanggaran ditangkap oleh cek arsitektur di CI.
4. **DILARANG** relasi GORM (`Preload`, `Joins`, association) lintas modul.
5. `platform/` **DILARANG** berisi logika bisnis atau bergantung pada modul.

### 3.2 Layer
- `domain`: aturan bisnis murni, tanpa import GORM/Fiber.
- `repository`: satu-satunya tempat GORM.
- `service`: orkestrasi, transaksi, publikasi event.
- `handler`: parsing, validasi, mapping DTO. Tanpa logika bisnis.
- Handler **DILARANG** memanggil repository langsung.
- Model GORM **DILARANG** keluar dari repository: map ke entity domain atau DTO.

### 3.3 GORM (WAJIB)
1. Seluruh akses data aplikasi **WAJIB** lewat GORM (`*gorm.DB`). **DILARANG** `database/sql`, `sqlx`, `pgx`, atau driver lain langsung di kode modul.
2. Transaksi lewat `uow.Tx` (membungkus `db.Transaction`). Repository menerima `*gorm.DB` dari konteks transaksi dan **DILARANG** membuka koneksi atau transaksi sendiri.
3. Row lock: `clause.Locking{Strength: "UPDATE"}`. Upsert: `clause.OnConflict`.
4. Optimistic locking: `Where("id = ? AND version = ?", id, v).Updates(...)`, lalu `RowsAffected == 0` dipetakan ke `VERSION_CONFLICT`.
5. `db.Raw`/`db.Exec` hanya bila builder GORM tidak bisa mengekspresikannya. Syaratnya: parameterized (tanpa string concat), berada di repository, dan diberi komentar alasan.
6. View `report_*` dibaca lewat model GORM read-only (`TableName()` = nama view).
7. Selalu `WithContext(ctx)`. Query di dalam loop (N+1) **DILARANG**; pakai `Where("id IN ?", ids)`.
8. `AutoMigrate` **DILARANG** di staging/produksi (lihat §4).
9. Soft delete hanya untuk tabel yang punya `deleted_at` di DBML.

### 3.4 Konsistensi & keandalan
1. Perubahan state dan penulisan `outbox_events` **WAJIB** dalam satu transaksi. **DILARANG** publish ke Asynq langsung dari service.
2. Konsumen event dan job **WAJIB** idempoten (kunci `event_id` / id entitas).
3. Saldo hanya berubah lewat `balance_ledger`, di transaksi yang sama dengan update `users.balance` ber-lock. `balance_ledger` dan `audit_logs` **DILARANG** di-UPDATE/DELETE.
4. Transisi status order **WAJIB** lewat state machine di `domain` (PRD §4.2). Setiap transisi menulis `order_status_histories`.
5. Satu provider dicoba maksimal sekali per order. `ref_id` unik per attempt. Respons ambigu (timeout) **WAJIB** dicek status dulu sebelum fallback.
6. Panggilan vendor **WAJIB** punya timeout dan melewati circuit breaker. **DILARANG** memanggil vendor di dalam transaksi DB.
7. Webhook dibalas cepat (< 2 detik). Proses berat dikerjakan di worker.

### 3.5 Adapter vendor
1. Setiap provider, PG, dan kanal notifikasi (SMTP, Kirimdev) adalah adapter di balik interface. Kode vendor **DILARANG** muncul di luar folder `adapters/`.
2. Setiap adapter **WAJIB** punya padanan `fake` dan lulus contract test bersama.
3. Status vendor dinormalisasi (`success | pending | failed` + `Retryable` + kode error internal). Kode dan pesan asli vendor disimpan, tetapi **DILARANG** ditampilkan ke pembeli.

### 3.6 Error, log, konfigurasi
1. Error domain memakai tipe di `platform/errs` yang dipetakan ke kode error kontrak. **DILARANG** mengirim pesan error internal atau vendor ke klien.
2. Log JSON (`slog`) dengan `request_id`, `module`, `user_id`, `invoice_no` bila ada. Data berikut **WAJIB** dimasking atau tidak dicatat: password, token, secret, kredensial vendor, SN/kode voucher, PII penuh.
3. Konfigurasi dari env (12-factor). Kredensial vendor disimpan terenkripsi di DB. Pengaturan bisnis disimpan di tabel `settings`. **DILARANG** hardcode nilai yang sudah punya key `settings`, misalnya `payment_recheck_threshold`, `cs_contact`, atau `min_deposit`.

## 4. Database & migrasi

1. Skema dikelola migrasi SQL bernomor (golang-migrate) dengan prefix modul (`0001_user_*`, `0002_rbac_*`, …). Setiap migrasi **WAJIB** punya `up` dan `down`.
2. Migrasi yang sudah ter-merge **DILARANG** diubah; buat migrasi baru.
3. Partial index, `UNIQUE NULLS NOT DISTINCT`, CHECK, dan `REVOKE UPDATE, DELETE` pada tabel append-only **WAJIB** ditulis di migrasi sesuai catatan di DBML.
4. Model GORM, migrasi, dan `database.dbml` **WAJIB** sinkron di PR yang sama.
5. Perubahan yang memblokir tabel besar (mis. tambah kolom `NOT NULL` tanpa default) **WAJIB** dipecah: tambah nullable → backfill → constraint.
6. FK lintas modul boleh ada di DB (integritas), tetapi tetap tanpa relasi GORM.
7. Gambar selalu lewat tabel `media` (`*_media_id`). **DILARANG** menambah kolom `*_url` gambar baru.

## 5. API

1. Semua respons memakai envelope kontrak (kecuali webhook vendor & export file). Daftar berhalaman memakai `meta`.
2. Setiap route admin/member **WAJIB** punya `rbac.Require(...)` sesuai kolom Permission di kontrak. Route tanpa permission eksplisit ditolak saat review.
3. Endpoint ★ **WAJIB** `StepUp` middleware. Endpoint 🔁 **WAJIB** `Idempotency-Key`.
4. Ownership dan masking PII dikerjakan di service. Data milik orang lain → `404`, bukan `403`.
5. Aksi admin yang mengubah order/saldo/role **WAJIB** meminta `reason` dan tercatat di `audit_logs`.
6. Breaking change hanya lewat versi path baru (`/v2`). Menambah field tidak dianggap breaking.
7. Respons API **WAJIB** divalidasi terhadap `openapi.yaml` di test API.

## 6. Frontend (Nuxt 4.6.0)

1. Browser **DILARANG** memegang access/refresh token. Semua panggilan lewat BFF Nitro (`/api/**`) dengan cookie httpOnly.
2. Tipe API **WAJIB** di-generate dari `openapi.yaml` ke `shared/types/api.d.ts`. **DILARANG** menulis tipe respons API secara manual.
3. Kode dipecah per Nuxt Layer (`core`, `storefront`, `account`, `admin`). Layer fitur hanya bergantung pada `core`.
4. Menu dan tombol memakai `useCan` / `v-can` dan `available_actions` dari API. Frontend tidak menentukan otorisasi.
5. Uang diformat lewat util `core` (rupiah); waktu ditampilkan WIB.
6. Mode render mengikuti `routeRules` di ARCHITECTURE §3.3; halaman privat tidak boleh di-cache.

## 7. Keamanan

1. Webhook: simpan raw ke `webhook_events`, verifikasi signature lewat adapter, idempoten per `source + event_id`, lalu cek ulang ke PG bila nominal ≥ `payment_recheck_threshold`.
2. Rahasia (kredensial vendor, secret TOTP, secret API key, callback secret, SN) disimpan terenkripsi AES-GCM dengan `*_key_id`. **DILARANG** dikembalikan di respons, kecuali sekali saat dibuat.
3. 2FA wajib untuk semua user ber-role `internal`. Permission sensitif memakai step-up.
4. Aturan anti-eskalasi RBAC (PRD §7.6) **WAJIB** ditegakkan di service dan diuji.
5. Input divalidasi di handler (validator). Batas ukuran body dan unggahan: media maks. 2 MB, tipe PNG/JPEG/WebP/SVG. SVG **WAJIB** disanitasi.

## 8. Pengujian

| Wajib ada untuk | Jenis test |
|---|---|
| Aturan `domain` (state machine, pricing, anti-eskalasi) | Unit test |
| Repository GORM | Integration test dengan PostgreSQL nyata (testcontainers), bukan mock DB |
| Setiap adapter vendor | Contract test bersama + adapter `fake` |
| Endpoint baru | API test yang memvalidasi respons terhadap `openapi.yaml`, termasuk kasus 401/403 |
| Alur kritis (checkout guest, refund, RBAC per role) | E2E Playwright memakai adapter `fake` |

- Bug fix **WAJIB** disertai test yang gagal sebelum perbaikan.
- **DILARANG** memanggil sandbox vendor sungguhan di CI.

## 9. Git & review *(usulan)*

1. Branch dari `main`: `feat/<topik>`, `fix/<topik>`, `docs/<topik>`, `chore/<topik>`.
2. Commit mengikuti Conventional Commits (`feat(order): ...`, `fix(wallet): ...`, `docs: ...`), dengan scope = nama modul.
3. PR kecil dan fokus. Deskripsi PR menyebut ID fitur PRD (mis. `PRV-03`) dan dokumen yang ikut berubah.
4. CI wajib hijau sebelum merge: `golangci-lint`, ESLint, test, cek arsitektur, validasi `openapi.yaml`, generate & diff tipe frontend.
5. Perubahan pada modul `wallet`, `payment`, `rbac`, `auth`, atau migrasi **WAJIB** di-review minimal satu orang selain penulis.
