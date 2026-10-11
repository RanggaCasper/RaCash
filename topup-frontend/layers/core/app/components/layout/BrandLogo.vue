<script setup lang="ts">
// Logo RaCash (design/public/Navbar.dc.html). `small` = versi header mobile.
defineProps<{ small?: boolean }>()
</script>

<template>
  <NuxtLink :to="HOME_PATH" class="flex flex-none items-center gap-2 text-highlighted" aria-label="RaCash beranda">
    <span
      class="grid place-items-center rounded-[9px] bg-primary font-extrabold text-[#0b0f17]"
      :class="small ? 'size-[30px] text-[15px]' : 'size-8 text-[17px]'"
    >R</span>
    <span class="font-extrabold tracking-tight" :class="small ? 'text-[17px]' : 'text-[19px]'">
      Ra<span class="text-primary">Cash</span>
    </span>
  </NuxtLink>
</template>
