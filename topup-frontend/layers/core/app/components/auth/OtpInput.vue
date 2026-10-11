<script setup lang="ts">
// Kotak kode 6 digit (desain): pindah otomatis, Backspace/panah, dan tempel kode penuh.
const props = defineProps<{ invalid?: boolean, disabled?: boolean }>()
const emit = defineEmits<{ complete: [code: string] }>()
const model = defineModel<string>({ default: '' })

const LENGTH = 6
const inputs = ref<HTMLInputElement[]>([])
const digits = computed(() => Array.from({ length: LENGTH }, (_, i) => model.value[i] ?? ''))

function setDigits(next: string[]) {
  model.value = next.join('').slice(0, LENGTH)
  if (model.value.length === LENGTH) emit('complete', model.value)
}

function focusAt(i: number) {
  inputs.value[Math.max(0, Math.min(LENGTH - 1, i))]?.focus()
}

function onInput(i: number, e: Event) {
  const d = (e.target as HTMLInputElement).value.replace(/\D/g, '').slice(-1)
  const next = [...digits.value]
  next[i] = d
  ;(e.target as HTMLInputElement).value = d
  setDigits(next)
  if (d) focusAt(i + 1)
}

function onKeydown(i: number, e: KeyboardEvent) {
  if (e.key === 'Backspace' && !digits.value[i]) {
    e.preventDefault()
    const next = [...digits.value]
    next[i - 1] = ''
    setDigits(next)
    focusAt(i - 1)
  }
  else if (e.key === 'ArrowLeft') focusAt(i - 1)
  else if (e.key === 'ArrowRight') focusAt(i + 1)
}

function onPaste(e: ClipboardEvent) {
  const text = (e.clipboardData?.getData('text') ?? '').replace(/\D/g, '').slice(0, LENGTH)
  if (!text) return
  e.preventDefault()
  setDigits(text.split(''))
  focusAt(text.length >= LENGTH ? LENGTH - 1 : text.length)
}

defineExpose({ focus: () => focusAt(0) })
</script>

<template>
  <div role="group" aria-label="Kode 6 digit" class="grid grid-cols-6 gap-1.5 sm:gap-2.5">
    <input
      v-for="(d, i) in digits"
      :key="i"
      ref="inputs"
      :value="d"
      :disabled="props.disabled"
      :aria-label="`Digit ${i + 1}`"
      :aria-invalid="props.invalid || undefined"
      :autocomplete="i === 0 ? 'one-time-code' : 'off'"
      inputmode="numeric"
      maxlength="1"
      class="h-[54px] w-full min-w-0 rounded-xl border-[1.5px] bg-muted text-center font-mono text-2xl font-bold text-highlighted outline-none transition-colors sm:h-[60px]"
      :class="props.invalid ? 'border-red-500' : d ? 'border-primary shadow-[0_0_0_3px_rgba(56,189,248,.15)]' : 'border-default focus:border-primary'"
      @input="onInput(i, $event)"
      @keydown="onKeydown(i, $event)"
      @paste="onPaste"
    >
  </div>
</template>
