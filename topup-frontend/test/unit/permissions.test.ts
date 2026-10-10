import { describe, expect, it } from 'vitest'
import { hasAnyPermission, hasPermission } from '../../layers/core/app/utils/permissions'
import { ApiError, toApiError } from '../../layers/core/app/utils/api-error'

describe('hasPermission', () => {
  it('mengecek permission eksplisit', () => {
    expect(hasPermission(['orders:read'], 'orders:read')).toBe(true)
    expect(hasPermission(['orders:read'], 'orders:refund')).toBe(false)
    expect(hasPermission([], 'orders:read')).toBe(false)
  })

  it('wildcard * meloloskan semua (super_admin)', () => {
    expect(hasPermission(['*'], 'roles:write')).toBe(true)
  })

  it('tidak menganggap prefix resource sebagai wildcard', () => {
    expect(hasPermission(['orders'], 'orders:read')).toBe(false)
  })

  it('hasAnyPermission', () => {
    expect(hasAnyPermission(['orders:read'], ['roles:read', 'orders:read'])).toBe(true)
    expect(hasAnyPermission(['orders:read'], ['roles:read'])).toBe(false)
  })
})

describe('toApiError', () => {
  it('membaca envelope gagal dari error $fetch', () => {
    const err = toApiError({
      status: 422,
      data: {
        success: false,
        error: { code: 'VALIDATION_ERROR', message: 'Data tidak valid.', details: [{ field: 'target.user_id', message: 'Wajib diisi.' }] },
        request_id: 'req_1',
      },
    })
    expect(err).toBeInstanceOf(ApiError)
    expect(err.code).toBe('VALIDATION_ERROR')
    expect(err.requestId).toBe('req_1')
    expect(err.fieldErrors()).toEqual({ 'target.user_id': 'Wajib diisi.' })
  })

  it('error tanpa envelope menjadi NETWORK_ERROR', () => {
    expect(toApiError(new TypeError('fetch failed')).code).toBe('NETWORK_ERROR')
  })
})
