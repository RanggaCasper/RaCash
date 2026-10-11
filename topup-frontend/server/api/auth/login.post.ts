import { defineEventHandler, readRawBody } from 'h3'

// POST /api/auth/login → Go API /api/v1/auth/login. Token disimpan di cookie httpOnly.
export default defineEventHandler(async (event) => {
  const body = await readRawBody(event, false)
  const res = await callApi(event, '/api/v1/auth/login', { method: 'POST', body: body as BodyInit | undefined })
  return finishLogin(event, res)
})
