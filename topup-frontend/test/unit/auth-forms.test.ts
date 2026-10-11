import { describe, expect, it } from 'vitest'
import {
  formatCountdown,
  maskEmailShort,
  normalizeIdPhone,
  passwordStrength,
  safeRedirect,
} from '../../layers/core/app/utils/auth-forms'

describe('passwordStrength', () => {
  it('kosong → skor 0 dan tidak diterima', () => {
    const s = passwordStrength('')
    expect(s.score).toBe(0)
    expect(s.label).toBe('')
    expect(s.acceptable).toBe(false)
  })

  it('mengikuti desain: 3 aturan + bonus panjang ≥ 12', () => {
    expect(passwordStrength('abcdefg1').score).toBe(2) // 8 karakter + angka
    expect(passwordStrength('Abcdefg1').score).toBe(3) // + huruf besar & kecil
    expect(passwordStrength('Abcdefghij12').score).toBe(4) // + panjang ≥ 12
    expect(passwordStrength('Abcdefghij12').label).toBe('Sangat kuat')
    expect(passwordStrength('abcdefgh').label).toBe('Lemah') // hanya panjang terpenuhi
  })

  it('acceptable = aturan server: ≥ 8, huruf, angka, ≤ 128', () => {
    expect(passwordStrength('abcdefg1').acceptable).toBe(true)
    expect(passwordStrength('abcdefgh').acceptable).toBe(false) // tanpa angka
    expect(passwordStrength('12345678').acceptable).toBe(false) // tanpa huruf
    expect(passwordStrength('abc123').acceptable).toBe(false) // pendek
    expect(passwordStrength(`a1${'x'.repeat(200)}`).acceptable).toBe(false) // terlalu panjang
  })
})

describe('normalizeIdPhone', () => {
  it('menerima berbagai penulisan nomor Indonesia', () => {
    for (const input of ['812 3456 7890', '0812-3456-7890', '+62 812 3456 7890', '6281234567890', '081234567890']) {
      expect(normalizeIdPhone(input)).toBe('081234567890')
    }
  })

  it('menolak nomor yang bukan seluler Indonesia', () => {
    for (const input of ['', '12345', '021 5551234', '+1 415 555 2671', '8123']) {
      expect(normalizeIdPhone(input)).toBeNull()
    }
  })
})

describe('maskEmailShort', () => {
  it('menyamarkan nama pengguna', () => {
    expect(maskEmailShort('bagas.w@gmail.com')).toBe('ba***@gmail.com')
    expect(maskEmailShort('a@x.id')).toBe('a***@x.id')
    expect(maskEmailShort('tanpa-at')).toBe('tanpa-at')
  })
})

describe('safeRedirect', () => {
  it('menerima path internal', () => {
    expect(safeRedirect('/account/security', '/account')).toBe('/account/security')
    expect(safeRedirect('/games?q=ml', '/account')).toBe('/games?q=ml')
  })

  it('menolak open redirect dan nilai bukan string', () => {
    for (const bad of ['https://evil.example', '//evil.example', '/\\evil.example', 'javascript:alert(1)', 'account', undefined, ['/x'], 42]) {
      expect(safeRedirect(bad, '/account')).toBe('/account')
    }
  })
})

describe('formatCountdown', () => {
  it('format m:ss', () => {
    expect(formatCountdown(60)).toBe('1:00')
    expect(formatCountdown(45)).toBe('0:45')
    expect(formatCountdown(5)).toBe('0:05')
    expect(formatCountdown(-3)).toBe('0:00')
  })
})
