import { defineStore } from 'pinia'
import type { Schema } from '#shared/types/schema'

/** Profil user yang login. Token tidak pernah disimpan di sini (ada di cookie httpOnly). */
export const useAuthStore = defineStore('auth', () => {
  const user = ref<Schema<'Me'> | null>(null)
  /** true setelah status sesi pernah dicek (berhasil atau tidak). */
  const loaded = ref(false)

  function setUser(u: Schema<'Me'> | null) {
    user.value = u
    loaded.value = true
  }

  function clear() {
    user.value = null
    loaded.value = true
  }

  return { user, loaded, setUser, clear }
})
