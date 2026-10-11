<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'

// Daftar (design/public/Masuk dan Daftar.dc.html): nama, email, WhatsApp, password
// dengan indikator kekuatan, persetujuan; lalu layar "Cek email kamu". Kode referral
// dan voucher sambutan di desain belum ada di backend.
definePageMeta({ layout: 'auth', middleware: 'guest' })
useSeoMeta({ title: 'Daftar — RaCash', robots: 'noindex' })

const { register, resendVerification } = useAuth()
const { public: { turnstileSiteKey } } = useRuntimeConfig()

const { handleSubmit, defineField, errors, meta, isSubmitting, setErrors, setFieldError, values } = useForm({
  validationSchema: toTypedSchema(z.object({
    name: z.string({ required_error: 'Wajib diisi.' }).trim().min(1, 'Wajib diisi.').max(120, 'Maksimal 120 karakter.'),
    email: z.string({ required_error: 'Wajib diisi.' }).email('Format email tidak valid.').max(254),
    phone: z.string({ required_error: 'Wajib diisi.' }).refine(v => normalizeIdPhone(v) !== null, 'Nomor WhatsApp tidak valid.'),
    password: z.string({ required_error: 'Wajib diisi.' }).refine(v => passwordStrength(v).acceptable, 'Min. 8 karakter, mengandung huruf dan angka.'),
    agree: z.boolean().refine(v => v, 'Setujui syarat & ketentuan untuk melanjutkan.'),
  })),
  initialValues: { name: '', email: '', phone: '', password: '', agree: false },
})
const [name, nameAttrs] = defineField('name')
const [email, emailAttrs] = defineField('email')
const [phone, phoneAttrs] = defineField('phone')
const [password, passwordAttrs] = defineField('password')
const [agree] = defineField('agree')

// Nomor: hanya angka dan spasi (awalan +62 ditampilkan terpisah).
watch(phone, (v) => {
  const clean = (v ?? '').replace(/[^\d ]/g, '')
  if (clean !== v) phone.value = clean
})

// Tombol dinonaktifkan sampai hidrasi (lihat catatan di login.vue).
const hydrated = ref(false)
onMounted(() => {
  hydrated.value = true
})

const captcha = ref('')
const captchaRef = ref<{ reset: () => void } | null>(null)
const formError = ref('')
const emailTaken = ref(false)
const sent = ref(false)
const resent = ref(false)
const cooldown = useCooldown(60)

watch(email, () => {
  emailTaken.value = false
})

const submit = handleSubmit(async (v) => {
  formError.value = ''
  emailTaken.value = false
  if (turnstileSiteKey && !captcha.value) {
    formError.value = 'Selesaikan verifikasi captcha terlebih dahulu.'
    return
  }
  try {
    await register(
      { name: v.name.trim(), email: v.email, phone: normalizeIdPhone(v.phone) ?? v.phone, password: v.password },
      captcha.value || undefined,
    )
    sent.value = true
    cooldown.start()
  }
  catch (e) {
    const err = toApiError(e)
    if (err.code === 'CONFLICT') {
      emailTaken.value = true
      setFieldError('email', 'Email sudah terdaftar.')
    }
    else if (err.code === 'VALIDATION_ERROR') {
      const fields = err.fieldErrors()
      formError.value = fields['X-Captcha-Token'] ?? ''
      setErrors(fields)
    }
    else formError.value = err.message
    captchaRef.value?.reset()
  }
})

async function resend() {
  if (cooldown.active.value) return
  formError.value = ''
  try {
    await resendVerification(values.email ?? '', captcha.value || undefined)
    resent.value = true
    cooldown.start()
  }
  catch (e) {
    formError.value = toApiError(e).message
  }
  finally {
    captchaRef.value?.reset()
  }
}

function editEmail() {
  sent.value = false
  resent.value = false
  formError.value = ''
}
</script>

<template>
  <div v-if="sent">
    <span class="grid size-14 place-items-center rounded-[14px] bg-primary/12 text-primary">
      <UIcon name="i-lucide-mail" class="size-[26px]" />
    </span>
    <h1 class="mt-4 text-2xl font-extrabold tracking-tight text-highlighted sm:text-[28px]">
      Cek email kamu
    </h1>
    <p class="mt-1.5 text-sm leading-relaxed text-muted">
      Kami mengirim link verifikasi ke <strong class="text-highlighted">{{ maskEmailShort(values.email ?? '') }}</strong>.
      Klik link tersebut untuk mengaktifkan akun.
    </p>

    <UAlert v-if="formError" class="mt-4" color="error" variant="subtle" icon="i-lucide-circle-x" :title="formError" role="alert" />
    <UAlert
      v-else-if="resent"
      class="mt-4"
      color="success"
      variant="subtle"
      icon="i-lucide-mail-check"
      title="Email verifikasi dikirim ulang."
      role="status"
    />

    <div class="mt-5 flex flex-col gap-2">
      <UButton
        block
        size="xl"
        variant="outline"
        color="neutral"
        class="h-12 font-bold"
        :disabled="cooldown.active.value"
        :label="cooldown.active.value ? `Kirim ulang dalam ${cooldown.label.value}` : 'Kirim ulang email'"
        @click="resend"
      />
      <UButton :to="LOGIN_PATH" block size="xl" variant="ghost" class="h-12 font-bold" label="Ke halaman masuk" />
    </div>
    <p class="mt-4 text-[13px] leading-relaxed text-muted">
      Tidak ada di inbox? Cek folder Spam atau Promosi. Salah email?
      <button type="button" class="cursor-pointer font-bold text-primary hover:underline" @click="editEmail">
        Ubah email
      </button>
    </p>
  </div>

  <div v-else>
    <AuthTabs active="register" />
    <h1 class="mt-6 text-2xl font-extrabold tracking-tight text-highlighted sm:text-[28px]">
      Buat akun gratis
    </h1>
    <p class="mt-1.5 text-sm text-muted">
      Langsung dapat harga member dan Saldo RaCash.
    </p>

    <UAlert v-if="formError" class="mt-4" color="error" variant="subtle" icon="i-lucide-circle-x" :title="formError" role="alert" />

    <form method="post" class="mt-5 flex flex-col gap-3.5" novalidate :data-hydrated="hydrated" @submit.prevent="submit">
      <UFormField label="Nama" :error="errors.name">
        <UInput v-model="name" v-bind="nameAttrs" size="xl" autocomplete="name" placeholder="Nama panggilan" class="w-full" />
      </UFormField>

      <UFormField label="Email" :error="errors.email">
        <UInput v-model="email" v-bind="emailAttrs" type="email" size="xl" autocomplete="email" placeholder="nama@email.com" class="w-full" />
        <template v-if="emailTaken" #error>
          Email sudah terdaftar.
          <LayoutLink :to="LOGIN_PATH" class="font-bold text-primary hover:underline">
            Masuk saja
          </LayoutLink>
        </template>
      </UFormField>

      <UFormField label="Nomor WhatsApp" :error="errors.phone" help="Untuk bukti transaksi dan informasi pesanan.">
        <UInput
          v-model="phone"
          v-bind="phoneAttrs"
          size="xl"
          inputmode="tel"
          autocomplete="tel-national"
          placeholder="812 3456 7890"
          class="w-full"
        >
          <template #leading>
            <span class="font-bold text-muted">+62</span>
          </template>
        </UInput>
      </UFormField>

      <UFormField label="Password" :error="errors.password">
        <AuthPasswordInput
          v-model="password"
          v-bind="passwordAttrs"
          autocomplete="new-password"
          placeholder="Min. 8 karakter"
          :invalid="!!errors.password"
        />
        <AuthPasswordStrength :password="password ?? ''" />
      </UFormField>

      <div>
        <UCheckbox v-model="agree" :ui="{ label: 'text-[13px] leading-relaxed text-toned' }">
          <template #label>
            Saya setuju dengan
            <LayoutLink :to="TERMS_PATH" class="text-primary hover:underline">
              Syarat &amp; Ketentuan
            </LayoutLink>
            dan
            <LayoutLink :to="PRIVACY_PATH" class="text-primary hover:underline">
              Kebijakan Privasi
            </LayoutLink>
            RaCash.
          </template>
        </UCheckbox>
        <p v-if="errors.agree" class="mt-1 text-xs text-error">
          {{ errors.agree }}
        </p>
      </div>

      <AuthCaptchaWidget ref="captchaRef" v-model="captcha" />

      <UButton
        type="submit"
        block
        size="xl"
        class="h-[52px] text-base font-extrabold"
        :loading="isSubmitting"
        :disabled="!hydrated || !meta.valid"
        label="Daftar"
      />
    </form>

    <p class="mt-5 text-center text-sm text-muted">
      Sudah punya akun?
      <LayoutLink :to="LOGIN_PATH" class="font-bold text-primary hover:underline">
        Masuk
      </LayoutLink>
    </p>
  </div>
</template>
