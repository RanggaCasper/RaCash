import { expect, test } from '@playwright/test'

// Layout dari design/public/Navbar.dc.html dan Beranda.dc.html, tanpa Go API
// (tamu). Desktop memakai menu atas, mobile (< 1024px) memakai tab bawah.

test.describe('header & navigasi', () => {
  test('desktop: logo, menu utama, pencarian, dan tombol Masuk/Daftar', async ({ page, isMobile }) => {
    test.skip(isMobile, 'khusus desktop')
    await page.goto('/')
    const nav = page.getByRole('navigation', { name: 'Menu utama' })
    await expect(nav.getByRole('link', { name: 'Beranda' })).toHaveAttribute('aria-current', 'page')
    for (const name of ['Lacak Pesanan', 'Promo']) {
      await expect(nav.getByRole('link', { name })).toBeVisible()
    }
    await expect(page.getByRole('search').getByPlaceholder('Cari game atau produk…')).toBeVisible()
    const header = page.getByRole('banner')
    await expect(header.getByRole('link', { name: 'Masuk' })).toHaveAttribute('href', '/login')
    await expect(header.getByRole('link', { name: 'Daftar' })).toHaveAttribute('href', '/register')
    await expect(page.getByRole('navigation', { name: 'Navigasi utama' })).toBeHidden()
  })

  test('mobile: tab bawah dengan tab Masuk, tanpa menu atas', async ({ page, isMobile }) => {
    test.skip(!isMobile, 'khusus mobile')
    await page.goto('/')
    const tabs = page.getByRole('navigation', { name: 'Navigasi utama' })
    await expect(tabs.getByRole('link')).toHaveText(['Beranda', 'Cari', 'Lacak', 'Promo', 'Masuk'])
    await expect(tabs.getByRole('link', { name: 'Beranda' })).toHaveAttribute('aria-current', 'page')
    await expect(page.getByRole('navigation', { name: 'Menu utama' })).toBeHidden()
  })

  test('mobile: tombol cari membuka kolom pencarian', async ({ page, isMobile }) => {
    test.skip(!isMobile, 'khusus mobile')
    await page.goto('/')
    const search = page.getByRole('button', { name: 'Cari' })
    await expect(search).toBeEnabled()
    await search.click()
    await expect(page.getByRole('search').getByPlaceholder('Cari game atau produk…')).toBeVisible()
  })

  test('pencarian menuju /games?q=', async ({ page, isMobile }) => {
    test.skip(isMobile, 'khusus desktop')
    await page.goto('/')
    const input = page.getByRole('search').getByPlaceholder('Cari game atau produk…')
    await input.fill('mobile legends')
    await input.press('Enter')
    await expect(page).toHaveURL(u => u.pathname === '/games' && u.searchParams.get('q') === 'mobile legends')
  })

  test('tombol "/" memfokuskan pencarian di desktop', async ({ page, isMobile }) => {
    test.skip(isMobile, 'khusus desktop')
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    await page.keyboard.press('/')
    await expect(page.getByRole('search').getByPlaceholder('Cari game atau produk…')).toBeFocused()
  })
})

test.describe('footer', () => {
  test('kolom tautan, metode pembayaran, dan hak cipta', async ({ page }) => {
    await page.goto('/')
    const footer = page.getByRole('contentinfo')
    for (const title of ['PRODUK', 'BANTUAN', 'PERUSAHAAN', 'LEGAL', 'METODE PEMBAYARAN']) {
      await expect(footer.getByText(title, { exact: true })).toBeVisible()
    }
    await expect(footer.getByRole('link', { name: 'Lacak Pesanan' })).toHaveAttribute('href', '/track')
    for (const logo of ['QRIS', 'BCA', 'GOPAY', 'INDOMARET']) {
      await expect(footer.getByText(logo, { exact: true })).toBeVisible()
    }
    await expect(footer.getByText('PT RaCash Digital Indonesia')).toBeVisible()
  })

  test('tombol bantuan WhatsApp tersembunyi bila nomor CS belum diatur', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('link', { name: /Bantuan WhatsApp/ })).toHaveCount(0)
  })
})
