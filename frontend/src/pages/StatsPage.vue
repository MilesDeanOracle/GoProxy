<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts/core'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { LineChart } from 'echarts/charts'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsCoreOption, EChartsType } from 'echarts/core'
import { useServerStore } from '../stores/server'

echarts.use([GridComponent, LegendComponent, TooltipComponent, LineChart, CanvasRenderer])

const server = useServerStore()
const chartEl = ref<HTMLDivElement | null>(null)
let chart: EChartsType | null = null

const protocolRows = computed(() => {
  const rows = new Map<string, { protocol: string; conns: number; upload: number; download: number }>()
  for (const conn of server.activeConnections) {
    const key = conn.protocol || 'UNKNOWN'
    const row = rows.get(key) ?? { protocol: key, conns: 0, upload: 0, download: 0 }
    row.conns += 1
    row.upload += conn.uploadBytes
    row.download += conn.downloadBytes
    rows.set(key, row)
  }
  return Array.from(rows.values()).sort((a, b) => b.conns - a.conns)
})

const peakUpload = computed(() => Math.max(0, ...server.trafficHistory.map((item) => item.uploadRate)))
const peakDownload = computed(() => Math.max(0, ...server.trafficHistory.map((item) => item.downloadRate)))

function formatBytes(value: number) {
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  if (value < 1024 * 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`
  return `${(value / 1024 / 1024 / 1024).toFixed(2)} GB`
}

function formatRate(value: number) {
  return `${formatBytes(value)}/s`
}

function renderChart() {
  if (!chartEl.value) return
  chart ??= echarts.init(chartEl.value)
  const option: EChartsCoreOption = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      formatter(params: unknown) {
        const items = Array.isArray(params) ? params : [params]
        return items
          .map((item) => `${item.marker}${item.seriesName}: ${formatRate(Number(item.value))}`)
          .join('<br/>')
      }
    },
    legend: {
      right: 10,
      top: 4,
      textStyle: { color: '#7d8590' }
    },
    grid: {
      top: 42,
      right: 20,
      bottom: 26,
      left: 54
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: server.trafficHistory.map((item) => item.time),
      axisLabel: { color: '#7d8590', fontSize: 10 },
      axisLine: { lineStyle: { color: '#2a3340' } },
      axisTick: { show: false }
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        color: '#7d8590',
        fontSize: 10,
        formatter: (value: number) => formatBytes(value)
      },
      splitLine: { lineStyle: { color: 'rgba(125,133,144,0.18)' } }
    },
    series: [
      {
        name: '上传',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: server.trafficHistory.map((item) => item.uploadRate),
        lineStyle: { color: '#3b82f6', width: 2 }
      },
      {
        name: '下载',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: server.trafficHistory.map((item) => item.downloadRate),
        lineStyle: { color: '#f59e0b', width: 2 }
      }
    ]
  }
  const instance = chart
  if (instance) {
    instance.setOption(option)
  }
}

function resizeChart() {
  chart?.resize()
}

watch(() => server.trafficHistory, () => nextTick(renderChart), { deep: true })

onMounted(async () => {
  await server.refresh()
  renderChart()
  window.addEventListener('resize', resizeChart)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', resizeChart)
  chart?.dispose()
  chart = null
})
</script>

<template>
  <section class="stats-page">
    <div class="section-actions">
      <div>
        <span class="section-kicker">TRAFFIC</span>
        <h2>流量统计</h2>
      </div>
      <span class="chip">最近 {{ server.trafficHistory.length }} 秒</span>
    </div>

    <div class="cards">
      <div class="card">
        <div class="card-label">当前连接</div>
        <div class="card-val green">{{ server.stats.activeConns }}</div>
        <div class="card-sub">累计 {{ server.stats.totalConns }} 次连接</div>
      </div>
      <div class="card">
        <div class="card-label">上传总量</div>
        <div class="card-val blue">{{ formatBytes(server.stats.uploadBytes) }}</div>
        <div class="card-sub">峰值 {{ formatRate(peakUpload) }}</div>
      </div>
      <div class="card">
        <div class="card-label">下载总量</div>
        <div class="card-val amber">{{ formatBytes(server.stats.downloadBytes) }}</div>
        <div class="card-sub">峰值 {{ formatRate(peakDownload) }}</div>
      </div>
      <div class="card">
        <div class="card-label">认证失败</div>
        <div class="card-val">{{ server.stats.authFailures }}</div>
        <div class="card-sub">本次运行内统计</div>
      </div>
    </div>

    <section class="panel">
      <div class="panel-head">
        <h3>实时速率</h3>
      </div>
      <div ref="chartEl" class="echarts-panel" />
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>协议分布</h3>
        <span class="tag">ACTIVE</span>
      </div>
      <table class="conn-table">
        <thead>
          <tr>
            <th>协议</th>
            <th>连接数</th>
            <th>上传</th>
            <th>下载</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="protocolRows.length === 0">
            <td colspan="4" class="table-empty">暂无活跃连接</td>
          </tr>
          <tr v-for="row in protocolRows" :key="row.protocol">
            <td>{{ row.protocol }}</td>
            <td>{{ row.conns }}</td>
            <td>{{ formatBytes(row.upload) }}</td>
            <td>{{ formatBytes(row.download) }}</td>
          </tr>
        </tbody>
      </table>
    </section>
  </section>
</template>
