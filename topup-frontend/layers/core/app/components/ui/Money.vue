<script setup lang="ts">
// <UiMoney :amount="19487" /> → "Rp 19.487"
const props = defineProps<{ amount: number | null | undefined }>()
const text = useMoney(() => props.amount)
</script>

<template>
  <span class="tabular-nums">{{ text }}</span>
</template>
