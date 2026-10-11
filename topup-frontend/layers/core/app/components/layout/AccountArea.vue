<script setup lang="ts">
// Area kanan header yang bergantung pada status login. Dirender di client
// (dibungkus ClientOnly oleh AppHeader) karena halaman publik di-cache SWR dan
// tidak boleh berisi data user. Desktop dan mobile berbeda (lihat desain).
const auth = useAuthStore()
const { user, logout } = useAuth()

const initials = computed(() => userInitials(user.value?.name))
const guest = computed(() => !auth.user && !auth.setupRequired)

const outline = 'inline-flex h-11 items-center justify-center rounded-[10px] border border-default px-4 text-sm font-semibold text-highlighted hover:bg-muted'
const solid = 'inline-flex h-11 items-center justify-center rounded-[10px] bg-primary px-4 text-sm font-bold text-[#0b0f17] hover:bg-sky-300'
</script>

<template>
  <!-- Tamu -->
  <div v-if="guest" class="flex flex-none gap-2 whitespace-nowrap">
    <LayoutLink :to="LOGIN_PATH" :class="[outline, 'max-lg:hidden']">
      Masuk
    </LayoutLink>
    <LayoutLink :to="REGISTER_PATH" :class="[solid, 'max-lg:hidden']">
      Daftar
    </LayoutLink>
    <LayoutLink :to="LOGIN_PATH" :class="[solid, 'h-10 px-3.5 text-[13px] lg:hidden']">
      Masuk
    </LayoutLink>
  </div>

  <!-- Token terbatas: staf internal wajib mengaktifkan 2FA dulu -->
  <div v-else-if="auth.setupRequired" class="flex flex-none items-center gap-2">
    <LayoutLink :to="SECURITY_PATH" :class="[outline, 'h-10 text-[13px]']">
      Aktifkan 2FA
    </LayoutLink>
    <button type="button" :class="[outline, 'h-10 cursor-pointer text-[13px]']" @click="logout">
      Keluar
    </button>
  </div>

  <!-- Sudah login -->
  <div v-else class="flex flex-none items-center gap-2">
    <LayoutLink
      :to="DEPOSIT_PATH"
      :aria-label="`Saldo ${formatRupiahTight(user?.balance ?? 0)}, isi saldo`"
      class="hidden h-10 items-center gap-2 rounded-[10px] border border-default bg-muted py-0 pr-1.5 pl-3 text-[13px] font-bold text-highlighted tabular-nums lg:flex"
    >
      <UIcon name="i-lucide-wallet" class="text-primary" />
      {{ formatRupiahTight(user?.balance ?? 0) }}
      <span class="grid size-7 place-items-center rounded-[7px] bg-primary text-[#0b0f17]">
        <UIcon name="i-lucide-plus" class="size-[15px]" />
      </span>
    </LayoutLink>
    <LayoutAccountMenu class="max-lg:hidden" />

    <LayoutLink
      :to="ACCOUNT_PATH"
      :aria-label="`Saldo ${formatRupiahTight(user?.balance ?? 0)}`"
      class="flex h-9 items-center gap-1.5 rounded-full border border-default bg-muted px-2.5 text-xs font-bold text-highlighted tabular-nums lg:hidden"
    >
      <UIcon name="i-lucide-wallet" class="size-3.5 text-primary" />
      {{ formatRupiahCompact(user?.balance ?? 0) }}
    </LayoutLink>
    <LayoutLink :to="ACCOUNT_PATH" aria-label="Akun" class="grid size-11 place-items-center lg:hidden">
      <span class="grid size-8 place-items-center rounded-full bg-linear-to-br from-sky-500 to-indigo-900 text-xs font-extrabold text-highlighted">
        {{ initials }}
      </span>
    </LayoutLink>
  </div>
</template>
