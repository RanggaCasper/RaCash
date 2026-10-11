# API CONTRACT — RaCash

> Website top-up multi provider & multi payment gateway

| Item | Keterangan |
|---|---|
| Versi kontrak | v1 — 0.5 (Draft) |
| Tanggal | 10 Oktober 2026 |
| Dokumen terkait | `PRD.md` (v0.5), `ARCHITECTURE.md` (v0.4), `database.dbml` (v0.3) |
| Sumber turunan | `api/openapi.yaml` dibuat dari dokumen ini |

---

## 1. Konvensi Umum

### 1.1 Base URL & grup

| Grup | Prefix | Autentikasi | Konsumen |
|---|---|---|---|
| Publik & member | `/api/v1` | Tidak ada / Bearer JWT | Nuxt (SSR & BFF) |
| Admin | `/api/admin/v1` | Bearer JWT + RBAC | Nuxt panel admin (via BFF) |
| Reseller H2H | `/api/h2h/v1` | API key + HMAC | Server reseller |
| Webhook | `/api/v1/webhooks` | Signature vendor | Payment gateway, provider |

- Host API: `https://api.<domain>`.
- Browser tidak memanggil Go API langsung. Nuxt BFF memetakan:
  - `https://<domain>/api/<path>` → `https://api.<domain>/api/v1/<path>`
  - `https://<domain>/api/admin/<path>` → `https://api.<domain>/api/admin/v1/<path>`
- Versi ada di path. Perubahan yang merusak kompatibilitas → `v2`. Penambahan field baru **tidak** dianggap merusak; klien wajib mengabaikan field yang tidak dikenal.

### 1.2 Format data

| Aspek | Aturan |
|---|---|
| Content type | `application/json; charset=utf-8` (kecuali export file) |
| Penamaan field | `snake_case` |
| Uang | `integer` dalam rupiah bulat, contoh `15750` = Rp15.750 |
| Persentase | `number` desimal, contoh `2.5` = 2,5% |
| Waktu | string RFC 3339 UTC, contoh `2026-10-10T14:47:00Z` |
| ID | string UUID v7, contoh `0192f5a1-7c3e-7b2a-9d41-3f0c2b9a1e77` |
| Nomor invoice | `RC-YYYYMMDD-XXXXXX`, contoh `RC-20261010-7F3K9Q` |
| Nilai kosong | field tetap dikirim dengan `null`, kecuali dinyatakan opsional |
| Boolean | `true` / `false` |

### 1.3 Envelope respons

**Sukses — objek tunggal**
```json
{
  "success": true,
  "data": { "...": "..." },
  "request_id": "req_01JA7Z8Q2N4K"
}
```

**Sukses — daftar berhalaman**
```json
{
  "success": true,
  "data": [ { "...": "..." } ],
  "meta": { "page": 1, "per_page": 20, "total": 134, "total_pages": 7 },
  "request_id": "req_01JA7Z8Q2N4K"
}
```

**Gagal**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Data yang dikirim tidak valid.",
    "details": [
      { "field": "target.user_id", "code": "required", "message": "User ID wajib diisi." }
    ]
  },
  "request_id": "req_01JA7Z8Q2N4K"
}
```

- `message` berbahasa Indonesia dan aman ditampilkan ke pengguna.
- `details` hanya ada untuk error validasi atau error yang punya informasi tambahan (mis. `required_permission`).
- `request_id` juga dikirim di header `X-Request-Id`.

### 1.4 Kode error

| HTTP | `error.code` | Arti |
|---|---|---|
| 400 | `BAD_REQUEST` | Format request salah (JSON rusak, parameter tidak dikenal) |
| 401 | `UNAUTHENTICATED` | Belum login / token tidak valid |
| 401 | `TOKEN_EXPIRED` | Access token kedaluwarsa → lakukan refresh |
| 401 | `INVALID_CREDENTIALS` | Email/password salah |
| 401 | `INVALID_SIGNATURE` | Signature H2H/webhook tidak valid |
| 403 | `PERMISSION_DENIED` | Tidak punya permission; `details[0].required_permission` berisi nama permission |
| 403 | `STEP_UP_REQUIRED` | Aksi sensitif butuh verifikasi 2FA ulang |
| 403 | `ACCOUNT_SUSPENDED` | Akun dinonaktifkan |
| 403 | `EMAIL_NOT_VERIFIED` | Email belum diverifikasi; dikembalikan saat login sehingga seluruh fitur member tertutup sampai verifikasi |
| 403 | `IP_NOT_ALLOWED` | IP tidak ada di whitelist (H2H) |
| 404 | `NOT_FOUND` | Data tidak ada atau bukan milik pemanggil |
| 409 | `CONFLICT` | Bentrok data (mis. email sudah terdaftar, kode duplikat) |
| 409 | `INVALID_STATE_TRANSITION` | Aksi tidak sah untuk status saat ini |
| 409 | `IDEMPOTENCY_CONFLICT` | `Idempotency-Key` sama dipakai dengan body berbeda |
| 409 | `VERSION_CONFLICT` | Data sudah diubah pihak lain (optimistic lock) |
| 422 | `VALIDATION_ERROR` | Validasi field gagal |
| 422 | `PRODUCT_UNAVAILABLE` | Produk nonaktif / tidak ada provider tersedia |
| 422 | `PAYMENT_METHOD_UNAVAILABLE` | Metode bayar nonaktif atau di luar batas nominal |
| 422 | `PROMO_INVALID` | Kode promo tidak berlaku |
| 422 | `INSUFFICIENT_BALANCE` | Saldo tidak cukup |
| 422 | `ACCOUNT_NOT_FOUND` | ID game tidak ditemukan saat cek akun |
| 422 | `PRICE_CHANGED` | Harga berubah sejak quote; `details` berisi harga baru |
| 429 | `RATE_LIMITED` | Terlalu banyak request; lihat header `Retry-After` |
| 502 | `UPSTREAM_ERROR` | Vendor (provider/PG) gagal merespons |
| 503 | `SERVICE_UNAVAILABLE` | Sistem sedang maintenance |
| 500 | `INTERNAL_ERROR` | Kesalahan tak terduga |

### 1.5 Paginasi, urutan, filter

| Parameter | Default | Keterangan |
|---|---|---|
| `page` | `1` | Mulai dari 1 |
| `per_page` | `20` | Maksimum `100` |
| `sort` | per endpoint | Nama field; awalan `-` untuk menurun, contoh `-created_at` |
| `q` | — | Pencarian teks bebas (bila didukung) |
| `from`, `to` | — | Rentang waktu RFC 3339 (inklusif `from`, eksklusif `to`) |
| filter lain | — | Nama field langsung, contoh `status=PAID&provider_id=...` ; banyak nilai dipisah koma `status=PAID,PROCESSING` |

### 1.6 Header

| Header | Arah | Keterangan |
|---|---|---|
| `Authorization: Bearer <access_token>` | request | Endpoint ber-JWT |
| `Idempotency-Key` | request | UUID; **wajib** di endpoint yang ditandai 🔁. Disimpan 24 jam. Key sama + body sama → respons yang sama diputar ulang; body beda → `409 IDEMPOTENCY_CONFLICT` |
| `X-Step-Up-Token` | request | Token hasil `POST /auth/step-up`; wajib untuk endpoint bertanda ★ |
| `X-Captcha-Token` | request | Token captcha untuk checkout guest & auth publik |
| `Accept-Language` | request | `id` (default) atau `en` untuk `error.message` |
| `X-Request-Id` | dua arah | Klien boleh mengirim; server selalu mengembalikan |
| `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `Retry-After` | respons | Informasi rate limit |

### 1.7 Autentikasi & otorisasi

- **Access token**: JWT, berlaku 15 menit. Claim: `sub` (user_id), `sid` (session_id), `pv` (perm_version), `iat`, `exp`.
- **Refresh token**: string opaque, berlaku 30 hari, **dirotasi** setiap dipakai. Pemakaian ulang refresh token lama (sudah dirotasi) → sesi pemilik token itu dicabut, sehingga token terbarunya ikut tidak berlaku.
- **Permission**: endpoint admin mencantumkan permission yang dibutuhkan. Tanpa permission → `403 PERMISSION_DENIED`.
- **★ Sensitif**: wajib `X-Step-Up-Token` (berlaku 5 menit, hanya untuk user yang sama). Tanpa token → `403 STEP_UP_REQUIRED`.
- **Ownership**: endpoint `/me/**` hanya mengembalikan data milik pemanggil; data milik orang lain → `404 NOT_FOUND`.

### 1.8 Rate limit (default)

| Cakupan | Batas |
|---|---|
| Publik per IP | 120 request/menit |
| `POST /orders` per IP | 10 request/menit |
| `POST /auth/login`, `/auth/register`, `/auth/forgot-password` per IP | 5 request/menit |
| Member/admin per user | 300 request/menit |
| H2H per API key | 600 request/menit (dapat diatur per reseller) |

---

## 2. Enum

### 2.1 `order.status`
| Nilai | Final | Arti |
|---|:-:|---|
| `PENDING_PAYMENT` | – | Menunggu pembayaran |
| `PAID` | – | Pembayaran diterima, menunggu diproses |
| `PROCESSING` | – | Sedang diproses provider |
| `SUCCESS` | ✓ | Produk terkirim |
| `FAILED` | – | Gagal di semua provider, menunggu refund |
| `REFUNDED` | ✓ | Dana dikembalikan |
| `EXPIRED` | ✓ | Tidak dibayar sampai batas waktu |

### 2.2 Enum lain
| Enum | Nilai |
|---|---|
| `payment.status` | `PENDING`, `PAID`, `EXPIRED`, `FAILED`, `REFUNDED` |
| `payment.purpose` | `ORDER`, `DEPOSIT` |
| `payment_method.group` | `QRIS`, `VIRTUAL_ACCOUNT`, `EWALLET`, `RETAIL`, `BALANCE` |
| `payment_method.fee_bearer` | `CUSTOMER`, `MERCHANT`, `SPLIT` |
| `order_attempt.status` | `SUCCESS`, `PENDING`, `FAILED` |
| `ledger.type` | `DEPOSIT`, `PURCHASE`, `REFUND`, `ADJUSTMENT` |
| `user.status` | `ACTIVE`, `SUSPENDED`, `PENDING_VERIFICATION` |
| `role.scope` | `internal`, `customer` |
| `routing_strategy` | `cheapest`, `priority`, `weighted` |
| `vendor.mode` | `sandbox`, `production` |
| `margin.type` | `FLAT`, `PERCENT` |
| `promo.type` | `FLAT`, `PERCENT` |

---

## 3. Endpoint Publik (`/api/v1`)

### 3.1 Ringkasan
| Method | Path | Keterangan |
|---|---|---|
| GET | `/categories` | Daftar kategori aktif |
| GET | `/brands` | Daftar brand; filter `category`, `q`, `popular=true` |
| GET | `/brands/{slug}` | Detail brand + skema form input |
| GET | `/brands/{slug}/products` | Produk aktif + harga sesuai tier pemanggil |
| POST | `/accounts/check` | Cek nickname akun game |
| GET | `/payment-methods` | Metode bayar yang tersedia untuk nominal tertentu |
| POST | `/orders/quote` | Hitung total (produk + promo + fee) tanpa membuat order |
| POST | `/orders` 🔁 | Checkout / buat invoice |
| GET | `/orders/{invoice_no}` | Lacak status order (data pribadi tersamar) |
| GET | `/banners` | Banner beranda |

### 3.2 `GET /brands/{slug}`
**200**
```json
{
  "success": true,
  "data": {
    "id": "0192f5a1-...",
    "slug": "mobile-legends",
    "name": "Mobile Legends",
    "category": { "slug": "game", "name": "Game" },
    "icon_url": "https://cdn.<domain>/brands/mlbb.png",
    "description": "Top up diamond Mobile Legends proses instan.",
    "supports_account_check": true,
    "input_schema": [
      { "key": "user_id", "label": "User ID", "type": "text", "required": true, "pattern": "^[0-9]{5,12}$" },
      { "key": "zone_id", "label": "Zone ID", "type": "text", "required": true, "pattern": "^[0-9]{3,5}$" }
    ]
  },
  "request_id": "req_..."
}
```
`input_schema[].type`: `text`, `number`, `phone`, `select` (dengan `options: [{value,label}]`).
Semua field `*_url` gambar (`icon_url`, `banner_url`, `image_url`) adalah URL hasil resolusi dari tabel `media` (lihat §8.10).

### 3.3 `GET /brands/{slug}/products`
Harga mengikuti tier pemanggil (guest bila tanpa token).

**200**
```json
{
  "success": true,
  "data": [
    {
      "id": "0192f5a2-...",
      "sku": "MLBB-86",
      "name": "86 Diamonds",
      "price": 21500,
      "original_price": 23000,
      "is_available": true,
      "badge": "POPULER",
      "sort_order": 1
    }
  ],
  "request_id": "req_..."
}
```

### 3.4 `POST /accounts/check`
**Request**
```json
{ "brand_slug": "mobile-legends", "target": { "user_id": "12345678", "zone_id": "2001" } }
```
**200** `{ "success": true, "data": { "nickname": "Pl***er01" } }`
**422** `ACCOUNT_NOT_FOUND`

### 3.5 `GET /payment-methods?amount=21500`
**200**
```json
{
  "success": true,
  "data": [
    {
      "code": "QRIS",
      "name": "QRIS",
      "group": "QRIS",
      "icon_url": "https://cdn.<domain>/pay/qris.png",
      "fee": 152,
      "total": 21652,
      "min_amount": 1000,
      "max_amount": 10000000,
      "is_available": true
    },
    {
      "code": "BALANCE",
      "name": "Saldo Akun",
      "group": "BALANCE",
      "fee": 0,
      "total": 21500,
      "is_available": false,
      "unavailable_reason": "LOGIN_REQUIRED"
    }
  ]
}
```
`unavailable_reason`: `LOGIN_REQUIRED`, `INSUFFICIENT_BALANCE`, `OUT_OF_RANGE`, `MAINTENANCE`.

### 3.6 `POST /orders/quote`
**Request**
```json
{
  "product_id": "0192f5a2-...",
  "payment_method": "QRIS",
  "promo_code": "HEMAT10"
}
```
**200**
```json
{
  "success": true,
  "data": {
    "product": { "id": "0192f5a2-...", "name": "86 Diamonds", "price": 21500 },
    "discount": 2150,
    "fee": 137,
    "total": 19487,
    "promo": { "code": "HEMAT10", "applied": true, "message": "Diskon 10% maks Rp5.000" },
    "quote_token": "qt_01JA80...",
    "quote_expires_at": "2026-10-10T14:57:00Z"
  }
}
```
`quote_token` mengunci harga selama 10 menit dan dipakai saat checkout.

### 3.7 `POST /orders` 🔁
Header: `Idempotency-Key` (wajib), `X-Captcha-Token` (wajib untuk guest), `Authorization` (opsional).

**Request**
```json
{
  "product_id": "0192f5a2-...",
  "target": { "user_id": "12345678", "zone_id": "2001" },
  "payment_method": "QRIS",
  "promo_code": "HEMAT10",
  "quote_token": "qt_01JA80...",
  "contact": { "email": "pembeli@example.com", "whatsapp": "081234567890" }
}
```
Aturan:
- `contact.email` atau `contact.whatsapp` wajib minimal satu untuk guest; member boleh kosong (memakai data akun).
- Jika `quote_token` kedaluwarsa dan harga berubah → `422 PRICE_CHANGED` (dengan harga baru di `details`).
- `payment_method = BALANCE` hanya untuk user login; order langsung berstatus `PAID`.
- `payment.expires_at` = waktu buat + `expiry_minutes` metode bayar, dan dikirim ke PG sebagai expiry charge. Setelah lewat, PG menolak pembayaran dan order menjadi `EXPIRED`.

**201**
```json
{
  "success": true,
  "data": {
    "invoice_no": "RC-20261010-7F3K9Q",
    "status": "PENDING_PAYMENT",
    "product": { "brand": "Mobile Legends", "name": "86 Diamonds" },
    "target_display": "12345678 (2001)",
    "amount": { "price": 21500, "discount": 2150, "fee": 137, "total": 19487 },
    "payment": {
      "method": "QRIS",
      "status": "PENDING",
      "instructions": {
        "type": "QR",
        "qr_string": "00020101021226...",
        "qr_image_url": null,
        "va_number": null,
        "checkout_url": null,
        "steps": ["Buka aplikasi e-wallet/m-banking", "Scan QR", "Bayar"]
      },
      "expires_at": "2026-10-10T15:47:00Z"
    },
    "created_at": "2026-10-10T14:47:00Z",
    "tracking_url": "https://<domain>/invoice/RC-20261010-7F3K9Q"
  }
}
```
`instructions.type`: `QR`, `VA`, `REDIRECT` (e-wallet/retail via `checkout_url`), `CODE` (kode bayar retail).

### 3.8 `GET /orders/{invoice_no}`
Publik. Data pribadi disamarkan. Untuk melihat SN/kode voucher lengkap, pembeli guest wajib menyertakan `?contact=<email atau no WA saat checkout>`; tanpa itu `delivery.serial_number` disamarkan.

**200**
```json
{
  "success": true,
  "data": {
    "invoice_no": "RC-20261010-7F3K9Q",
    "status": "SUCCESS",
    "product": { "brand": "Mobile Legends", "name": "86 Diamonds" },
    "target_display": "1234****(2001)",
    "amount": { "price": 21500, "discount": 2150, "fee": 137, "total": 19487 },
    "payment": { "method": "QRIS", "status": "PAID", "paid_at": "2026-10-10T14:49:10Z" },
    "delivery": { "serial_number": "RCP-88****21", "message": "Diamond berhasil dikirim." },
    "timeline": [
      { "status": "PENDING_PAYMENT", "at": "2026-10-10T14:47:00Z" },
      { "status": "PAID", "at": "2026-10-10T14:49:10Z" },
      { "status": "PROCESSING", "at": "2026-10-10T14:49:12Z" },
      { "status": "SUCCESS", "at": "2026-10-10T14:49:25Z" }
    ]
  }
}
```

---

## 4. Autentikasi (`/api/v1/auth`)

| Method | Path | Keterangan |
|---|---|---|
| POST | `/auth/register` | Daftar member |
| POST | `/auth/verify-email` | Verifikasi email dengan token |
| POST | `/auth/resend-verification` | Kirim ulang email verifikasi |
| POST | `/auth/login` | Login email + password |
| POST | `/auth/login/2fa` | Lanjutan login bila 2FA aktif |
| POST | `/auth/refresh` | Tukar refresh token (rotasi) |
| POST | `/auth/logout` | Cabut sesi saat ini |
| POST | `/auth/logout-all` | Cabut semua sesi |
| POST | `/auth/forgot-password` | Kirim tautan reset |
| POST | `/auth/reset-password` | Set password baru |
| POST | `/auth/2fa/setup` | Buat secret TOTP (mengembalikan `otpauth_url`) |
| POST | `/auth/2fa/enable` | Aktifkan 2FA dengan kode pertama; kembalikan recovery codes |
| POST | `/auth/2fa/disable` ★ | Matikan 2FA |
| POST | `/auth/step-up` | Verifikasi TOTP → `step_up_token` |

### 4.0 Aturan umum endpoint auth publik

- **Captcha**: `register`, `resend-verification`, `login`, dan `forgot-password` wajib `X-Captcha-Token`. Tanpa token atau token ditolak → `422 VALIDATION_ERROR` dengan `details[].field = "X-Captcha-Token"` dan `code` `captcha_required` / `captcha_invalid`. Bila layanan captcha tidak dapat dihubungi → `503 SERVICE_UNAVAILABLE` (gagal tertutup).
- **Rate limit per IP** (per menit): `register` 5, `login` 10, `resend-verification` 5, `forgot-password` 5, `verify-email` 10, `reset-password` 10, `refresh` 30. Tambahan per email: `login` 10 percobaan / 15 menit (`429 RATE_LIMITED`); `resend-verification` dan `forgot-password` 3 permintaan / jam (kelebihannya tetap dibalas `200` tetapi tidak mengirim email).
- **Anti-enumerasi**: `resend-verification` dan `forgot-password` selalu `200 { "success": true, "data": null }`, apa pun status emailnya. Login dengan email tidak dikenal dan password salah mengembalikan `401 INVALID_CREDENTIALS` yang sama.
- **Token sekali pakai**: token verifikasi email (24 jam) dan reset password (1 jam) hanya disimpan sebagai hash. Token yang salah, kedaluwarsa, atau sudah dipakai → `422 VALIDATION_ERROR` dengan `details[].field = "token"`. Reset password berhasil mencabut seluruh sesi user.
- **Email**: dikirim modul `notification` setelah event `email_verification.requested` / `password_reset.requested` (ARCHITECTURE §2.5); endpoint tidak menunggu pengiriman.
- **Sementara**: login user yang `two_factor_enabled` atau ber-role internal belum dapat diselesaikan sebelum alur 2FA (§4, `/auth/login/2fa`, `/auth/2fa/*`) selesai; endpoint login menjawab `503 SERVICE_UNAVAILABLE` (gagal tertutup) alih-alih menerbitkan token penuh.

### 4.1 `POST /auth/register`
```json
{ "name": "Rangga", "email": "user@example.com", "phone": "081234567890", "password": "S3cure!pass" }
```
**201** `{ "success": true, "data": { "user_id": "...", "status": "PENDING_VERIFICATION" } }`
Password minimal 8 karakter, mengandung huruf dan angka. **409** bila email sudah terdaftar.

### 4.2 `POST /auth/login`
```json
{ "email": "user@example.com", "password": "S3cure!pass" }
```
**200 (tanpa 2FA)**
```json
{
  "success": true,
  "data": {
    "two_factor_required": false,
    "access_token": "eyJhbGciOi...",
    "access_token_expires_at": "2026-10-10T15:02:00Z",
    "refresh_token": "rt_9b1f...",
    "refresh_token_expires_at": "2026-11-09T14:47:00Z",
    "user": { "id": "...", "name": "Rangga", "email": "user@example.com", "tier": "MEMBER", "roles": ["member"] }
  }
}
```
**200 (2FA aktif)**
```json
{ "success": true, "data": { "two_factor_required": true, "challenge_token": "ch_4c2a...", "challenge_expires_at": "2026-10-10T14:52:00Z" } }
```
Lanjut ke `POST /auth/login/2fa` dengan `{ "challenge_token": "...", "code": "123456" }` → respons sama seperti login tanpa 2FA. Recovery code boleh dipakai di field `code`.

**403 `EMAIL_NOT_VERIFIED`** bila email belum diverifikasi (pakai `POST /auth/resend-verification`).
**200 dengan `two_factor_setup_required: true`** bila user ber-role internal belum mengaktifkan 2FA; token yang diberikan hanya bisa mengakses `/auth/2fa/*` sampai 2FA aktif.

### 4.3 `POST /auth/refresh`
```json
{ "refresh_token": "rt_9b1f..." }
```
**200** → token baru (format sama dengan login). Refresh token lama langsung tidak berlaku.

### 4.4 `POST /auth/step-up`
```json
{ "code": "123456" }
```
**200**
```json
{ "success": true, "data": { "step_up_token": "su_77ab...", "expires_at": "2026-10-10T14:52:00Z" } }
```

---

## 5. Member (`/api/v1/me`) — Bearer JWT

| Method | Path | Permission | Keterangan |
|---|---|---|---|
| GET | `/me` | — | Profil, tier, saldo, status 2FA |
| PATCH | `/me` | — | Ubah nama/no HP |
| PUT | `/me/password` | — | Ganti password (`current_password`, `new_password`) |
| GET | `/me/permissions` | — | Permission efektif & role |
| GET | `/me/sessions` | — | Daftar sesi aktif |
| DELETE | `/me/sessions/{id}` | — | Cabut sesi |
| GET | `/me/orders` | `self:orders` | Riwayat order; filter `status`, `from`, `to`, `q` |
| GET | `/me/orders/{invoice_no}` | `self:orders` | Detail order lengkap (SN tidak disamarkan) |
| GET | `/me/ledger` | `self:balance` | Mutasi saldo; filter `type`, `from`, `to` |
| POST | `/me/deposits` 🔁 | `self:deposit` | Buat deposit saldo |
| GET | `/me/deposits` | `self:deposit` | Riwayat deposit |
| GET | `/me/deposits/{id}` | `self:deposit` | Detail & instruksi bayar |

### 5.1 `GET /me/permissions`
```json
{
  "success": true,
  "data": {
    "roles": [ { "code": "admin", "name": "Admin" } ],
    "permissions": ["dashboard:view", "orders:read", "orders:retry", "orders:refund"],
    "perm_version": 12
  }
}
```
Untuk `super_admin`, `permissions` berisi `["*"]`.

### 5.2 `GET /me/ledger`
```json
{
  "success": true,
  "data": [
    {
      "id": "...",
      "type": "PURCHASE",
      "amount": -19487,
      "balance_after": 80513,
      "reference": { "type": "order", "id": "RC-20261010-7F3K9Q" },
      "description": "Pembelian 86 Diamonds",
      "created_at": "2026-10-10T14:47:00Z"
    }
  ],
  "meta": { "page": 1, "per_page": 20, "total": 1, "total_pages": 1 }
}
```

### 5.3 `POST /me/deposits` 🔁
```json
{ "amount": 100000, "payment_method": "VA_BCA" }
```
**201** → objek deposit dengan `payment.instructions` (format sama dengan order 3.7). Minimal deposit diatur di `settings` (default Rp10.000).

---

## 6. Reseller

### 6.1 Pengaturan reseller (`/api/v1/me`, butuh `h2h:access`)
| Method | Path | Keterangan |
|---|---|---|
| GET | `/me/api-keys` | Daftar API key (secret tidak pernah ditampilkan ulang) |
| POST | `/me/api-keys` ★ | Buat API key; `secret` hanya ditampilkan sekali |
| DELETE | `/me/api-keys/{id}` ★ | Cabut API key |
| GET | `/me/h2h-settings` | Callback URL & IP whitelist (callback secret tidak pernah ditampilkan) |
| PUT | `/me/h2h-settings` ★ | Ubah callback URL & IP whitelist |
| POST | `/me/h2h-settings/callback-secret` ★ | Buat/rotasi callback secret; nilai hanya ditampilkan sekali, secret lama langsung tidak berlaku |
| GET | `/me/h2h-callbacks` | Log pengiriman callback (status, percobaan) |

**`POST /me/api-keys` — 201**
```json
{ "success": true, "data": { "id": "...", "key": "pk_live_3f9a...", "secret": "sk_live_c81e...", "created_at": "..." } }
```

### 6.2 Autentikasi H2H (`/api/h2h/v1`)

| Header | Isi |
|---|---|
| `X-Api-Key` | API key (`pk_live_...`) |
| `X-Timestamp` | Unix epoch detik; selisih maksimum ±300 detik dari waktu server |
| `X-Signature` | `hex(HMAC_SHA256(secret, string_to_sign))` |

```
string_to_sign = METHOD + "\n" + PATH_WITH_QUERY + "\n" + X-Timestamp + "\n" + hex(SHA256(body))
```
Body kosong → SHA256 dari string kosong. Request dari IP di luar whitelist → `403 IP_NOT_ALLOWED`.

### 6.3 Endpoint H2H
| Method | Path | Keterangan |
|---|---|---|
| GET | `/products` | Semua produk aktif + harga reseller; filter `brand`, `category` |
| GET | `/balance` | Saldo reseller |
| POST | `/orders` 🔁 | Beli produk, dibayar dari saldo |
| GET | `/orders/{ref_id}` | Status berdasarkan `ref_id` reseller |

**`POST /orders` — Request**
```json
{
  "ref_id": "RS-000981",
  "sku": "MLBB-86",
  "target": { "user_id": "12345678", "zone_id": "2001" },
  "max_price": 21000
}
```
- `ref_id` unik per reseller; dipakai sebagai kunci idempotensi (`Idempotency-Key` opsional di H2H).
- `max_price` opsional; jika harga saat ini lebih tinggi → `422 PRICE_CHANGED`.

**202**
```json
{
  "success": true,
  "data": {
    "ref_id": "RS-000981",
    "invoice_no": "RC-20261010-K2M8PD",
    "status": "PROCESSING",
    "price": 20500,
    "balance_after": 479500,
    "serial_number": null,
    "message": null,
    "created_at": "2026-10-10T14:47:00Z"
  }
}
```

### 6.4 Callback ke reseller (keluar)
Dikirim ke `callback_url` reseller saat order mencapai `SUCCESS`, `FAILED`, atau `REFUNDED`.

`POST <callback_url>` dengan header `X-Timestamp`, `X-Signature`, `X-Event-Id`.
- `X-Signature` memakai rumus yang sama dengan §6.2, tetapi kuncinya adalah **callback secret** milik reseller (dibuat lewat `POST /me/h2h-settings/callback-secret`), bukan secret API key. Path yang dipakai = path dari `callback_url`.
- Callback hanya dikirim bila reseller sudah punya `callback_url` dan callback secret.
```json
{
  "event": "order.succeeded",
  "event_id": "evt_01JA81...",
  "data": {
    "ref_id": "RS-000981",
    "invoice_no": "RC-20261010-K2M8PD",
    "status": "SUCCESS",
    "sku": "MLBB-86",
    "price": 20500,
    "serial_number": "RCP-889120021",
    "message": "Diamond berhasil dikirim.",
    "updated_at": "2026-10-10T14:47:20Z"
  }
}
```
- Reseller wajib membalas HTTP 2xx dalam 10 detik.
- Retry: 1 m, 5 m, 15 m, 1 j, 6 j (maks. 5 kali). Reseller harus idempoten berdasarkan `event_id`.
- Nilai `event`: `order.succeeded`, `order.failed`, `order.refunded`.

---

## 7. Webhook Masuk (`/api/v1/webhooks`)

| Method | Path | Pengirim |
|---|---|---|
| POST | `/webhooks/payments/{gateway_code}` | Payment gateway, contoh `xendit` |
| POST | `/webhooks/providers/{provider_code}` | Provider, contoh `digiflazz` |

Aturan:
- Body & header **mengikuti format vendor masing-masing**; tidak memakai envelope §1.3. Adapter yang menerjemahkan.
- Signature diverifikasi adapter; gagal → `401` dan dicatat.
- IP pengirim dicek terhadap allowlist vendor (jika vendor menyediakan).
- Idempoten berdasarkan `source + event_id` vendor; duplikat langsung dibalas sukses.
- Balasan sukses mengikuti format yang diminta tiap vendor (`WebhookAck` adapter); default `200 {"success": true}`.
- Proses berat dilakukan asinkron; target waktu balas < 2 detik.
- Webhook PG dengan nominal ≥ `settings.payment_recheck_threshold` (default Rp1.000.000) dikonfirmasi ulang via `CheckStatus` sebelum ditandai `PAID`.
- Webhook `PAID` untuk order/deposit yang sudah `EXPIRED` tidak diproses; sistem mengirim alert ke admin.

---

## 8. Admin (`/api/admin/v1`) — Bearer JWT + RBAC

Kolom **Permission** adalah permission yang wajib dimiliki. ★ = butuh `X-Step-Up-Token`. 🔁 = wajib `Idempotency-Key`.

### 8.1 Dashboard & laporan
| Method | Path | Permission | Keterangan |
|---|---|---|---|
| GET | `/dashboard/summary` | `dashboard:view` | Jumlah transaksi, success rate, omzet; `profit` hanya muncul bila punya `dashboard:view_profit` |
| GET | `/dashboard/vendor-health` | `dashboard:view` | Success rate & latensi per provider/PG, status circuit breaker |
| GET | `/reports/sales` | `reports:read` | Penjualan per hari/kategori/brand/produk; param `group_by` |
| GET | `/reports/sales/export` | `reports:export` | File CSV/XLSX; param `format` |
| GET | `/reconciliations` | `reports:read` | Daftar hasil rekonsiliasi |
| POST | `/reconciliations` | `reconciliation:run` | Jalankan rekonsiliasi untuk tanggal tertentu (asinkron, `202`) |
| GET | `/reconciliations/{id}` | `reports:read` | Detail selisih |

**`GET /dashboard/summary?from=...&to=...` — 200**
```json
{
  "success": true,
  "data": {
    "orders_total": 1250,
    "orders_success": 1229,
    "success_rate": 98.32,
    "revenue": 31250000,
    "profit": 2150000,
    "pending_manual": 3,
    "by_status": { "SUCCESS": 1229, "FAILED": 4, "REFUNDED": 4, "PROCESSING": 2, "PENDING_PAYMENT": 11 }
  }
}
```

### 8.2 Katalog
| Method | Path | Permission |
|---|---|---|
| GET | `/categories` | `catalog:read` |
| POST | `/categories` | `catalog:write` |
| PATCH | `/categories/{id}` | `catalog:write` |
| DELETE | `/categories/{id}` | `catalog:write` |
| GET | `/brands` | `catalog:read` |
| POST | `/brands` | `catalog:write` |
| GET | `/brands/{id}` | `catalog:read` |
| PATCH | `/brands/{id}` | `catalog:write` |
| DELETE | `/brands/{id}` | `catalog:write` |
| GET | `/products` | `catalog:read` |
| POST | `/products` | `catalog:write` |
| GET | `/products/{id}` | `catalog:read` |
| PATCH | `/products/{id}` | `catalog:write` |
| DELETE | `/products/{id}` | `catalog:write` |
| PATCH | `/products/bulk` | `catalog:write` |
| GET | `/banners` | `catalog:read` |
| POST | `/banners` | `catalog:write` |
| PATCH | `/banners/{id}` | `catalog:write` |
| DELETE | `/banners/{id}` | `catalog:write` |
| GET | `/products/{id}/providers` | `providers:read` |
| PUT | `/products/{id}/providers` | `providers:write` |

**`PUT /products/{id}/providers`** — set ulang mapping produk ke provider
```json
{
  "routing_strategy": "priority",
  "max_attempts": 3,
  "mappings": [
    { "provider_id": "...", "provider_sku": "ML86", "weight": null, "is_active": true }
  ]
}
```
- Strategi `priority` mengurutkan provider berdasarkan `providers.priority` (global, diatur di §8.4); tidak ada prioritas per produk.
- `max_attempts` = jumlah maksimum provider **berbeda** yang dicoba; provider yang sama tidak dicoba ulang.

Untuk kategori dan brand, gambar dikirim sebagai referensi media, contoh `PATCH /brands/{id}`: `{ "icon_media_id": "...", "banner_media_id": "..." }`.
**200** mengembalikan mapping beserta `cost_price`, `is_available`, `last_synced_at` hasil sinkron terakhir.

### 8.3 Harga & promo
| Method | Path | Permission |
|---|---|---|
| GET | `/pricing-rules` | `pricing:read` |
| POST | `/pricing-rules` | `pricing:write` |
| PATCH | `/pricing-rules/{id}` | `pricing:write` |
| DELETE | `/pricing-rules/{id}` | `pricing:write` |
| POST | `/pricing/simulate` | `pricing:read` |
| GET | `/tiers` | `pricing:read` |
| PATCH | `/tiers/{id}` | `pricing:write` |
| GET | `/promos` | `promos:read` |
| POST | `/promos` | `promos:write` |
| PATCH | `/promos/{id}` | `promos:write` |
| DELETE | `/promos/{id}` | `promos:write` |

**`POST /pricing-rules`**
```json
{
  "scope": "brand",
  "scope_id": "0192f5a1-...",
  "tier_id": "...",
  "margin_type": "PERCENT",
  "margin_value": 4.5,
  "min_margin": 300,
  "is_active": true
}
```
`scope`: `global`, `category`, `brand`, `product`. `tier_id: null` = berlaku untuk semua tier.
Urutan resolusi: `product` → `brand` → `category` → `global`. Aturan `global` hanya dipakai bila kategori produk belum punya aturan.

### 8.4 Provider
| Method | Path | Permission | Keterangan |
|---|---|---|---|
| GET | `/providers` | `providers:read` | Daftar + saldo terakhir + status circuit breaker |
| POST | `/providers` | `providers:write` | Daftarkan provider (adapter harus sudah ada di kode) |
| GET | `/providers/{id}` | `providers:read` | |
| PATCH | `/providers/{id}` | `providers:write` | Aktif/nonaktif, prioritas global (dipakai strategi `priority`), mode |
| PUT | `/providers/{id}/credentials` ★ | `providers:manage_credentials` | Kredensial tidak pernah dikembalikan dalam respons |
| POST | `/providers/{id}/test-connection` | `providers:write` | Uji kredensial & cek saldo |
| POST | `/providers/{id}/sync-prices` | `catalog:sync` | Sinkron harga (asinkron, `202`) |
| POST | `/providers/{id}/reset-circuit` | `providers:write` | Tutup circuit breaker manual |

**`GET /providers` — item**
```json
{
  "id": "...",
  "code": "digiflazz",
  "name": "Digiflazz",
  "mode": "production",
  "is_active": true,
  "priority": 1,
  "balance": 4250000,
  "balance_min_alert": 1000000,
  "balance_checked_at": "2026-10-10T14:40:00Z",
  "circuit_state": "CLOSED",
  "success_rate_24h": 99.1,
  "credentials_configured": true
}
```
`circuit_state`: `CLOSED`, `OPEN`, `HALF_OPEN`.

### 8.5 Payment gateway & metode bayar
| Method | Path | Permission |
|---|---|---|
| GET | `/payment-gateways` | `payments:read` |
| POST | `/payment-gateways` | `payments:write` |
| PATCH | `/payment-gateways/{id}` | `payments:write` |
| PUT | `/payment-gateways/{id}/credentials` ★ | `payments:manage_credentials` |
| POST | `/payment-gateways/{id}/test-connection` | `payments:write` |
| GET | `/payment-methods` | `payments:read` |
| POST | `/payment-methods` | `payments:write` |
| PATCH | `/payment-methods/{id}` | `payments:write` |
| PUT | `/payment-methods/{id}/gateways` | `payments:write` |

**`PATCH /payment-methods/{id}`**
```json
{ "icon_media_id": "...", "fee_flat": 0, "fee_percent": 0.7, "fee_bearer": "SPLIT", "fee_customer_share": 50, "min_amount": 1000, "max_amount": 10000000, "expiry_minutes": 60, "is_active": true }
```
- `fee_customer_share` (0–100) wajib bila `fee_bearer = SPLIT`: persentase fee yang ditagihkan ke pembeli, sisanya ditanggung merchant. Diabaikan untuk `CUSTOMER` (100%) dan `MERCHANT` (0%).
- `expiry_minutes` adalah satu-satunya sumber masa berlaku invoice/deposit untuk metode ini.

### 8.6 Order
| Method | Path | Permission | Keterangan |
|---|---|---|---|
| GET | `/orders` | `orders:read` | Filter: `status`, `provider_id`, `payment_method`, `user_id`, `from`, `to`, `q` (invoice/target) |
| GET | `/orders/{id}` | `orders:read` | Detail + attempts + payment + timeline; PII disamarkan tanpa `customers:view_pii` |
| GET | `/orders/{id}/attempts` | `orders:read` | Raw request/response provider (sudah dimasking) |
| POST | `/orders/{id}/retry` 🔁 | `orders:retry` | Proses ulang dengan routing normal |
| POST | `/orders/{id}/switch-provider` 🔁 | `orders:switch_provider` | Proses ulang ke provider tertentu |
| POST | `/orders/{id}/mark` 🔁 ★ | `orders:mark_status` | Tandai `SUCCESS`/`FAILED` manual |
| POST | `/orders/{id}/refund` 🔁 ★ | `orders:refund` | Refund ke saldo (atau sumber dana bila didukung) |
| POST | `/orders/{id}/resend-notification` | `notifications:resend` | Kirim ulang notifikasi |
| GET | `/orders/export` | `reports:export` | CSV/XLSX |

**Aksi & status yang sah**

| Aksi | Status awal yang diizinkan |
|---|---|
| `retry` | `PAID`, `FAILED` (belum refund), `PROCESSING` yang macet > 15 menit |
| `switch-provider` | sama dengan `retry` |
| `mark` → `SUCCESS` | `PROCESSING`, `FAILED` |
| `mark` → `FAILED` | `PROCESSING` |
| `refund` | `FAILED` |

Transisi ini sama dengan state machine di PRD §4.2.

Selain itu → `409 INVALID_STATE_TRANSITION`.

**`POST /orders/{id}/switch-provider`**
```json
{ "provider_id": "...", "reason": "Digiflazz gangguan untuk MLBB", "version": 4 }
```

**`POST /orders/{id}/mark`**
```json
{ "status": "SUCCESS", "serial_number": "RCP-889120021", "reason": "Dicek manual di dashboard provider", "version": 5 }
```

**`POST /orders/{id}/refund`**
```json
{ "destination": "BALANCE", "reason": "Gagal di semua provider", "version": 6 }
```
`destination`: `BALANCE` (default), `SOURCE` (hanya jika PG mendukung), atau `MANUAL`. Guest tanpa akun → pembeli menghubungi admin, Finance mentransfer dana di luar sistem lalu mencatatnya dengan `destination: "MANUAL"` (wajib menyertakan `reason` berisi bukti/referensi transfer).

`version` = versi order yang terakhir dibaca klien; jika berbeda → `409 VERSION_CONFLICT`. Semua aksi di atas mewajibkan `reason` (min. 5 karakter) dan tercatat di audit log.

**`GET /orders/{id}` — 200 (ringkas)**
```json
{
  "success": true,
  "data": {
    "id": "...",
    "invoice_no": "RC-20261010-7F3K9Q",
    "status": "PROCESSING",
    "version": 4,
    "user": { "id": null, "type": "GUEST" },
    "contact": { "email": "pe****@example.com", "whatsapp": "0812****890" },
    "product": { "id": "...", "sku": "MLBB-86", "name": "86 Diamonds", "brand": "Mobile Legends" },
    "target": { "user_id": "12345678", "zone_id": "2001" },
    "amount": { "price": 21500, "cost": 20120, "discount": 2150, "fee": 137, "total": 19487, "margin": -770 },
    "payment": { "id": "...", "gateway": "xendit", "method": "QRIS", "status": "PAID", "external_id": "qr_1234...", "paid_at": "..." },
    "attempts": [
      { "id": "...", "provider": "digiflazz", "ref_id": "RC-20261010-7F3K9Q-1", "status": "PENDING", "error_code": null, "created_at": "..." }
    ],
    "available_actions": ["retry", "switch-provider", "mark", "resend-notification"],
    "timeline": [ { "status": "PAID", "at": "...", "actor": "system" } ]
  }
}
```
`available_actions` sudah difilter berdasarkan status **dan** permission pemanggil, sehingga frontend cukup merender dari daftar ini.

### 8.7 User & saldo
| Method | Path | Permission | Keterangan |
|---|---|---|---|
| GET | `/users` | `users:read` | Filter `role`, `tier`, `status`, `q` |
| GET | `/users/{id}` | `users:read` | |
| PATCH | `/users/{id}` | `users:write` | Nama, tier. Tier `RESELLER_*` hanya untuk pemegang role `reseller` (lainnya → `422 VALIDATION_ERROR`) |
| POST | `/users/{id}/suspend` | `users:suspend` | `{ "reason": "..." }` |
| POST | `/users/{id}/activate` | `users:suspend` | |
| POST | `/users/{id}/reset-2fa` ★ | `users:write` | |
| PUT | `/users/{id}/roles` ★ | `users:assign_role` | Set ulang daftar role |
| GET | `/users/{id}/ledger` | `balance:read` | |
| POST | `/users/{id}/balance-adjustments` 🔁 ★ | `balance:adjust` | Penyesuaian saldo manual |
| POST | `/staff` ★ | `users:write` + `users:assign_role` | Undang staf internal (email undangan) |

**`PUT /users/{id}/roles`**
```json
{ "role_ids": ["...operator", "...customer_service"], "reason": "Penempatan shift malam" }
```
Assign role `reseller` otomatis mengubah tier ke `RESELLER_SILVER` (bila belum ber-tier reseller); mencabutnya mengembalikan tier ke `MEMBER`. Penolakan anti-eskalasi → `403 PERMISSION_DENIED` dengan `details[0].code = "ESCALATION_BLOCKED"`. Mengubah role diri sendiri → `403` dengan `details[0].code = "SELF_ASSIGNMENT_BLOCKED"`. Mencabut Super Admin terakhir → `409 CONFLICT` dengan `details[0].code = "LAST_SUPER_ADMIN"`.

**`POST /users/{id}/balance-adjustments`**
```json
{ "amount": -50000, "reason": "Koreksi deposit ganda INV-...", "reference": "RC-20261009-AB12CD" }
```

### 8.8 RBAC
| Method | Path | Permission | Keterangan |
|---|---|---|---|
| GET | `/permissions` | `roles:read` | Katalog permission (dikelompokkan per resource) |
| GET | `/roles` | `roles:read` | Daftar role + jumlah pemegang |
| POST | `/roles` ★ | `roles:write` | Buat role kustom |
| GET | `/roles/{id}` | `roles:read` | Detail + permission |
| PATCH | `/roles/{id}` ★ | `roles:write` | Nama/deskripsi (role sistem: hanya deskripsi) |
| DELETE | `/roles/{id}` ★ | `roles:write` | Ditolak jika `is_system` atau masih dipakai |
| PUT | `/roles/{id}/permissions` ★ | `roles:write` | Set ulang permission role |
| GET | `/roles/{id}/users` | `roles:read` | Pemegang role |

**`GET /permissions` — 200**
```json
{
  "success": true,
  "data": [
    {
      "resource": "orders",
      "label": "Order",
      "permissions": [
        { "code": "orders:read", "description": "Lihat order", "is_sensitive": false },
        { "code": "orders:refund", "description": "Refund order", "is_sensitive": true }
      ]
    }
  ]
}
```

**`POST /roles`**
```json
{ "code": "cs_senior", "name": "CS Senior", "description": "CS dengan akses refund", "scope": "internal", "permissions": ["dashboard:view", "orders:read", "orders:retry", "orders:refund"] }
```
`code` hanya huruf kecil, angka, garis bawah; unik.

**`PUT /roles/{id}/permissions`**
```json
{ "permissions": ["dashboard:view", "orders:read", "orders:retry"], "reason": "Cabut akses refund" }
```
**200** → role terbaru beserta `affected_users` (jumlah user yang `perm_version`-nya dinaikkan).

### 8.9 Audit & pengaturan
| Method | Path | Permission | Keterangan |
|---|---|---|---|
| GET | `/audit-logs` | `audit:read` | Filter `actor_id`, `action`, `entity`, `entity_id`, `from`, `to` |
| GET | `/audit-logs/{id}` | `audit:read` | Dengan `before` & `after` |
| GET | `/settings` | `settings:write` | Pengaturan sistem (nilai rahasia disamarkan) |
| PATCH | `/settings` ★ | `settings:write` | Mis. `min_deposit`, `maintenance_mode`, `payment_recheck_threshold`, `cs_contact`, `admin_alert_whatsapp`, `kirimdev_api_key` (rahasia) |
| GET | `/notifications/logs` | `orders:read` | Log email/WA (Kirimdev) |

**Item audit log**
```json
{
  "id": "...",
  "actor": { "id": "...", "name": "Admin A", "roles": ["admin"] },
  "action": "order.refund",
  "entity": "order",
  "entity_id": "...",
  "reason": "Gagal di semua provider",
  "before": { "status": "FAILED" },
  "after": { "status": "REFUNDED" },
  "ip": "103.10.***.***",
  "user_agent": "Mozilla/5.0 ...",
  "created_at": "2026-10-10T14:55:00Z"
}
```

### 8.10 Media
| Method | Path | Permission | Keterangan |
|---|---|---|---|
| GET | `/media` | `media:read` | Pustaka media; filter `q`, `mime_type`; paginasi |
| POST | `/media` | `media:write` | Unggah gambar (`multipart/form-data`, field `file`, `alt_text` opsional). Maks. 2 MB; PNG/JPEG/WebP/SVG. File dengan checksum sama mengembalikan media yang sudah ada |
| GET | `/media/{id}` | `media:read` | Detail + daftar pemakaian (`usages`) |
| PATCH | `/media/{id}` | `media:write` | Ubah `alt_text` |
| DELETE | `/media/{id}` | `media:write` | Ditolak `409 CONFLICT` (`details[0].code = "MEDIA_IN_USE"`) bila masih dipakai |

**`POST /media` — 201**
```json
{
  "success": true,
  "data": {
    "id": "0192f5b0-...",
    "url": "https://cdn.<domain>/media/2026/10/mlbb-icon.webp",
    "mime_type": "image/webp",
    "size_bytes": 18342,
    "width": 256,
    "height": 256,
    "alt_text": "Ikon Mobile Legends",
    "created_at": "2026-10-10T14:47:00Z"
  }
}
```
`id` ini dipakai di field `*_media_id` (kategori, brand, banner, metode bayar).

---

## 9. Sistem

| Method | Path | Keterangan |
|---|---|---|
| GET | `/healthz` | Liveness — selalu `200` jika proses hidup |
| GET | `/readyz` | Readiness — cek PostgreSQL & Redis |
| GET | `/metrics` | Prometheus. Hanya dilayani di port internal `METRICS_ADDR` (default `:9090`), tidak di host API publik |

---

## 10. Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 0.1 | 2026-10-10 | Draft awal: publik, auth, member, reseller H2H, webhook, admin, RBAC |
| 0.2 | 2026-10-10 | Nama aplikasi RaCash; prefix invoice `RC-` |
| 0.3 | 2026-10-10 | Endpoint media (§8.10) & `*_media_id`; refund `MANUAL`; callback secret reseller; `fee_customer_share` & `expiry_minutes`; prioritas provider global (hapus `priority` per mapping); `EMAIL_NOT_VERIFIED` & wajib 2FA internal saat login; contoh vendor Xendit/Digiflazz; CRUD banner admin; `openapi.yaml` pertama dibuat |
| 0.4 | 2026-10-10 | `/metrics` dipindah ke port internal `METRICS_ADDR` |
| 0.5 | 2026-10-12 | Auth (§4): format error captcha, batas rate limit per endpoint, respons endpoint tanpa data, perilaku anti-enumerasi, event `email_verification.requested` & `password_reset.requested`, batas sementara 2FA |