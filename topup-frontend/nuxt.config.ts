// Aplikasi root. Fitur dipecah per Nuxt Layer di layers/ (core, storefront,
// account) yang otomatis didaftarkan oleh Nuxt (ARCHITECTURE §3).
export default defineNuxtConfig({
  compatibilityDate: '2026-10-01',
  devtools: { enabled: true },

  modules: ['@nuxt/eslint'],

  runtimeConfig: {
    // Hanya server (BFF). Override lewat NUXT_API_BASE, NUXT_COOKIE_SECURE, dst.
    apiBase: 'http://localhost:8080',
    cookieSecure: true,
    // Origin tambahan yang boleh mengirim request non-GET ke /api/** (dipisah koma).
    allowedOrigins: '',
    public: {
      siteName: 'RaCash',
    },
  },

  // Strategi rendering (ARCHITECTURE §3.3). Halaman ber-cache (SWR) TIDAK boleh
  // merender data per user di server; status login dimuat di client.
  routeRules: {
    '/': { swr: 60 },
    '/games/**': { swr: 300 },
    '/invoice/**': { headers: { 'cache-control': 'no-store' } },
    '/track': { headers: { 'cache-control': 'no-store' } },
    '/account/**': { headers: { 'cache-control': 'no-store' } },
    '/login': { headers: { 'cache-control': 'no-store' } },
    '/api/**': { headers: { 'cache-control': 'no-store' } },
  },

  typescript: {
    strict: true,
  },

  nitro: {
    // Paksa satu salinan Pinia di bundle server. Tanpa ini, build di Windows bisa
    // berisi Pinia ter-bundle (dipakai @pinia/nuxt) DAN Pinia eksternal (dipakai
    // store), sehingga store gagal dengan "no active Pinia" saat SSR.
    externals: { inline: ['pinia'] },
  },

  eslint: {
    config: { stylistic: false },
  },
})
