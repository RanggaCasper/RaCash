<script setup lang="ts">
// Beranda (SSR + SWR 60 dtk). Konten katalog menyusul di issue #10 & #20.
useSeoMeta({
  title: 'RaCash — Top Up Game & Voucher',
  description: 'Top up diamond game, voucher, pulsa, dan e-wallet dengan proses instan.',
})
</script>

<template>
  <UContainer class="py-16 text-center">
    <h1 class="text-4xl font-bold">
      Top up cepat, harga bersahabat
    </h1>
    <p class="mt-4 text-muted">
      Diamond game, voucher, pulsa, dan e-wallet. Mulai dari <UiMoney :amount="1000" />.
    </p>
  </UContainer>
</template>
