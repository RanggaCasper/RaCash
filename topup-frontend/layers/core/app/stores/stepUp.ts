import { defineStore } from 'pinia'

/** Status modal step-up 2FA dan koordinator token (lihat utils/step-up.ts). */
export const useStepUpStore = defineStore('stepUp', () => {
  const visible = ref(false)
  /** true bila token sebelumnya baru saja ditolak server. */
  const retry = ref(false)

  let pending: { resolve: (g: StepUpGrant) => void, reject: (e: unknown) => void } | null = null

  const coordinator = new StepUpCoordinator(opts => new Promise<StepUpGrant>((resolve, reject) => {
    retry.value = opts.retry
    visible.value = true
    pending = { resolve, reject }
  }))

  /** Dipanggil modal setelah POST /auth/step-up berhasil. */
  function grant(g: StepUpGrant) {
    pending?.resolve(g)
    pending = null
    visible.value = false
  }

  /** Dipanggil modal saat user membatalkan. */
  function cancel() {
    pending?.reject(new StepUpCancelled())
    pending = null
    visible.value = false
  }

  /** Menjalankan request; bila 403 STEP_UP_REQUIRED, tampilkan modal lalu ulangi. */
  function run<T>(fn: (stepUpToken?: string) => Promise<T>, isStepUpRequired: (e: unknown) => boolean) {
    return coordinator.run(fn, isStepUpRequired)
  }

  return { visible, retry, grant, cancel, run }
})
