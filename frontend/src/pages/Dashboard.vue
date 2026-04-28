<script setup lang="ts">
import { Power, Square, RefreshCw } from 'lucide-vue-next'
import { NAlert, NButton, NIcon, NSpin } from 'naive-ui'
import StatusBadge from '../components/StatusBadge.vue'
import { useServerStore } from '../stores/server'

const server = useServerStore()

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
</script>

<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h1>仪表盘</h1>
        <p>当前代理服务状态与会话统计</p>
      </div>
      <div class="header-actions">
        <NButton secondary :loading="server.loading" @click="server.refresh">
          <template #icon>
            <NIcon :component="RefreshCw" />
          </template>
          刷新
        </NButton>
        <NButton
          v-if="!server.status.running"
          type="primary"
          :loading="server.loading"
          @click="server.start"
        >
          <template #icon>
            <NIcon :component="Power" />
          </template>
          启动服务
        </NButton>
        <NButton v-else type="error" secondary :loading="server.loading" @click="server.stop">
          <template #icon>
            <NIcon :component="Square" />
          </template>
          停止服务
        </NButton>
      </div>
    </div>

    <NAlert v-if="server.error" type="error" class="page-alert">
      {{ server.error }}
    </NAlert>

    <NSpin :show="server.loading">
      <div class="status-strip">
        <StatusBadge :running="server.status.running" />
        <span>SOCKS5：{{ server.status.socks5Addr || '-' }}</span>
        <span>HTTP：{{ server.status.httpAddr || '-' }}</span>
      </div>

      <div class="metric-grid">
        <div class="metric-card">
          <span>活跃连接</span>
          <strong>{{ server.status.activeConns }}</strong>
        </div>
        <div class="metric-card">
          <span>总连接数</span>
          <strong>{{ server.status.totalConns }}</strong>
        </div>
        <div class="metric-card">
          <span>上行流量</span>
          <strong>{{ formatBytes(server.stats.uploadBytes) }}</strong>
        </div>
        <div class="metric-card">
          <span>下行流量</span>
          <strong>{{ formatBytes(server.stats.downloadBytes) }}</strong>
        </div>
      </div>

      <div class="session-summary">
        <span>当前会话总流量</span>
        <strong>{{ formatBytes(server.totalBytes) }}</strong>
      </div>
    </NSpin>
  </section>
</template>
