<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'

definePageMeta({ middleware: 'guest' })
useSeoMeta({ title: 'Masuk — RaCash', robots: 'noindex' })

const route = useRoute()
const { login, verifyTwoFactor } = useAuth()

const { handleSubmit, defineField, errors, isSubmitting, setErrors } = useForm({
  validationSchema: toTypedSchema(z.object({
    email: z.string({ required_error: 'Wajib diisi.' }).email('Format email tidak valid.'),
    password: z.string({ required_error: 'Wajib diisi.' }).min(1, 'Wajib diisi.'),
  })),
})
const [email, emailAttrs] = defineField('email')
const [password, passwordAttrs] = defineField('password')

// Sebelum hidrasi, klik submit memicu submit form native. Tombol dinonaktifkan
// sampai mounted, dan form memakai method="post" agar isi form (password)
// tidak pernah masuk ke URL.
const hydrated = ref(false)
onMounted(() => {
  hydrated.value = true
})

const challengeToken = ref<string | null>(null)
const otp = ref('')
const formError = ref('')

async function done() {
  const redirect = typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/')
    ? route.query.redirect
    : ACCOUNT_PATH
  await navigateTo(redirect)
}

async function onOutcome(outcome: LoginOutcome) {
  if (outcome.status === 'two_factor') challengeToken.value = outcome.challengeToken
  else if (outcome.status === 'two_factor_setup') await navigateTo(SECURITY_PATH)
  else await done()
}

function showError(e: unknown) {
  const err = toApiError(e)
  if (err.code === 'VALIDATION_ERROR') setErrors(err.fieldErrors())
  else formError.value = err.message
}

const submit = handleSubmit(async (values) => {
  formError.value = ''
  try {
    await onOutcome(await login(values.email, values.password))
  }
  catch (e) {
    showError(e)
  }
})

async function submitOtp() {
  if (!challengeToken.value) return
  formError.value = ''
  try {
    await onOutcome(await verifyTwoFactor(challengeToken.value, otp.value))
  }
  catch (e) {
    showError(e)
  }
}
</script>

<template>
  <UContainer class="flex justify-center py-16">
    <UCard class="w-full max-w-sm">
      <template #header>
        <h1 class="text-xl font-semibold">
          Masuk
        </h1>
      </template>

      <UAlert v-if="formError" color="error" variant="subtle" :title="formError" class="mb-4" />

      <form v-if="!challengeToken" method="post" class="space-y-4" novalidate @submit.prevent="submit">
        <UFormField label="Email" :error="errors.email">
          <UInput v-model="email" v-bind="emailAttrs" type="email" autocomplete="email" class="w-full" />
        </UFormField>
        <UFormField label="Password" :error="errors.password">
          <UInput v-model="password" v-bind="passwordAttrs" type="password" autocomplete="current-password" class="w-full" />
        </UFormField>
        <UButton type="submit" block :loading="isSubmitting" :disabled="!hydrated" label="Masuk" />
      </form>

      <form v-else method="post" class="space-y-4" @submit.prevent="submitOtp">
        <UFormField label="Kode autentikator (6 digit) atau recovery code">
          <UInput v-model="otp" autocomplete="one-time-code" class="w-full" />
        </UFormField>
        <UButton type="submit" block :disabled="!hydrated" label="Verifikasi" />
      </form>
    </UCard>
  </UContainer>
</template>
