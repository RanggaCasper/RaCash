import type { MaybeRefOrGetter } from 'vue'

/** useMoney(() => order.total) → computed "Rp 19.487". */
export function useMoney(amount: MaybeRefOrGetter<number | null | undefined>) {
  return computed(() => {
    const v = toValue(amount)
    return v === null || v === undefined ? '-' : formatRupiah(v)
  })
}
