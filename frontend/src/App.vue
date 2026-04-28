<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  BarChart3,
  FileText,
  LayoutDashboard,
  Moon,
  Network,
  Power,
  Settings,
  Shield,
  SlidersHorizontal,
  Square,
  Sun
} from 'lucide-vue-next'
import { darkTheme, NConfigProvider, NIcon, NMessageProvider } from 'naive-ui'
import { onEvent } from './backend/api'
import Dashboard from './pages/Dashboard.vue'
import ActiveConnectionsPage from './pages/ActiveConnectionsPage.vue'
import ConfigPage from './pages/ConfigPage.vue'
import LogsPage from './pages/LogsPage.vue'
import { useConfigStore } from './stores/config'
import { useLogStore } from './stores/logs'
import { useServerStore } from './stores/server'
import type { LogEntry, ServerStatus } from './types'

type PageKey = 'dashboard' | 'connections' | 'logs' | 'stats' | 'config' | 'auth' | 'settings'

interface NavItem {
  key: PageKey
  label: string
  icon: typeof LayoutDashboard
  disabled?: boolean
}

const config = useConfigStore()
const server = useServerStore()
const logs = useLogStore()

const navGroups: Array<{ title: string; items: NavItem[] }> = [
  {
    title: '监控',
    items: [
      { key: 'dashboard', label: '仪表盘', icon: LayoutDashboard },
      { key: 'connections', label: '活跃连接', icon: Network },
      { key: 'logs', label: '实时日志', icon: FileText },
      { key: 'stats', label: '流量统计', icon: BarChart3, disabled: true }
    ]
  },
  {
    title: '管理',
    items: [
      { key: 'config', label: '服务配置', icon: SlidersHorizontal },
      { key: 'auth', label: '认证管理', icon: Shield, disabled: true },
      { key: 'settings', label: '应用设置', icon: Settings, disabled: true }
    ]
  }
]

const pageLabels: Record<PageKey, string> = {
  dashboard: '仪表盘',
  connections: '活跃连接',
  logs: '实时日志',
  stats: '流量统计',
  config: '服务配置',
  auth: '认证管理',
  settings: '应用设置'
}

const initialHash = window.location.hash.replace('#', '') as PageKey
const enabledKeys = navGroups.flatMap((group) => group.items).filter((item) => !item.disabled).map((item) => item.key)
const activePage = ref<PageKey>(enabledKeys.includes(initialHash) ? initialHash : 'dashboard')
const systemDark = ref(window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? true)
const serverActionLocked = ref(false)

const currentTheme = computed<'dark' | 'light'>(() => {
  const selected = config.draft?.ui.theme ?? 'dark'
  if (selected === 'auto') return systemDark.value ? 'dark' : 'light'
  return selected
})

const naiveTheme = computed(() => (currentTheme.value === 'dark' ? darkTheme : null))
const activeLabel = computed(() => pageLabels[activePage.value])
const dashboardListenState = computed(() => (server.status.running ? '监听状态 · RUNNING' : '监听状态 · STOPPED'))
const socksChip = computed(() => formatAddr('SOCKS5', server.status.socks5Addr, config.draft?.server.socks5.port))
const httpChip = computed(() => formatAddr('HTTP', server.status.httpAddr, config.draft?.server.http.port))

function selectPage(item: NavItem) {
  if (item.disabled) return
  activePage.value = item.key
  window.location.hash = item.key
}

function formatAddr(label: string, addr: string, fallbackPort?: number) {
  if (addr) return `${label} · ${addr}`
  if (fallbackPort) return `${label} · :${fallbackPort}`
  return `${label} · -`
}

async function toggleTheme() {
  if (!config.draft) return
  config.draft.ui.theme = currentTheme.value === 'dark' ? 'light' : 'dark'
  await config.save(server.status.running)
}

async function toggleServer() {
  if (server.loading || serverActionLocked.value) return
  serverActionLocked.value = true
  if (server.status.running) {
    try {
      await server.stop()
    } finally {
      window.setTimeout(() => {
        serverActionLocked.value = false
      }, 1200)
    }
    return
  }
  try {
    await server.start()
  } finally {
    window.setTimeout(() => {
      serverActionLocked.value = false
    }, 1200)
  }
}

onMounted(async () => {
  await Promise.all([config.load(), server.refresh(), logs.load()])
  onEvent<LogEntry>('proxy:log', logs.append)
  onEvent<ServerStatus>('proxy:status', server.setStatus)

  const media = window.matchMedia?.('(prefers-color-scheme: dark)')
  media?.addEventListener('change', (event) => {
    systemDark.value = event.matches
  })
})
</script>

<template>
  <NConfigProvider :theme="naiveTheme">
    <NMessageProvider>
      <div class="app-shell" :data-theme="currentTheme">
        <aside class="sidebar">
          <div class="nav-logo">
            <span class="live-dot" :class="{ stopped: !server.status.running }" />
            <div>
              <strong>ProxyServer</strong>
              <small>v1.0.0</small>
            </div>
          </div>

          <nav class="nav">
            <template v-for="group in navGroups" :key="group.title">
              <div class="nav-section">{{ group.title }}</div>
              <button
                v-for="item in group.items"
                :key="item.key"
                class="nav-item"
                :class="{ active: activePage === item.key, disabled: item.disabled }"
                type="button"
                @click="selectPage(item)"
              >
                <NIcon :component="item.icon" />
                <span>{{ item.label }}</span>
              </button>
            </template>
          </nav>

          <div class="nav-status">
            <div class="status-pill" :class="{ stopped: !server.status.running }">
              <span class="blink" />
              <span>{{ server.status.running ? '服务运行中' : '服务已停止' }}</span>
            </div>
          </div>
        </aside>

        <main class="main">
          <header class="topbar">
            <span class="topbar-title">{{ activeLabel }}</span>
            <span v-if="activePage === 'dashboard'" class="chip listen-chip" :class="{ running: server.status.running }">
              {{ dashboardListenState }}
            </span>
            <span class="chip">{{ socksChip }}</span>
            <span class="chip">{{ httpChip }}</span>
            <div class="topbar-right">
              <button class="icon-btn" type="button" :title="currentTheme === 'dark' ? '切换到白天模式' : '切换到黑暗模式'" @click="toggleTheme">
                <NIcon :component="currentTheme === 'dark' ? Sun : Moon" />
              </button>
              <button
                class="btn"
                :class="server.status.running ? 'btn-stop' : 'btn-start'"
                type="button"
                :disabled="server.loading || serverActionLocked"
                @click="toggleServer"
              >
                <NIcon :component="server.status.running ? Square : Power" />
                <span>{{ server.status.running ? '停止服务' : '启动服务' }}</span>
              </button>
            </div>
          </header>

          <div class="content">
            <Dashboard v-if="activePage === 'dashboard'" />
            <ActiveConnectionsPage v-else-if="activePage === 'connections'" />
            <LogsPage v-else-if="activePage === 'logs'" />
            <ConfigPage v-else />
          </div>
        </main>
      </div>
    </NMessageProvider>
  </NConfigProvider>
</template>
