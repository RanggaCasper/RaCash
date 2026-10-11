import { expect, test } from '@playwright/test'

// Smoke test tanpa Go API: halaman publik, dan redirect login.

test('beranda tampil dengan tombol Masuk', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { level: 1 })).toContainText('Top up cepat')
  await expect(page.getByRole('link', { name: 'Masuk' })).toBeVisible()
})

test('/account tanpa login diarahkan ke /login dengan redirect', async ({ page }) => {
  await page.goto('/account')
  await expect(page).toHaveURL(u => u.pathname === '/login' && u.searchParams.get('redirect') === '/account')
  await expect(page.getByRole('heading', { name: 'Masuk' })).toBeVisible()
})

test('form login memvalidasi input di client', async ({ page }) => {
  await page.goto('/login')
  // Tombol aktif setelah hidrasi; mengisi form sebelum itu bisa tertimpa saat hidrasi.
  await expect(page.getByRole('button', { name: 'Masuk' })).toBeEnabled()
  await page.getByLabel('Email').fill('bukan-email')
  await page.getByRole('button', { name: 'Masuk' }).click()
  await expect(page.getByText('Format email tidak valid.')).toBeVisible()
  await expect(page.getByText('Wajib diisi.')).toBeVisible()
})
