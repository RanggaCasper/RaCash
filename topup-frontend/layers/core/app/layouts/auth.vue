<script setup lang="ts">
// Layout halaman masuk/daftar (design/public/Masuk dan Daftar.dc.html): dua kolom di
// desktop (panel manfaat + form), satu kolom di mobile. Tanpa header/footer situs.
const { public: { siteLegalName } } = useRuntimeConfig()

const perks = [
  { icon: 'i-lucide-badge-percent', text: 'Harga member lebih murah' },
  { icon: 'i-lucide-wallet', text: 'Saldo RaCash untuk bayar sekali ketuk' },
  { icon: 'i-lucide-receipt-text', text: 'Riwayat pesanan tersimpan rapi' },
]
</script>

<template>
  <div class="grid min-h-screen bg-default text-default lg:grid-cols-2">
    <aside
      class="relative hidden flex-col overflow-hidden border-r border-default p-12 lg:flex"
      style="background: repeating-linear-gradient(135deg, rgba(255, 255, 255, .03) 0 10px, transparent 10px 20px), radial-gradient(90% 70% at 20% 10%, rgba(14, 116, 180, .55), transparent 60%), #0f1420"
    >
      <NuxtLink :to="HOME_PATH" class="flex w-fit items-center gap-2 text-highlighted" aria-label="RaCash beranda">
        <span class="grid size-9 place-items-center rounded-[10px] bg-primary text-[19px] font-extrabold text-[#0b0f17]">R</span>
        <span class="text-[21px] font-extrabold tracking-tight">Ra<span class="text-primary">Cash</span></span>
      </NuxtLink>

      <div class="mt-auto max-w-[440px]">
        <h2 class="text-[40px] leading-[1.1] font-extrabold tracking-[-.025em] text-balance text-highlighted">
          Harga member, saldo, dan riwayat dalam satu akun
        </h2>
        <ul class="mt-6 flex flex-col gap-3">
          <li v-for="p in perks" :key="p.text" class="flex items-center gap-3">
            <span class="grid size-10 flex-none place-items-center rounded-[10px] bg-primary/12 text-primary">
              <UIcon :name="p.icon" class="size-[19px]" />
            </span>
            <span class="text-[15px] text-toned">{{ p.text }}</span>
          </li>
        </ul>
      </div>

      <div class="mt-10 text-xs text-muted">
        © {{ new Date().getFullYear() }} {{ siteLegalName }}
      </div>
    </aside>

    <main class="flex min-w-0 flex-col">
      <header class="flex h-16 items-center gap-2 px-4 sm:px-8">
        <LayoutBrandLogo small class="lg:hidden" />
        <NuxtLink :to="HOME_PATH" class="ml-auto flex min-h-11 items-center gap-1.5 text-[13px] font-semibold text-muted hover:text-highlighted">
          Lanjut sebagai tamu
          <UIcon name="i-lucide-arrow-right" />
        </NuxtLink>
      </header>
      <div class="flex flex-1 justify-center px-4 pt-2 pb-10 sm:px-8 sm:pt-6 sm:pb-14 lg:items-center">
        <div class="w-full max-w-[420px]">
          <slot />
        </div>
      </div>
    </main>
  </div>
</template>
