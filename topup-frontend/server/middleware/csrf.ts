// CSRF: request non-GET ke /api/** wajib membawa Origin (atau Referer) situs ini
// atau origin di NUXT_ALLOWED_ORIGINS. Cookie SameSite=Lax menjadi lapisan kedua.
import { defineEventHandler, getRequestHeader, getRequestHost, getRequestURL, setResponseStatus } from 'h3'

export default defineEventHandler((event) => {
  if (!getRequestURL(event).pathname.startsWith('/api/') || isSafeMethod(event.method)) return

  let origin = getRequestHeader(event, 'origin')
  if (!origin) {
    const referer = getRequestHeader(event, 'referer')
    if (referer) {
      try {
        origin = new URL(referer).origin
      }
      catch { /* referer rusak → ditolak */ }
    }
  }
  const allowed = parseAllowedOrigins(String(useRuntimeConfig(event).allowedOrigins ?? ''))
  if (!isAllowedOrigin(origin, getRequestHost(event, { xForwardedHost: true }), allowed)) {
    setResponseStatus(event, 403)
    return errorEnvelope('PERMISSION_DENIED', 'Permintaan ditolak (origin tidak valid).')
  }
})
