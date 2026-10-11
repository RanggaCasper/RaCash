import { expect, test, type Page, type Route } from '@playwright/test'

// Halaman Masuk / Daftar / Lupa password (design/public/Masuk dan Daftar.dc.html).
// Go API ditiru lewat page.route; sesi dimuat di client lewat /api/me yang ditiru.

const ok = (data: unknown, status = 200) => ({ status, contentType: 'application/json', body: JSON.stringify({ success: true, data, request_id: 'r' }) })
const fail = (status: number, code: string, message: string, details: unknown[] = []) => ({
  status, contentType: 'application/json',
  body: JSON.stringify({ success: false, error: { code, message, details }, request_id: 'r' }),
})

const me = {
  id: 'u1', name: 'Rangga', email: 'a@racash.id', phone: null,
  tier: { id: 't1', code: 'MEMBER', name: 'Member' }, balance: 0, status: 'ACTIVE',
  two_factor_enabled: false, email_verified_at: null, roles: ['member'],
}
const loggedIn = { two_factor_required: false, user: { id: 'u1', name: 'Rangga', email: 'a@racash.id', tier: 'MEMBER', roles: ['member'] } }

/** Setelah login berhasil, client memuat profil dan permission. */
async function mockSession(page: Page) {
  await page.route('**/api/me', (r: Route) => r.fulfill(ok(me)))
  await page.route('**/api/me/permissions', (r: Route) => r.fulfill(ok({ roles: [], permissions: [], perm_version: 1 })))
}

/** Form siap diisi setelah hidrasi; mengisi sebelum itu bisa tertimpa saat hidrasi. */
async function gotoHydrated(page: Page, path: string) {
  await page.goto(path)
  await page.locator('form[data-hydrated="true"]').waitFor()
}

async function fillLogin(page: Page, email: string, password: string) {
  await gotoHydrated(page, '/login')
  const submit = page.getByRole('button', { name: 'Masuk', exact: true })
  await expect(submit).toBeEnabled() // aktif setelah hidrasi
  await page.getByLabel('Email', { exact: true }).fill(email)
  await page.getByLabel('Password', { exact: true }).fill(password)
  return submit
}

test.describe('Masuk', () => {
  test('tab, field, tautan lupa password, dan tautan daftar', async ({ page, isMobile }) => {
    await page.goto('/login')
    const tabs = page.getByRole('navigation', { name: 'Masuk atau daftar' })
    await expect(tabs.getByRole('link', { name: 'Masuk' })).toHaveAttribute('aria-current', 'page')
    await expect(tabs.getByRole('link', { name: 'Daftar' })).toHaveAttribute('href', '/register')
    await expect(page.getByRole('heading', { name: 'Selamat datang lagi' })).toBeVisible()
    await expect(page.getByLabel('Email', { exact: true })).toBeVisible()
    await expect(page.getByLabel('Password', { exact: true })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Lupa password?' })).toHaveAttribute('href', '/forgot-password')
    // Panel manfaat hanya di desktop.
    const perks = page.getByRole('heading', { name: 'Harga member, saldo, dan riwayat dalam satu akun' })
    if (isMobile) await expect(perks).toBeHidden()
    else await expect(perks).toBeVisible()
  })

  test('tombol mata menampilkan dan menyembunyikan password', async ({ page }) => {
    await gotoHydrated(page, '/login')
    const pw = page.getByLabel('Password', { exact: true })
    await expect(pw).toHaveAttribute('type', 'password')
    await page.getByRole('button', { name: 'Tampilkan password' }).click()
    await expect(pw).toHaveAttribute('type', 'text')
    await page.getByRole('button', { name: 'Sembunyikan password' }).click()
    await expect(pw).toHaveAttribute('type', 'password')
  })

  test('kredensial salah menampilkan pesan error', async ({ page }) => {
    await page.route('**/api/auth/login', r => r.fulfill(fail(401, 'INVALID_CREDENTIALS', 'Email atau password salah.')))
    const submit = await fillLogin(page, 'a@racash.id', 'salah123')
    await submit.click()
    await expect(page.getByRole('alert')).toContainText('Email atau password salah.')
    await expect(page).toHaveURL(/\/login$/)
  })

  test('email belum diverifikasi menawarkan kirim ulang', async ({ page }) => {
    let resent = 0
    await page.route('**/api/auth/login', r => r.fulfill(fail(403, 'EMAIL_NOT_VERIFIED', 'Email belum diverifikasi.')))
    await page.route('**/api/auth/resend-verification', (r) => {
      resent++
      return r.fulfill(ok(null))
    })
    const submit = await fillLogin(page, 'a@racash.id', 'S3cure!pass')
    await submit.click()
    await expect(page.getByRole('alert')).toContainText('Email belum diverifikasi.')
    await page.getByRole('button', { name: 'Kirim ulang email verifikasi' }).click()
    await expect(page.getByRole('status')).toContainText('link verifikasi sudah dikirim')
    expect(resent).toBe(1)
    await expect(page.getByRole('button', { name: /Kirim ulang dalam 1:/ })).toBeDisabled()
  })

  test('login berhasil → /account, dan redirect eksternal diabaikan', async ({ page }) => {
    await mockSession(page)
    await page.route('**/api/auth/login', r => r.fulfill(ok(loggedIn)))
    await gotoHydrated(page, '/login?redirect=//evil.example')
    const submit = page.getByRole('button', { name: 'Masuk', exact: true })
    await expect(submit).toBeEnabled()
    await page.getByLabel('Email', { exact: true }).fill('a@racash.id')
    await page.getByLabel('Password', { exact: true }).fill('S3cure!pass')
    await submit.click()
    await expect(page).toHaveURL(u => u.pathname === '/account')
  })

  test('akun ber-2FA: kode 6 digit lalu masuk; kode salah ditolak', async ({ page }) => {
    await mockSession(page)
    await page.route('**/api/auth/login', r => r.fulfill(ok({ two_factor_required: true, challenge_token: 'ch_1', challenge_expires_at: new Date().toISOString() })))
    const codes: string[] = []
    await page.route('**/api/auth/login/2fa', (r) => {
      const { code } = r.request().postDataJSON() as { code: string }
      codes.push(code)
      return code === '123456' ? r.fulfill(ok(loggedIn)) : r.fulfill(fail(401, 'INVALID_CREDENTIALS', 'Kode 2FA salah.'))
    })

    const submit = await fillLogin(page, 'a@racash.id', 'S3cure!pass')
    await submit.click()
    await expect(page.getByRole('heading', { name: 'Verifikasi 2 langkah' })).toBeVisible()

    const verify = page.getByRole('button', { name: 'Verifikasi' })
    await expect(verify).toBeDisabled() // belum 6 digit
    await page.getByLabel('Digit 1').click()
    await page.keyboard.type('000000')
    await expect(verify).toBeEnabled()
    await verify.click()
    await expect(page.getByRole('alert')).toContainText('Kode salah')

    await page.getByLabel('Digit 1').click()
    await page.keyboard.type('123456')
    await verify.click()
    await expect(page).toHaveURL(u => u.pathname === '/account')
    expect(codes).toEqual(['000000', '123456'])
  })

  test('2FA: beralih ke kode cadangan', async ({ page }) => {
    await page.route('**/api/auth/login', r => r.fulfill(ok({ two_factor_required: true, challenge_token: 'ch_1' })))
    const submit = await fillLogin(page, 'a@racash.id', 'S3cure!pass')
    await submit.click()
    await page.getByRole('button', { name: 'Pakai kode cadangan' }).click()
    await expect(page.getByLabel('Kode cadangan')).toBeVisible()
    await page.getByRole('button', { name: 'Pakai kode dari aplikasi' }).click()
    await expect(page.getByLabel('Digit 1')).toBeVisible()
  })

  test('tempel kode 6 digit mengisi semua kotak', async ({ page, context }) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write']).catch(() => {})
    await page.route('**/api/auth/login', r => r.fulfill(ok({ two_factor_required: true, challenge_token: 'ch_1' })))
    const submit = await fillLogin(page, 'a@racash.id', 'S3cure!pass')
    await submit.click()
    await page.getByLabel('Digit 1').evaluate((el) => {
      const dt = new DataTransfer()
      dt.setData('text', '654321')
      el.dispatchEvent(new ClipboardEvent('paste', { clipboardData: dt, bubbles: true, cancelable: true }))
    })
    for (const [i, d] of [...'654321'].entries()) {
      await expect(page.getByLabel(`Digit ${i + 1}`)).toHaveValue(d)
    }
  })
})

test.describe('Daftar', () => {
  async function fillRegister(page: Page, overrides: Partial<Record<'name' | 'email' | 'phone' | 'password', string>> = {}) {
    await gotoHydrated(page, '/register')
    await expect(page.getByRole('heading', { name: 'Buat akun gratis' })).toBeVisible()
    await page.getByLabel('Nama', { exact: true }).fill(overrides.name ?? 'Bagas')
    await page.getByLabel('Email', { exact: true }).fill(overrides.email ?? 'bagas@example.com')
    await page.getByLabel('Nomor WhatsApp').fill(overrides.phone ?? '812 3456 7890')
    await page.getByLabel('Password', { exact: true }).fill(overrides.password ?? 'S3cure!pass')
    await page.getByRole('checkbox').check()
  }

  test('tombol Daftar baru aktif setelah form valid; indikator kekuatan bereaksi', async ({ page }) => {
    await gotoHydrated(page, '/register')
    const submit = page.getByRole('button', { name: 'Daftar', exact: true })
    await expect(submit).toBeDisabled()

    await page.getByLabel('Password', { exact: true }).fill('abc')
    await expect(page.getByTestId('password-strength')).not.toContainText('Kuat')
    await page.getByLabel('Password', { exact: true }).fill('Abcdefg1')
    await expect(page.getByTestId('password-strength')).toContainText('Kuat')
    await page.getByLabel('Password', { exact: true }).fill('Abcdefghij12')
    await expect(page.getByTestId('password-strength')).toContainText('Sangat kuat')

    await page.getByLabel('Nama', { exact: true }).fill('Bagas')
    await page.getByLabel('Email', { exact: true }).fill('bagas@example.com')
    await page.getByLabel('Nomor WhatsApp').fill('812 3456 7890')
    await expect(submit).toBeDisabled() // belum menyetujui syarat
    await page.getByRole('checkbox').check()
    await expect(submit).toBeEnabled()
  })

  test('nomor WhatsApp hanya menerima angka dan spasi', async ({ page }) => {
    await gotoHydrated(page, '/register')
    const phone = page.getByLabel('Nomor WhatsApp')
    await phone.fill('abc 0812-345')
    await expect(phone).toHaveValue(' 0812345')
  })

  test('daftar berhasil → layar cek email, kirim ulang dengan hitung mundur', async ({ page }) => {
    let body: Record<string, string> = {}
    let resent = 0
    await page.route('**/api/auth/register', (r) => {
      body = r.request().postDataJSON() as Record<string, string>
      return r.fulfill(ok({ user_id: 'u1', status: 'PENDING_VERIFICATION' }, 201))
    })
    await page.route('**/api/auth/resend-verification', (r) => {
      resent++
      return r.fulfill(ok(null))
    })
    await fillRegister(page)
    const submit = page.getByRole('button', { name: 'Daftar', exact: true })
    await expect(submit).toBeEnabled()
    await submit.click()

    await expect(page.getByRole('heading', { name: 'Cek email kamu' })).toBeVisible()
    await expect(page.getByText('ba***@example.com')).toBeVisible()
    // Nomor dikirim dalam format server (08…), password tidak berubah.
    expect(body).toMatchObject({ name: 'Bagas', email: 'bagas@example.com', phone: '081234567890', password: 'S3cure!pass' })
    // Hitung mundur 60 detik sebelum boleh kirim ulang.
    await expect(page.getByRole('button', { name: /Kirim ulang dalam 0:|Kirim ulang dalam 1:/ })).toBeDisabled()
    expect(resent).toBe(0)

    await page.getByRole('button', { name: 'Ubah email' }).click()
    await expect(page.getByRole('heading', { name: 'Buat akun gratis' })).toBeVisible()
    await expect(page.getByLabel('Email', { exact: true })).toHaveValue('bagas@example.com') // isian tidak hilang
  })

  test('email sudah terdaftar → pesan error dan tautan Masuk saja', async ({ page }) => {
    await page.route('**/api/auth/register', r => r.fulfill(fail(409, 'CONFLICT', 'Email sudah terdaftar.')))
    await fillRegister(page)
    const submit = page.getByRole('button', { name: 'Daftar', exact: true })
    await expect(submit).toBeEnabled()
    await submit.click()
    await expect(page.getByText('Email sudah terdaftar.')).toBeVisible()
    await expect(page.getByRole('link', { name: 'Masuk saja' })).toHaveAttribute('href', '/login')
  })

  test('error validasi server ditampilkan di field-nya', async ({ page }) => {
    await page.route('**/api/auth/register', r => r.fulfill(fail(422, 'VALIDATION_ERROR', 'Data yang dikirim tidak valid.', [
      { field: 'phone', code: 'phone_id', message: 'Nomor tidak valid.' },
    ])))
    await fillRegister(page)
    const submit = page.getByRole('button', { name: 'Daftar', exact: true })
    await expect(submit).toBeEnabled()
    await submit.click()
    await expect(page.getByText('Nomor tidak valid.')).toBeVisible()
  })
})

test.describe('Lupa password', () => {
  test('mengirim link dan menampilkan konfirmasi tanpa membocorkan akun', async ({ page }) => {
    let sent = ''
    await page.route('**/api/auth/forgot-password', (r) => {
      sent = (r.request().postDataJSON() as { email: string }).email
      return r.fulfill(ok(null))
    })
    await gotoHydrated(page, '/forgot-password')
    const submit = page.getByRole('button', { name: 'Kirim link reset' })
    await expect(submit).toBeEnabled()
    await page.getByLabel('Email', { exact: true }).fill('bagas@example.com')
    await submit.click()
    await expect(page.getByRole('status')).toContainText('terdaftar, link reset sudah dikirim')
    expect(sent).toBe('bagas@example.com')
    await expect(page.getByRole('link', { name: 'Kembali ke Masuk' })).toHaveAttribute('href', '/login')
  })
})
