/**
 * Hitung mundur untuk tombol "Kirim ulang" (desain: 60 detik).
 *   const cd = useCooldown(60); cd.start(); cd.active.value; cd.label.value // "0:45"
 */
export function useCooldown(seconds = 60) {
  const remaining = ref(0)
  let timer: ReturnType<typeof setInterval> | null = null

  function stop() {
    if (timer) clearInterval(timer)
    timer = null
  }

  function start() {
    stop()
    remaining.value = seconds
    timer = setInterval(() => {
      remaining.value -= 1
      if (remaining.value <= 0) {
        remaining.value = 0
        stop()
      }
    }, 1000)
  }

  onBeforeUnmount(stop)

  return {
    remaining,
    active: computed(() => remaining.value > 0),
    label: computed(() => formatCountdown(remaining.value)),
    start,
  }
}
