<script setup lang="ts">
const { user, logout } = useAuth()
</script>

<template>
  <div class="flex min-h-screen">
    <aside class="w-60 border-r border-default p-4">
      <NuxtLink to="/admin" class="mb-6 block text-lg font-bold text-primary">
        RaCash Admin
      </NuxtLink>
      <nav class="flex flex-col gap-1">
        <UButton v-can="'dashboard:view'" to="/admin" variant="ghost" icon="i-lucide-layout-dashboard" label="Dashboard" />
        <UButton v-can="'orders:read'" to="/admin/orders" variant="ghost" icon="i-lucide-receipt" label="Order" />
        <UButton v-can="'roles:read'" to="/admin/roles" variant="ghost" icon="i-lucide-shield" label="Role & Permission" />
      </nav>
    </aside>
    <div class="flex flex-1 flex-col">
      <header class="flex h-14 items-center justify-end gap-3 border-b border-default px-6">
        <span class="text-sm text-muted">{{ user?.name }}</span>
        <UButton size="sm" variant="outline" label="Keluar" @click="logout" />
      </header>
      <main class="flex-1 p-6">
        <slot />
      </main>
    </div>
  </div>
</template>
