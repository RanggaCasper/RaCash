# ARCHITECTURE — Website Top-Up Multi Provider & Multi Payment Gateway

| Item | Keterangan |
|---|---|
| Versi | 0.3 (Draft) |
| Tanggal | 10 Oktober 2026 |
| Dokumen terkait | `PRD.md` (v0.5), `API_CONTRACT.md` (v0.4), `database.dbml` (v0.2) |

---

## 1. Gambaran Umum

Sistem terdiri dari dua aplikasi utama dan dua proses pendukung:

| Komponen | Teknologi | Peran |
|---|---|---|
| **Frontend** | Nuxt 4.6.0 (Vue 3, TypeScript) | Storefront (SSR/SWR), area member, panel admin; Nitro sebagai BFF |
| **API** | Go, GoFiber, GORM | REST API, webhook PG, callback provider, API H2H reseller |
| **Worker** | Go (binary yang sama, entrypoint berbeda) | Fulfillment, polling status, sinkronisasi harga, rekonsiliasi, notifikasi, outbox relay |
| **Data** | PostgreSQL, Redis, object storage S3-compatible | Penyimpanan utama; cache, lock, rate limit, antrean (Asynq); file media |

Layanan eksternal: provider (Digiflazz), payment gateway (Xendit), email (SMTP), dan WhatsApp lewat [Kirimdev](https://kirimdev.com/) untuk notifikasi pembeli, OTP, dan alert admin.

```
                     ┌──────────────────────────────┐
  Browser ──HTTPS──▶ │  Reverse proxy (Caddy/Nginx) │
                     └──────┬───────────────┬───────┘
                            │ www.domain    │ api.domain
                            ▼               ▼
                   ┌────────────────┐   ┌───────────────────────┐
                   │ Nuxt 4 (Nitro) │──▶│  Go API (GoFiber)     │◀── Webhook PG
                   │ SSR + BFF      │   │  modular monolith     │◀── Callback provider
                   └────────────────┘   └──────────┬────────────┘◀── Reseller H2H
                                                   │
                         ┌─────────────────────────┼───────────────────┐
                         ▼                         ▼                   ▼
                   ┌──────────┐            ┌──────────────┐     ┌────────────┐
                   │PostgreSQL│◀───────────│ Go Worker    │────▶│ Redis      │
                   └──────────┘            │ (Asynq)      │     │ cache/queue│
                                           └──────┬───────┘     └────────────┘
                                                  ▼
                         Digiflazz · Xendit · Email · WhatsApp (Kirimdev)
```

Prinsip utama:
1. **Modular monolith** — satu repo & satu deployable backend, tetapi batas antar modul ditegakkan seketat microservice. Bila suatu saat perlu dipecah, modul bisa diangkat menjadi service tanpa bongkar ulang.
2. **Adapter untuk dunia luar** — setiap provider & PG adalah adapter di balik interface; alur inti tidak tahu vendor mana yang dipakai.
3. **Async untuk pekerjaan lambat** — webhook dibalas cepat; fulfillment, polling, dan notifikasi berjalan di worker.
4. **Kontrak dulu** — DBML → `API_CONTRACT.md`/`openapi.yaml` → implementasi; tipe frontend di-generate dari OpenAPI.

---

## 2. Backend — Modular Architecture

### 2.1 Daftar modul

| Modul | Tanggung jawab | Tabel yang dimiliki |
|---|---|---|
| `auth` | Registrasi, login, JWT, sesi & refresh token, verifikasi email, reset password, 2FA & step-up | `sessions`, `email_verifications`, `password_resets`, `user_totp`, `user_recovery_codes` |
| `user` | Profil user, status (aktif/suspend), tier, undangan staf | `users`, `tiers`, `staff_invitations` |
| `rbac` | Role, permission, assignment, `Authorizer`, middleware `Require`; sinkron tier saat role `reseller` di-assign/dicabut | `roles`, `permissions`, `role_permissions`, `user_roles` |
| `media` | Unggah, simpan ke object storage, pustaka media yang dipakai ulang | `media` |
| `catalog` | Kategori, brand, produk, banner, skema input per brand | `categories`, `brands`, `products`, `banners` |
| `provider` | Adapter supplier, routing per produk, mapping produk↔SKU provider, router & circuit breaker, sinkron harga, riwayat harga, saldo provider | `providers`, `product_routing`, `product_provider_map`, `provider_price_histories` |
| `pricing` | Hitung harga jual: modal + aturan margin + tier | `pricing_rules` |
| `promo` | Kode promo, validasi, kuota | `promos`, `promo_usages` |
| `payment` | Adapter PG, metode bayar, routing PG, charge, webhook PG | `payment_gateways`, `payment_methods`, `payment_method_gateway_map`, `payments` |
| `order` | Checkout, invoice, state machine order, riwayat status | `orders`, `order_status_histories` |
| `fulfillment` | Eksekusi pembelian ke provider, fallback, polling, callback provider | `order_attempts` |
| `wallet` | Saldo, ledger append-only, deposit, refund ke saldo | `balance_ledger`, `deposits` |
| `reseller` | API key, IP whitelist, signature HMAC, endpoint H2H, callback ke reseller | `api_keys`, `reseller_settings`, `reseller_callbacks` |
| `notification` | Email & WhatsApp (adapter Kirimdev), template, alert admin, retry kirim | `notification_logs` |
| `report` | Dashboard, laporan, export, rekonsiliasi | `reconciliation_runs`, `reconciliation_items` |
| `audit` | Pencatatan aksi admin & perubahan sensitif | `audit_logs` |
| `setting` | Konfigurasi sistem (key-value terenkripsi bila perlu) | `settings` |

Tabel infrastruktur milik `platform`: `webhook_events`, `outbox_events`, `idempotency_keys`.

### 2.2 Aturan batas modul

1. **Kepemilikan tabel** — hanya modul pemilik yang boleh query/menulis tabelnya. Modul lain meminta data lewat kontrak publik modul tersebut.
2. **Kontrak publik vs internal** — setiap modul punya package root (kontrak publik) dan sub-package `internal/`. Aturan `internal` milik Go memastikan modul lain **tidak bisa** meng-import implementasi (ditegakkan oleh compiler).
3. **Tanpa siklus dependensi** — arah dependensi mengikuti tabel 2.4. Kebutuhan arah sebaliknya diselesaikan dengan **domain event**.
4. **Foreign key lintas modul** — boleh ada di database untuk integritas, tetapi tidak ada relasi GORM (`Preload`/`Joins`) lintas modul.
5. **Pengecualian `report`** — boleh membaca lintas tabel secara **read-only** melalui view SQL khusus laporan (`report_*`), tidak boleh menulis.

### 2.3 Struktur folder

```
topup-backend/
├── cmd/
│   ├── api/main.go             # HTTP server
│   ├── worker/main.go          # Asynq worker + scheduler + outbox relay
│   └── migrate/main.go         # migrasi + seed (role, permission, kategori dasar)
├── internal/
│   ├── app/
│   │   ├── bootstrap.go        # wiring DI: platform → modul
│   │   ├── module.go           # interface Module
│   │   └── registry.go         # urutan registrasi modul
│   ├── platform/               # shared kernel (tanpa logika bisnis)
│   │   ├── config/  database/  cache/  queue/  logger/
│   │   ├── httpx/              # envelope respons, paginasi, error handler Fiber
│   │   ├── errs/               # tipe error domain + kode error
│   │   ├── event/              # bus, outbox writer, outbox relay
│   │   ├── crypto/             # AES-GCM, HMAC, hashing
│   │   ├── idempotency/  ratelimit/  validator/  clock/  money/
│   │   └── uow/                # unit of work (transaksi DB lintas repository)
│   └── modules/
│       ├── order/
│       │   ├── order.go        # kontrak publik: Service interface, DTO, error
│       │   ├── events.go       # event yang dipublikasikan modul
│       │   ├── module.go       # New(deps) → Module; RegisterRoutes/Jobs/Handlers
│       │   └── internal/
│       │       ├── domain/     # entity, state machine, aturan bisnis murni
│       │       ├── repository/ # GORM model + implementasi repository
│       │       ├── service/    # implementasi Service
│       │       └── handler/    # handler HTTP (Fiber) + DTO request/response
│       ├── provider/
│       │   ├── provider.go
│       │   ├── adapter.go      # interface Provider (adapter)
│       │   ├── module.go
│       │   ├── adapters/
│       │   │   ├── digiflazz/  # MVP
│       │   │   └── fake/       # adapter palsu untuk test & sandbox lokal
│       │   └── internal/ ...
│       ├── payment/            # struktur sama, adapters/{xendit,fake}
│       ├── notification/       # adapters/{smtp,kirimdev,fake}
│       └── ...                 # modul lain dengan pola yang sama
├── migrations/                 # SQL bernomor (golang-migrate), satu prefix per modul
├── api/openapi.yaml            # dihasilkan dari API_CONTRACT.md
├── docs/dbml/schema.dbml
├── deployments/                # Dockerfile, docker-compose, Caddyfile
└── test/                       # integration & e2e
```

### 2.4 Kontrak modul

```go
// internal/app/module.go
type Module interface {
    Name() string
    RegisterRoutes(r fiber.Router, mw Middlewares) // dipanggil di cmd/api
    RegisterJobs(mux *asynq.ServeMux)              // dipanggil di cmd/worker
    RegisterSubscribers(bus event.Bus)             // listener domain event
}

// Opsional: modul dengan job terjadwal (cron, UTC).
type Scheduled interface {
    RegisterSchedules(s *asynq.Scheduler) error
}
```

```go
// internal/modules/order/order.go — kontrak publik
package order

type Service interface {
    Create(ctx context.Context, in CreateInput) (OrderDTO, error)
    GetByInvoice(ctx context.Context, invoiceNo string) (OrderDTO, error)
    MarkProcessing(ctx context.Context, orderID string) error
    MarkSuccess(ctx context.Context, orderID string, r FulfillmentResult) error
    MarkFailed(ctx context.Context, orderID string, reason string) error
}
```

**Arah dependensi (A → B = A memanggil kontrak B):**

| Modul | Bergantung pada |
|---|---|
| `auth` | `user`, `rbac` |
| `rbac` | `user` |
| `media` | — |
| `catalog` | `media` |
| `provider` | `catalog` |
| `pricing` | `catalog`, `provider`, `user` |
| `payment` | `media` |
| `wallet` | `payment` |
| `order` | `catalog`, `pricing`, `promo`, `payment`, `wallet` |
| `fulfillment` | `order`, `provider` |
| `reseller` | `order`, `wallet`, `pricing`, `user` |
| `notification`, `audit` | — (hanya mendengarkan event) |
| `report` | read-only view |

Semua modul boleh bergantung pada `platform`. `platform` tidak boleh bergantung pada modul mana pun.

Pengecekan arah dependensi dijalankan di CI (mis. `go-arch-lint` atau test yang membaca `go list -deps`).

### 2.5 Komunikasi antar modul

| Pola | Dipakai untuk | Contoh |
|---|---|---|
| **Panggilan sinkron** via interface | Butuh jawaban langsung, dalam satu request | `order` → `pricing.Quote()` saat checkout |
| **Domain event** via outbox | Reaksi atas kejadian, arah dependensi terbalik, efek samping | `payment` menerbitkan `PaymentPaid` → `order` & `wallet` bereaksi |

**Outbox pattern:** event ditulis ke tabel `outbox_events` di transaksi DB yang sama dengan perubahan data. Relay di worker membaca outbox dan memasukkannya ke Asynq, lalu menandai terkirim. Konsumen wajib **idempoten** (kunci: `event_id`).

**Katalog event:**

| Event | Penerbit | Pendengar |
|---|---|---|
| `payment.paid` | payment | order (purpose=order), wallet (purpose=deposit). Bila order/deposit sudah `EXPIRED` → tidak diproses, terbitkan `payment.anomaly` |
| `payment.expired` | payment | order (→ `EXPIRED`), wallet (deposit → `EXPIRED`) |
| `payment.anomaly` | payment | notification (alert admin) |
| `order.paid` | order | fulfillment, notification |
| `order.succeeded` | order | notification, reseller (callback), report |
| `order.failed` | order | wallet (refund ke saldo, **hanya** bila order punya `user_id`), notification (guest: pesan untuk menghubungi CS dari `settings.cs_contact`), reseller |
| `order.refunded` | order | notification, reseller |
| `provider.balance_low` | provider | notification (alert admin) |
| `provider.circuit_opened` | provider | notification, catalog (nonaktifkan produk bila tak ada provider lain) |
| `rbac.permissions_changed` | rbac | auth (invalidasi cache), audit |
| `admin.action_performed` | semua modul (aksi admin) | audit |

### 2.6 Dependency injection

Wiring manual dan eksplisit di `internal/app/bootstrap.go` (tanpa container reflection), agar urutan dependensi mudah dibaca:

```go
func Bootstrap(cfg config.Config) (*App, error) {
    p := platform.New(cfg) // db, redis, logger, bus, queue, crypto

    userM    := user.New(p)
    rbacM    := rbac.New(p, userM.Service())
    authM    := auth.New(p, userM.Service(), rbacM.Authorizer())
    mediaM   := media.New(p)
    catalogM := catalog.New(p, mediaM.Service())
    provM    := provider.New(p, catalogM.Service(), provider.Adapters(cfg)...)
    pricingM := pricing.New(p, catalogM.Service(), provM.Service(), userM.Service())
    promoM   := promo.New(p)
    payM     := payment.New(p, mediaM.Service(), payment.Adapters(cfg)...)
    walletM  := wallet.New(p, payM.Service())
    orderM   := order.New(p, catalogM.Service(), pricingM.Service(), promoM.Service(),
                          payM.Service(), walletM.Service())
    fulfilM  := fulfillment.New(p, orderM.Service(), provM.Service())
    // reseller, notification, report, audit, setting ...

    return &App{Modules: []app.Module{userM, rbacM, authM, mediaM, catalogM, provM, pricingM,
        promoM, payM, walletM, orderM, fulfilM /* ... */}}, nil
}
```

`cmd/api` memanggil `RegisterRoutes` semua modul; `cmd/worker` memanggil `RegisterJobs` dan `RegisterSubscribers`.

### 2.7 Layer di dalam modul

```
handler  ──▶  service  ──▶  repository  ──▶  GORM / PostgreSQL
   │             │
   │             └──▶ kontrak modul lain, adapter, event bus
   └── parsing request, validasi, mapping DTO, RBAC (via middleware)
```

- **domain**: entity & aturan murni (mis. transisi status order) — tanpa GORM, tanpa Fiber, mudah di-unit-test.
- **repository**: interface di `service`, implementasi GORM di `repository`. Mendukung `uow.Tx` agar beberapa repository berjalan dalam satu transaksi.

**GORM wajib** untuk seluruh akses data aplikasi:
- Semua query & penulisan lewat `*gorm.DB`; dilarang memakai `database/sql`, `sqlx`, atau `pgx` langsung di kode modul.
- Transaksi lewat `db.Transaction(...)` yang dibungkus `uow.Tx`; repository menerima `*gorm.DB` dari konteks transaksi, bukan membuka koneksi sendiri.
- Row lock memakai `clause.Locking{Strength: "UPDATE"}` (setara `SELECT ... FOR UPDATE`); upsert memakai `clause.OnConflict`.
- Optimistic locking: `Where("version = ?", v).Updates(...)` lalu cek `RowsAffected == 0` → `VERSION_CONFLICT`.
- View laporan `report_*` dibaca lewat model GORM read-only (`TableName()` → nama view).
- `db.Raw`/`db.Exec` hanya untuk kasus yang tidak bisa diekspresikan builder GORM, wajib parameterized, dan diletakkan di repository.
- Skema tetap dikelola **migrasi SQL bernomor** (golang-migrate), karena partial index, `UNIQUE NULLS NOT DISTINCT`, CHECK, dan `GRANT` tidak bisa diekspresikan `AutoMigrate`. `AutoMigrate` dilarang di staging/produksi; model GORM harus sesuai dengan `database.dbml`.
- **service**: orkestrasi use case, transaksi, publikasi event.
- **handler**: tipis; tidak ada logika bisnis.

### 2.8 Adapter provider & payment gateway

```go
// provider/adapter.go
type Provider interface {
    Code() string
    FetchPriceList(ctx context.Context) ([]ProviderProduct, error)
    Purchase(ctx context.Context, req PurchaseRequest) (PurchaseResult, error)
    CheckStatus(ctx context.Context, refID string) (PurchaseResult, error)
    Balance(ctx context.Context) (int64, error)
    ParseCallback(h map[string]string, body []byte) (PurchaseResult, error)
}

// payment/adapter.go
type Gateway interface {
    Code() string
    SupportedMethods() []string
    CreateCharge(ctx context.Context, req ChargeRequest) (ChargeResult, error)
    CheckStatus(ctx context.Context, externalID string) (PaymentStatus, error)
    ParseWebhook(h map[string]string, body []byte) (WebhookEvent, error)
    WebhookAck(ok bool) (status int, body []byte) // format balasan tiap PG berbeda
    Refund(ctx context.Context, req RefundRequest) error
}
```

- Kredensial adapter disimpan terenkripsi di DB dan dimuat saat dipakai (bisa diganti dari panel tanpa restart; cache singkat di memori).
- Setiap adapter wajib lulus **contract test** bersama (suite yang sama dijalankan ke semua adapter memakai server HTTP tiruan).
- `PurchaseResult.Status` dinormalisasi ke `success | pending | failed`, plus `Retryable bool` dan `ErrorCode` internal; kode error asli vendor tetap disimpan.
- Circuit breaker & rate limit per adapter disimpan di Redis agar konsisten antar instance.

### 2.9 Alur kunci (sequence)

**Checkout → bayar → fulfillment**
```
Browser   Nuxt BFF      order        pricing/promo   payment       PG        worker/fulfillment   provider
  │ POST    │             │               │             │           │               │                │
  │────────▶│────────────▶│ Quote ───────▶│             │           │               │                │
  │         │             │◀──────────────│             │           │               │                │
  │         │             │ CreateCharge ──────────────▶│──────────▶│               │                │
  │         │             │ (tx: orders + payments)     │◀──────────│               │                │
  │◀────────│◀────────────│ invoice                      │           │               │                │
  │                                                     │◀─webhook──│               │                │
  │                                                     │ tx: payment PAID + outbox payment.paid      │
  │                       │◀────── payment.paid (event) │           │               │                │
  │                       │ tx: order PAID + outbox order.paid      │               │                │
  │                       │───────────────────────────── order.paid ──────────────▶│                │
  │                       │◀──────────── MarkProcessing ───────────────────────────│── Purchase ───▶│
  │                       │◀──────────── MarkSuccess / fallback / MarkFailed ──────│◀── result ─────│
```

**Penanganan webhook (PG maupun provider):**
1. Simpan raw ke `webhook_events` (unik `source + event_id`); jika duplikat → balas sukses tanpa proses ulang.
2. Verifikasi signature lewat adapter; gagal → simpan `signature_valid=false`, balas 401.
3. Bila nominal ≥ `settings.payment_recheck_threshold` (default Rp1.000.000), konfirmasi ulang lewat `CheckStatus` ke PG.
4. Ubah status + tulis outbox dalam satu transaksi; balas sesuai `WebhookAck` adapter.

**Konkurensi:** perubahan status order memakai optimistic locking (`version`); job fulfillment memakai lock Redis per `order_id` agar satu order tidak diproses dua worker.

### 2.10 RBAC di backend
Mengikuti PRD bagian 7: JWT berisi `sub`, `sid`, `pv` (perm_version); middleware `rbac.Load()` mengambil permission set dari Redis (`rbac:perm:{user_id}:{pv}`), `rbac.Require("orders:refund")` di setiap route, `rbac.StepUp()` untuk permission sensitif. Ownership & masking PII dikerjakan di service.

### 2.11 Job & penjadwalan (worker)

| Job | Pemicu | Keterangan |
|---|---|---|
| `fulfillment:execute` | event `order.paid` | Router provider + fallback ke provider lain (tidak mengulang provider yang sama) |
| `fulfillment:poll` | terjadwal tiap 1 menit | Order `PROCESSING` tanpa callback > N menit |
| `payment:poll` | terjadwal tiap 2 menit | Payment `PENDING` mendekati/lewat batas waktu |
| `order:expire` | terjadwal tiap 1 menit | Order & deposit lewat `expires_at` → `EXPIRED` (payment ikut `EXPIRED`; charge di PG sudah kedaluwarsa di waktu yang sama) |
| `provider:sync_prices` | terjadwal (mis. tiap 30 menit) + manual | Perbarui `product_provider_map` |
| `provider:check_balance` | terjadwal tiap 10 menit | Event `provider.balance_low` |
| `report:reconcile` | harian 02:00 WIB | Rekonsiliasi provider & PG |
| `notification:send` | event | Retry dengan backoff |
| `reseller:callback` | event | Kirim callback ke reseller, retry backoff |
| `outbox:relay` | loop terus-menerus | Outbox → Asynq |

### 2.12 Konfigurasi
Semua konfigurasi dari environment variable (12-factor), dibaca sekali saat startup ke struct `config.Config`. Rahasia (kunci enkripsi, JWT secret) tidak pernah disimpan di repo. Kredensial vendor disimpan terenkripsi di DB, bukan di env.

---

## 3. Frontend — Nuxt 4.6.0

### 3.1 Prinsip
- **Nuxt Layers** untuk memecah frontend sesuai area, sejalan dengan modularitas backend.
- **Struktur direktori Nuxt 4** (`app/` sebagai srcDir, `server/` dan `shared/` di root).
- **TypeScript strict**; tipe API di-generate dari `openapi.yaml` (`openapi-typescript`) ke `shared/types/api.d.ts`.
- **BFF**: browser tidak pernah memegang access/refresh token; token disimpan di cookie httpOnly dan diteruskan oleh Nitro ke Go API.

### 3.2 Struktur folder

```
topup-frontend/
├── nuxt.config.ts
├── app/                         # aplikasi root (app.vue, error.vue)
├── layers/
│   ├── core/                    # fondasi bersama
│   │   ├── nuxt.config.ts
│   │   └── app/
│   │       ├── components/ui/   # komponen dasar
│   │       ├── composables/     # useApi, useAuth, useCan, useMoney, useToast
│   │       ├── middleware/      # auth.ts, guest.ts, permission.ts
│   │       ├── plugins/         # api.ts ($api), can.ts (directive v-can)
│   │       ├── stores/          # Pinia: auth, permissions
│   │       └── utils/           # format rupiah, tanggal WIB, masking
│   ├── storefront/              # halaman publik: beranda, brand, checkout, invoice, lacak
│   ├── account/                 # /akun/**: dashboard member, riwayat, saldo, deposit, API key
│   └── admin/                   # /admin/**: seluruh panel admin
├── server/                      # Nitro = BFF
│   ├── api/
│   │   ├── auth/login.post.ts   # login → set cookie httpOnly
│   │   ├── auth/logout.post.ts
│   │   ├── auth/refresh.post.ts
│   │   └── [...path].ts         # proxy generik ke Go API + sisipkan Bearer token
│   ├── middleware/csrf.ts       # cek Origin untuk method non-GET
│   └── utils/session.ts         # baca/tulis cookie, refresh otomatis
├── shared/
│   └── types/api.d.ts           # hasil generate dari openapi.yaml
└── public/
```

Stack pendukung (rekomendasi): Pinia, Tailwind CSS + Nuxt UI, VeeValidate + Zod, `@nuxt/image`, `@nuxtjs/i18n` (opsional), Vitest + `@nuxt/test-utils`, Playwright untuk e2e.

### 3.3 Strategi rendering (`routeRules`)

| Rute | Mode | Alasan |
|---|---|---|
| `/` | SSR + SWR 60 dtk | SEO, konten berubah pelan |
| `/g/:brand` (halaman produk game) | SSR + SWR 300 dtk; harga diambil ulang di client | SEO; harga harus segar saat checkout |
| `/invoice/:no`, `/lacak` | SSR tanpa cache | Data privat per transaksi |
| `/akun/**` | SSR tanpa cache, butuh login | Data privat |
| `/admin/**` | `ssr: false` (SPA) | Tidak perlu SEO, interaksi berat |
| `/api/**` | Nitro handler (BFF) | Proxy ke Go API |

### 3.4 Alur autentikasi (BFF)

```
Browser ──POST /api/auth/login──▶ Nitro ──POST /api/v1/auth/login──▶ Go API
        ◀── Set-Cookie: at (httpOnly, 15m), rt (httpOnly, path=/api/auth, 30 hari) ──
Browser ──GET /api/me/orders (cookie)──▶ Nitro ──Authorization: Bearer at──▶ Go API
                                         └─ jika 401 TOKEN_EXPIRED → refresh sekali → ulangi
```
- Cookie: `HttpOnly`, `Secure`, `SameSite=Lax`.
- CSRF: request non-GET ke `/api/**` wajib `Origin` sama dengan domain situs.
- Endpoint publik (katalog) saat SSR dipanggil Nitro langsung ke Go API melalui jaringan internal.
- Webhook PG, callback provider, dan API H2H **tidak** lewat Nuxt — langsung ke `api.domain`.

### 3.5 RBAC di frontend
- Setelah login, `GET /api/v1/me/permissions` disimpan di store `permissions`.
- Composable `useCan('orders:refund')` dan directive `v-can="'orders:refund'"` untuk menyembunyikan tombol/menu.
- Route middleware `permission` membaca `definePageMeta({ permission: 'orders:read' })`.
- Jika API mengembalikan `403 STEP_UP_REQUIRED`, tampilkan modal TOTP lalu ulangi aksi.
- Frontend hanya untuk UX; keputusan akhir selalu di backend.

### 3.6 Penanganan data & state
- `useApi()` membungkus `$fetch`/`useFetch` dengan base `/api`, pemetaan envelope (`data`, `meta`, `error`), dan toast error standar per kode error.
- Uang selalu `number` dalam rupiah bulat (sesuai kontrak); format tampilan di `utils`.
- Waktu dari API dalam UTC; ditampilkan dalam WIB (`Asia/Jakarta`).
- Status invoice di halaman invoice diperbarui dengan polling ringan (mis. tiap 5 detik, berhenti di status final).

---

## 4. Data

- PostgreSQL tunggal; migrasi bernomor urut global dengan nama modul di nama file: `NNNN_<modul>_<deskripsi>` (mis. `0001_platform_init`, `0002_user_init`). Buat lewat `go run ./cmd/migrate create <modul>_<deskripsi>`.
- Primary key `uuid` (v7, terurut waktu) untuk entitas yang terekspos ke publik; nomor invoice terpisah (`RC-YYYYMMDD-XXXXXX`).
- Gambar disimpan di object storage S3-compatible; metadata di tabel `media` dan direferensikan lewat `*_media_id` sehingga satu file bisa dipakai ulang. Respons API tetap mengirim URL (`icon_url`, dll.) hasil resolusi dari `media`.
- Uang disimpan `BIGINT` dalam rupiah (tanpa desimal).
- Waktu disimpan `timestamptz` (UTC).
- `balance_ledger` append-only; saldo di `users.balance` diperbarui dalam transaksi yang sama dengan penulisan ledger, dengan `SELECT ... FOR UPDATE`.
- Data sensitif (kredensial vendor, secret TOTP) dienkripsi AES-GCM; kunci dari env/secret manager, mendukung rotasi via `key_id`.

**Redis keyspace:**

| Prefix | Isi | TTL |
|---|---|---|
| `cache:catalog:*` | katalog & daftar produk | 5 menit / invalidasi saat update |
| `rbac:perm:{uid}:{pv}` | permission set user | 1 jam |
| `cb:{adapter}` | state circuit breaker | dinamis |
| `lock:order:{id}` | lock fulfillment | 2 menit |
| `rl:{scope}:{key}` | rate limit | sesuai jendela |
| `idem:{key}` | cache respons idempotency (cadangan tabel) | 24 jam |
| `asynq:*` | antrean | dikelola Asynq |

---

## 5. Keamanan (ringkas)

| Area | Penerapan |
|---|---|
| Autentikasi | JWT akses 15 menit (HS256/EdDSA), refresh token opaque dengan rotasi & deteksi reuse |
| Otorisasi | RBAC deny-by-default + ownership check di service |
| Webhook | Signature per adapter, IP allowlist, idempoten, re-check status untuk nominal besar |
| H2H | API key + HMAC-SHA256 + timestamp ±5 menit + IP whitelist |
| Input | Validasi DTO (`go-playground/validator`), batas ukuran body, sanitasi |
| Rate limit | Per IP & per user di Redis; captcha (mis. Turnstile) di checkout guest |
| Rahasia | Env/secret manager; kredensial vendor terenkripsi di DB |
| Audit | Semua aksi admin & perubahan RBAC tercatat |
| Header | HSTS, CSP, X-Content-Type-Options lewat reverse proxy & Nuxt |

---

## 6. Observability

- **Log**: JSON terstruktur (`slog`), field wajib `request_id`, `user_id`, `module`, `invoice_no` bila ada; data sensitif dimasking.
- **Trace**: OpenTelemetry di API & worker (span per panggilan adapter).
- **Metrik**: Prometheus — success rate & latensi per provider/PG, ukuran antrean, outbox lag, saldo provider.
- **Alert**: dikirim ke WhatsApp admin (`settings.admin_alert_whatsapp`) lewat Kirimdev melalui modul `notification`.
- **Health**: `GET /healthz` (liveness), `GET /readyz` (DB & Redis).

---

## 7. Pengujian

| Level | Cakupan |
|---|---|
| Unit | `domain` & `service` dengan repository/adapters tiruan |
| Contract test adapter | Suite bersama untuk semua adapter provider & PG |
| Integration | Repository + PostgreSQL/Redis nyata (testcontainers) |
| Arsitektur | Cek arah dependensi modul & larangan import `internal` |
| API | Validasi respons terhadap `openapi.yaml` |
| E2E | Playwright: checkout guest dengan adapter `fake`, panel admin per role |

---

## 8. Deployment

```
docker-compose (staging / produksi awal)
├── caddy        # TLS otomatis, www.domain → nuxt, api.domain → api
├── nuxt         # node .output/server/index.mjs
├── api          # /app/api
├── worker       # /app/worker (bisa >1 replika)
├── postgres
├── redis
└── minio        # object storage lokal/staging; produksi boleh S3/R2
```

- Build Go multi-stage → image distroless; `api` dan `worker` dari image yang sama dengan command berbeda.
- Migrasi dijalankan sebagai job (`/app/migrate up`) sebelum rollout.
- CI: lint (`golangci-lint`, ESLint), test, cek arsitektur, generate & diff `openapi.yaml`, build image.
- Lingkungan: `local` (adapter fake), `staging` (sandbox vendor), `production`.

---

## 9. Keputusan Arsitektur (ADR ringkas)

| # | Keputusan | Alasan | Alternatif yang ditolak |
|---|---|---|---|
| 1 | Modular monolith | Tim kecil, deploy sederhana, transaksi DB mudah; batas modul tetap tegas | Microservice (overhead operasional) |
| 2 | Outbox + Asynq untuk event | Konsisten dengan DB tanpa broker tambahan | Kafka/RabbitMQ (berlebihan di tahap awal) |
| 3 | Adapter per vendor | Vendor bisa ditambah/diganti tanpa ubah alur inti | Integrasi langsung di service |
| 4 | Nuxt 4 + Layers | SSR/SWR untuk SEO storefront, admin SPA dalam satu codebase | Dua aplikasi frontend terpisah |
| 5 | BFF via Nitro | Token aman di cookie httpOnly, tidak terekspos ke JavaScript | Token di localStorage |
| 6 | JWT berisi `perm_version`, bukan daftar permission | Perubahan RBAC berlaku seketika, token kecil | Permission di dalam JWT |
| 7 | Uang sebagai integer rupiah | Hindari error pembulatan float | `decimal`/`float` |