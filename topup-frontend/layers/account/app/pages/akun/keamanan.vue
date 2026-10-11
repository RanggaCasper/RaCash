<script setup lang="ts">
import { renderSVG } from 'uqr'
import type { Schema } from '#shared/types/schema'

// Setup, aktivasi, dan penonaktifan 2FA TOTP (USR-06). Juga satu-satunya halaman
// yang bisa dibuka staf internal sebelum 2FA aktif (token terbatas).
definePageMeta({ middleware: 'auth' })
useSeoMeta({ title: 'Keamanan — RaCash', robots: 'noindex' })

const api = useApi()
const auth = useAuthStore()
const { fetchSession } = useAuth()
const toast = useToast()

const status = ref<Schema<'TwoFactorStatus'> | null>(null)
const setup = ref<Schema<'TwoFactorSetup'> | null>(null)
const recoveryCodes = ref<string[]>([])
const saved = ref(false)
const code = ref('')
const formError = ref('')
const busy = ref(false)

const qrSvg = computed(() => (setup.value ? renderSVG(setup.value.otpauth_url, { border: 1 }) : ''))
const enabled = computed(() => status.value?.enabled === true)

async function loadStatus() {
  status.value = await api.get<Schema<'TwoFactorStatus'>>('/auth/2fa/status')
}

onMounted(loadStatus)

async function run(action: () => Promise<void>) {
  busy.value = true
  formError.value = ''
  try {
    await action()
  }
  catch (e) {
    const err = toApiError(e)
    // Pembatalan modal step-up bukan error yang perlu ditampilkan.
    if (err.code !== 'STEP_UP_REQUIRED') formError.value = err.fieldErrors().code ?? err.message
  }
  finally {
    busy.value = false
  }
}

const startSetup = () => run(async () => {
  setup.value = await api.post<Schema<'TwoFactorSetup'>>('/auth/2fa/setup', undefined, { silent: true })
  code.value = ''
})

const confirmEnable = () => run(async () => {
  const r = await api.post<Schema<'RecoveryCodes'>>('/auth/2fa/enable', { code: code.value }, { silent: true })
  recoveryCodes.value = r.recovery_codes
  saved.value = false
  setup.value = null
  code.value = ''
  await loadStatus()
  // Token terbatas langsung berlaku penuh; muat ulang profil dan permission.
  await fetchSession()
})

const disable = () => run(async () => {
  // 403 STEP_UP_REQUIRED ditangani useApi: modal TOTP, lalu request diulang.
  await api.post('/auth/2fa/disable', undefined, { silent: true })
  toast.add({ title: '2FA dimatikan.', color: 'success' })
  await loadStatus()
  await fetchSession()
})

async function copyCodes() {
  await navigator.clipboard.writeText(recoveryCodes.value.join('\n'))
  toast.add({ title: 'Recovery code disalin.', color: 'success' })
}

async function finish() {
  recoveryCodes.value = []
  if (auth.user) await navigateTo('/akun')
}
</script>

<template>
  <UContainer class="max-w-xl py-10">
    <h1 class="text-2xl font-semibold">
      Keamanan akun
    </h1>

    <UAlert
      v-if="auth.setupRequired || (status?.required && !enabled)"
      color="warning"
      variant="subtle"
      class="mt-4"
      title="Akun Anda wajib mengaktifkan 2FA"
      description="Selesaikan pengaturan autentikasi dua langkah untuk mengakses fitur lainnya."
    />
    <UAlert v-if="formError" color="error" variant="subtle" class="mt-4" :title="formError" />

    <!-- Recovery code: ditampilkan sekali setelah 2FA aktif -->
    <UCard v-if="recoveryCodes.length" class="mt-6">
      <template #header>
        <h2 class="font-semibold">
          Simpan recovery code Anda
        </h2>
      </template>
      <p class="text-sm text-muted">
        Kode ini satu-satunya cara masuk bila Anda kehilangan ponsel. Tiap kode hanya berlaku sekali
        dan <strong>tidak akan ditampilkan lagi</strong>.
      </p>
      <ul class="mt-4 grid grid-cols-2 gap-2 font-mono text-sm" data-testid="recovery-codes">
        <li v-for="c in recoveryCodes" :key="c">
          {{ c }}
        </li>
      </ul>
      <div class="mt-4 flex flex-wrap items-center gap-3">
        <UButton variant="outline" icon="i-lucide-copy" label="Salin semua" @click="copyCodes" />
        <UCheckbox v-model="saved" label="Saya sudah menyimpannya di tempat aman" />
      </div>
      <template #footer>
        <UButton :disabled="!saved" label="Selesai" @click="finish" />
      </template>
    </UCard>

    <template v-else-if="status">
      <!-- Belum aktif -->
      <UCard v-if="!enabled" class="mt-6">
        <template #header>
          <h2 class="font-semibold">
            Autentikasi dua langkah (2FA)
          </h2>
        </template>

        <div v-if="!setup" class="space-y-4">
          <p class="text-sm text-muted">
            Lindungi akun Anda dengan kode dari aplikasi autentikator (Google Authenticator, Authy, dsb.).
          </p>
          <UButton :loading="busy" label="Mulai pengaturan" @click="startSetup" />
        </div>

        <form v-else method="post" class="space-y-4" @submit.prevent="confirmEnable">
          <p class="text-sm">
            1. Pindai QR code dengan aplikasi autentikator Anda.
          </p>
          <!-- eslint-disable-next-line vue/no-v-html -- SVG dibuat lokal oleh uqr dari URL milik server -->
          <div class="w-48 rounded bg-white p-2" data-testid="qr" v-html="qrSvg" />
          <p class="text-sm text-muted">
            Atau masukkan kunci ini secara manual:
            <code class="break-all font-mono" data-testid="secret">{{ setup.secret }}</code>
          </p>
          <p class="text-sm">
            2. Masukkan kode 6 digit yang tampil di aplikasi.
          </p>
          <UFormField label="Kode autentikator">
            <UInput v-model="code" inputmode="numeric" autocomplete="one-time-code" maxlength="6" class="w-full" />
          </UFormField>
          <UButton type="submit" :loading="busy" :disabled="code.length !== 6" label="Aktifkan 2FA" />
        </form>
      </UCard>

      <!-- Sudah aktif -->
      <UCard v-else class="mt-6">
        <template #header>
          <h2 class="font-semibold">
            2FA aktif
          </h2>
        </template>
        <p class="text-sm text-muted">
          Sisa recovery code: <strong>{{ status.recovery_codes_remaining }}</strong>
        </p>
        <p v-if="status.required" class="mt-2 text-sm text-muted">
          2FA wajib untuk akun internal dan tidak dapat dimatikan.
        </p>
        <template #footer>
          <UButton
            v-if="!status.required"
            color="error"
            variant="outline"
            :loading="busy"
            label="Matikan 2FA"
            @click="disable"
          />
        </template>
      </UCard>
    </template>
  </UContainer>
</template>
