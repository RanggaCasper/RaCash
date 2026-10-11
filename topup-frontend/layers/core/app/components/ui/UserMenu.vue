<script setup lang="ts">
// Menu akun di header. Dirender di client (ClientOnly di layout) karena
// halaman publik di-cache SWR dan tidak boleh berisi data user.
const { user, loggedIn, logout } = useAuth()
</script>

<template>
  <div class="flex items-center gap-2">
    <template v-if="loggedIn">
      <UButton :to="user ? '/akun' : SECURITY_PATH" variant="ghost" icon="i-lucide-user" :label="user?.name ?? 'Akun'" />
      <UButton v-can="'dashboard:view'" to="/admin" variant="ghost" label="Admin" />
      <UButton variant="outline" label="Keluar" @click="logout" />
    </template>
    <UButton v-else to="/masuk" label="Masuk" />
  </div>
</template>
