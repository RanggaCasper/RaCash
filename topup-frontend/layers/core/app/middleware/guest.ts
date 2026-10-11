// Halaman khusus tamu (mis. /login): user yang sudah login diarahkan ke /account.
export default defineNuxtRouteMiddleware(async () => {
  const auth = useAuthStore()
  if (!auth.loaded && useCookie(SESSION_HINT).value) await useAuth().fetchSession()
  if (auth.setupRequired) return navigateTo(SECURITY_PATH)
  if (auth.user) return navigateTo(ACCOUNT_PATH)
})
