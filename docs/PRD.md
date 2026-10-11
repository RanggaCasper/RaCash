# PRD — Website Top-Up Multi Provider & Multi Payment Gateway

| Item | Keterangan |
|---|---|
| Versi dokumen | 0.6 (Draft) — Panel admin dipisah menjadi aplikasi Nuxt tersendiri |
| Tanggal | 10 Oktober 2026 |
| Pemilik | Rangga |
| Status | Draft — menunggu review |

---

## 1. Ringkasan

Website top-up untuk produk digital (diamond/currency game, voucher game, pulsa, paket data, token PLN, e-wallet) yang **tidak terikat ke satu supplier maupun satu payment gateway**. Sistem memilih supplier (provider) terbaik per produk berdasarkan harga, stok, dan kesehatan provider, serta otomatis *fallback* ke provider lain jika transaksi gagal. Pembayaran dapat diproses lewat beberapa payment gateway (PG) sekaligus, dengan routing metode bayar ke PG yang paling murah/stabil.

### 1.1 Masalah yang diselesaikan
- Ketergantungan pada satu supplier → saat supplier gangguan, penjualan berhenti total.
- Harga modal berbeda antar supplier → margin tidak optimal bila hanya pakai satu.
- Fee & ketersediaan metode bayar berbeda antar PG → biaya transaksi tidak efisien.
- Proses manual (cek status, refund, rekonsiliasi) memakan waktu admin.

### 1.2 Tujuan
1. Transaksi berhasil (sukses terkirim) ≥ 98% dari transaksi yang sudah dibayar.
2. Waktu dari pembayaran terkonfirmasi hingga produk terkirim: median < 30 detik untuk produk instan.
3. Penambahan provider/PG baru cukup dengan menulis satu *adapter* tanpa mengubah alur inti.
4. Rekonsiliasi harian otomatis; selisih saldo/transaksi terdeteksi < 24 jam.

### 1.3 Non-goals (di luar cakupan v1)
- Aplikasi mobile native.
- Marketplace multi-seller (penjual pihak ketiga).
- Fitur kripto sebagai metode pembayaran.
- Program afiliasi berjenjang (MLM).

---

## 2. Persona

| Persona | Kebutuhan utama |
|---|---|
| **Pembeli tamu (guest)** | Top-up cepat tanpa daftar; cukup ID game + nomor WA/email untuk notifikasi. |
| **Member** | Riwayat transaksi, saldo deposit, harga member lebih murah, poin/promo. |
| **Reseller** | Harga khusus per tier, API H2H (host-to-host) untuk jualan ulang, laporan transaksi. |
| **Staf internal** | Operator, Finance, Customer Service — akses panel admin sesuai role (lihat bagian 7). |
| **Super Admin** | Kelola user internal, role & permission, kredensial provider & PG. |

> Persona di atas bukan role yang di-hardcode. Hak akses setiap user ditentukan oleh **role** yang ia miliki dalam sistem RBAC (bagian 7), sedangkan **tier** hanya menentukan harga.

---

## 3. Ruang Lingkup Fitur

Prioritas: **P0** = wajib di MVP, **P1** = rilis berikutnya, **P2** = nice to have.

### 3.1 Katalog & Produk
| ID | Fitur | Prioritas |
|---|---|---|
| CAT-01 | Kategori (Game, Voucher, Pulsa, Data, PLN, E-Wallet) & brand (mis. Mobile Legends, Free Fire, Genshin) | P0 |
| CAT-02 | Produk internal (SKU internal) yang dipetakan ke banyak SKU provider (`product_provider_map`) | P0 |
| CAT-03 | Form input dinamis per brand (User ID, Zone ID, Server, nomor HP) dengan validasi format | P0 |
| CAT-04 | Cek nickname/akun game sebelum bayar (jika provider mendukung) | P1 |
| CAT-05 | Sinkronisasi otomatis daftar harga & status produk dari provider (terjadwal + manual) | P0 |
| CAT-06 | Produk otomatis nonaktif jika semua provider yang dipetakan sedang gangguan/kosong | P0 |
| CAT-07 | Banner, produk populer, flash sale | P1 |
| CAT-08 | Pustaka media terpusat (tabel `media`): gambar diunggah sekali lalu dipakai ulang untuk ikon kategori/brand, banner, dan ikon metode bayar | P0 |

### 3.2 Harga & Margin
| ID | Fitur | Prioritas |
|---|---|---|
| PRC-01 | Harga jual = harga modal (provider terpilih) + margin (nominal / persen) | P0 |
| PRC-02 | Margin bertingkat: produk → brand → kategori → global (yang lebih spesifik menang). Margin default diatur per kategori; aturan global hanya dipakai bila kategori belum punya aturan | P0 |
| PRC-03 | Harga berbeda per tier user (Guest, Member, Reseller Silver/Gold/Platinum) | P0 |
| PRC-04 | Mode harga: **fixed** (admin kunci) atau **dinamis** (ikut modal termurah) | P1 |
| PRC-05 | Proteksi rugi: transaksi tidak diteruskan ke provider jika modal aktual > harga jual − batas minimum margin | P0 |
| PRC-06 | Kode promo/voucher diskon (kuota, periode, min. transaksi, per user) | P1 |

### 3.3 Checkout & Transaksi
| ID | Fitur | Prioritas |
|---|---|---|
| TRX-01 | Checkout guest dengan nomor WA/email | P0 |
| TRX-02 | Pemilihan metode bayar (VA, QRIS, e-wallet, retail, saldo deposit) dengan fee ditampilkan transparan | P0 |
| TRX-03 | Halaman invoice dengan countdown kedaluwarsa & instruksi pembayaran. Masa berlaku invoice = `expiry_minutes` metode bayar | P0 |
| TRX-04 | Lacak transaksi via nomor invoice (tanpa login) | P0 |
| TRX-05 | Notifikasi status via email & WhatsApp (WA lewat gateway pihak ketiga [Kirimdev](https://kirimdev.com/), di balik adapter agar bisa diganti) | P0 (email) / P1 (WA) |
| TRX-06 | Bayar pakai saldo deposit (member/reseller) | P0 |
| TRX-07 | Rate limit & captcha pada checkout untuk mencegah spam invoice | P0 |

### 3.4 Multi Provider (Supplier)
| ID | Fitur | Prioritas |
|---|---|---|
| PRV-01 | Abstraksi `Provider` interface; satu adapter per supplier | P0 |
| PRV-02 | Strategi routing per produk: `cheapest`, `priority` (urutan manual), `weighted` | P0 (cheapest, priority) / P1 (weighted) |
| PRV-03 | Fallback otomatis ke provider **berikutnya** bila gagal dengan error yang boleh di-retry. Error tidak pernah diulang ke provider yang sama; bila tidak ada provider lain → langsung `FAILED` | P0 |
| PRV-04 | Health check & circuit breaker per provider (auto-disable sementara bila error rate melewati ambang) | P0 |
| PRV-05 | Monitoring saldo deposit di tiap provider + alert saldo minimum | P0 |
| PRV-06 | Penerimaan callback/webhook provider + polling status sebagai cadangan | P0 |
| PRV-07 | Penyimpanan raw request/response provider untuk audit (dengan masking data sensitif) | P0 |

Provider MVP: **Digiflazz**. Kandidat tambahan setelah MVP: VIP Reseller, Apigames, Tokovoucher. Router & fallback tetap dibangun di MVP agar provider kedua cukup ditambah sebagai adapter.

### 3.5 Multi Payment Gateway
| ID | Fitur | Prioritas |
|---|---|---|
| PAY-01 | Abstraksi `PaymentGateway` interface; satu adapter per PG | P0 |
| PAY-02 | Metode bayar internal (mis. `QRIS`, `VA_BCA`) dipetakan ke satu/lebih PG dengan prioritas | P0 |
| PAY-03 | Routing metode bayar ke PG berdasarkan prioritas, fee, dan status aktif | P0 |
| PAY-04 | Verifikasi signature webhook di setiap adapter; webhook idempoten | P0 |
| PAY-05 | Fee bisa dibebankan ke pembeli, ke penjual, atau dibagi (`SPLIT`: persentase beban pembeli diatur per metode bayar di `fee_customer_share`, sisanya ditanggung penjual) | P0 |
| PAY-06 | Batas nominal min/maks per metode bayar | P0 |
| PAY-07 | Refund otomatis ke saldo deposit; refund ke sumber dana jika PG mendukung. Guest (tanpa akun saldo) menghubungi admin/CS dan refund diproses manual | P0 (saldo) / P2 (sumber dana) |
| PAY-08 | Deposit saldo member/reseller lewat PG | P0 |

PG MVP: **Xendit**. Kandidat tambahan setelah MVP: Midtrans, Tripay, Duitku.

### 3.6 Akun & Reseller
| ID | Fitur | Prioritas |
|---|---|---|
| USR-01 | Registrasi/login email + password, verifikasi email, reset password | P0 |
| USR-02 | Login OTP WhatsApp (via Kirimdev) / Google OAuth | P1 |
| USR-03 | Dashboard member: saldo, riwayat transaksi, mutasi saldo | P0 |
| USR-04 | Upgrade tier reseller (manual oleh admin atau otomatis berdasarkan volume) | P1 |
| USR-05 | API H2H untuk reseller: API key, IP whitelist, signature HMAC, callback (push) status order ke callback URL reseller | P1 |
| USR-06 | 2FA (TOTP): wajib untuk akun internal, opsional untuk reseller | P0 (internal) / P1 (reseller) |

### 3.7 Panel Admin
| ID | Fitur | Prioritas |
|---|---|---|
| ADM-01 | Dashboard: omzet, profit, jumlah transaksi, success rate per provider & per PG | P0 |
| ADM-02 | CRUD kategori, brand, produk, mapping produk–provider | P0 |
| ADM-03 | Kelola provider & PG: kredensial (terenkripsi), aktif/nonaktif, prioritas, mode sandbox/production | P0 |
| ADM-04 | Daftar transaksi dengan filter; aksi manual: retry, pindah provider, tandai sukses/gagal, refund | P0 |
| ADM-05 | Rekonsiliasi harian (transaksi internal vs laporan provider & PG) + laporan selisih | P1 |
| ADM-06 | Manajemen RBAC: CRUD role, atur permission per role, assign role ke user (detail di bagian 7) | P0 |
| ADM-07 | Audit log setiap aksi admin | P0 |
| ADM-08 | Export laporan (CSV/Excel) | P1 |

---

## 4. Alur Utama

### 4.1 Alur pembelian (guest, bayar via PG)
1. Pembeli memilih brand → produk → mengisi ID akun → memilih metode bayar.
2. Sistem menghitung harga (tier + promo + fee) dan membuat `order` berstatus `PENDING_PAYMENT`.
3. Sistem memilih PG untuk metode bayar tersebut → membuat tagihan → menyimpan `payment` dan menampilkan invoice.
4. PG mengirim webhook → signature diverifikasi → `payment` = `PAID`, `order` = `PAID`.
5. Job *fulfillment* masuk antrean → router memilih provider → kirim order ke provider → `order` = `PROCESSING`.
6. Provider mengembalikan hasil (langsung/callback/polling):
   - Sukses → `order` = `SUCCESS`, simpan SN/kode voucher, kirim notifikasi.
   - Gagal dan error boleh di-retry → coba provider berikutnya (fallback).
   - Gagal final (tidak ada provider lain / error tidak boleh di-retry) → `order` = `FAILED` → refund otomatis ke saldo → `REFUNDED`. Untuk guest (tanpa akun saldo), order tetap `FAILED` dan notifikasi gagal berisi pesan untuk menghubungi Customer Service (kontak dari `settings.cs_contact`); Finance memproses refund manual (`destination: MANUAL`) lalu order menjadi `REFUNDED`.
7. Invoice kedaluwarsa tanpa pembayaran → `order` = `EXPIRED` dan `payment` = `EXPIRED`. Masa berlaku charge di PG disamakan dengan `orders.expires_at`, sehingga PG menolak pembayaran setelah kedaluwarsa.

### 4.2 State machine order

```
PENDING_PAYMENT ──bayar──▶ PAID ──kirim──▶ PROCESSING ──sukses──▶ SUCCESS
      │                                     │      ▲
      └──timeout──▶ EXPIRED      gagal final│      │ retry / switch-provider (admin)
                                            ▼      │
                                          FAILED ──┘
                                            ├──mark SUCCESS (admin)──▶ SUCCESS
                                            └──refund──▶ REFUNDED

Manual admin dari PROCESSING: mark SUCCESS ──▶ SUCCESS, mark FAILED ──▶ FAILED
```

Aturan:
- **Transisi otomatis** (sistem) hanya maju: `PENDING_PAYMENT → PAID → PROCESSING → SUCCESS | FAILED → REFUNDED`, atau `PENDING_PAYMENT → EXPIRED`.
- **Transisi manual admin** hanya yang tercantum di diagram (`FAILED → PROCESSING` lewat retry/switch-provider, `FAILED/PROCESSING → SUCCESS` lewat mark, `PROCESSING → FAILED` lewat mark). Wajib `reason` dan permission terkait, serta tercatat di `order_status_histories` dan `audit_logs`. Tabel lengkapnya di `API_CONTRACT.md` §8.6.
- Semua transisi dikunci dengan *optimistic locking* (kolom `version`).
- `SUCCESS`, `EXPIRED`, `REFUNDED` adalah status final.
- `EXPIRED` berarti pembayaran gagal: PG menolak pembayaran setelah kedaluwarsa. Bila webhook `PAID` tetap datang setelah `EXPIRED` (anomali/race), order **tidak** diproses dan sistem mengirim alert ke admin untuk ditangani manual.
- Aturan yang sama berlaku untuk deposit: deposit `EXPIRED` = pembayaran gagal, saldo tidak bertambah.

### 4.3 Alur fallback provider
```
for provider in router.Candidates(product):        // urut sesuai strategi
    if circuitBreaker.Open(provider) { continue }
    if provider.Cost(product) > maxAllowedCost { continue }   // proteksi rugi
    res := provider.Purchase(ctx, req)                         // ref_id unik per attempt
    log attempt
    if res.Success  { return SUCCESS }
    if res.Pending  { return PROCESSING }  // tunggu callback / polling
    if !res.Retryable { break }
return FAILED
```
Satu `order` bisa punya banyak `order_attempt`; setiap attempt memakai `ref_id` unik agar tidak terjadi double-charge di sisi provider. Setiap provider hanya dicoba **satu kali** per order (error retryable tidak diulang ke provider yang sama); `product_routing.max_attempts` membatasi jumlah provider berbeda yang dicoba. Dengan satu provider di MVP (Digiflazz), kegagalan langsung menjadi `FAILED`. Urutan strategi `priority` memakai `providers.priority` (global).

---

## 5. Arsitektur Teknis

### 5.1 Stack
- **Backend:** Go + GoFiber, GORM, JWT auth — **Modular Architecture (modular monolith)**: setiap domain (auth, rbac, catalog, pricing, provider, payment, order, fulfillment, wallet, dst.) adalah modul mandiri dengan layer handler/service/repository berbasis interface, berkomunikasi lewat kontrak publik dan domain event. Detail di `ARCHITECTURE.md`.
- **Database:** PostgreSQL.
- **Cache & antrean:** Redis (cache harga/produk, rate limit, distributed lock) + worker queue (mis. Asynq) untuk fulfillment, polling status, sinkronisasi harga.
- **Frontend:** Nuxt 4.6.1 (Vue 3, TypeScript), mobile-first. Dipecah per Nuxt Layer (core, storefront, account); storefront di-render SSR/SWR untuk SEO. Panel admin adalah aplikasi Nuxt terpisah di direktori sendiri (lihat butir 25 di bagian 13). Nuxt server (Nitro) berperan sebagai BFF untuk menyimpan token di cookie httpOnly.
- **Alur kerja:** DBML schema → kontrak OpenAPI → implementasi.

### 5.2 Interface inti (gambaran)

```go
type Provider interface {
    Code() string
    FetchPriceList(ctx context.Context) ([]ProviderProduct, error)
    Purchase(ctx context.Context, req PurchaseRequest) (PurchaseResult, error)
    CheckStatus(ctx context.Context, refID string) (PurchaseResult, error)
    Balance(ctx context.Context) (int64, error)
    ParseCallback(headers map[string]string, body []byte) (PurchaseResult, error) // sekaligus verifikasi signature
}

type PaymentGateway interface {
    Code() string
    SupportedMethods() []string
    CreateCharge(ctx context.Context, req ChargeRequest) (ChargeResult, error)
    CheckStatus(ctx context.Context, externalID string) (PaymentStatus, error)
    ParseWebhook(headers map[string]string, body []byte) (WebhookEvent, error) // sekaligus verifikasi signature
    Refund(ctx context.Context, req RefundRequest) error // boleh return ErrNotSupported
}
```

Registry adapter di-inject saat startup; provider/PG aktif dibaca dari database sehingga bisa diubah tanpa deploy ulang.

### 5.3 Komponen
```
[Web/Client] ──▶ [API GoFiber] ──▶ [Service layer] ──▶ [PostgreSQL]
                      │                  │
         webhook PG ──┤                  ├──▶ [Redis: cache, lock, rate limit]
   callback provider ─┘                  └──▶ [Queue] ──▶ [Worker: fulfillment,
                                                          polling, sync harga,
                                                          rekonsiliasi, notifikasi]
```

---

## 6. Model Data (garis besar)

| Tabel | Kolom penting |
|---|---|
| `users` | id, name, email, phone, password_hash, tier_id, status, balance (cache), perm_version, two_factor_enabled, email_verified_at |
| `tiers` | id, code (`GUEST`, `MEMBER`, `RESELLER_*`), name, rank, is_default, auto_upgrade_min_volume |
| `roles` | id, code, name, description, scope (`internal`/`customer`), is_system |
| `permissions` | id, code (`resource:action`), resource, action, description, is_sensitive |
| `role_permissions` | role_id, permission_id (PK gabungan) |
| `user_roles` | user_id, role_id, assigned_by, assigned_at, expires_at (nullable, belum dipakai di v1) |
| `media` | id, storage_key, url, mime_type, size_bytes, width, height, alt_text, uploaded_by |
| `categories` | id, slug, name, icon_media_id, sort_order, is_active |
| `brands` | id, category_id, slug, name, icon_media_id, banner_media_id, input_schema (JSON), supports_account_check, is_active |
| `products` | id, brand_id, sku, name, fixed_price (null = dinamis), original_price, is_active, is_available |
| `banners` | id, title, image_media_id, link_url, starts_at, ends_at, is_active |
| `providers` | id, code, name, mode, credentials_encrypted, is_active, priority (global), balance, circuit_state, config |
| `product_routing` | product_id, strategy, max_attempts (jumlah provider berbeda) |
| `product_provider_map` | product_id, provider_id, provider_sku, cost_price, is_available, is_active, weight, last_synced_at |
| `pricing_rules` | scope (global/category/brand/product), scope_id, tier_id, margin_type, margin_value, min_margin |
| `promos`, `promo_usages` | code, type, value, max_discount, quota, used_count, per_user_limit, rules (JSON), starts_at, ends_at; pemakaian per order |
| `payment_gateways` | id, code, name, mode, credentials_encrypted, is_active |
| `payment_methods` | id, code, name, group, icon_media_id, fee_flat, fee_percent, fee_bearer, fee_customer_share, min_amount, max_amount, expiry_minutes |
| `payment_method_gateway_map` | payment_method_id, gateway_id, gateway_method_code, priority, is_active |
| `payments` | id, purpose (`ORDER`/`DEPOSIT`), order_id / deposit_id, payment_method_id, gateway_id, external_id, amount, fee, status, expires_at, paid_at |
| `orders` | id, invoice_no, source (`WEB`/`H2H`), user_id (nullable), product_id, product_snapshot, target_data (JSON), contact_email, contact_whatsapp, tier_id, price, estimated_cost, actual_cost, discount, fee, total, status, refund_destination, version, expires_at |
| `order_status_histories` | order_id, from_status, to_status, actor_type, actor_id, reason |
| `order_attempts` | id, order_id, attempt_no, provider_id, ref_id, status, error_code, request_payload, response_payload |
| `deposits` | id, user_id, amount, status, payment_method_id, expires_at, paid_at |
| `balance_ledger` | id, user_id, type, amount, balance_before, balance_after, reference_type, reference_id |
| `webhook_events` | id, source_type, source_code, event_id, signature_valid, payload, processed_at (unik per source+event_id) |
| `audit_logs` | id, actor_id, actor_roles, action, entity, entity_id, reason, before, after, ip |

Skema lengkap (termasuk tabel auth, reseller, notifikasi, rekonsiliasi, setting, outbox) ada di `database.dbml`, yang menjadi sumber kebenaran bila berbeda dengan tabel di atas.

Prinsip: saldo user hanya berubah lewat `balance_ledger` (append-only) di dalam transaksi DB yang sama.

---

## 7. Role-Based Access Control (RBAC)

### 7.1 Prinsip
- **Deny by default** — tanpa permission eksplisit, akses ditolak (HTTP 403).
- **Permission granular** dengan format `resource:action`, contoh `orders:refund`, `providers:manage_credentials`.
- **Role = kumpulan permission.** Role disimpan di database dan bisa dibuat/diubah lewat panel tanpa deploy ulang.
- **Multi-role** — satu user boleh punya lebih dari satu role; permission efektif = gabungan (union) semua role-nya.
- **Role ≠ tier.** Role mengatur *apa yang boleh dilakukan*; tier mengatur *harga yang didapat*.
- **Role `reseller` ↔ tier reseller.** Pemegang role `reseller` hanya mendapat akses sesuai permission role reseller, dan harga sesuai tier reseller. Saat role `reseller` di-assign, tier user otomatis diset ke tier reseller default (`RESELLER_SILVER`) bila belum ber-tier reseller. Tier reseller (Silver/Gold/Platinum) hanya boleh dimiliki pemegang role `reseller`; saat role dicabut, tier kembali ke `MEMBER`.
- **Katalog permission berasal dari kode** (di-seed saat migrasi), bukan dibuat bebas lewat UI, sehingga setiap permission pasti punya pengecekan di backend.

### 7.2 Kebutuhan fungsional RBAC
| ID | Fitur | Prioritas |
|---|---|---|
| RBAC-01 | Seed role sistem & katalog permission saat migrasi; role sistem (`is_system`) tidak bisa dihapus | P0 |
| RBAC-02 | CRUD role kustom (nama, deskripsi, scope `internal`/`customer`) | P0 |
| RBAC-03 | Atur permission per role lewat checklist yang dikelompokkan per resource | P0 |
| RBAC-04 | Assign/cabut role ke user; user boleh punya banyak role | P0 |
| RBAC-05 | Middleware pengecekan permission di setiap route admin & member | P0 |
| RBAC-06 | Pengecekan kepemilikan data (ownership) di service layer: member/reseller hanya bisa akses datanya sendiri | P0 |
| RBAC-07 | Anti-eskalasi: user tidak bisa memberi permission yang tidak ia miliki, tidak bisa mengubah role dirinya sendiri, dan hanya Super Admin yang bisa assign role Super Admin | P0 |
| RBAC-08 | Permission sensitif (`is_sensitive`) mewajibkan 2FA step-up (verifikasi TOTP ≤ 5 menit terakhir) | P0 |
| RBAC-09 | Masking data per permission: tanpa `customers:view_pii`, email/nomor HP pembeli ditampilkan tersamar (mis. `0812****789`) | P0 |
| RBAC-10 | Sistem menjaga minimal satu Super Admin aktif (tidak bisa dicabut/dinonaktifkan jika tinggal satu) | P0 |
| RBAC-11 | Setiap perubahan role, permission, dan assignment tercatat di `audit_logs` (before/after) | P0 |
| RBAC-12 | Perubahan permission berlaku segera tanpa user perlu login ulang | P0 |
| RBAC-13 | Frontend menampilkan menu & tombol sesuai permission efektif user (`GET /me/permissions`); backend tetap jadi penentu akhir | P0 |
| RBAC-14 | Role dengan masa berlaku (`expires_at`) untuk akses sementara — tidak di v1 (kolom disiapkan, belum dipakai) | P2 |

### 7.3 Role bawaan (system roles)
| Role | Scope | Deskripsi |
|---|---|---|
| `super_admin` | internal | Akses penuh (wildcard `*`). Satu-satunya yang bisa kelola role & kredensial provider/PG. |
| `admin` | internal | Operasional harian penuh, kecuali kredensial, manajemen role, dan pengaturan sistem. |
| `operator` | internal | Kelola katalog, harga, mapping provider, dan menangani transaksi bermasalah (retry/pindah provider). |
| `finance` | internal | Refund, penyesuaian saldo, rekonsiliasi, laporan keuangan. |
| `customer_service` | internal | Lihat transaksi & user, kirim ulang notifikasi, retry transaksi. Data pribadi pembeli tersamar. |
| `reseller` | customer | Fitur member + harga reseller + akses API H2H. |
| `member` | customer | Akses dashboard pribadi, saldo, dan riwayat transaksi sendiri. |

Guest tidak punya role; hanya bisa mengakses endpoint publik.

### 7.4 Katalog permission (awal)
| Resource | Permission |
|---|---|
| Dashboard | `dashboard:view`, `dashboard:view_profit` |
| Katalog | `catalog:read`, `catalog:write`, `catalog:sync` |
| Media | `media:read`, `media:write` |
| Harga | `pricing:read`, `pricing:write` |
| Provider | `providers:read`, `providers:write`, `providers:manage_credentials`★ |
| Payment gateway | `payments:read`, `payments:write`, `payments:manage_credentials`★ |
| Order | `orders:read`, `orders:retry`, `orders:switch_provider`, `orders:mark_status`★, `orders:refund`★ |
| User | `users:read`, `users:write`, `users:suspend`, `users:assign_role`★ |
| Data pribadi | `customers:view_pii` |
| Saldo | `balance:read`, `balance:adjust`★ |
| Promo | `promos:read`, `promos:write` |
| Laporan | `reports:read`, `reports:export`, `reconciliation:run` |
| Notifikasi | `notifications:resend` |
| RBAC | `roles:read`, `roles:write`★ |
| Audit | `audit:read` |
| Sistem | `settings:write`★ |
| Customer | `self:orders`, `self:balance`, `self:deposit`, `h2h:access` |

★ = sensitif (`is_sensitive = true`): wajib 2FA step-up dan selalu diaudit.

### 7.5 Matriks role × permission (default)
| Permission | super_admin | admin | operator | finance | customer_service |
|---|:-:|:-:|:-:|:-:|:-:|
| `dashboard:view` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `dashboard:view_profit` | ✓ | ✓ | – | ✓ | – |
| `catalog:*`, `pricing:*` | ✓ | ✓ | ✓ | read | read |
| `media:read/write` | ✓ | ✓ | ✓ | – | – |
| `providers:read/write` | ✓ | ✓ | ✓ | read | – |
| `providers:manage_credentials` | ✓ | – | – | – | – |
| `payments:read/write` | ✓ | ✓ | read | ✓ | – |
| `payments:manage_credentials` | ✓ | – | – | – | – |
| `orders:read` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `orders:retry` | ✓ | ✓ | ✓ | – | ✓ |
| `orders:switch_provider` | ✓ | ✓ | ✓ | – | – |
| `orders:mark_status` | ✓ | ✓ | ✓ | – | – |
| `orders:refund` | ✓ | ✓ | – | ✓ | – |
| `users:read` | ✓ | ✓ | – | ✓ | ✓ |
| `users:write`, `users:suspend` | ✓ | ✓ | – | – | – |
| `users:assign_role` | ✓ | ✓ (hanya role customer) | – | – | – |
| `customers:view_pii` | ✓ | ✓ | – | ✓ | – |
| `balance:read` | ✓ | ✓ | – | ✓ | ✓ |
| `balance:adjust` | ✓ | – | – | ✓ | – |
| `promos:*` | ✓ | ✓ | – | read | read |
| `reports:*`, `reconciliation:run` | ✓ | ✓ | – | ✓ | – |
| `notifications:resend` | ✓ | ✓ | ✓ | – | ✓ |
| `roles:read` | ✓ | ✓ | – | – | – |
| `roles:write` | ✓ | – | – | – | – |
| `audit:read` | ✓ | ✓ | – | ✓ | – |
| `settings:write` | ✓ | – | – | – | – |

Role customer: `member` = `self:orders`, `self:balance`, `self:deposit`; `reseller` = semua milik `member` + `h2h:access`.

Matriks ini hanya nilai awal (seed); Super Admin dapat mengubahnya dari panel.

### 7.6 Implementasi teknis
**Pengecekan di route (GoFiber):**
```go
admin := app.Group("/api/admin/v1", authMW.JWT(), rbacMW.Load())
admin.Get("/orders", rbacMW.Require("orders:read"), orderHandler.List)
admin.Post("/orders/:id/refund",
    rbacMW.Require("orders:refund"), rbacMW.StepUp2FA(), orderHandler.Refund)
admin.Put("/users/:id/roles", rbacMW.Require("users:assign_role"), userHandler.AssignRoles)
```

**Interface:**
```go
type Authorizer interface {
    Permissions(ctx context.Context, userID uint) (PermissionSet, error) // dari cache, fallback DB
    Can(ctx context.Context, userID uint, perm string) (bool, error)
    Invalidate(ctx context.Context, userID uint) error
}

type PermissionSet map[string]struct{} // "*" untuk super_admin
```

**Alur per request:**
1. JWT hanya berisi `user_id`, `session_id`, dan `perm_version` — **bukan** daftar permission (agar token tetap kecil dan perubahan role tidak menunggu token kedaluwarsa).
2. Middleware `Load()` mengambil permission set dari Redis (`rbac:perm:{user_id}:{perm_version}`); jika tidak ada, query DB lalu simpan ke cache.
3. Setiap perubahan role/permission menaikkan `users.perm_version` untuk user terdampak (untuk perubahan permission sebuah role: semua pemegang role itu). Cache lama otomatis tidak terpakai → berlaku seketika (RBAC-12).
4. `Require(perm)` mengecek permission; `*` meloloskan semua.
5. Ownership & masking dikerjakan di service layer, bukan di middleware, karena butuh konteks data.

**Aturan anti-eskalasi (di service `RoleService`):**
- Saat mengubah permission role: setiap permission yang ditambahkan harus dimiliki oleh pelaku.
- Saat assign role: seluruh permission role tersebut harus subset dari permission pelaku; pelaku tidak boleh menjadi target.
- Role `super_admin` hanya bisa di-assign/dicabut oleh `super_admin`.
- Penghapusan role ditolak jika masih ada user yang memegangnya (harus dipindahkan dulu) atau jika `is_system = true`.

**Respons error:** `401` jika belum login/token tidak valid; `403` dengan kode `PERMISSION_DENIED` dan nama permission yang dibutuhkan; `403` dengan kode `STEP_UP_REQUIRED` jika perlu verifikasi 2FA ulang.

### 7.7 Kriteria penerimaan (contoh)
- User `customer_service` memanggil `POST /orders/:id/refund` → 403 `PERMISSION_DENIED`.
- Admin mencabut `orders:retry` dari role `operator` → request berikutnya dari operator langsung 403 tanpa logout.
- `admin` mencoba assign role `super_admin` ke user lain → ditolak.
- Super Admin terakhir mencoba menonaktifkan dirinya → ditolak.
- `customer_service` membuka detail order → nomor HP & email tersamar.
- Member A mengakses `GET /me/orders` → hanya melihat order miliknya; mencoba ID order milik member B → 404.

---

## 8. Kebutuhan Non-Fungsional

### 8.1 Keamanan
- Verifikasi signature untuk semua webhook PG & callback provider; tolak jika tidak valid, catat di `webhook_events`.
- IP whitelist untuk endpoint callback jika provider/PG menyediakan daftar IP.
- Kredensial provider/PG dienkripsi at-rest (AES-GCM, key dari env/secret manager).
- Idempotency key pada endpoint checkout & API reseller.
- Rate limiting per IP & per user; captcha di checkout guest.
- Validasi & sanitasi input; tidak menampilkan raw error provider ke pembeli.
- Akses dikendalikan RBAC (bagian 7): deny-by-default, permission dicek di setiap endpoint, perubahan role/permission diaudit.
- 2FA wajib untuk semua user dengan role `internal`; aksi sensitif butuh konfirmasi ulang 2FA (step-up).

### 8.2 Keandalan
- Webhook diproses idempoten (unik `source + event_id`) dan dibalas cepat (proses berat via queue).
- Polling status cadangan untuk order `PROCESSING` dan payment `PENDING` yang tidak menerima callback dalam batas waktu.
- Circuit breaker per provider/PG; retry dengan exponential backoff.
- Backup database harian, retensi minimal 30 hari.

### 8.3 Performa
- Halaman katalog & produk dari cache; p95 respons API < 300 ms (di luar panggilan ke pihak ketiga).
- Target awal: 50 transaksi/menit sustained tanpa degradasi.

### 8.4 Observability
- Structured logging dengan `invoice_no` / `request_id` sebagai correlation ID.
- Metrik: success rate & latensi per provider/PG, antrean worker, saldo provider.
- Alert (WhatsApp admin via Kirimdev): saldo provider di bawah ambang, success rate turun, webhook gagal berturut-turut, selisih rekonsiliasi.

### 8.5 Kepatuhan
- Kebijakan privasi & syarat layanan; penyimpanan data pribadi minimal.
- Kelengkapan legal usaha mengikuti persyaratan onboarding masing-masing PG.

---

## 9. Endpoint API (garis besar)

**Publik**
- `GET /api/v1/categories`, `GET /api/v1/brands/:slug`, `GET /api/v1/brands/:slug/products`
- `POST /api/v1/accounts/check` — cek nickname
- `GET /api/v1/payment-methods?amount=`
- `POST /api/v1/orders` — checkout (header `Idempotency-Key`)
- `GET /api/v1/orders/:invoice_no` — lacak transaksi

**Webhook**
- `POST /api/v1/webhooks/payments/:gateway`
- `POST /api/v1/webhooks/providers/:provider`

**Member** (JWT)
- `POST /api/v1/auth/register|login|refresh`, `GET /api/v1/me`
- `GET /api/v1/me/orders`, `GET /api/v1/me/ledger`, `POST /api/v1/me/deposits`

**Reseller H2H** (API key + HMAC) — P1
- `GET /api/h2h/v1/products`, `POST /api/h2h/v1/orders`, `GET /api/h2h/v1/orders/:ref_id`, `GET /api/h2h/v1/balance`

**RBAC** (JWT + permission `roles:*` / `users:assign_role`)
- `GET|POST /api/admin/v1/roles`, `GET|PATCH|DELETE /api/admin/v1/roles/:id`
- `PUT /api/admin/v1/roles/:id/permissions` — set daftar permission sebuah role
- `GET /api/admin/v1/permissions` — katalog permission (read-only, dari seed)
- `PUT /api/admin/v1/users/:id/roles` — assign/cabut role user
- `GET /api/v1/me/permissions` — daftar permission efektif user login (untuk render menu frontend)

**Admin** (JWT + RBAC, tiap route punya permission sendiri)
- `/api/admin/v1/products`, `/media`, `/providers`, `/payment-gateways`, `/payment-methods`, `/orders`, `/orders/:id/retry|refund|mark`, `/users`, `/promos`, `/reports`, `/audit-logs`

Detail request/response ada di `API_CONTRACT.md` (sumber untuk `openapi.yaml`).

---

## 10. Metrik Keberhasilan

| Metrik | Target |
|---|---|
| Success rate fulfillment (dari order PAID) | ≥ 98% |
| Median waktu PAID → SUCCESS (produk instan) | < 30 detik |
| Order yang butuh intervensi manual | < 1% |
| Konversi checkout → paid | dipantau, baseline di bulan pertama |
| Selisih rekonsiliasi yang tidak terjelaskan | 0 per hari |

---

## 11. Rencana Rilis

| Fase | Isi | Estimasi |
|---|---|---|
| **M0 — Fondasi** | DBML, OpenAPI, scaffolding, auth, 2FA (TOTP) internal + step-up, RBAC inti (seed role & permission, middleware, assignment, anti-eskalasi, audit), katalog, media | 3 minggu |
| **M1 — MVP** | Provider Digiflazz + PG Xendit, routing `cheapest`/`priority`, fallback, checkout guest, webhook, refund ke saldo, panel admin dasar, notifikasi email | 4 minggu |
| **M2 — Member & Saldo** | Deposit, bayar pakai saldo, tier harga, promo, notifikasi WA | 2 minggu |
| **M3 — Reseller & Ops** | API H2H + callback, 2FA reseller, rekonsiliasi otomatis, laporan & export, routing `weighted` | 3 minggu |
| **M4 — Ekspansi** | Provider/PG tambahan, cek nickname lebih luas, flash sale | berkelanjutan |

---

## 12. Risiko & Mitigasi

| Risiko | Mitigasi |
|---|---|
| Provider sukses tapi callback tidak sampai → order nyangkut | Polling status cadangan + aksi manual admin |
| Double fulfillment akibat retry | `ref_id` unik per attempt, cek status sebelum fallback jika respons ambigu (timeout) |
| Harga modal naik mendadak → rugi | Sinkronisasi harga terjadwal + proteksi rugi (PRC-05) |
| Saldo provider habis | Alert saldo minimum + provider otomatis dilewati router |
| Webhook palsu | Verifikasi signature + IP whitelist + cek status ke PG sebelum menandai PAID bila nominal ≥ `settings.payment_recheck_threshold` (default Rp1.000.000) |
| Chargeback / penipuan pembayaran | Batas nominal per metode, pemantauan pola transaksi, blacklist |
| Perubahan API pihak ketiga | Adapter terisolasi + uji kontrak di sandbox |
| Eskalasi hak akses oleh staf internal | Aturan anti-eskalasi RBAC (RBAC-07), 2FA step-up, audit log perubahan role |
| Gateway WA (Kirimdev) gangguan atau nomor pengirim diblokir | Adapter terisolasi agar gateway bisa diganti, nomor khusus notifikasi, hanya pesan transaksional; email tetap kanal utama (P0) dan WA pelengkap |
| Super Admin terkunci keluar (lockout) | Minimal satu Super Admin aktif dijaga sistem + prosedur recovery via CLI di server |

---

## 13. Open Questions

1. ~~Frontend~~ — **diputuskan:** Nuxt 4.6 (lihat `ARCHITECTURE.md`). Dipakai 4.6.1 karena 4.6.0 gagal merender SSR pada build produksi di Windows.
2. ~~Late payment setelah `EXPIRED`~~ — **direvisi:** tidak ada late payment; charge di PG kedaluwarsa bersamaan dengan order sehingga PG menolak pembayaran (lihat 4.2).
3. ~~Refund guest tanpa akun saldo~~ — **diputuskan:** pembeli menghubungi admin; refund diproses manual (PAY-07).
4. ~~Provider & PG untuk MVP~~ — **diputuskan:** Digiflazz & Xendit (satu masing-masing).
5. ~~Kanal WhatsApp~~ — **direvisi:** gateway pihak ketiga Kirimdev (TRX-05).
6. ~~Callback H2H~~ — **diputuskan:** push callback ke reseller (USR-05).
7. ~~Margin default per kategori~~ — **diputuskan:** ya, tiap kategori punya margin default (PRC-02). Nilainya diatur admin.
8. ~~Role sementara di v1~~ — **diputuskan:** tidak (RBAC-14 tetap P2).
9. ~~Pembatasan akses per kategori/brand~~ — **diputuskan:** tidak, cukup RBAC global di v1.
10. ~~Late payment pada order guest~~ — **gugur:** mengikuti no. 2.
11. ~~Implementasi gateway WhatsApp~~ — **direvisi:** Kirimdev (bukan Baileys), juga dipakai untuk alert admin dan OTP WA.
12. ~~Deposit yang kedaluwarsa~~ — **diputuskan:** `EXPIRED` = pembayaran gagal, saldo tidak bertambah.
13. ~~2FA vs MVP~~ — **diputuskan:** 2FA internal + step-up dipindah ke M0 (P0); 2FA reseller tetap P1.
14. ~~Retry dengan satu provider~~ — **diputuskan:** tidak diulang ke provider yang sama; langsung `FAILED` (PRV-03).
15. ~~Transisi manual admin~~ — **diputuskan:** dimasukkan ke state machine (4.2).
16. ~~Margin global vs kategori~~ — **diputuskan:** global dipakai bila kategori belum punya aturan (PRC-02).
17. ~~Role reseller vs tier~~ — **diputuskan:** role reseller membawa tier reseller (7.1).
18. ~~Prioritas provider~~ — **diputuskan:** `providers.priority` (global).
19. ~~Masa berlaku invoice~~ — **diputuskan:** `payment_methods.expiry_minutes`.
20. ~~Rasio fee `SPLIT`~~ — **diputuskan:** kolom `payment_methods.fee_customer_share` (PAY-05).
21. ~~Ambang cek ulang PG~~ — **diputuskan:** `settings.payment_recheck_threshold`, default Rp1.000.000.
22. ~~Tanda tangan callback reseller~~ — **diputuskan:** memakai `callback_secret` khusus per reseller, terpisah dari API key (`API_CONTRACT.md` §6.4).
23. ~~Verifikasi email~~ — **diputuskan:** email wajib terverifikasi untuk login, sehingga berlaku ke seluruh fitur member.
24. ~~Gambar~~ — **diputuskan:** tabel `media` terpusat dan dipakai ulang (CAT-08).
25. ~~Lokasi panel admin~~ — **diputuskan:** panel admin dibangun sebagai aplikasi Nuxt baru di direktori terpisah (bukan layer di `topup-frontend`). Storefront tidak memproxy `/api/admin/**`; aplikasi admin punya BFF sendiri.