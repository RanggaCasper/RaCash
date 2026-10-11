// @ts-check
import withNuxt from './.nuxt/eslint.config.mjs'

export default withNuxt({
  ignores: ['shared/types/api.d.ts', 'playwright-report/**', 'test-results/**'],
})
