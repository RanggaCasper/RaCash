import { defineConfig, devices } from '@playwright/test'

const port = Number(process.env.E2E_PORT ?? 3100)

// E2E browser. Server Nuxt hasil build dijalankan otomatis; Go API tidak
// dibutuhkan untuk smoke test ini (halaman publik & redirect login).
export default defineConfig({
  testDir: 'test/e2e',
  fullyParallel: true,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    baseURL: `http://localhost:${port}`,
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'mobile', use: { ...devices['Pixel 7'] } },
  ],
  webServer: {
    command: 'node .output/server/index.mjs',
    port,
    reuseExistingServer: !process.env.CI,
    env: {
      PORT: String(port),
      NUXT_API_BASE: process.env.NUXT_API_BASE ?? 'http://127.0.0.1:1',
      NUXT_COOKIE_SECURE: 'false',
    },
  },
})
