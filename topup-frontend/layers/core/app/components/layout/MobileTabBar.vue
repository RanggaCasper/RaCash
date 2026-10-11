<script setup lang="ts">
// Tab navigasi bawah untuk mobile (< lg). Tab terakhir: Masuk atau Akun.
const route = useRoute()
const auth = useAuthStore()

const lastTab = computed(() => (auth.user || auth.setupRequired ? TAB_ACCOUNT : TAB_LOGIN))
const tabs = computed(() => [...MOBILE_TABS, lastTab.value])
const loginTab = [...MOBILE_TABS, TAB_LOGIN]

const base = 'relative flex min-h-14 flex-col items-center justify-center gap-[3px] text-[11px] font-semibold'
</script>

<template>
  <nav
    class="grid grid-cols-5 border-t border-default bg-muted/95 pb-1.5 backdrop-blur-md lg:hidden"
    aria-label="Navigasi utama"
  >
    <!-- Status login hanya diketahui di client (halaman publik di-cache SWR) -->
    <ClientOnly>
      <LayoutLink
        v-for="t in tabs"
        :key="t.to"
        :to="t.to"
        :aria-current="isActivePath(route.path, t.to) ? 'page' : undefined"
        :class="[base, isActivePath(route.path, t.to) ? 'text-primary' : 'text-muted']"
      >
        <UIcon :name="t.icon" class="size-5" />
        {{ t.label }}
      </LayoutLink>
      <template #fallback>
        <LayoutLink
          v-for="t in loginTab"
          :key="t.to"
          :to="t.to"
          :aria-current="isActivePath(route.path, t.to) ? 'page' : undefined"
          :class="[base, isActivePath(route.path, t.to) ? 'text-primary' : 'text-muted']"
        >
          <UIcon :name="t.icon" class="size-5" />
          {{ t.label }}
        </LayoutLink>
      </template>
    </ClientOnly>
  </nav>
</template>
