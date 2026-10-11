<script setup lang="ts">
// Panel admin (SPA, routeRules ssr: false). Isi dashboard di issue #19.
definePageMeta({ layout: 'admin', permission: 'dashboard:view' })
useSeoMeta({ title: 'Dashboard — RaCash Admin', robots: 'noindex' })
</script>

<template>
  <div>
    <h1 class="text-2xl font-semibold">
      Dashboard
    </h1>
    <p class="mt-2 text-muted">
      Ringkasan transaksi akan tampil di sini.
    </p>
  </div>
</template>
