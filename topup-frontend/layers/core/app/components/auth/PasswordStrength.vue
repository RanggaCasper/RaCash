<script setup lang="ts">
// Indikator kekuatan password (desain): 4 batang, tiga aturan, dan label.
const props = defineProps<{ password: string }>()
const strength = computed(() => passwordStrength(props.password))

const barColor = computed(() => {
  switch (strength.value.score) {
    case 1: return 'bg-red-500'
    case 2: return 'bg-amber-500'
    default: return 'bg-green-500'
  }
})
const labelColor = computed(() => ({
  'text-red-500': strength.value.score === 1,
  'text-amber-500': strength.value.score === 2,
  'text-green-500': strength.value.score >= 3,
}))
</script>

<template>
  <div class="mt-2 flex flex-col gap-1.5" data-testid="password-strength">
    <div class="grid grid-cols-4 gap-1" aria-hidden="true">
      <span
        v-for="i in 4"
        :key="i"
        class="h-1 rounded-sm transition-colors"
        :class="i <= strength.score ? barColor : 'bg-accented'"
      />
    </div>
    <ul class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs font-medium">
      <li
        v-for="r in strength.rules"
        :key="r.label"
        class="inline-flex items-center gap-1"
        :class="r.ok ? 'text-green-400' : 'text-muted'"
      >
        <UIcon :name="r.ok ? 'i-lucide-circle-check' : 'i-lucide-circle'" class="size-3" />
        {{ r.label }}
      </li>
      <li class="ml-auto font-bold" :class="labelColor" aria-live="polite">
        {{ strength.label }}
      </li>
    </ul>
  </div>
</template>
