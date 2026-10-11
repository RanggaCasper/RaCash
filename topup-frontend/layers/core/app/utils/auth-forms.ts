// Fungsi murni untuk form login/daftar (design/public/Masuk dan Daftar.dc.html).

export interface PasswordRule {
  label: string
  ok: boolean
}

export interface PasswordStrength {
  rules: PasswordRule[]
  /** 0 (kosong) sampai 4. */
  score: number
  label: '' | 'Lemah' | 'Cukup' | 'Kuat' | 'Sangat kuat'
  /** Memenuhi aturan server (API_CONTRACT §4.1): ≥ 8 karakter, mengandung huruf dan angka. */
  acceptable: boolean
}

const LABELS: PasswordStrength['label'][] = ['', 'Lemah', 'Cukup', 'Kuat', 'Sangat kuat']

/** Indikator kekuatan password seperti di desain; `acceptable` mengikuti aturan server. */
export function passwordStrength(pw: string): PasswordStrength {
  const rules: PasswordRule[] = [
    { label: 'Min. 8 karakter', ok: pw.length >= 8 },
    { label: 'Ada angka', ok: /\d/.test(pw) },
    { label: 'Huruf besar & kecil', ok: /\p{Lu}/u.test(pw) && /\p{Ll}/u.test(pw) },
  ]
  const score = pw ? rules.filter(r => r.ok).length + (pw.length >= 12 ? 1 : 0) : 0
  const acceptable = pw.length >= 8 && pw.length <= 128 && /\p{L}/u.test(pw) && /\d/.test(pw)
  return { rules, score, label: LABELS[score]!, acceptable }
}

/**
 * Nomor WhatsApp/HP Indonesia dari input pengguna → format yang diterima server
 * (`08…`), atau null bila bukan nomor seluler yang valid.
 *   "812 3456 7890" → "081234567890";  "+62 812-3456-7890" → "081234567890".
 */
export function normalizeIdPhone(input: string): string | null {
  let digits = input.replace(/\D/g, '')
  if (digits.startsWith('62')) digits = digits.slice(2)
  else if (digits.startsWith('0')) digits = digits.slice(1)
  if (!/^8\d{8,12}$/.test(digits)) return null
  return `0${digits}`
}

/** Tampilan tersamar email untuk pesan konfirmasi: "bagas.w@gmail.com" → "ba***@gmail.com". */
export function maskEmailShort(email: string): string {
  const at = email.indexOf('@')
  if (at < 1) return email
  return `${email.slice(0, Math.min(2, at))}***${email.slice(at)}`
}

/**
 * Tujuan redirect setelah login dari query `?redirect=`. Hanya path internal yang
 * diterima (mencegah open redirect ke situs lain); selain itu pakai `fallback`.
 */
export function safeRedirect(target: unknown, fallback: string): string {
  if (typeof target !== 'string') return fallback
  if (!target.startsWith('/') || target.startsWith('//') || target.includes('\\')) return fallback
  return target
}

/** Detik → "0:45" untuk hitung mundur kirim ulang. */
export function formatCountdown(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds))
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}
