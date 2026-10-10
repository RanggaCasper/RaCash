import type { Pinia } from 'pinia'
import type { Directive } from 'vue'

// v-can="'orders:refund'" → elemen disembunyikan bila tidak punya permission.
// Hanya UX; backend tetap penentu akhir.
export default defineNuxtPlugin({
  name: 'can-directive',
  // Plugin layer tidak dijamin berjalan setelah Pinia; store juga diambil malas
  // dengan instance Pinia eksplisit agar aman saat SSR.
  dependsOn: ['pinia'],
  setup(nuxtApp) {
    const can = (permission: string) => usePermissionsStore(nuxtApp.$pinia as Pinia).can(permission)

    const directive: Directive<HTMLElement, string> = {
      mounted: (el, binding) => {
        el.style.display = can(binding.value) ? '' : 'none'
      },
      updated: (el, binding) => {
        el.style.display = can(binding.value) ? '' : 'none'
      },
      getSSRProps: binding => (can(binding.value) ? {} : { style: { display: 'none' } }),
    }

    nuxtApp.vueApp.directive('can', directive)
  },
})
