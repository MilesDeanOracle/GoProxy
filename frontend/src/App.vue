<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Activity, FileText, Settings } from 'lucide-vue-next'
import { darkTheme, NConfigProvider, NIcon, NMessageProvider } from 'naive-ui'
import { onEvent } from './backend/api'
import Dashboard from './pages/Dashboard.vue'
import ConfigPage from './pages/ConfigPage.vue'
import LogsPage from './pages/LogsPage.vue'
import { useConfigStore } from './stores/config'
import { useLogStore } from './stores/logs'
import { useServerStore } from './stores/server'
import type { LogEntry, ServerStatus } from './types'

type PageKey = 'dashboard' | 'config' | 'logs'

const config = useConfigStore()
const server = useServerStore()
const logs = useLogStore()

const navItems: Array<{ key: PageKey; label: string; icon: typeof Activity }> = [
  { key: 'dashboard', label: '仪表盘', icon: Activity },
  { key: 'config', label: '服务配置', icon: Settings },
  { key: 'logs', label: '实时日志', icon: FileText }
]

const initialHash = window.location.hash.replace('#', '') as PageKey
const activePage = ref<PageKey>(navItems.some((item) => item.key === initialHash) ? initialHash : 'dashboard')

function selectPage(value: PageKey) {
  activePage.value = value
  window.location.hash = value
}

const naiveTheme = computed(() => {
  if (config.draft?.ui.theme === 'dark') return darkTheme
  return null
})

onMounted(async () => {
  await Promise.all([config.load(), server.refresh(), logs.load()])
  onEvent<LogEntry>('proxy:log', logs.append)
  onEvent<ServerStatus>('proxy:status', server.setStatus)
})
</script>

<template>
  <NConfigProvider :theme="naiveTheme">
    <NMessageProvider>
      <div class="app-shell">
        <aside class="sidebar">
          <div class="brand">
            <span class="brand-mark">PS</span>
            <div>
              <strong>ProxyServer</strong>
              <small>Desktop Proxy</small>
            </div>
          </div>
          <nav class="nav">
            <button
              v-for="item in navItems"
              :key="item.key"
              class="nav-item"
              :class="{ active: activePage === item.key }"
              @click="selectPage(item.key)"
            >
              <NIcon :component="item.icon" />
              <span>{{ item.label }}</span>
            </button>
          </nav>
        </aside>

        <main class="main-content">
          <Dashboard v-if="activePage === 'dashboard'" />
          <ConfigPage v-else-if="activePage === 'config'" />
          <LogsPage v-else />
        </main>
      </div>
    </NMessageProvider>
  </NConfigProvider>
</template>
