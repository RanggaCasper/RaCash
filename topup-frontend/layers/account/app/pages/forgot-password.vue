<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'

// Lupa password (design/public/Masuk dan Daftar.dc.html, layar "forgot"). Server selalu
// menjawab sukses (anti-enumerasi), jadi pesannya berbunyi "kalau terdaftar".
definePageMeta({ layout: 'auth', middleware: 'guest' })
useSeoMeta({ title: 'Lupa password — RaCash', robots: 'noindex' })

const { forgotPassword } = useAuth()
const { public: { turnstileSiteKey } } = useRuntimeConfig()

const { handleSubmit, defineField, errors, isSubmitting, setErrors, values } = useForm({
  validationSchema: toTypedSchema(z.object({
    email: z.string({ required_error: 'Wajib diisi.' }).email('Format email tidak valid.'),
  })),
})
const [email, emailAttrs] = defineField('email')

const hydrated = ref(false)
onMounted(() => {
  hydrated.value = true
})

const captcha = ref('')
const captchaRef = ref<{ reset: () => void } | null>(null)
const formError = ref('')
const sent = ref(false)
const cooldown = useCooldown(60)

async function send(v: { email: string }) {
  formError.value = ''
  if (turnstileSiteKey && !captcha.value) {
    formError.value = 'Selesaikan verifikasi captcha terlebih dahulu.'
    return
  }
  try {
    await forgotPassword(v.email, captcha.value || undefined)
    sent.value = true
    cooldown.start()
  }
  catch (e) {
    const err = toApiError(e)
    if (err.code === 'VALIDATION_ERROR') {
      const fields = err.fieldErrors()
      formError.value = fields['X-Captcha-Token'] ?? ''
      setErrors(fields)
    }
    else formError.value = err.message
  }
  finally {
    captchaRef.value?.reset()
  }
}

const submit = handleSubmit(send)
const resend = () => values.email && !cooldown.active.value && send({ email: values.email })
</script>

<template>
  <div>
    <span class="grid size-14 place-items-center rounded-[14px] bg-primary/12 text-primary">
      <UIcon name="i-lucide-key-round" class="size-[26px]" />
    </span>
    <h1 class="mt-4 text-2xl font-extrabold tracking-tight text-highlighted sm:text-[28px]">
      Lupa password
    </h1>

    <UAlert v-if="formError" class="mt-4" color="error" variant="subtle" icon="i-lucide-circle-x" :title="formError" role="alert" />

    <template v-if="!sent">
      <p class="mt-1.5 text-sm leading-relaxed text-muted">
        Masukkan email akunmu. Kami kirim link untuk membuat password baru.
      </p>
      <form method="post" class="mt-5 flex flex-col gap-3.5" novalidate :data-hydrated="hydrated" @submit.prevent="submit">
        <UFormField label="Email" :error="errors.email">
          <UInput
            v-model="email"
            v-bind="emailAttrs"
            type="email"
            size="xl"
            autocomplete="email"
            placeholder="nama@email.com"
            class="w-full"
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
          label="Kirim link reset"
        />
      </form>
    </template>

    <template v-else>
      <UAlert
        class="mt-4"
        color="success"
        variant="subtle"
        icon="i-lucide-mail-check"
        role="status"
      >
        <template #title>
          Kalau <strong>{{ maskEmailShort(values.email ?? '') }}</strong> terdaftar, link reset sudah dikirim.
        </template>
        <template #description>
          Link berlaku 1 jam.
        </template>
      </UAlert>
      <UButton
        block
        size="xl"
        variant="outline"
        color="neutral"
        class="mt-3.5 h-12 font-bold"
        :disabled="cooldown.active.value"
        :label="cooldown.active.value ? `Kirim ulang dalam ${cooldown.label.value}` : 'Kirim ulang link'"
        @click="resend"
      />
    </template>

    <UButton :to="LOGIN_PATH" block variant="ghost" size="lg" class="mt-3 font-bold" icon="i-lucide-arrow-left" label="Kembali ke Masuk" />
  </div>
</template>
