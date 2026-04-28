<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { Search, Trash2 } from 'lucide-vue-next'
import { NButton, NCheckbox, NIcon, NInput } from 'naive-ui'
import { useLogStore } from '../stores/logs'
import type { LogEntry } from '../types'

const logs = useLogStore()
const scroller = ref<HTMLElement | null>(null)

const levels: Array<{ label: string; value: 'ALL' | LogEntry['level'] }> = [
  { label: '全部', value: 'ALL' },
  { label: 'INFO', value: 'INFO' },
  { label: 'WARN', value: 'WARN' },
  { label: 'ERROR', value: 'ERROR' },
  { label: 'DEBUG', value: 'DEBUG' }
]

const visibleCount = computed(() => logs.filteredEntries.length)

function levelClass(level: LogEntry['level']) {
  return level.toLowerCase()
}

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
  <section class="logs-page">
    <div class="panel log-panel">
      <div class="tabs">
        <button
          v-for="level in levels"
          :key="level.value"
          class="tab"
          :class="{ active: logs.level === level.value }"
          type="button"
          @click="logs.level = level.value"
        >
          {{ level.label }}
        </button>
      </div>

      <div class="panel-head log-head">
        <h3>实时日志</h3>
        <span class="tag">{{ visibleCount }} MATCHED</span>
        <div class="log-tools">
          <NInput v-model:value="logs.keyword" clearable placeholder="搜索日志" size="small">
            <template #prefix>
              <NIcon :component="Search" />
            </template>
          </NInput>
          <NCheckbox v-model:checked="logs.autoScroll">自动滚动</NCheckbox>
          <NButton secondary size="small" @click="logs.clearDisplay">
            <template #icon>
              <NIcon :component="Trash2" />
            </template>
            清空
          </NButton>
        </div>
      </div>

      <div ref="scroller" class="log-list terminal-list">
        <div v-for="(entry, index) in logs.filteredEntries" :key="`${entry.time}-${index}`" class="log-row">
          <span class="log-time">{{ entry.time }}</span>
          <span class="log-level-pill" :class="levelClass(entry.level)">{{ entry.level }}</span>
          <span class="log-source">{{ entry.source }}</span>
          <span class="log-message">{{ entry.message }}</span>
        </div>
        <div v-if="logs.filteredEntries.length === 0" class="empty-log">暂无日志</div>
      </div>
    </div>
  </section>
</template>
