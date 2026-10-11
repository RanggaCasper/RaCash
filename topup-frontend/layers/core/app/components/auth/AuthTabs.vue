<script setup lang="ts">
// Tab Masuk | Daftar di atas form (desain). Dua halaman terpisah, jadi berupa tautan.
defineProps<{ active: 'login' | 'register' }>()

const base = 'flex h-11 items-center justify-center rounded-[9px] text-sm font-bold transition-colors'
</script>

<template>
  <nav class="grid grid-cols-2 rounded-xl border border-default bg-muted p-1" aria-label="Masuk atau daftar">
    <LayoutLink
      :to="LOGIN_PATH"
      :aria-current="active === 'login' ? 'page' : undefined"
      :class="[base, active === 'login' ? 'bg-elevated text-highlighted' : 'text-muted hover:text-highlighted']"
    >
      Masuk
    </LayoutLink>
    <LayoutLink
      :to="REGISTER_PATH"
      :aria-current="active === 'register' ? 'page' : undefined"
      :class="[base, active === 'register' ? 'bg-elevated text-highlighted' : 'text-muted hover:text-highlighted']"
    >
      Daftar
    </LayoutLink>
  </nav>
</template>
