import type { MaybeRefOrGetter } from 'vue'

/** useCan('orders:refund') → computed boolean. Hanya untuk UX; backend penentu akhir. */
export function useCan(permission: MaybeRefOrGetter<string>) {
  const perms = usePermissionsStore()
  return computed(() => perms.can(toValue(permission)))
}
