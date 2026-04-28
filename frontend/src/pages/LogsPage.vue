<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { Trash2 } from 'lucide-vue-next'
import { NButton, NCheckbox, NIcon, NInput, NSelect } from 'naive-ui'
import { useLogStore } from '../stores/logs'

const logs = useLogStore()
const scroller = ref<HTMLElement | null>(null)

const levelOptions = [
  { label: '全部', value: 'ALL' },
  { label: 'DEBUG', value: 'DEBUG' },
  { label: 'INFO', value: 'INFO' },
  { label: 'WARN', value: 'WARN' },
  { label: 'ERROR', value: 'ERROR' }
]

const visibleCount = computed(() => logs.filteredEntries.length)

watch(
  () => logs.entries.length,
  async () => {
    if (!logs.autoScroll) return
    await nextTick()
    if (scroller.value) {
      scroller.value.scrollTop = scroller.value.scrollHeight
    }
  }
)
</script>

<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h1>实时日志</h1>
        <p>{{ visibleCount }} 条匹配记录</p>
      </div>
      <div class="header-actions">
        <NButton secondary @click="logs.clearDisplay">
          <template #icon>
            <NIcon :component="Trash2" />
          </template>
          清空显示
        </NButton>
      </div>
    </div>

    <div class="log-toolbar">
      <NSelect v-model:value="logs.level" class="log-level" :options="levelOptions" />
      <NInput v-model:value="logs.keyword" clearable placeholder="搜索日志" />
      <NCheckbox v-model:checked="logs.autoScroll">自动滚动</NCheckbox>
    </div>

    <div ref="scroller" class="log-list">
      <div v-for="(entry, index) in logs.filteredEntries" :key="`${entry.time}-${index}`" class="log-row">
        <span class="log-time">{{ entry.time }}</span>
        <span class="log-level-pill" :class="entry.level.toLowerCase()">{{ entry.level }}</span>
        <span class="log-source">{{ entry.source }}</span>
        <span class="log-message">{{ entry.message }}</span>
      </div>
      <div v-if="logs.filteredEntries.length === 0" class="empty-log">暂无日志</div>
    </div>
  </section>
</template>
