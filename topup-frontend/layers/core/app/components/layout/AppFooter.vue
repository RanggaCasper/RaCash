<script setup lang="ts">
// Footer situs (design/public/Beranda.dc.html): merek, tautan, metode pembayaran.
const { public: { siteLegalName } } = useRuntimeConfig()
const year = new Date().getFullYear()
</script>

<template>
  <footer class="mt-14 border-t border-default bg-[#0e131d]">
    <div class="mx-auto max-w-[1248px] px-4 pt-8 pb-6 lg:px-6">
      <div class="grid grid-cols-2 gap-7 lg:grid-cols-[1.4fr_repeat(4,1fr)]">
        <div class="col-span-2 lg:col-span-1">
          <LayoutBrandLogo />
          <p class="mt-3 max-w-[300px] text-[13px] leading-relaxed text-muted">
            Top up game dan produk digital, diproses otomatis 24 jam.
          </p>
          <div class="mt-3.5 flex gap-2">
            <a
              v-for="s in SOCIAL_LINKS"
              :key="s.label"
              href="#"
              :aria-label="s.label"
              class="grid size-10 place-items-center rounded-[10px] border border-default text-muted hover:text-primary"
            >
              <UIcon :name="s.icon" />
            </a>
          </div>
        </div>

        <div v-for="col in FOOTER_COLUMNS" :key="col.title">
          <div class="text-xs font-bold tracking-[.08em] text-muted">
            {{ col.title }}
          </div>
          <ul class="mt-2.5 flex flex-col gap-0.5">
            <li v-for="l in col.links" :key="l.label">
              <LayoutLink :to="l.to" class="block py-1.5 text-sm text-toned hover:text-primary">
                {{ l.label }}
              </LayoutLink>
            </li>
          </ul>
        </div>
      </div>

      <div class="mt-7 border-t border-elevated pt-5">
        <div class="text-xs font-bold tracking-[.08em] text-muted">
          METODE PEMBAYARAN
        </div>
        <ul class="mt-2.5 flex flex-wrap gap-2">
          <li
            v-for="p in PAYMENT_LOGOS"
            :key="p"
            class="grid h-8 place-items-center rounded-md bg-[#f3f5f9] px-2.5 font-mono text-[11px] font-extrabold text-[#0b0f17]"
          >
            {{ p }}
          </li>
        </ul>
      </div>

      <div class="mt-5 flex flex-wrap justify-between gap-2 text-xs text-muted">
        <span>© {{ year }} {{ siteLegalName }}</span>
        <span>Semua waktu dalam WIB</span>
      </div>
    </div>
  </footer>
</template>
