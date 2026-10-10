import { defineEventHandler, readRawBody } from 'h3'

// POST /api/auth/login/2fa → Go API /api/v1/auth/login/2fa (lanjutan login bila 2FA aktif).
export default defineEventHandler(async (event) => {
  const body = await readRawBody(event, false)
  const res = await callApi(event, '/api/v1/auth/login/2fa', { method: 'POST', body: body as BodyInit | undefined })
  return finishLogin(event, res)
})
