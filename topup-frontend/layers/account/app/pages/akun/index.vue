<script setup lang="ts">
// Dashboard member (SSR tanpa cache, butuh login). Isi lengkap di issue #26.
definePageMeta({ middleware: 'auth' })
useSeoMeta({ title: 'Akun — RaCash', robots: 'noindex' })

const { user } = useAuth()
</script>

<template>
  <UContainer class="py-10">
    <h1 class="text-2xl font-semibold">
      Halo, {{ user?.name }}
    </h1>
    <UCard class="mt-6 max-w-sm">
      <p class="text-sm text-muted">
        Saldo
      </p>
      <p class="text-2xl font-bold">
        <UiMoney :amount="user?.balance" />
      </p>
    </UCard>
  </UContainer>
</template>
