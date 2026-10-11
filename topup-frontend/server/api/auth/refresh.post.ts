import { defineEventHandler, setResponseStatus } from 'h3'

// POST /api/auth/refresh — rotasi token dikerjakan di server; browser hanya menerima cookie baru.
export default defineEventHandler(async (event) => {
  const token = await refreshSession(event)
  if (!token) {
    setResponseStatus(event, 401)
    return errorEnvelope('UNAUTHENTICATED', 'Sesi berakhir. Silakan login kembali.')
  }
  return { success: true, data: { refreshed: true }, request_id: '' }
})
