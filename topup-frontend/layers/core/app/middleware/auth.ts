// Halaman yang butuh login: definePageMeta({ middleware: 'auth' }).
export default defineNuxtRouteMiddleware(async (to) => {
  const auth = useAuthStore()
  if (!auth.loaded) await useAuth().fetchSession()
  // Token terbatas: hanya halaman setup 2FA yang boleh dibuka.
  if (auth.setupRequired) return to.path === SECURITY_PATH ? undefined : navigateTo(SECURITY_PATH)
  if (!auth.user) return navigateTo({ path: LOGIN_PATH, query: { redirect: to.fullPath } })
})
