// Halaman yang butuh login: definePageMeta({ middleware: 'auth' }).
export default defineNuxtRouteMiddleware(async (to) => {
  const auth = useAuthStore()
  if (!auth.loaded) await useAuth().fetchSession()
  if (!auth.user) return navigateTo({ path: '/masuk', query: { redirect: to.fullPath } })
})
