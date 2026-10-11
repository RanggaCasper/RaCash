<script setup lang="ts">
const { public: { siteName } } = useRuntimeConfig()
</script>

<template>
  <div class="flex min-h-screen flex-col">
    <header class="border-b border-default">
      <UContainer class="flex h-16 items-center justify-between">
        <NuxtLink to="/" class="text-xl font-bold text-primary">
          {{ siteName }}
        </NuxtLink>
        <ClientOnly>
          <UiUserMenu />
        </ClientOnly>
      </UContainer>
    </header>
    <main class="flex-1">
      <slot />
    </main>
    <footer class="border-t border-default py-6 text-center text-sm text-muted">
      © {{ new Date().getFullYear() }} {{ siteName }}
    </footer>
  </div>
</template>
