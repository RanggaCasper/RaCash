// Halaman khusus tamu (mis. /masuk): user yang sudah login diarahkan ke /akun.
export default defineNuxtRouteMiddleware(async () => {
  const auth = useAuthStore()
  if (!auth.loaded && useCookie(SESSION_HINT).value) await useAuth().fetchSession()
  if (auth.setupRequired) return navigateTo(SECURITY_PATH)
  if (auth.user) return navigateTo('/akun')
})
