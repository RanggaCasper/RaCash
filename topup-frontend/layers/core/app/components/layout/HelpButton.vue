<script setup lang="ts">
// Tombol bantuan WhatsApp melayang (design/public/Beranda.dc.html). Nomor CS
// dari NUXT_PUBLIC_SUPPORT_WHATSAPP (format internasional tanpa +, mis. 6281234567890);
// tanpa nomor, tombol tidak ditampilkan.
const { public: { supportWhatsapp } } = useRuntimeConfig()

const href = computed(() => {
  const digits = String(supportWhatsapp ?? '').replace(/\D/g, '')
  return digits ? `https://wa.me/${digits}` : ''
})
</script>

<template>
  <div v-if="href" class="pointer-events-none flex justify-end px-4 pb-3 lg:px-6">
    <a
      :href="href"
      target="_blank"
      rel="noopener noreferrer"
      aria-label="Bantuan WhatsApp untuk pesanan bermasalah"
      class="pointer-events-auto flex h-12 items-center gap-2 rounded-full bg-green-500 pr-4 pl-3 text-sm font-bold text-green-950 shadow-[0_8px_24px_rgba(34,197,94,.35)] hover:bg-green-400 hover:text-green-950"
    >
      <UIcon name="i-lucide-message-circle" class="size-5" />
      <span class="lg:hidden">Bantuan</span>
      <span class="max-lg:hidden">Pesanan bermasalah?</span>
    </a>
  </div>
</template>
