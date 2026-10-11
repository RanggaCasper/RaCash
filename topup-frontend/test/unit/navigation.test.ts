import { describe, expect, it } from 'vitest'
import { hasStaticRoute, isActivePath } from '../../layers/core/app/utils/navigation'

describe('isActivePath', () => {
  it('beranda hanya cocok persis', () => {
    expect(isActivePath('/', '/')).toBe(true)
    expect(isActivePath('/promo', '/')).toBe(false)
  })

  it('cocok dengan path sendiri dan anak-anaknya, bukan awalan kata', () => {
    expect(isActivePath('/games', '/games')).toBe(true)
    expect(isActivePath('/games/mobile-legends', '/games')).toBe(true)
    expect(isActivePath('/gamesx', '/games')).toBe(false)
    expect(isActivePath('/track', '/promo')).toBe(false)
  })
})

describe('hasStaticRoute', () => {
  const routes = [{ path: '/' }, { path: '/login' }, { path: '/account/security' }]

  it('mengenali route statis, mengabaikan query dan hash', () => {
    expect(hasStaticRoute(routes, '/login')).toBe(true)
    expect(hasStaticRoute(routes, '/login?redirect=/account')).toBe(true)
    expect(hasStaticRoute(routes, '/account/security#2fa')).toBe(true)
  })

  it('halaman yang belum dibuat → false', () => {
    expect(hasStaticRoute(routes, '/promo')).toBe(false)
    expect(hasStaticRoute(routes, '/games?category=pln')).toBe(false)
  })
})
