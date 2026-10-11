// Fungsi murni BFF (tanpa h3/Nitro) agar bisa di-unit-test.

/** Header request dari browser yang boleh diteruskan ke Go API. Cookie TIDAK diteruskan. */
export const FORWARD_REQUEST_HEADERS = [
  'accept',
  'accept-language',
  'content-type',
  'idempotency-key',
  'x-step-up-token',
  'x-captcha-token',
  'x-request-id',
] as const

/** Header respons Go API yang diteruskan ke browser. */
export const FORWARD_RESPONSE_HEADERS = [
  'content-type',
  'content-disposition',
  'x-request-id',
  'retry-after',
  'x-ratelimit-limit',
  'x-ratelimit-remaining',
  'idempotent-replayed',
] as const

/**
 * Path yang ditangani route BFF khusus dan TIDAK boleh lewat proxy generik,
 * karena responsnya berisi token (harus disimpan di cookie, bukan dikirim ke browser).
 */
const AUTH_TOKEN_PATHS = new Set(['auth/login', 'auth/login/2fa', 'auth/refresh', 'auth/logout'])

/**
 * Memetakan path setelah /api/ ke path Go API (API_CONTRACT §1.1):
 *   admin/<p> → /api/admin/v1/<p>,  <p> → /api/v1/<p>.
 * null = tidak boleh diproxy (webhook & H2H langsung ke api.domain, route token auth).
 */
export function upstreamPath(rest: string): string | null {
  const path = rest.replace(/^\/+/, '').replace(/\/+$/, '')
  if (path === '' || path.split('/').some(s => s === '..' || s === '.')) return null
  const first = path.split('/')[0]
  if (first === 'webhooks' || first === 'h2h') return null
  if (AUTH_TOKEN_PATHS.has(path)) return null
  if (first === 'admin') {
    const sub = path.slice('admin'.length).replace(/^\//, '')
    return sub ? `/api/admin/v1/${sub}` : null
  }
  return `/api/v1/${path}`
}

/** Mengambil header yang diizinkan (nama dibandingkan case-insensitive). */
export function pickHeaders(
  src: Record<string, string | undefined>,
  allow: readonly string[],
): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(src)) {
    const key = k.toLowerCase()
    if (v !== undefined && allow.includes(key)) out[key] = v
  }
  return out
}

/**
 * Cek CSRF: request non-GET ke /api/** wajib datang dari origin situs sendiri
 * (host sama dengan request) atau origin yang diizinkan eksplisit.
 */
export function isAllowedOrigin(origin: string | undefined, requestHost: string, allowed: readonly string[]): boolean {
  if (!origin || origin === 'null') return false
  let parsed: URL
  try {
    parsed = new URL(origin)
  }
  catch {
    return false
  }
  if (parsed.host === requestHost) return true
  return allowed.some(a => a.replace(/\/+$/, '') === parsed.origin)
}

export function parseAllowedOrigins(csv: string): string[] {
  return csv.split(',').map(s => s.trim()).filter(Boolean)
}

export function isSafeMethod(method: string): boolean {
  return ['GET', 'HEAD', 'OPTIONS'].includes(method.toUpperCase())
}

/** 401 dengan kode TOKEN_EXPIRED → access token perlu di-refresh. */
export function isTokenExpired(status: number, body: unknown): boolean {
  if (status !== 401 || typeof body !== 'object' || body === null) return false
  const err = (body as { error?: { code?: unknown } }).error
  return err?.code === 'TOKEN_EXPIRED'
}

export interface ErrorEnvelope {
  success: false
  error: { code: string, message: string }
  request_id: string
}

/** Envelope gagal sesuai API_CONTRACT §1.3 untuk error yang berasal dari BFF sendiri. */
export function errorEnvelope(code: string, message: string, requestId = ''): ErrorEnvelope {
  return { success: false, error: { code, message }, request_id: requestId }
}

/** Sisa umur token (detik) dari timestamp RFC 3339, dengan batas bawah 0. */
export function secondsUntil(iso: string | undefined, now: number, fallback: number): number {
  if (!iso) return fallback
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return fallback
  return Math.max(0, Math.floor((t - now) / 1000))
}
