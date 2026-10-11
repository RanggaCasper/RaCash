# topup-frontend

Frontend RaCash: Nuxt 4.6.1 (Vue 3, TypeScript strict), Nuxt UI 4 + Tailwind 4, Pinia, VeeValidate + Zod. Nitro berperan sebagai BFF. Desainnya ada di [`docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md) §3, aturan wajibnya di [`docs/RULES.md`](../docs/RULES.md) §6.

## Menjalankan

```bash
npm install                  # postinstall menjalankan nuxt prepare
cp .env.example .env         # NUXT_API_BASE → Go API, NUXT_COOKIE_SECURE=false untuk http lokal
npm run dev                  # http://localhost:3000
```

| Perintah | Keterangan |
|---|---|
| `npm run build` / `npm run preview` | Build produksi (`.output/`) / jalankan hasil build |
| `npm run typecheck` | `nuxt typecheck` (vue-tsc) |
| `npm run lint` | ESLint (`@nuxt/eslint`) |
| `npm run gen:api` | Generate `shared/types/api.d.ts` dari `../docs/openapi.yaml` — **jangan diedit manual** |
| `npm test` | Unit test (util client & BFF, tanpa build) |
| `npm run test:bff` | BFF end-to-end: build Nuxt + Go API tiruan, tanpa browser (±1 menit) |
| `npm run test:e2e` | Playwright (butuh `npm run build` dulu dan `npx playwright install chromium`) |
| `npx vitest run test/unit/format.test.ts` | Satu file test |

## Struktur

```
app/                  app.vue, error.vue
layers/core           UI kit, Pinia store, useApi/useAuth/useCan/useMoney, middleware, v-can, util
layers/storefront     halaman publik (beranda, brand, checkout, invoice, lacak)
layers/account        /login, /account/**
server/               BFF: api/auth/*, api/[...path].ts (proxy), middleware/csrf.ts, utils/session.ts
shared/types          api.d.ts (generated) + schema.ts (helper Schema<'Me'>)
test/{unit,bff,e2e}
```

Layer di `layers/` didaftarkan otomatis oleh Nuxt. Layer fitur hanya boleh bergantung pada `core`.

## Pola penting

- **Panggilan API:** `const api = useApi(); await api.get<Schema<'Me'>>('/me')`. Path ditulis tanpa `/api`. Panel admin bukan bagian dari aplikasi ini (aplikasi Nuxt terpisah). `useApi` mengurai envelope, melempar `ApiError`, dan menampilkan toast kecuali `{ silent: true }`. Di SSR, `useApi` memakai `useRequestFetch` agar cookie ikut terkirim.
- **Tipe:** pakai `Schema<'NamaSchema'>` dari `#shared/types/schema`. Jangan menulis tipe respons API manual.
- **Auth:**
  - Token hanya ada di cookie httpOnly (`at`, `rt`) yang dikelola `server/utils/session.ts`; client tidak pernah melihatnya.
  - Halaman privat memakai `definePageMeta({ middleware: 'auth' })` atau `{ permission: 'orders:read' }`.
- **Halaman SWR (`/`, `/g/**`):** HTML-nya di-cache dan dibagi ke semua pengunjung. Jangan render data user di server; bungkus dengan `<ClientOnly>`.
- **RBAC di UI:** `v-can="'orders:refund'"`, `useCan(...)`, dan `available_actions` dari API. Ini hanya UX; backend tetap penentu akhir.

## Jebakan yang sudah ditemui

- **Nuxt 4.6.0** gagal merender SSR pada build produksi di Windows (`Either manifest or precomputed data must be provided`). Karena itu dipakai **4.6.1**.
- **`nitro.externals.inline: ['pinia']` wajib dipertahankan.** Tanpanya, build di Windows bisa berisi dua salinan Pinia, sehingga store gagal dengan "no active Pinia" saat SSR.
- **TypeScript dikunci di 5.x.** TypeScript 7 (native) tidak punya compiler API JS yang dipakai `openapi-typescript` dan `vue-tsc`.
- **Zod dikunci di 3.x,** karena `@vee-validate/zod` 4.15 hanya mendukung Zod 3.
- **Plugin di layer yang memakai store** wajib `dependsOn: ['pinia']`.
- **Form:** pakai `method="post"` dan nonaktifkan tombol submit sampai `onMounted`. Klik sebelum hidrasi memicu submit native, yang bisa memasukkan isi form ke URL.
