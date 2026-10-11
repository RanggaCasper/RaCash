<script setup lang="ts">
// Input password dengan tombol tampilkan/sembunyikan (desain: ikon mata di kanan).
defineProps<{
  autocomplete?: 'current-password' | 'new-password'
  placeholder?: string
  icon?: string
  invalid?: boolean
}>()
const model = defineModel<string>({ default: '' })
const visible = ref(false)
</script>

<template>
  <UInput
    v-model="model"
    :type="visible ? 'text' : 'password'"
    :autocomplete="autocomplete"
    :placeholder="placeholder"
    :icon="icon"
    :color="invalid ? 'error' : undefined"
    size="xl"
    class="w-full"
  >
    <template #trailing>
      <UButton
        type="button"
        variant="ghost"
        color="neutral"
        size="sm"
        :icon="visible ? 'i-lucide-eye-off' : 'i-lucide-eye'"
        :aria-label="visible ? 'Sembunyikan password' : 'Tampilkan password'"
        :aria-pressed="visible"
        @click="visible = !visible"
      />
    </template>
  </UInput>
</template>
