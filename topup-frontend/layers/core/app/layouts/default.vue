<script setup lang="ts">
// Layout utama storefront dan area member (design/public/*.dc.html):
// header lengket, konten, footer, tombol bantuan, dan tab bawah di mobile.
</script>

<template>
  <div class="flex min-h-screen flex-col bg-default text-default">
    <LayoutAppHeader />
    <main class="flex-1">
      <slot />
    </main>
    <LayoutAppFooter />
    <div class="pointer-events-none sticky bottom-0 z-40">
      <LayoutHelpButton />
      <div class="pointer-events-auto">
        <LayoutMobileTabBar />
      </div>
    </div>
  </div>
</template>
