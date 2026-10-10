/** Wildcard super_admin (API_CONTRACT §5.1). */
export const WILDCARD_PERMISSION = '*'

/**
 * Cek permission efektif. Hanya untuk UX (sembunyikan menu/tombol);
 * keputusan akhir selalu di backend (RULES §6.4).
 */
export function hasPermission(granted: readonly string[], permission: string): boolean {
  return granted.includes(WILDCARD_PERMISSION) || granted.includes(permission)
}

/** true bila punya minimal satu dari daftar permission. */
export function hasAnyPermission(granted: readonly string[], permissions: readonly string[]): boolean {
  return permissions.some(p => hasPermission(granted, p))
}
