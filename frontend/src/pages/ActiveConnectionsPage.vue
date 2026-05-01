<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { NSpin } from 'naive-ui'
import { useConfigStore } from '../stores/config'
import { useServerStore } from '../stores/server'
import type { ActiveConnection } from '../types'

const server = useServerStore()
const config = useConfigStore()
let timer: number | undefined

const maxConnections = computed(() => config.draft?.relay.maxConnections ?? 1000)
const previousBytes = ref(new Map<number, { uploadBytes: number; downloadBytes: number }>())
const connectionRates = ref(new Map<number, { uploadRate: number; downloadRate: number }>())
const rows = computed(() =>
  server.activeConnections.map((conn) => ({
    ...conn,
    uploadRate: connectionRates.value.get(conn.id)?.uploadRate ?? 0,
    downloadRate: connectionRates.value.get(conn.id)?.downloadRate ?? 0
  }))
)

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

function formatProtocol(protocol: string): string {
  return protocol === 'socks5' ? 'SOCKS5' : 'HTTP'
}

function protocolClass(protocol: string): string {
  return protocol === 'socks5' ? 's5' : 'hc'
}

function shortTime(value: string): string {
  if (!value) return '--'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value.slice(11, 19) || value
  return parsed.toLocaleTimeString('zh-CN', { hour12: false })
}

function updateConnectionRates(connections: ActiveConnection[]) {
  const nextPrevious = new Map<number, { uploadBytes: number; downloadBytes: number }>()
  const nextRates = new Map<number, { uploadRate: number; downloadRate: number }>()

  for (const conn of connections) {
    const previous = previousBytes.value.get(conn.id)
    nextPrevious.set(conn.id, {
      uploadBytes: conn.uploadBytes,
      downloadBytes: conn.downloadBytes
    })
    nextRates.set(conn.id, {
      uploadRate: previous ? Math.max(0, conn.uploadBytes - previous.uploadBytes) : 0,
      downloadRate: previous ? Math.max(0, conn.downloadBytes - previous.downloadBytes) : 0
    })
  }

  previousBytes.value = nextPrevious
  connectionRates.value = nextRates
}

onMounted(async () => {
  await server.refresh()
  updateConnectionRates(server.activeConnections)
  timer = window.setInterval(() => {
    void server.refresh().then(() => {
      updateConnectionRates(server.activeConnections)
    })
  }, 1000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <section class="connections-page">
    <NSpin :show="server.loading">
      <div class="panel active-panel">
        <div class="panel-head">
          <h3>活跃连接</h3>
          <span class="tag ml">{{ server.status.activeConns }} / {{ maxConnections }}</span>
        </div>
        <table class="conn-table active-conn-table">
          <thead>
            <tr>
              <th>协议</th>
              <th>客户端</th>
              <th>目标</th>
              <th>实时上行</th>
              <th>实时下行</th>
              <th>累计上行</th>
              <th>累计下行</th>
              <th>建立时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="conn in rows" :key="conn.id">
              <td><span class="proto" :class="protocolClass(conn.protocol)">{{ formatProtocol(conn.protocol) }}</span></td>
              <td>{{ conn.clientAddr }}</td>
              <td>{{ conn.targetAddr || '-' }}</td>
              <td>{{ formatRate(conn.uploadRate) }}</td>
              <td>{{ formatRate(conn.downloadRate) }}</td>
              <td>{{ formatBytes(conn.uploadBytes) }}</td>
              <td>{{ formatBytes(conn.downloadBytes) }}</td>
              <td>{{ shortTime(conn.openedAt) }}</td>
            </tr>
            <tr v-if="rows.length === 0">
              <td colspan="8" class="table-empty">暂无活跃连接</td>
            </tr>
          </tbody>
        </table>
      </div>
    </NSpin>
  </section>
</template>
