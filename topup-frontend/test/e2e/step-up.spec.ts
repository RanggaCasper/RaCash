import { expect, test, type Page, type Route } from '@playwright/test'

// Modal step-up 2FA (RBAC-08) end-to-end di browser, dengan Go API ditiru lewat
// page.route. Navigasi ke /account/security dilakukan di sisi client karena BFF
// tanpa Go API tidak bisa memuat sesi saat SSR.

const ok = (data: unknown) => ({ status: 200, contentType: 'application/json', body: JSON.stringify({ success: true, data, request_id: 'r' }) })
const fail = (status: number, code: string, message = code) => ({
  status, contentType: 'application/json',
  body: JSON.stringify({ success: false, error: { code, message }, request_id: 'r' }),
})

const me = {
  id: 'u1', name: 'Rangga', email: 'a@racash.id', phone: null,
  tier: { id: 't1', code: 'MEMBER', name: 'Member' }, balance: 0, status: 'ACTIVE',
  two_factor_enabled: true, email_verified_at: null, roles: ['member'],
}

interface Calls { disable: Array<string | undefined>, stepUp: string[] }

async function mockApi(page: Page, opts: { validCode?: string } = {}): Promise<Calls> {
  const calls: Calls = { disable: [], stepUp: [] }
  const validCode = opts.validCode ?? '123456'
  await page.context().addCookies([{ name: 'rc_session', value: '1', url: 'http://localhost:3100' }])

  await page.route('**/api/me', (r: Route) => r.fulfill(ok(me)))
  await page.route('**/api/me/permissions', (r: Route) => r.fulfill(ok({ roles: [], permissions: [], perm_version: 1 })))
  await page.route('**/api/auth/2fa/status', (r: Route) => r.fulfill(ok({ enabled: true, required: false, recovery_codes_remaining: 7 })))
  await page.route('**/api/auth/step-up', async (r: Route) => {
    const { code } = r.request().postDataJSON() as { code: string }
    calls.stepUp.push(code)
    if (code !== validCode) return r.fulfill(fail(422, 'VALIDATION_ERROR', 'Kode salah atau sudah dipakai.'))
    await r.fulfill(ok({ step_up_token: 'su_token', expires_at: new Date(Date.now() + 300_000).toISOString() }))
  })
  await page.route('**/api/auth/2fa/disable', async (r: Route) => {
    const token = r.request().headers()['x-step-up-token']
    calls.disable.push(token)
    if (token !== 'su_token') return r.fulfill(fail(403, 'STEP_UP_REQUIRED', 'Aksi ini memerlukan verifikasi 2FA ulang.'))
    await r.fulfill(ok(null))
  })
  return calls
}

async function openSecurityPage(page: Page) {
  await page.goto('/')
  await page.getByRole('link', { name: 'Rangga' }).click()
  await page.getByRole('link', { name: 'Keamanan akun' }).click()
  await expect(page.getByRole('heading', { name: 'Keamanan akun' })).toBeVisible()
  await expect(page.getByText('2FA aktif')).toBeVisible()
}

test('403 STEP_UP_REQUIRED → modal TOTP → request diulang dengan X-Step-Up-Token', async ({ page }) => {
  const calls = await mockApi(page)
  await openSecurityPage(page)

  await page.getByRole('button', { name: 'Matikan 2FA' }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.getByText('Verifikasi 2FA')).toBeVisible()

  await dialog.getByLabel('Kode autentikator').fill('123456')
  await dialog.getByRole('button', { name: 'Verifikasi' }).click()

  await expect(page.getByText('2FA dimatikan.', { exact: true })).toBeVisible()
  await expect(dialog).toBeHidden()
  expect(calls.stepUp).toEqual(['123456'])
  // Percobaan pertama tanpa token (ditolak), pengulangan dengan token.
  expect(calls.disable).toEqual([undefined, 'su_token'])
})

test('kode salah menampilkan error di modal dan modal tetap terbuka', async ({ page }) => {
  const calls = await mockApi(page)
  await openSecurityPage(page)

  await page.getByRole('button', { name: 'Matikan 2FA' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Kode autentikator').fill('000000')
  await dialog.getByRole('button', { name: 'Verifikasi' }).click()

  await expect(dialog.getByText('Kode salah atau sudah dipakai.')).toBeVisible()
  expect(calls.disable).toEqual([undefined]) // belum diulang
})

test('membatalkan modal tidak menampilkan error dan tidak mengulang request', async ({ page }) => {
  const calls = await mockApi(page)
  await openSecurityPage(page)

  await page.getByRole('button', { name: 'Matikan 2FA' }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button', { name: 'Batal' }).click()

  await expect(dialog).toBeHidden()
  await expect(page.getByText('Aksi ini memerlukan verifikasi 2FA ulang.')).toHaveCount(0)
  expect(calls.disable).toEqual([undefined])
  expect(calls.stepUp).toEqual([])
})
