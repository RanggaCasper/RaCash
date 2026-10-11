import { describe, expect, it } from 'vitest'
import { formatDateTimeWIB, formatRupiah, formatRupiahCompact, formatRupiahTight, maskEmail, maskPhone, userInitials } from '../../layers/core/app/utils/format'

// Intl id-ID memakai spasi non-breaking (U+00A0) setelah "Rp".
const NBSP = new RegExp(String.fromCharCode(0xa0), 'g')
const nbsp = (s: string) => s.replace(NBSP, ' ')

describe('formatRupiah', () => {
  it('memformat integer rupiah tanpa desimal', () => {
    expect(nbsp(formatRupiah(15750))).toBe('Rp 15.750')
    expect(nbsp(formatRupiah(1000000))).toBe('Rp 1.000.000')
    expect(nbsp(formatRupiah(0))).toBe('Rp 0')
  })
})

describe('formatDateTimeWIB', () => {
  it('mengonversi UTC ke WIB (UTC+7)', () => {
    const out = formatDateTimeWIB('2026-10-10T14:47:00Z')
    expect(out).toContain('21.47')
    expect(out).toContain('2026')
    expect(out.endsWith('WIB')).toBe(true)
  })

  it('melewati tengah malam dengan benar', () => {
    expect(formatDateTimeWIB('2026-10-10T18:30:00Z')).toContain('11 Okt 2026')
  })

  it('mengembalikan "-" untuk tanggal tidak valid', () => {
    expect(formatDateTimeWIB('bukan-tanggal')).toBe('-')
  })
})

describe('masking PII', () => {
  it('menyamarkan nomor HP', () => {
    expect(maskPhone('081234567890')).toBe('0812****890')
    expect(maskPhone('0812')).toBe('****')
  })

  it('menyamarkan email', () => {
    expect(maskEmail('pembeli@example.com')).toBe('pe****@example.com')
    expect(maskEmail('rusak')).toBe('****')
  })
})

describe('formatRupiahTight & formatRupiahCompact (header)', () => {
  it('tanpa spasi setelah Rp', () => {
    expect(formatRupiahTight(1254300)).toBe('Rp1.254.300')
    expect(formatRupiahTight(0)).toBe('Rp0')
  })

  it('menyingkat jutaan, membiarkan angka kecil utuh', () => {
    expect(formatRupiahCompact(1254300)).toBe('Rp1,25 jt')
    expect(formatRupiahCompact(2000000)).toBe('Rp2 jt')
    expect(formatRupiahCompact(125000)).toBe('Rp125.000')
  })
})

describe('userInitials', () => {
  it('mengambil inisial nama depan dan belakang', () => {
    expect(userInitials('Bagas Wicaksono')).toBe('BW')
    expect(userInitials('Siti Nur Aini')).toBe('SA')
  })

  it('satu kata → dua huruf; kosong → ?', () => {
    expect(userInitials('rangga')).toBe('RA')
    expect(userInitials('  ')).toBe('?')
    expect(userInitials(undefined)).toBe('?')
  })
})
