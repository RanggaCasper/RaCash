import { fileURLToPath } from 'node:url'

// Layer fondasi: UI kit, state, composable API/auth/RBAC, util format.
// Layer fitur (storefront, account, admin) hanya boleh bergantung pada core.
export default defineNuxtConfig({
  modules: ['@nuxt/ui', '@pinia/nuxt', '@vee-validate/nuxt', '@nuxt/image'],
  css: [fileURLToPath(new URL('./app/assets/css/main.css', import.meta.url))],
  // @pinia/nuxt hanya memindai app/stores di root; store layer didaftarkan eksplisit.
  pinia: {
    storesDirs: [fileURLToPath(new URL('./app/stores', import.meta.url))],
  },
  ui: {
    colorMode: true,
  },
})
