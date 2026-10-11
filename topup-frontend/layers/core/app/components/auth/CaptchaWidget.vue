<script setup lang="ts">
// Widget Cloudflare Turnstile untuk endpoint auth publik. Aktif hanya bila
// NUXT_PUBLIC_TURNSTILE_SITE_KEY diisi; tanpa itu (lokal) tidak menampilkan apa pun
// dan token tetap kosong, sesuai backend yang mematikan captcha di APP_ENV=local.
interface TurnstileApi {
  render: (el: HTMLElement, opts: Record<string, unknown>) => string
  reset: (id?: string) => void
  remove: (id?: string) => void
}

const model = defineModel<string>({ default: '' })
const { public: { turnstileSiteKey } } = useRuntimeConfig()

const el = ref<HTMLElement | null>(null)
let widgetId: string | undefined

const SCRIPT_SRC = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'

function loadScript(): Promise<TurnstileApi> {
  const w = window as unknown as { turnstile?: TurnstileApi }
  if (w.turnstile) return Promise.resolve(w.turnstile)
  return new Promise((resolve, reject) => {
    const s = document.querySelector<HTMLScriptElement>(`script[src="${SCRIPT_SRC}"]`) ?? document.createElement('script')
    s.addEventListener('load', () => resolve((window as unknown as { turnstile: TurnstileApi }).turnstile))
    s.addEventListener('error', () => reject(new Error('Gagal memuat captcha.')))
    if (!s.parentNode) {
      s.src = SCRIPT_SRC
      s.async = true
      document.head.appendChild(s)
    }
  })
}

onMounted(async () => {
  if (!turnstileSiteKey || !el.value) return
  try {
    const api = await loadScript()
    widgetId = api.render(el.value, {
      'sitekey': turnstileSiteKey,
      'theme': 'dark',
      'callback': (token: string) => { model.value = token },
      'expired-callback': () => { model.value = '' },
      'error-callback': () => { model.value = '' },
    })
  }
  catch {
    model.value = ''
  }
})

onBeforeUnmount(() => {
  const w = window as unknown as { turnstile?: TurnstileApi }
  if (widgetId) w.turnstile?.remove(widgetId)
})

/** Token captcha sekali pakai: panggil setelah submit agar user mendapat token baru. */
function reset() {
  model.value = ''
  const w = window as unknown as { turnstile?: TurnstileApi }
  if (widgetId) w.turnstile?.reset(widgetId)
}

defineExpose({ reset })
</script>

<template>
  <div v-if="turnstileSiteKey" ref="el" class="min-h-[65px]" data-testid="captcha" />
</template>
