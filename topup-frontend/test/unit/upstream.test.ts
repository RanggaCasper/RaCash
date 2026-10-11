import { describe, expect, it } from 'vitest'
import {
  FORWARD_REQUEST_HEADERS,
  isAllowedOrigin,
  isTokenExpired,
  parseAllowedOrigins,
  pickHeaders,
  secondsUntil,
  upstreamPath,
} from '../../server/utils/upstream'

describe('upstreamPath', () => {
  it('memetakan path publik/member ke /api/v1', () => {
    expect(upstreamPath('me/orders')).toBe('/api/v1/me/orders')
    expect(upstreamPath('brands/mobile-legends/products')).toBe('/api/v1/brands/mobile-legends/products')
  })

  it('memetakan path admin ke /api/admin/v1', () => {
    expect(upstreamPath('admin/orders/123/refund')).toBe('/api/admin/v1/orders/123/refund')
    expect(upstreamPath('admin')).toBeNull()
  })

  it('menolak webhook, H2H, dan path traversal', () => {
    expect(upstreamPath('webhooks/payments/xendit')).toBeNull()
    expect(upstreamPath('h2h/orders')).toBeNull()
    expect(upstreamPath('me/../admin/roles')).toBeNull()
    expect(upstreamPath('')).toBeNull()
  })

  it('tidak memproxy endpoint yang mengembalikan token', () => {
    for (const p of ['auth/login', 'auth/login/2fa', 'auth/refresh', 'auth/logout']) {
      expect(upstreamPath(p)).toBeNull()
    }
    expect(upstreamPath('auth/register')).toBe('/api/v1/auth/register')
  })
})

describe('pickHeaders', () => {
  it('hanya meneruskan header yang diizinkan, tanpa cookie/authorization', () => {
    const out = pickHeaders({
      'Content-Type': 'application/json',
      'Idempotency-Key': 'k',
      'cookie': 'at=secret',
      'authorization': 'Bearer dari-browser',
      'x-step-up-token': 'su',
    }, FORWARD_REQUEST_HEADERS)
    expect(out).toEqual({ 'content-type': 'application/json', 'idempotency-key': 'k', 'x-step-up-token': 'su' })
  })
})

describe('isAllowedOrigin (CSRF)', () => {
  it('menerima origin dengan host yang sama', () => {
    expect(isAllowedOrigin('https://racash.id', 'racash.id', [])).toBe(true)
  })

  it('menolak origin lain, kosong, atau "null"', () => {
    expect(isAllowedOrigin('https://evil.example', 'racash.id', [])).toBe(false)
    expect(isAllowedOrigin(undefined, 'racash.id', [])).toBe(false)
    expect(isAllowedOrigin('null', 'racash.id', [])).toBe(false)
    expect(isAllowedOrigin('https://racash.id.evil.example', 'racash.id', [])).toBe(false)
  })

  it('menerima origin dari daftar izin', () => {
    const allowed = parseAllowedOrigins(' https://admin.racash.id/ , https://x.test')
    expect(isAllowedOrigin('https://admin.racash.id', 'racash.id', allowed)).toBe(true)
  })
})

describe('isTokenExpired', () => {
  it('hanya 401 TOKEN_EXPIRED', () => {
    expect(isTokenExpired(401, { error: { code: 'TOKEN_EXPIRED' } })).toBe(true)
    expect(isTokenExpired(401, { error: { code: 'UNAUTHENTICATED' } })).toBe(false)
    expect(isTokenExpired(403, { error: { code: 'TOKEN_EXPIRED' } })).toBe(false)
    expect(isTokenExpired(401, null)).toBe(false)
  })
})

describe('secondsUntil', () => {
  it('menghitung sisa umur token', () => {
    const now = Date.parse('2026-10-10T14:47:00Z')
    expect(secondsUntil('2026-10-10T15:02:00Z', now, 1)).toBe(900)
    expect(secondsUntil('2026-10-10T14:00:00Z', now, 1)).toBe(0)
    expect(secondsUntil(undefined, now, 42)).toBe(42)
  })
})
