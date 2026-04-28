<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { NAlert, NSpin } from 'naive-ui'
import { useConfigStore } from '../stores/config'
import { useLogStore } from '../stores/logs'
import { useServerStore } from '../stores/server'
import type { LogEntry } from '../types'

const server = useServerStore()
const config = useConfigStore()
const logs = useLogStore()

const samples = ref<Array<{ up: number; down: number }>>(Array.from({ length: 60 }, () => ({ up: 0, down: 0 })))
const lastBytes = ref({ up: 0, down: 0 })
const lastClientBytes = ref(new Map<string, { uploadBytes: number; downloadBytes: number }>())
const clientRates = ref(new Map<string, { uploadRate: number; downloadRate: number }>())
const chartTime = ref('--')
const logLevel = ref<'ALL' | LogEntry['level']>('ALL')
const hoverIndex = ref<number | null>(null)
let timer: number | undefined

const chartWidth = 760
const chartHeight = 220
const chartBounds = {
  left: 58,
  top: 18,
  right: 18,
  bottom: 32
}
const chartPlotWidth = chartWidth - chartBounds.left - chartBounds.right
const chartPlotHeight = chartHeight - chartBounds.top - chartBounds.bottom

const logTabs: Array<{ label: string; value: 'ALL' | LogEntry['level'] }> = [
  { label: '全部', value: 'ALL' },
  { label: 'INFO', value: 'INFO' },
  { label: 'WARN', value: 'WARN' },
  { label: 'ERROR', value: 'ERROR' },
  { label: 'DEBUG', value: 'DEBUG' }
]

const maxConnections = computed(() => config.draft?.relay.maxConnections ?? 1000)
const uploadRate = computed(() => samples.value.at(-1)?.up ?? 0)
const downloadRate = computed(() => samples.value.at(-1)?.down ?? 0)
const totalTraffic = computed(() => server.stats.uploadBytes + server.stats.downloadBytes)
const chartMax = computed(() => Math.max(1, ...samples.value.flatMap((point) => [point.up, point.down])))
const chartPointsUp = computed(() => buildChartPoints(samples.value.map((point) => point.up)))
const chartPointsDown = computed(() => buildChartPoints(samples.value.map((point) => point.down)))
const chartPathUp = computed(() => buildSmoothPath(chartPointsUp.value))
const chartPathDown = computed(() => buildSmoothPath(chartPointsDown.value))
const chartYTicks = computed(() => {
  return Array.from({ length: 5 }, (_, index) => {
    const ratio = index / 4
    const value = chartMax.value * (1 - ratio)
    return {
      y: chartBounds.top + ratio * chartPlotHeight,
      label: formatAxisRate(value)
    }
  })
})
const chartXTicks = computed(() => {
  const ticks = [
    { index: 0, label: '-59s' },
    { index: 15, label: '-45s' },
    { index: 30, label: '-30s' },
    { index: 45, label: '-15s' },
    { index: 59, label: 'now' }
  ]
  return ticks.map((tick) => ({
    x: chartBounds.left + (tick.index / 59) * chartPlotWidth,
    label: tick.label
  }))
})
const hoverSample = computed(() => {
  if (hoverIndex.value === null) return null
  const sample = samples.value[hoverIndex.value]
  const point = chartPointsUp.value[hoverIndex.value]
  if (!sample || !point) return null
  const tooltipX = point.x > chartWidth - 180 ? point.x - 162 : point.x + 12
  return {
    x: point.x,
    upY: point.y,
    downY: chartPointsDown.value[hoverIndex.value]?.y ?? point.y,
    tooltipX,
    tooltipY: chartBounds.top + 8,
    up: sample.up,
    down: sample.down
  }
})
const clientRows = computed(() => {
  const grouped = new Map<
    string,
    {
      clientIp: string
      count: number
      uploadBytes: number
      downloadBytes: number
      uploadRate: number
      downloadRate: number
    }
  >()
  for (const conn of server.activeConnections) {
    const clientIp = normalizeClientIp(conn.clientAddr)
    const current = grouped.get(clientIp) ?? {
      clientIp,
      count: 0,
      uploadBytes: 0,
      downloadBytes: 0,
      uploadRate: 0,
      downloadRate: 0
    }
    current.count += 1
    current.uploadBytes += conn.uploadBytes
    current.downloadBytes += conn.downloadBytes
    grouped.set(clientIp, current)
  }
  for (const client of grouped.values()) {
    const rate = clientRates.value.get(client.clientIp)
    if (rate) {
      client.uploadRate = rate.uploadRate
      client.downloadRate = rate.downloadRate
    }
  }
  return [...grouped.values()].sort((a, b) => b.count - a.count || b.downloadRate - a.downloadRate).slice(0, 6)
})
const dashboardLogs = computed(() => {
  return [...logs.entries]
    .reverse()
    .filter((entry) => logLevel.value === 'ALL' || entry.level === logLevel.value)
    .slice(0, 40)
})

function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let next = value / 1024
  let unit = units[0]
  for (let i = 1; i < units.length && next >= 1024; i += 1) {
    next /= 1024
    unit = units[i]
  }
  return `${next.toFixed(next >= 10 ? 1 : 2)} ${unit}`
}

function formatRate(value: number): string {
  return `${formatBytes(value)}/s`
}

function formatAxisRate(value: number): string {
  return formatRate(value).replace(' ', '')
}

function connectionTraffic(upload: number, download: number): string {
  return `${formatBytes(upload)} ↑ ${formatBytes(download)} ↓`
}

function normalizeClientIp(address: string): string {
  if (!address) return '-'
  const bracketMatch = address.match(/^\[([^\]]+)\]:(\d+)$/)
  if (bracketMatch) return bracketMatch[1]
  const lastColon = address.lastIndexOf(':')
  if (lastColon > -1 && address.indexOf(':') === lastColon) {
    return address.slice(0, lastColon)
  }
  return address
}

function clientTrafficTitle(upload: number, download: number): string {
  return `总上行 ${formatBytes(upload)} / 总下行 ${formatBytes(download)}`
}

function shortTime(value: string): string {
  if (!value) return '--'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value.slice(11, 19) || value
  return parsed.toLocaleTimeString('zh-CN', { hour12: false })
}

function levelClass(level: LogEntry['level']) {
  return level.toLowerCase()
}

function buildChartPoints(values: number[]): Array<{ x: number; y: number; value: number }> {
  const max = chartMax.value
  return values.map((value, index) => {
    const x = chartBounds.left + (index / Math.max(values.length - 1, 1)) * chartPlotWidth
    const y = chartBounds.top + chartPlotHeight - (value / max) * chartPlotHeight
    return { x, y, value }
  })
}

function buildSmoothPath(points: Array<{ x: number; y: number }>): string {
  if (points.length === 0) return ''
  if (points.length === 1) return `M ${points[0].x.toFixed(1)} ${points[0].y.toFixed(1)}`

  let path = `M ${points[0].x.toFixed(1)} ${points[0].y.toFixed(1)}`
  for (let index = 1; index < points.length; index += 1) {
    const prev = points[index - 1]
    const curr = points[index]
    const midX = (prev.x + curr.x) / 2
    path += ` C ${midX.toFixed(1)} ${prev.y.toFixed(1)}, ${midX.toFixed(1)} ${curr.y.toFixed(1)}, ${curr.x.toFixed(1)} ${curr.y.toFixed(1)}`
  }
  return path
}

function onChartMouseMove(event: MouseEvent) {
  const svg = event.currentTarget as SVGSVGElement
  const rect = svg.getBoundingClientRect()
  const x = ((event.clientX - rect.left) / rect.width) * chartWidth
  const y = ((event.clientY - rect.top) / rect.height) * chartHeight

  if (
    x < chartBounds.left ||
    x > chartWidth - chartBounds.right ||
    y < chartBounds.top ||
    y > chartHeight - chartBounds.bottom
  ) {
    hoverIndex.value = null
    return
  }

  const ratio = (x - chartBounds.left) / chartPlotWidth
  hoverIndex.value = Math.max(0, Math.min(samples.value.length - 1, Math.round(ratio * (samples.value.length - 1))))
}

function onChartMouseLeave() {
  hoverIndex.value = null
}

async function tick() {
  await server.refresh()
  const up = Math.max(0, server.stats.uploadBytes - lastBytes.value.up)
  const down = Math.max(0, server.stats.downloadBytes - lastBytes.value.down)
  lastBytes.value = { up: server.stats.uploadBytes, down: server.stats.downloadBytes }
  samples.value = [...samples.value.slice(1), { up, down }]
  updateClientRates()
  chartTime.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
}

function updateClientRates() {
  const totals = new Map<string, { uploadBytes: number; downloadBytes: number }>()
  for (const conn of server.activeConnections) {
    const clientIp = normalizeClientIp(conn.clientAddr)
    const current = totals.get(clientIp) ?? { uploadBytes: 0, downloadBytes: 0 }
    current.uploadBytes += conn.uploadBytes
    current.downloadBytes += conn.downloadBytes
    totals.set(clientIp, current)
  }

  const nextRates = new Map<string, { uploadRate: number; downloadRate: number }>()
  for (const [clientIp, total] of totals) {
    const previous = lastClientBytes.value.get(clientIp)
    nextRates.set(clientIp, {
      uploadRate: previous ? Math.max(0, total.uploadBytes - previous.uploadBytes) : 0,
      downloadRate: previous ? Math.max(0, total.downloadBytes - previous.downloadBytes) : 0
    })
  }
  clientRates.value = nextRates
  lastClientBytes.value = totals
}

onMounted(async () => {
  await Promise.all([server.refresh(), logs.load()])
  lastBytes.value = { up: server.stats.uploadBytes, down: server.stats.downloadBytes }
  updateClientRates()
  timer = window.setInterval(() => {
    void tick()
  }, 1000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <section class="dashboard">
    <NAlert v-if="server.error" type="error" class="page-alert">
      {{ server.error }}
    </NAlert>
    <NAlert v-if="!server.status.running" type="warning" class="page-alert">
      代理服务未启动，Proxifier 连接 127.0.0.1:8080 会被系统拒绝。请点击右上角“启动服务”后再测试。
    </NAlert>

    <NSpin :show="server.loading">
      <div class="dashboard-stack">
        <div class="cards">
          <article class="card">
            <div class="card-label">当前连接数</div>
            <div class="card-val green">{{ server.status.activeConns }}</div>
            <div class="card-sub trend up">上限 {{ maxConnections }}</div>
          </article>
          <article class="card">
            <div class="card-label">上传速率</div>
            <div class="card-val blue">{{ formatRate(uploadRate) }}</div>
            <div class="card-sub">本次运行 {{ formatBytes(server.stats.uploadBytes) }}</div>
          </article>
          <article class="card">
            <div class="card-label">下载速率</div>
            <div class="card-val amber">{{ formatRate(downloadRate) }}</div>
            <div class="card-sub">本次运行 {{ formatBytes(server.stats.downloadBytes) }}</div>
          </article>
          <article class="card">
            <div class="card-label">总连接数</div>
            <div class="card-val">{{ server.status.totalConns }}</div>
            <div class="card-sub">总流量 {{ formatBytes(totalTraffic) }}</div>
          </article>
        </div>

        <div class="panels">
          <section class="panel">
            <div class="panel-head">
              <h3>实时流量速率</h3>
              <span class="tag ml">{{ chartTime }}</span>
            </div>
            <div class="chart-wrap">
              <div class="chart-legend">
                <span><i class="legend-dot up" />上传</span>
                <span><i class="legend-dot down" />下载</span>
              </div>
              <svg
                class="traffic-chart"
                :viewBox="`0 0 ${chartWidth} ${chartHeight}`"
                preserveAspectRatio="none"
                @mousemove="onChartMouseMove"
                @mouseleave="onChartMouseLeave"
              >
                <line
                  v-for="tick in chartYTicks"
                  :key="tick.y"
                  :x1="chartBounds.left"
                  :y1="tick.y"
                  :x2="chartWidth - chartBounds.right"
                  :y2="tick.y"
                  class="chart-grid"
                />
                <line
                  :x1="chartBounds.left"
                  :y1="chartBounds.top"
                  :x2="chartBounds.left"
                  :y2="chartHeight - chartBounds.bottom"
                  class="chart-axis"
                />
                <line
                  :x1="chartBounds.left"
                  :y1="chartHeight - chartBounds.bottom"
                  :x2="chartWidth - chartBounds.right"
                  :y2="chartHeight - chartBounds.bottom"
                  class="chart-axis"
                />
                <text
                  v-for="tick in chartYTicks"
                  :key="tick.label"
                  :x="chartBounds.left - 8"
                  :y="tick.y + 4"
                  text-anchor="end"
                  class="chart-tick-label"
                >
                  {{ tick.label }}
                </text>
                <text
                  v-for="tick in chartXTicks"
                  :key="tick.label"
                  :x="tick.x"
                  :y="chartHeight - 10"
                  text-anchor="middle"
                  class="chart-tick-label"
                >
                  {{ tick.label }}
                </text>
                <path :d="chartPathUp" class="chart-line chart-up" />
                <path :d="chartPathDown" class="chart-line chart-down" />
                <g v-if="hoverSample">
                  <line
                    :x1="hoverSample.x"
                    :y1="chartBounds.top"
                    :x2="hoverSample.x"
                    :y2="chartHeight - chartBounds.bottom"
                    class="chart-hover-line"
                  />
                  <circle :cx="hoverSample.x" :cy="hoverSample.upY" r="4" class="chart-point up" />
                  <circle :cx="hoverSample.x" :cy="hoverSample.downY" r="4" class="chart-point down" />
                  <g :transform="`translate(${hoverSample.tooltipX}, ${hoverSample.tooltipY})`">
                    <rect width="150" height="58" rx="6" class="chart-tooltip-bg" />
                    <text x="10" y="20" class="chart-tooltip-title">当前采样</text>
                    <text x="10" y="38" class="chart-tooltip-up">上传 {{ formatRate(hoverSample.up) }}</text>
                    <text x="10" y="52" class="chart-tooltip-down">下载 {{ formatRate(hoverSample.down) }}</text>
                  </g>
                </g>
              </svg>
            </div>
          </section>
        </div>

        <section class="panel client-panel">
          <div class="panel-head">
            <h3>当前连接客户端</h3>
            <span class="tag ml">{{ clientRows.length }} CLIENTS</span>
          </div>
          <div class="client-table">
            <div class="client-table-head">
              <span>客户端 IP</span>
              <span>连接数</span>
              <span>实时上行</span>
              <span>实时下行</span>
              <span>总上 / 下行</span>
            </div>
            <div v-for="client in clientRows" :key="client.clientIp" class="client-item">
              <strong>{{ client.clientIp }}</strong>
              <span>{{ client.count }}</span>
              <span>{{ formatRate(client.uploadRate) }}</span>
              <span>{{ formatRate(client.downloadRate) }}</span>
              <span :title="clientTrafficTitle(client.uploadBytes, client.downloadBytes)">
                {{ connectionTraffic(client.uploadBytes, client.downloadBytes) }}
              </span>
            </div>
            <div v-if="clientRows.length === 0" class="empty-log compact">暂无客户端连接</div>
          </div>
        </section>

        <section class="panel dashboard-log-panel">
          <div class="tabs">
            <button
              v-for="tab in logTabs"
              :key="tab.value"
              class="tab"
              :class="{ active: logLevel === tab.value }"
              type="button"
              @click="logLevel = tab.value"
            >
              {{ tab.label }}
            </button>
          </div>
          <div class="panel-head log-head">
            <h3>实时日志</h3>
            <span class="tag ml">自动滚动</span>
          </div>
          <div class="dashboard-log-list">
            <div v-for="(entry, index) in dashboardLogs" :key="`${entry.time}-${index}`" class="log-item">
              <span class="log-time">{{ shortTime(entry.time) }}</span>
              <span class="log-lv" :class="levelClass(entry.level)">{{ entry.level }}</span>
              <span class="log-msg">{{ entry.message }}</span>
            </div>
            <div v-if="dashboardLogs.length === 0" class="empty-log compact">暂无日志</div>
          </div>
        </section>
      </div>
    </NSpin>
  </section>
</template>
