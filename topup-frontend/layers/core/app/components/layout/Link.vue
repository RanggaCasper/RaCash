<script setup lang="ts">
// <LayoutLink to="/promo">: NuxtLink bila halamannya sudah ada, <a> biasa bila belum.
// Atribut (class, aria-*) diteruskan ke elemen akar.
const props = defineProps<{ to: string }>()
const router = useRouter()
const routable = computed(() => hasStaticRoute(router.getRoutes(), props.to))
</script>

<template>
  <NuxtLink v-if="routable" :to="to">
    <slot />
  </NuxtLink>
  <a v-else :href="to">
    <slot />
  </a>
</template>
