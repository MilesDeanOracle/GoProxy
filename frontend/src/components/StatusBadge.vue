<script setup lang="ts">
defineProps<{
  running: boolean
}>()
</script>

<template>
  <span class="status-badge" :class="{ running }">
    <span class="status-dot" />
    {{ running ? '运行中' : '已停止' }}
  </span>
</template>
