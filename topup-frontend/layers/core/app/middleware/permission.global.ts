// definePageMeta({ permission: 'orders:read' }) → wajib login dan punya permission.
// Tanpa permission → halaman error 403. Hanya UX; backend tetap mengecek ulang.
export default defineNuxtRouteMiddleware(async (to) => {
  const permission = to.meta.permission
  if (!permission) return

  const auth = useAuthStore()
  if (!auth.loaded) await useAuth().fetchSession()
  if (auth.setupRequired) return navigateTo(SECURITY_PATH)
  if (!auth.user) return navigateTo({ path: '/masuk', query: { redirect: to.fullPath } })

  if (!usePermissionsStore().can(permission)) {
    return abortNavigation(createError({ statusCode: 403, statusMessage: 'Anda tidak memiliki akses ke halaman ini.' }))
  }
})
