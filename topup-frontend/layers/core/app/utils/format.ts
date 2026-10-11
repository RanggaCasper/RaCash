// Format tampilan. Uang dari API selalu integer rupiah; waktu dari API UTC,
// ditampilkan dalam WIB (ARCHITECTURE §3.6).

const rupiah = new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
})

/** 15750 → "Rp 15.750" (spasi mengikuti locale id-ID). */
export function formatRupiah(amount: number): string {
  return rupiah.format(amount)
}

/** 1254300 → "Rp1.254.300" (tanpa spasi; dipakai di header sempit). */
export function formatRupiahTight(amount: number): string {
  return formatRupiah(amount).replace(/\s/g, '')
}

const decimal2 = new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 })

/** 1254300 → "Rp1,25 jt"; di bawah sejuta tetap utuh: 125000 → "Rp125.000". */
export function formatRupiahCompact(amount: number): string {
  if (Math.abs(amount) >= 1_000_000) return `Rp${decimal2.format(amount / 1_000_000)} jt`
  return formatRupiahTight(amount)
}

/** "Bagas Wicaksono" → "BW"; satu kata → dua huruf pertama; kosong → "?". */
export function userInitials(name: string | null | undefined): string {
  const parts = (name ?? '').trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  const letters = parts.length === 1 ? parts[0]!.slice(0, 2) : parts[0]!.charAt(0) + parts[parts.length - 1]!.charAt(0)
  return letters.toUpperCase()
}

const wibDateTime = new Intl.DateTimeFormat('id-ID', {
  timeZone: 'Asia/Jakarta',
  day: '2-digit',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
})

/** "2026-10-10T14:47:00Z" → "10 Okt 2026, 21.47 WIB". */
export function formatDateTimeWIB(iso: string | Date): string {
  const d = typeof iso === 'string' ? new Date(iso) : iso
  if (Number.isNaN(d.getTime())) return '-'
  return `${wibDateTime.format(d)} WIB`
}

/** "081234567890" → "0812****890". */
export function maskPhone(phone: string): string {
  const p = phone.replace(/\s+/g, '')
  if (p.length <= 7) return '*'.repeat(p.length)
  return `${p.slice(0, 4)}****${p.slice(-3)}`
}

/** "pembeli@example.com" → "pe****@example.com". */
export function maskEmail(email: string): string {
  const [local, domain] = email.split('@')
  if (!local || !domain) return '****'
  return `${local.slice(0, 2)}****@${domain}`
}
