import { defineStore } from 'pinia'
import type { Schema } from '#shared/types/schema'

/** Permission efektif dari GET /me/permissions (RBAC-13). */
export const usePermissionsStore = defineStore('permissions', () => {
  const roles = ref<Schema<'MyPermissions'>['roles']>([])
  const permissions = ref<string[]>([])
  const permVersion = ref(0)

  function set(p: Schema<'MyPermissions'>) {
    roles.value = p.roles
    permissions.value = p.permissions
    permVersion.value = p.perm_version
  }

  function clear() {
    roles.value = []
    permissions.value = []
    permVersion.value = 0
  }

  function can(permission: string) {
    return hasPermission(permissions.value, permission)
  }

  return { roles, permissions, permVersion, set, clear, can }
})
