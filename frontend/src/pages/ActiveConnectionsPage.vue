<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { NSpin } from 'naive-ui'
import { useConfigStore } from '../stores/config'
import { useServerStore } from '../stores/server'

const server = useServerStore()
const config = useConfigStore()
let timer: number | undefined

const maxConnections = computed(() => config.draft?.relay.maxConnections ?? 1000)
const rows = computed(() => server.activeConnections)

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

onMounted(async () => {
  await server.refresh()
  timer = window.setInterval(() => {
    void server.refresh()
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
              <th>上行</th>
              <th>下行</th>
              <th>建立时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="conn in rows" :key="conn.id">
              <td><span class="proto" :class="protocolClass(conn.protocol)">{{ formatProtocol(conn.protocol) }}</span></td>
              <td>{{ conn.clientAddr }}</td>
              <td>{{ conn.targetAddr || '-' }}</td>
              <td>{{ formatBytes(conn.uploadBytes) }}</td>
              <td>{{ formatBytes(conn.downloadBytes) }}</td>
              <td>{{ shortTime(conn.openedAt) }}</td>
            </tr>
            <tr v-if="rows.length === 0">
              <td colspan="6" class="table-empty">暂无活跃连接</td>
            </tr>
          </tbody>
        </table>
      </div>
    </NSpin>
  </section>
</template>
