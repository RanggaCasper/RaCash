<script setup lang="ts">
import type { Schema } from '#shared/types/schema'

// Modal TOTP untuk aksi sensitif (RBAC-08). Muncul saat API menjawab
// 403 STEP_UP_REQUIRED; useApi mengulang request setelah kode diverifikasi.
const stepUp = useStepUpStore()
const api = useApi()

const code = ref('')
const error = ref('')
const busy = ref(false)

watch(() => stepUp.visible, (open) => {
  if (!open) return
  code.value = ''
  error.value = stepUp.retry ? 'Verifikasi sebelumnya tidak berlaku. Masukkan kode terbaru.' : ''
})

async function submit() {
  if (busy.value) return
  if (!/^\d{6}$/.test(code.value)) {
    error.value = 'Masukkan 6 digit kode dari aplikasi autentikator.'
    return
  }
  busy.value = true
  error.value = ''
  try {
    const r = await api.post<Schema<'StepUpResult'>>('/auth/step-up', { code: code.value }, { silent: true })
    stepUp.grant({ token: r.step_up_token, expiresAt: r.expires_at })
  }
  catch (e) {
    const err = toApiError(e)
    error.value = err.fieldErrors().code ?? err.message
    code.value = ''
  }
  finally {
    busy.value = false
  }
}

function onOpenChange(open: boolean) {
  if (!open) stepUp.cancel()
}
</script>

<template>
  <UModal
    :open="stepUp.visible"
    title="Verifikasi 2FA"
    description="Aksi ini sensitif. Masukkan kode 6 digit dari aplikasi autentikator Anda."
    @update:open="onOpenChange"
  >
    <template #body>
      <form id="step-up-form" method="post" class="space-y-4" @submit.prevent="submit">
        <UFormField label="Kode autentikator" :error="error">
          <UInput
            v-model="code"
            inputmode="numeric"
            autocomplete="one-time-code"
            maxlength="6"
            placeholder="123456"
            autofocus
            class="w-full"
          />
        </UFormField>
      </form>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton variant="ghost" label="Batal" @click="stepUp.cancel()" />
        <UButton type="submit" form="step-up-form" :loading="busy" label="Verifikasi" />
      </div>
    </template>
  </UModal>
</template>
