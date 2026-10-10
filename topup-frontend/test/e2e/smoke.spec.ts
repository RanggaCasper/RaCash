import { expect, test } from '@playwright/test'

// Smoke test tanpa Go API: halaman publik, redirect login, dan SPA admin.

test('beranda tampil dengan tombol Masuk', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { level: 1 })).toContainText('Top up cepat')
  await expect(page.getByRole('link', { name: 'Masuk' })).toBeVisible()
})

test('/akun tanpa login diarahkan ke /masuk dengan redirect', async ({ page }) => {
  await page.goto('/akun')
  await expect(page).toHaveURL(u => u.pathname === '/masuk' && u.searchParams.get('redirect') === '/akun')
  await expect(page.getByRole('heading', { name: 'Masuk' })).toBeVisible()
})

test('/admin (SPA) tanpa login diarahkan ke /masuk', async ({ page }) => {
  await page.goto('/admin')
  await expect(page).toHaveURL(u => u.pathname === '/masuk' && u.searchParams.get('redirect') === '/admin')
})

test('form login memvalidasi input di client', async ({ page }) => {
  await page.goto('/masuk')
  await page.getByLabel('Email').fill('bukan-email')
  await page.getByRole('button', { name: 'Masuk' }).click()
  await expect(page.getByText('Format email tidak valid.')).toBeVisible()
  await expect(page.getByText('Wajib diisi.')).toBeVisible()
})
