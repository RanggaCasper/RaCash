<script setup lang="ts">
import type { NuxtError } from '#app'

const props = defineProps<{ error: NuxtError }>()

const title = computed(() => {
  switch (props.error.statusCode) {
    case 403: return 'Akses ditolak'
    case 404: return 'Halaman tidak ditemukan'
    default: return 'Terjadi kesalahan'
  }
})

useHead({ title })
</script>

<template>
  <UApp>
    <UContainer class="flex min-h-screen flex-col items-center justify-center gap-4 text-center">
      <p class="text-6xl font-bold text-primary">
        {{ error.statusCode }}
      </p>
      <h1 class="text-2xl font-semibold">
        {{ title }}
      </h1>
      <p v-if="error.statusCode === 403" class="text-muted">
        {{ error.statusMessage }}
      </p>
      <UButton label="Kembali ke beranda" @click="clearError({ redirect: '/' })" />
    </UContainer>
  </UApp>
</template>
