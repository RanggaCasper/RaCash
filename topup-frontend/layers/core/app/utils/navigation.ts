// Data navigasi bersama untuk header, tab bawah (mobile), dan footer.
// Sumber desain: design/public/Navbar.dc.html dan Beranda.dc.html.
import { ACCOUNT_PATH, GAMES_PATH, HOME_PATH, LOGIN_PATH, PROMO_PATH, TRACK_PATH } from './routes'

export interface NavLink {
  label: string
  to: string
}

export interface TabItem extends NavLink {
  icon: string
}

export interface FooterColumn {
  title: string
  links: NavLink[]
}

/** Menu utama desktop. "Semua Game" baru tampil di layar ≥ xl (mengikuti desain). */
export const MAIN_NAV: Array<NavLink & { xlOnly?: boolean }> = [
  { label: 'Beranda', to: HOME_PATH },
  { label: 'Lacak Pesanan', to: TRACK_PATH },
  { label: 'Promo', to: PROMO_PATH },
  { label: 'Semua Game', to: GAMES_PATH, xlOnly: true },
]

/** Tab bawah mobile; tab terakhir (Masuk / Akun) bergantung pada status login. */
export const MOBILE_TABS: TabItem[] = [
  { label: 'Beranda', to: HOME_PATH, icon: 'i-lucide-house' },
  { label: 'Cari', to: GAMES_PATH, icon: 'i-lucide-search' },
  { label: 'Lacak', to: TRACK_PATH, icon: 'i-lucide-package-search' },
  { label: 'Promo', to: PROMO_PATH, icon: 'i-lucide-badge-percent' },
]
export const TAB_LOGIN: TabItem = { label: 'Masuk', to: LOGIN_PATH, icon: 'i-lucide-log-in' }
export const TAB_ACCOUNT: TabItem = { label: 'Akun', to: ACCOUNT_PATH, icon: 'i-lucide-circle-user' }

const category = (slug: string) => `${GAMES_PATH}?category=${slug}`

export const FOOTER_COLUMNS: FooterColumn[] = [
  {
    title: 'PRODUK',
    links: [
      { label: 'Game Mobile', to: category('mobile-games') },
      { label: 'Game PC', to: category('pc-games') },
      { label: 'Voucher', to: category('vouchers') },
      { label: 'Pulsa & Data', to: category('pulsa-data') },
      { label: 'Token PLN', to: category('pln') },
    ],
  },
  {
    title: 'BANTUAN',
    links: [
      { label: 'Lacak Pesanan', to: TRACK_PATH },
      { label: 'Cara Top Up', to: '/how-to-top-up' },
      { label: 'FAQ', to: '/faq' },
      { label: 'Hubungi Kami', to: '/contact' },
    ],
  },
  {
    title: 'PERUSAHAAN',
    links: [
      { label: 'Tentang RaCash', to: '/about' },
      { label: 'Program Reseller', to: '/reseller' },
      { label: 'Karier', to: '/careers' },
      { label: 'Blog', to: '/blog' },
    ],
  },
  {
    title: 'LEGAL',
    links: [
      { label: 'Syarat & Ketentuan', to: '/terms' },
      { label: 'Kebijakan Privasi', to: '/privacy' },
      { label: 'Kebijakan Refund', to: '/refund-policy' },
    ],
  },
]

export const PAYMENT_LOGOS = [
  'QRIS', 'BCA', 'BRI', 'BNI', 'MANDIRI', 'PERMATA',
  'DANA', 'OVO', 'GOPAY', 'SHOPEEPAY', 'ALFAMART', 'INDOMARET',
] as const

export const SOCIAL_LINKS = [
  { label: 'Instagram', icon: 'i-lucide-instagram' },
  { label: 'YouTube', icon: 'i-lucide-youtube' },
  { label: 'Twitter/X', icon: 'i-lucide-twitter' },
  { label: 'Facebook', icon: 'i-lucide-facebook' },
] as const

/**
 * Apakah `to` (tanpa query/hash) adalah route statis yang sudah terdaftar? Tautan ke
 * halaman yang belum dibuat dirender sebagai <a> biasa, sehingga router tidak
 * memperingatkan "No match found" dan tidak mem-prefetch rute yang tidak ada.
 */
export function hasStaticRoute(routes: ReadonlyArray<{ path: string }>, to: string): boolean {
  const path = to.split(/[?#]/)[0]
  return routes.some(r => r.path === path)
}

/** Item aktif di navigasi: beranda hanya cocok persis; lainnya cocok dengan path dan anak-anaknya. */
export function isActivePath(current: string, target: string): boolean {
  if (target === HOME_PATH) return current === HOME_PATH
  return current === target || current.startsWith(`${target}/`)
}
