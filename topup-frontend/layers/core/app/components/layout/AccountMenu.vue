<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'

// Menu avatar desktop. Dropdown Nuxt UI: menutup dengan Esc / klik di luar.
// "Keluar" selalu paling bawah, berwarna merah (design/public/Navbar.dc.html).
const { user, logout } = useAuth()
const router = useRouter()

// Halaman yang belum ada memakai href biasa (lihat LayoutLink).
const link = (path: string) => (hasStaticRoute(router.getRoutes(), path) ? { to: path } : { href: path })

const initials = computed(() => userInitials(user.value?.name))
const isReseller = computed(() => user.value?.roles.includes('reseller') === true)

const items = computed<DropdownMenuItem[][]>(() => {
  const u = user.value
  if (!u) return []
  return [
    [{ type: 'label', label: u.name }, { type: 'label', label: u.email, class: 'font-normal text-muted' }],
    [
      { label: 'Dashboard', icon: 'i-lucide-layout-dashboard', ...link(ACCOUNT_PATH) },
      { label: 'Riwayat Pesanan', icon: 'i-lucide-receipt-text', ...link(ORDERS_PATH) },
      { label: 'Isi Saldo', icon: 'i-lucide-circle-plus', ...link(DEPOSIT_PATH) },
      ...(isReseller.value ? [{ label: 'API & Callback', icon: 'i-lucide-code-xml', ...link(API_ACCESS_PATH) }] : []),
      { label: 'Keamanan', icon: 'i-lucide-shield-check', ...link(SECURITY_PATH) },
    ],
    [{ label: 'Keluar', icon: 'i-lucide-log-out', color: 'error', onSelect: () => logout() }],
  ]
})
</script>

<template>
  <UDropdownMenu :items="items" :content="{ align: 'end' }" :ui="{ content: 'w-64' }">
    <button
      type="button"
      aria-label="Menu akun"
      class="flex h-11 cursor-pointer items-center gap-2 rounded-[10px] py-0 pr-2 pl-1 text-highlighted hover:bg-elevated"
    >
      <span class="grid size-[34px] place-items-center rounded-full bg-linear-to-br from-sky-500 to-indigo-900 text-[13px] font-extrabold">
        {{ initials }}
      </span>
      <span class="text-left leading-tight">
        <span class="block text-[13px] font-bold">{{ user?.name?.split(' ')[0] }}</span>
        <span class="block text-[11px] font-bold" :class="isReseller ? 'text-amber-400' : 'text-muted'">{{ user?.tier.name }}</span>
      </span>
      <UIcon name="i-lucide-chevron-down" class="text-muted" />
    </button>
  </UDropdownMenu>
</template>
