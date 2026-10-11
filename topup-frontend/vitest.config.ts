import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    projects: [
      {
        // Fungsi murni (util client & BFF) — cepat, tanpa build Nuxt.
        test: {
          name: 'unit',
          include: ['test/unit/**/*.test.ts'],
          environment: 'node',
        },
      },
      {
        // BFF end-to-end: build Nuxt sungguhan + Go API tiruan (tanpa browser).
        test: {
          name: 'bff',
          include: ['test/bff/**/*.test.ts'],
          environment: 'node',
          testTimeout: 60_000,
          hookTimeout: 600_000,
        },
      },
    ],
  },
})
