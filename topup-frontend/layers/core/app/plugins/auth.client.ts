// Status login dimuat di CLIENT agar HTML halaman ber-cache (SWR) tidak pernah
// berisi data user. Halaman privat (/akun, /admin) memuatnya lewat middleware.
export default defineNuxtPlugin({
  name: 'auth-init',
  dependsOn: ['pinia'],
  parallel: true,
  async setup() {
    const auth = useAuthStore()
    if (auth.loaded) return
    if (useCookie(SESSION_HINT).value) await useAuth().fetchSession()
    else auth.clear()
  },
})
