<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { Search, Trash2 } from 'lucide-vue-next'
import { NButton, NCheckbox, NIcon, NInput } from 'naive-ui'
import { useLogStore } from '../stores/logs'
import type { LogEntry } from '../types'

interface RouteLogRow {
  time: string
  sourceIp: string
  accessIp: string
  rule: string
  action: string
  raw: LogEntry
}

const logs = useLogStore()
const scroller = ref<HTMLElement | null>(null)
const viewMode = ref<'all' | 'route'>('all')

const levels: Array<{ label: string; value: 'ALL' | LogEntry['level'] }> = [
  { label: '全部', value: 'ALL' },
  { label: 'INFO', value: 'INFO' },
  { label: 'WARN', value: 'WARN' },
  { label: 'ERROR', value: 'ERROR' },
  { label: 'DEBUG', value: 'DEBUG' }
]

const routeRows = computed(() => {
  const query = logs.keyword.trim().toLowerCase()
  return logs.entries
    .filter((entry) => entry.source === 'route')
    .filter((entry) => logs.level === 'ALL' || entry.level === logs.level)
    .map(parseRouteLog)
    .filter((row) => {
      if (query.length === 0) return true
      return [row.time, row.sourceIp, row.accessIp, row.rule, row.action, row.raw.message].some((value) =>
        value.toLowerCase().includes(query)
      )
    })
})

const visibleCount = computed(() => (viewMode.value === 'route' ? routeRows.value.length : logs.filteredEntries.length))

function levelClass(level: LogEntry['level']) {
  return level.toLowerCase()
}

function parseRouteLog(entry: LogEntry): RouteLogRow {
  const matched = entry.message.match(
    /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:[+-]\d{2}:\d{2}|Z))-(.*?)-(.*?)-触发规则\((.*?)\)-动作\((.*?)\)$/
  )
  if (!matched) {
    return {
      time: entry.time,
      sourceIp: '-',
      accessIp: '-',
      rule: '-',
      action: entry.message,
      raw: entry
    }
  }
  return {
    time: matched[1],
    sourceIp: matched[2] || '-',
    accessIp: matched[3] || '-',
    rule: matched[4] || '-',
    action: matched[5] || '-',
    raw: entry
  }
}

function actionClass(action: string): string {
  if (action.includes('拦截')) return 'blocked'
  if (action.includes('网卡')) return 'interface'
  return 'direct'
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
        <button class="tab log-kind-tab" :class="{ active: viewMode === 'all' }" type="button" @click="viewMode = 'all'">
          全部日志
        </button>
        <button class="tab log-kind-tab" :class="{ active: viewMode === 'route' }" type="button" @click="viewMode = 'route'">
          规则日志
        </button>
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
        <h3>{{ viewMode === 'route' ? '规则日志' : '实时日志' }}</h3>
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
        <template v-if="viewMode === 'route'">
          <div class="route-log-head">
            <span>时间</span>
            <span>来源 IP</span>
            <span>访问 IP</span>
            <span>触发规则</span>
            <span>动作</span>
          </div>
          <div v-for="(row, index) in routeRows" :key="`${row.raw.time}-${index}`" class="route-log-row">
            <span class="route-log-time">{{ row.time }}</span>
            <span class="route-log-ip">{{ row.sourceIp }}</span>
            <span class="route-log-ip">{{ row.accessIp }}</span>
            <span class="route-log-rule">{{ row.rule }}</span>
            <span class="route-log-action" :class="actionClass(row.action)">{{ row.action }}</span>
          </div>
          <div v-if="routeRows.length === 0" class="empty-log">暂无规则日志</div>
        </template>

        <template v-else>
          <div v-for="(entry, index) in logs.filteredEntries" :key="`${entry.time}-${index}`" class="log-row">
            <span class="log-time">{{ entry.time }}</span>
            <span class="log-level-pill" :class="levelClass(entry.level)">{{ entry.level }}</span>
            <span class="log-source">{{ entry.source }}</span>
            <span class="log-message">{{ entry.message }}</span>
          </div>
          <div v-if="logs.filteredEntries.length === 0" class="empty-log">暂无日志</div>
        </template>
      </div>
    </div>
  </section>
</template>
