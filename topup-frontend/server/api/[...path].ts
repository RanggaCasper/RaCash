// Proxy generik BFF: /api/<path> → Go API /api/v1/<path>. Menyisipkan Bearer dari cookie httpOnly, me-refresh sekali
// bila access token kedaluwarsa, dan tidak pernah meneruskan cookie browser.
import { defineEventHandler, getRequestURL, getRouterParam, readRawBody, setResponseStatus } from 'h3'

export default defineEventHandler(async (event) => {
  const target = upstreamPath(getRouterParam(event, 'path') ?? '')
  if (!target) {
    setResponseStatus(event, 404)
    return errorEnvelope('NOT_FOUND', 'Data tidak ditemukan.')
  }

  const url = target + getRequestURL(event).search
  const body = isSafeMethod(event.method) ? undefined : await readRawBody(event, false)
  const send = (token?: string) => callApi(event, url, { method: event.method, body: body as BodyInit | undefined, token })

  // Cookie access token sudah habis (maxAge = masa berlaku token) tetapi refresh
  // token masih ada → refresh lebih dulu agar request tidak gagal 401.
  let token = getAccessToken(event)
  if (!token && getRefreshToken(event)) token = (await refreshSession(event)) ?? undefined

  let res = await send(token)
  if (res.status === 401 && getRefreshToken(event) && isTokenExpired(401, await readJson(res))) {
    const fresh = await refreshSession(event)
    if (fresh) res = await send(fresh)
  }

  relayResponseMeta(event, res)
  return Buffer.from(await res.arrayBuffer())
})
