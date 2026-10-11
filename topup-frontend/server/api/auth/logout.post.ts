import { defineEventHandler } from 'h3'

// POST /api/auth/logout — cabut sesi di Go API (best effort) lalu hapus cookie.
export default defineEventHandler(async (event) => {
  const token = getAccessToken(event)
  if (token) await callApi(event, '/api/v1/auth/logout', { method: 'POST', token })
  clearAuthSession(event)
  return { success: true, data: null, request_id: '' }
})
