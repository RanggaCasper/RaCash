// Test BFF end-to-end: Nuxt di-build & dijalankan sungguhan, Go API diganti
// server HTTP tiruan. Memastikan token tidak pernah sampai ke browser.
import { createServer, type IncomingMessage, type Server } from 'node:http'
import { fileURLToPath } from 'node:url'
import { fetch, setup, url } from '@nuxt/test-utils/e2e'
import { afterAll, beforeAll, beforeEach, describe, expect, it } from 'vitest'

const MOCK_PORT = 39181
const MOCK = `http://127.0.0.1:${MOCK_PORT}`

interface Seen {
  method: string
  path: string
  headers: IncomingMessage['headers']
  body: string
}

const seen: Seen[] = []
let refreshCalls = 0
let mock: Server

const tokens = (at: string, rt: string) => ({
  access_token: at,
  access_token_expires_at: new Date(Date.now() + 15 * 60_000).toISOString(),
  refresh_token: rt,
  refresh_token_expires_at: new Date(Date.now() + 30 * 86_400_000).toISOString(),
})

function json(res: import('node:http').ServerResponse, status: number, body: unknown) {
  res.writeHead(status, { 'content-type': 'application/json', 'x-request-id': 'req_mock' })
  res.end(JSON.stringify(body))
}

const fail = (code: string) => ({ success: false, error: { code, message: code }, request_id: 'req_mock' })

beforeAll(async () => {
  mock = createServer((req, res) => {
    let body = ''
    req.on('data', (c) => { body += c })
    req.on('end', async () => {
      const path = req.url ?? ''
      seen.push({ method: req.method ?? '', path, headers: req.headers, body })
      const auth = req.headers.authorization

      if (path === '/api/v1/auth/login') {
        const { email, password } = JSON.parse(body || '{}')
        if (email === '2fa@racash.id') return json(res, 200, { success: true, data: { two_factor_required: true, challenge_token: 'ch_1' }, request_id: 'r' })
        if (email === 'staf@racash.id') return json(res, 200, { success: true, data: { two_factor_required: false, two_factor_setup_required: true, ...tokens('at-limited', 'rt-limited'), user: { id: 'u2', name: 'Staf' } }, request_id: 'r' })
        if (password !== 'rahasia') return json(res, 401, fail('INVALID_CREDENTIALS'))
        return json(res, 200, { success: true, data: { two_factor_required: false, ...tokens('at-valid', 'rt-login'), user: { id: 'u1', name: 'Rangga' } }, request_id: 'r' })
      }
      if (path === '/api/v1/auth/login/2fa') {
        const { challenge_token, code } = JSON.parse(body || '{}')
        if (challenge_token !== 'ch_1' || code !== '123456') return json(res, 401, fail('INVALID_CREDENTIALS'))
        return json(res, 200, { success: true, data: { two_factor_required: false, ...tokens('at-2fa', 'rt-2fa'), user: { id: 'u1', name: 'Rangga' } }, request_id: 'r' })
      }
      if (path === '/api/v1/auth/refresh') {
        refreshCalls++
        const { refresh_token } = JSON.parse(body || '{}')
        await new Promise(r => setTimeout(r, 300)) // beri waktu request paralel saling tumpang tindih
        if (!String(refresh_token).startsWith('rt-ok')) return json(res, 401, fail('UNAUTHENTICATED'))
        return json(res, 200, { success: true, data: tokens('at-valid', `${refresh_token}-next`), request_id: 'r' })
      }
      if (path === '/api/v1/auth/logout') return json(res, 200, { success: true, data: null, request_id: 'r' })
      if (path === '/api/v1/me') {
        if (auth === 'Bearer at-valid') return json(res, 200, { success: true, data: { id: 'u1', name: 'Rangga' }, request_id: 'r' })
        if (auth === 'Bearer at-expired') return json(res, 401, fail('TOKEN_EXPIRED'))
        return json(res, 401, fail('UNAUTHENTICATED'))
      }
      return json(res, 200, { success: true, data: { path, method: req.method, body }, request_id: 'r' })
    })
  })
  await new Promise<void>(r => mock.listen(MOCK_PORT, '127.0.0.1', () => r()))
})

afterAll(() => new Promise<void>(r => mock.close(() => r())))

await setup({
  rootDir: fileURLToPath(new URL('../..', import.meta.url)),
  server: true,
  browser: false,
  nuxtConfig: {
    runtimeConfig: { apiBase: MOCK, cookieSecure: false },
  },
})

const origin = () => new URL(url('/')).origin
const post = (path: string, body: unknown, headers: Record<string, string> = {}) =>
  fetch(path, { method: 'POST', headers: { 'content-type': 'application/json', origin: origin(), ...headers }, body: JSON.stringify(body) })
const cookies = (res: Response) => res.headers.getSetCookie()
const cookie = (res: Response, name: string) => cookies(res).find(c => c.startsWith(`${name}=`))

beforeEach(() => {
  seen.length = 0
  refreshCalls = 0
})

describe('CSRF', () => {
  it('menolak POST tanpa Origin', async () => {
    const res = await fetch('/api/auth/login', { method: 'POST', headers: { 'content-type': 'application/json' }, body: '{}' })
    expect(res.status).toBe(403)
    expect((await res.json()).error.code).toBe('PERMISSION_DENIED')
    expect(seen).toHaveLength(0)
  })

  it('menolak POST dari origin lain', async () => {
    const res = await post('/api/orders', {}, { origin: 'https://evil.example' })
    expect(res.status).toBe(403)
    expect(seen).toHaveLength(0)
  })

  it('GET tidak dicek origin', async () => {
    const res = await fetch('/api/categories')
    expect(res.status).toBe(200)
  })
})

describe('login', () => {
  it('menyimpan token di cookie httpOnly dan tidak mengirimnya ke browser', async () => {
    const res = await post('/api/auth/login', { email: 'a@racash.id', password: 'rahasia' })
    expect(res.status).toBe(200)
    const body = await res.json()
    expect(body.data.user.name).toBe('Rangga')
    expect(JSON.stringify(body)).not.toMatch(/at-valid|rt-login|access_token|refresh_token/)

    const at = cookie(res, 'at')!
    const rt = cookie(res, 'rt')!
    expect(at).toMatch(/^at=at-valid;/)
    expect(at).toMatch(/HttpOnly/i)
    expect(at).toMatch(/SameSite=Lax/i)
    expect(rt).toMatch(/^rt=rt-login;/)
    expect(rt).toMatch(/HttpOnly/i)
    expect(cookie(res, 'rc_session')).not.toMatch(/HttpOnly/i)
  })

  it('2FA mengembalikan challenge tanpa membuat sesi', async () => {
    const res = await post('/api/auth/login', { email: '2fa@racash.id', password: 'x' })
    expect((await res.json()).data).toMatchObject({ two_factor_required: true, challenge_token: 'ch_1' })
    expect(cookie(res, 'at')).toBeUndefined()
  })

  it('login/2fa menyimpan token di cookie dan menyembunyikannya dari browser', async () => {
    const res = await post('/api/auth/login/2fa', { challenge_token: 'ch_1', code: '123456' })
    expect(res.status).toBe(200)
    expect(JSON.stringify(await res.json())).not.toMatch(/at-2fa|rt-2fa|access_token|refresh_token/)
    expect(cookie(res, 'at')).toMatch(/^at=at-2fa;.*HttpOnly/i)
    expect(cookie(res, 'rt')).toMatch(/^rt=rt-2fa;.*HttpOnly/i)

    const bad = await post('/api/auth/login/2fa', { challenge_token: 'ch_1', code: '000000' })
    expect(bad.status).toBe(401)
    expect(cookie(bad, 'at')).toBeUndefined()
  })

  it('staf tanpa 2FA: token terbatas disimpan di cookie, flag setup diteruskan ke browser', async () => {
    const res = await post('/api/auth/login', { email: 'staf@racash.id', password: 'rahasia' })
    const body = await res.json()
    expect(body.data.two_factor_setup_required).toBe(true)
    expect(JSON.stringify(body)).not.toMatch(/at-limited|rt-limited|access_token|refresh_token/)
    expect(cookie(res, 'at')).toMatch(/^at=at-limited;.*HttpOnly/i)
  })

  it('meneruskan error Go API apa adanya', async () => {
    const res = await post('/api/auth/login', { email: 'a@racash.id', password: 'salah' })
    expect(res.status).toBe(401)
    expect((await res.json()).error.code).toBe('INVALID_CREDENTIALS')
    expect(cookie(res, 'at')).toBeUndefined()
  })
})

describe('proxy', () => {
  it('menyisipkan Bearer dari cookie dan tidak meneruskan cookie browser', async () => {
    const res = await fetch('/api/me', { headers: { cookie: 'at=at-valid; rt=rt-x; lain=1' } })
    expect(res.status).toBe(200)
    const call = seen.find(s => s.path === '/api/v1/me')!
    expect(call.headers.authorization).toBe('Bearer at-valid')
    expect(call.headers.cookie).toBeUndefined()
    expect(call.headers['x-forwarded-for']).toBeTruthy()
  })

  it('tidak memproxy /api/admin/** (bukan bagian storefront)', async () => {
    const res = await fetch('/api/admin/orders')
    expect(res.status).toBe(404)
    expect(seen).toHaveLength(0)
  })

  it('meneruskan body & header yang diizinkan (Idempotency-Key, step-up)', async () => {
    const res = await post('/api/orders', { product_id: 'p1' }, { 'idempotency-key': 'idem-1', 'x-step-up-token': 'su-1', 'authorization': 'Bearer palsu' })
    expect(res.status).toBe(200)
    const call = seen.find(s => s.path === '/api/v1/orders')!
    expect(JSON.parse(call.body)).toEqual({ product_id: 'p1' })
    expect(call.headers['idempotency-key']).toBe('idem-1')
    expect(call.headers['x-step-up-token']).toBe('su-1')
    expect(call.headers.authorization).toBeUndefined() // header Authorization dari browser diabaikan
  })

  it('step-up dan endpoint 2FA lewat proxy dengan Bearer dari cookie', async () => {
    for (const path of ['/api/auth/step-up', '/api/auth/2fa/setup', '/api/auth/2fa/enable', '/api/auth/2fa/disable']) {
      seen.length = 0
      const res = await post(path, { code: '123456' }, { cookie: 'at=at-valid; rt=rt-x', 'x-step-up-token': 'su_1' })
      expect(res.status).toBe(200)
      const call = seen.find(s => s.path === path.replace('/api/', '/api/v1/'))!
      expect(call.headers.authorization).toBe('Bearer at-valid')
      expect(call.headers['x-step-up-token']).toBe('su_1')
    }
  })

  it('tidak memproxy webhook & H2H', async () => {
    expect((await fetch('/api/webhooks/payments/xendit')).status).toBe(404)
    expect((await fetch('/api/h2h/products')).status).toBe(404)
    expect(seen).toHaveLength(0)
  })
})

describe('refresh token', () => {
  it('request paralel dengan token kedaluwarsa hanya me-refresh sekali', async () => {
    const headers = { cookie: 'at=at-expired; rt=rt-ok-parallel' }
    const results = await Promise.all([1, 2, 3].map(() => fetch('/api/me', { headers })))
    expect(results.map(r => r.status)).toEqual([200, 200, 200])
    expect(refreshCalls).toBe(1)
    expect(cookie(results[0]!, 'rt')).toMatch(/^rt=rt-ok-parallel-next;/)
  })

  it('refresh lebih dulu bila cookie access token sudah habis', async () => {
    const res = await fetch('/api/me', { headers: { cookie: 'rt=rt-ok-proactive' } })
    expect(res.status).toBe(200)
    expect(refreshCalls).toBe(1)
    expect(seen.filter(s => s.path === '/api/v1/me')).toHaveLength(1)
  })

  it('refresh ditolak → cookie sesi dihapus', async () => {
    const res = await fetch('/api/me', { headers: { cookie: 'at=at-expired; rt=rt-dicabut' } })
    expect(res.status).toBe(401)
    expect(cookie(res, 'at')).toMatch(/Max-Age=0|Expires=Thu, 01 Jan 1970/i)
    expect(cookie(res, 'rt')).toMatch(/Max-Age=0|Expires=Thu, 01 Jan 1970/i)
  })

  it('/api/auth/refresh tanpa sesi → 401', async () => {
    const res = await post('/api/auth/refresh', {})
    expect(res.status).toBe(401)
    expect((await res.json()).error.code).toBe('UNAUTHENTICATED')
  })
})

describe('logout', () => {
  it('mencabut sesi di Go API dan menghapus cookie', async () => {
    const res = await post('/api/auth/logout', {}, { cookie: 'at=at-valid; rt=rt-x' })
    expect(res.status).toBe(200)
    expect(seen.find(s => s.path === '/api/v1/auth/logout')?.headers.authorization).toBe('Bearer at-valid')
    expect(cookie(res, 'at')).toMatch(/Max-Age=0|Expires=Thu, 01 Jan 1970/i)
  })
})

describe('halaman', () => {
  it('beranda dirender SSR', async () => {
    const html = await (await fetch('/')).text()
    expect(html).toContain('Top up cepat')
  })

  it('/account tanpa login diarahkan ke /login', async () => {
    const res = await fetch('/account', { redirect: 'manual' })
    expect([301, 302, 307]).toContain(res.status)
    expect(res.headers.get('location')).toContain('/login')
  })
})
