<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'

// Masuk (design/public/Masuk dan Daftar.dc.html): email + password, lalu langkah
// 2FA bila akun memakainya. Google dan OTP WhatsApp di desain belum didukung backend.
definePageMeta({ layout: 'auth', middleware: 'guest' })
useSeoMeta({ title: 'Masuk — RaCash', robots: 'noindex' })

const route = useRoute()
const { login, verifyTwoFactor, resendVerification } = useAuth()
const { public: { turnstileSiteKey } } = useRuntimeConfig()

const { handleSubmit, defineField, errors, isSubmitting, setErrors, values } = useForm({
  validationSchema: toTypedSchema(z.object({
    email: z.string({ required_error: 'Wajib diisi.' }).email('Format email tidak valid.'),
    password: z.string({ required_error: 'Wajib diisi.' }).min(1, 'Wajib diisi.'),
  })),
})
const [email, emailAttrs] = defineField('email')
const [password, passwordAttrs] = defineField('password')

// Tombol dinonaktifkan sampai hidrasi: klik sebelum itu memicu submit native yang
// bisa memasukkan isi form (password) ke URL. Form juga memakai method="post".
const hydrated = ref(false)
onMounted(() => {
  hydrated.value = true
})

const captcha = ref('')
const captchaRef = ref<{ reset: () => void } | null>(null)

type Step = 'credentials' | 'two_factor'
const step = ref<Step>('credentials')
const challengeToken = ref('')
const otp = ref('')
const recovery = ref('')
const useRecovery = ref(false)
const otpInvalid = ref(false)
const busy = ref(false)

const formError = ref('')
const unverified = ref(false)
const resent = ref(false)
const cooldown = useCooldown(60)

async function done() {
  await navigateTo(safeRedirect(route.query.redirect, ACCOUNT_PATH))
}

async function onOutcome(outcome: LoginOutcome) {
  if (outcome.status === 'two_factor') {
    challengeToken.value = outcome.challengeToken
    step.value = 'two_factor'
  }
  else if (outcome.status === 'two_factor_setup') await navigateTo(SECURITY_PATH)
  else await done()
}

function showError(e: unknown) {
  const err = toApiError(e)
  unverified.value = err.code === 'EMAIL_NOT_VERIFIED'
  if (err.code === 'VALIDATION_ERROR') {
    const fields = err.fieldErrors()
    formError.value = fields['X-Captcha-Token'] ?? ''
    setErrors(fields)
  }
  else formError.value = err.message
}

const submit = handleSubmit(async (v) => {
  formError.value = ''
  unverified.value = false
  resent.value = false
  if (turnstileSiteKey && !captcha.value) {
    formError.value = 'Selesaikan verifikasi captcha terlebih dahulu.'
    return
  }
  try {
    await onOutcome(await login(v.email, v.password, captcha.value || undefined))
  }
  catch (e) {
    showError(e)
    captchaRef.value?.reset()
  }
})

async function submitCode() {
  const code = useRecovery.value ? recovery.value.trim() : otp.value
  if (!code || busy.value) return
  busy.value = true
  formError.value = ''
  otpInvalid.value = false
  try {
    await onOutcome(await verifyTwoFactor(challengeToken.value, code))
  }
  catch (e) {
    const err = toApiError(e)
    otpInvalid.value = err.code === 'INVALID_CREDENTIALS'
    formError.value = err.code === 'INVALID_CREDENTIALS' ? 'Kode salah. Periksa kode lalu coba lagi.' : err.message
    // Challenge hangus (kedaluwarsa / 5 kali salah): ulangi dari awal.
    if (err.code === 'UNAUTHENTICATED') step.value = 'credentials'
    otp.value = ''
  }
  finally {
    busy.value = false
  }
}

function backToCredentials() {
  step.value = 'credentials'
  formError.value = ''
  otp.value = ''
  recovery.value = ''
  useRecovery.value = false
  otpInvalid.value = false
}

async function resend() {
  if (!values.email || cooldown.active.value) return
  try {
    await resendVerification(values.email, captcha.value || undefined)
    resent.value = true
    cooldown.start()
  }
  catch (e) {
    showError(e)
  }
  finally {
    captchaRef.value?.reset()
  }
}
</script>

<template>
  <div v-if="step === 'two_factor'">
    <button
      type="button"
      class="-ml-2.5 mb-2 grid size-11 cursor-pointer place-items-center rounded-[10px] text-highlighted hover:bg-elevated"
      aria-label="Kembali"
      @click="backToCredentials"
    >
      <UIcon name="i-lucide-arrow-left" class="size-5" />
    </button>
    <span class="grid size-14 place-items-center rounded-[14px] bg-primary/12 text-primary">
      <UIcon name="i-lucide-shield-check" class="size-[26px]" />
    </span>
    <h1 class="mt-4 text-2xl font-extrabold tracking-tight text-highlighted sm:text-[28px]">
      Verifikasi 2 langkah
    </h1>
    <p class="mt-1.5 text-sm leading-relaxed text-muted">
      {{ useRecovery ? 'Masukkan salah satu kode cadangan yang kamu simpan saat mengaktifkan 2FA.' : 'Masukkan 6 digit kode dari aplikasi authenticator kamu.' }}
    </p>

    <UAlert v-if="formError" class="mt-4" color="error" variant="subtle" icon="i-lucide-circle-x" :title="formError" role="alert" />

    <form method="post" class="mt-5" @submit.prevent="submitCode">
      <AuthOtpInput v-if="!useRecovery" v-model="otp" :invalid="otpInvalid" :disabled="busy" />
      <UInput
        v-else
        v-model="recovery"
        size="xl"
        class="w-full"
        placeholder="ABCDE-FGH23"
        autocomplete="off"
        aria-label="Kode cadangan"
        :ui="{ base: 'font-mono uppercase' }"
      />
      <UButton
        type="submit"
        block
        size="xl"
        class="mt-4 h-[52px] font-extrabold"
        :loading="busy"
        :disabled="!hydrated || (useRecovery ? !recovery.trim() : otp.length < 6)"
        label="Verifikasi"
      />
    </form>
    <button
      type="button"
      class="mt-3 min-h-10 cursor-pointer text-[13px] font-semibold text-muted hover:text-highlighted"
      @click="useRecovery = !useRecovery; formError = ''; otpInvalid = false"
    >
      {{ useRecovery ? 'Pakai kode dari aplikasi' : 'Pakai kode cadangan' }}
    </button>
  </div>

  <div v-else>
    <AuthTabs active="login" />
    <h1 class="mt-6 text-2xl font-extrabold tracking-tight text-highlighted sm:text-[28px]">
      Selamat datang lagi
    </h1>
    <p class="mt-1.5 text-sm text-muted">
      Masuk untuk harga member dan Saldo RaCash.
    </p>

    <UAlert
      v-if="formError"
      class="mt-4"
      color="error"
      variant="subtle"
      icon="i-lucide-circle-x"
      :title="formError"
      role="alert"
    >
      <template v-if="unverified" #description>
        <button
          type="button"
          class="mt-1 cursor-pointer font-semibold text-primary disabled:cursor-not-allowed disabled:text-dimmed"
          :disabled="cooldown.active.value"
          @click="resend"
        >
          {{ cooldown.active.value ? `Kirim ulang dalam ${cooldown.label.value}` : 'Kirim ulang email verifikasi' }}
        </button>
      </template>
    </UAlert>
    <UAlert
      v-if="resent"
      class="mt-4"
      color="success"
      variant="subtle"
      icon="i-lucide-mail-check"
      title="Jika emailmu terdaftar, link verifikasi sudah dikirim."
      role="status"
    />

    <form method="post" class="mt-5 flex flex-col gap-3.5" novalidate :data-hydrated="hydrated" @submit.prevent="submit">
      <UFormField label="Email" :error="errors.email">
        <UInput
          v-model="email"
          v-bind="emailAttrs"
          type="email"
          size="xl"
          icon="i-lucide-at-sign"
          autocomplete="username"
          placeholder="nama@email.com"
          class="w-full"
        />
      </UFormField>
      <UFormField :error="errors.password">
        <template #label>
          <span class="flex w-full items-center justify-between">
            Password
          </span>
        </template>
        <template #hint>
          <LayoutLink :to="FORGOT_PASSWORD_PATH" class="text-[13px] font-semibold text-primary hover:underline">
            Lupa password?
          </LayoutLink>
        </template>
        <AuthPasswordInput
          v-model="password"
          v-bind="passwordAttrs"
          autocomplete="current-password"
          placeholder="••••••••"
          icon="i-lucide-lock"
          :invalid="!!errors.password"
        />
      </UFormField>

      <AuthCaptchaWidget ref="captchaRef" v-model="captcha" />

      <UButton
        type="submit"
        block
        size="xl"
        class="h-[52px] text-base font-extrabold"
        :loading="isSubmitting"
        :disabled="!hydrated"
        label="Masuk"
      />
    </form>

    <p class="mt-5 text-center text-sm text-muted">
      Belum punya akun?
      <LayoutLink :to="REGISTER_PATH" class="font-bold text-primary hover:underline">
        Daftar gratis
      </LayoutLink>
    </p>
  </div>
</template>
