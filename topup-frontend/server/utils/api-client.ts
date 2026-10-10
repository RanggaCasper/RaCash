// Klien Go API untuk handler BFF.
import type { H3Event } from 'h3'
import { getRequestHeaders, getRequestIP, setResponseHeader, setResponseStatus } from 'h3'

export interface UpstreamInit {
  method: string
  body?: BodyInit
  headers?: Record<string, string>
  token?: string
}

/** Memanggil Go API; error jaringan → 503 SERVICE_UNAVAILABLE (envelope). */
export async function callApi(event: H3Event, path: string, init: UpstreamInit): Promise<Response> {
  const apiBase = useRuntimeConfig(event).apiBase as string
  const headers: Record<string, string> = {
    ...pickHeaders(getRequestHeaders(event), FORWARD_REQUEST_HEADERS),
    ...init.headers,
  }
  // IP asli pembeli untuk rate limit per IP di Go API.
  const ip = getRequestIP(event, { xForwardedFor: true })
  if (ip) headers['x-forwarded-for'] = ip
  if (init.token) headers.authorization = `Bearer ${init.token}`
  try {
    return await fetch(`${apiBase}${path}`, { method: init.method, headers, body: init.body, redirect: 'manual' })
  }
  catch {
    return Response.json(errorEnvelope('SERVICE_UNAVAILABLE', 'Layanan sedang tidak tersedia. Coba lagi nanti.'), { status: 503 })
  }
}

/** Menyalin status & header yang diizinkan dari respons Go API ke respons BFF. */
export function relayResponseMeta(event: H3Event, res: Response) {
  setResponseStatus(event, res.status)
  for (const name of FORWARD_RESPONSE_HEADERS) {
    const v = res.headers.get(name)
    if (v !== null) setResponseHeader(event, name, v)
  }
}

export async function readJson(res: Response): Promise<unknown> {
  try {
    return await res.clone().json()
  }
  catch {
    return null
  }
}
