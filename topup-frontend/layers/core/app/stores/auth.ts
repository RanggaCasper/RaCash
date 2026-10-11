import { defineStore } from 'pinia'
import type { Schema } from '#shared/types/schema'

/** Profil user yang login. Token tidak pernah disimpan di sini (ada di cookie httpOnly). */
export const useAuthStore = defineStore('auth', () => {
  const user = ref<Schema<'Me'> | null>(null)
  /** true setelah status sesi pernah dicek (berhasil atau tidak). */
  const loaded = ref(false)
  /**
   * Login berhasil tetapi token terbatas: staf internal wajib mengaktifkan 2FA dulu
   * (API_CONTRACT §4.7). Profil belum bisa dimuat; hanya /account/security yang boleh dibuka.
   */
  const setupRequired = ref(false)

  function setUser(u: Schema<'Me'> | null) {
    user.value = u
    setupRequired.value = false
    loaded.value = true
  }

  function setSetupRequired() {
    user.value = null
    setupRequired.value = true
    loaded.value = true
  }

  function clear() {
    user.value = null
    setupRequired.value = false
    loaded.value = true
  }

  return { user, loaded, setupRequired, setUser, setSetupRequired, clear }
})
