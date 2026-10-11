// Sesi BFF: access & refresh token disimpan di cookie httpOnly. Browser tidak
// pernah memegang token (ARCHITECTURE §3.4).
import type { H3Event } from 'h3'
import { deleteCookie, getCookie, setCookie } from 'h3'

export const ACCESS_COOKIE = 'at'
export const REFRESH_COOKIE = 'rt'
/** Penanda non-httpOnly tanpa isi rahasia: client tahu ada sesi tanpa memanggil /me. */
export const SESSION_HINT_COOKIE = 'rc_session'

const ACCESS_FALLBACK_SECONDS = 15 * 60
const REFRESH_FALLBACK_SECONDS = 30 * 24 * 60 * 60

export interface TokenPair {
  access_token: string
  access_token_expires_at?: string
  refresh_token: string
  refresh_token_expires_at?: string
}

function cookieBase(event: H3Event) {
  return {
    httpOnly: true,
    secure: Boolean(useRuntimeConfig(event).cookieSecure),
    sameSite: 'lax' as const,
    // path "/" (bukan /api/auth) agar SSR halaman /account & proxy /api/** bisa
    // me-refresh access token yang kedaluwarsa secara transparan.
    path: '/',
  }
}

export function setAuthSession(event: H3Event, t: TokenPair) {
  const now = Date.now()
  const base = cookieBase(event)
  const refreshAge = secondsUntil(t.refresh_token_expires_at, now, REFRESH_FALLBACK_SECONDS)
  setCookie(event, ACCESS_COOKIE, t.access_token, {
    ...base,
    maxAge: secondsUntil(t.access_token_expires_at, now, ACCESS_FALLBACK_SECONDS),
  })
  setCookie(event, REFRESH_COOKIE, t.refresh_token, { ...base, maxAge: refreshAge })
  setCookie(event, SESSION_HINT_COOKIE, '1', { ...base, httpOnly: false, maxAge: refreshAge })
  event.context.accessToken = t.access_token
  event.context.refreshToken = t.refresh_token
}

export function clearAuthSession(event: H3Event) {
  const base = cookieBase(event)
  for (const name of [ACCESS_COOKIE, REFRESH_COOKIE, SESSION_HINT_COOKIE]) {
    deleteCookie(event, name, { ...base, httpOnly: name !== SESSION_HINT_COOKIE })
  }
  event.context.accessToken = undefined
  event.context.refreshToken = undefined
  event.context.sessionCleared = true
}

export function getAccessToken(event: H3Event): string | undefined {
  if (event.context.sessionCleared) return undefined
  return event.context.accessToken ?? getCookie(event, ACCESS_COOKIE)
}

export function getRefreshToken(event: H3Event): string | undefined {
  if (event.context.sessionCleared) return undefined
  return event.context.refreshToken ?? getCookie(event, REFRESH_COOKIE)
}

// Refresh token dirotasi setiap dipakai dan pemakaian ulang token lama mencabut
// seluruh sesi (API_CONTRACT §1.7). Beberapa request paralel dengan access token
// kedaluwarsa karenanya TIDAK boleh me-refresh masing-masing:
//  - inflight: request bersamaan menunggu satu panggilan refresh yang sama;
//  - recent: request yang datang sedikit kemudian dengan refresh token lama
//    (browser belum menerima cookie baru) memakai hasil rotasi yang sama.
const inflight = new Map<string, Promise<TokenPair | null>>()
const recent = new Map<string, { pair: TokenPair, until: number }>()
const RECENT_TTL_MS = 30_000

/** Me-refresh sesi; mengembalikan access token baru, atau null bila sesi tidak valid lagi. */
export async function refreshSession(event: H3Event): Promise<string | null> {
  const rt = getRefreshToken(event)
  if (!rt) return null

  const now = Date.now()
  for (const [k, v] of recent) if (v.until <= now) recent.delete(k)
  const cached = recent.get(rt)
  if (cached) {
    setAuthSession(event, cached.pair)
    return cached.pair.access_token
  }

  let pending = inflight.get(rt)
  if (!pending) {
    pending = callRefresh(useRuntimeConfig(event).apiBase as string, rt).finally(() => inflight.delete(rt))
    inflight.set(rt, pending)
  }
  const pair = await pending
  if (!pair) {
    clearAuthSession(event)
    return null
  }
  recent.set(rt, { pair, until: Date.now() + RECENT_TTL_MS })
  setAuthSession(event, pair)
  return pair.access_token
}

/** null = refresh token ditolak (401/403); error jaringan dilempar agar sesi tidak dihapus. */
async function callRefresh(apiBase: string, refreshToken: string): Promise<TokenPair | null> {
  const res = await fetch(`${apiBase}/api/v1/auth/refresh`, {
    method: 'POST',
    headers: { 'content-type': 'application/json', 'accept': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  })
  if (res.status === 401 || res.status === 403) return null
  if (!res.ok) throw new Error(`refresh gagal: HTTP ${res.status}`)
  const body = await res.json() as { data?: Partial<TokenPair> }
  const d = body.data
  if (!d?.access_token || !d.refresh_token) return null
  return d as TokenPair
}

/** Hanya untuk test. */
export function resetRefreshState() {
  inflight.clear()
  recent.clear()
}
