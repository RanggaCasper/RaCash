<script setup lang="ts">
// Header situs (design/public/Navbar.dc.html): logo, menu utama, pencarian, dan
// area akun. Menyesuaikan lebar layar dengan kelas responsif (aman untuk SSR).
const route = useRoute()

const query = ref(typeof route.query.q === 'string' ? route.query.q : '')
const searchOpen = ref(false)
const desktopInput = ref<HTMLInputElement | null>(null)
const mobileInput = ref<HTMLInputElement | null>(null)

async function search() {
  const q = query.value.trim()
  if (!q) return
  searchOpen.value = false
  await navigateTo({ path: GAMES_PATH, query: { q } })
}

async function toggleSearch() {
  searchOpen.value = !searchOpen.value
  if (searchOpen.value) {
    await nextTick()
    mobileInput.value?.focus()
  }
}

// Tekan "/" untuk fokus ke pencarian (seperti petunjuk di kolom pencarian).
function onKeydown(e: KeyboardEvent) {
  if (e.key !== '/' || e.ctrlKey || e.metaKey || e.altKey) return
  const el = e.target as HTMLElement | null
  if (el && (el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName))) return
  e.preventDefault()
  if (desktopInput.value && desktopInput.value.offsetParent !== null) desktopInput.value.focus()
  else void toggleSearch()
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
watch(() => route.fullPath, () => {
  searchOpen.value = false
})

const linkBase = 'rounded-[10px] px-3 py-2.5 hover:text-highlighted'
</script>

<template>
  <header class="sticky top-0 z-30 border-b border-default bg-default/90 backdrop-blur-md">
    <div class="mx-auto flex h-16 max-w-[1248px] items-center gap-6 px-4 lg:px-6">
      <LayoutBrandLogo class="lg:hidden" small />
      <LayoutBrandLogo class="max-lg:hidden" />

      <nav class="hidden flex-none gap-1 text-sm font-semibold whitespace-nowrap lg:flex" aria-label="Menu utama">
        <LayoutLink
          v-for="l in MAIN_NAV"
          :key="l.to"
          :to="l.to"
          :aria-current="isActivePath(route.path, l.to) ? 'page' : undefined"
          :class="[linkBase, l.xlOnly && 'max-xl:hidden', isActivePath(route.path, l.to) ? 'bg-elevated text-highlighted' : 'text-muted']"
        >
          {{ l.label }}
        </LayoutLink>
      </nav>

      <form
        role="search"
        class="ml-auto hidden h-11 max-w-[380px] min-w-0 flex-1 items-center gap-2.5 rounded-[10px] border border-default bg-muted px-3.5 text-muted lg:flex"
        @submit.prevent="search"
      >
        <UIcon name="i-lucide-search" class="size-[18px] flex-none" />
        <input
          ref="desktopInput"
          v-model="query"
          type="search"
          placeholder="Cari game atau produk…"
          aria-label="Cari"
          class="min-w-0 flex-1 bg-transparent text-sm text-highlighted outline-none placeholder:text-muted"
        >
        <kbd class="rounded-md border border-default px-1.5 py-0.5 font-mono text-[11px]">/</kbd>
      </form>

      <div class="ml-auto flex flex-none items-center gap-1 lg:ml-0">
        <button
          type="button"
          class="grid size-11 cursor-pointer place-items-center rounded-[10px] text-highlighted hover:bg-elevated lg:hidden"
          aria-label="Cari"
          :aria-expanded="searchOpen"
          @click="toggleSearch"
        >
          <UIcon name="i-lucide-search" class="size-5" />
        </button>
        <ClientOnly>
          <LayoutAccountArea />
          <template #fallback>
            <div class="h-11 w-24" aria-hidden="true" />
          </template>
        </ClientOnly>
      </div>
    </div>

    <!-- Pencarian mobile -->
    <form
      v-if="searchOpen"
      role="search"
      class="border-t border-default px-4 py-3 lg:hidden"
      @submit.prevent="search"
    >
      <label class="flex h-11 items-center gap-2.5 rounded-[10px] border border-default bg-muted px-3.5 text-muted">
        <UIcon name="i-lucide-search" class="size-[18px] flex-none" />
        <input
          ref="mobileInput"
          v-model="query"
          type="search"
          placeholder="Cari game atau produk…"
          aria-label="Cari"
          class="min-w-0 flex-1 bg-transparent text-sm text-highlighted outline-none placeholder:text-muted"
          @keydown.esc="searchOpen = false"
        >
      </label>
    </form>
  </header>
</template>
