<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  BarChart3,
  FileText,
  Info,
  LayoutDashboard,
  Moon,
  Network,
  Power,
  Route as RouteIcon,
  Settings,
  Shield,
  SlidersHorizontal,
  Square,
  Sun
} from 'lucide-vue-next'
import { darkTheme, NConfigProvider, NDialogProvider, NIcon, NMessageProvider, NPopover } from 'naive-ui'
import { getLocalIPAddresses, onEvent } from './backend/api'
import Dashboard from './pages/Dashboard.vue'
import ActiveConnectionsPage from './pages/ActiveConnectionsPage.vue'
import AuthPage from './pages/AuthPage.vue'
import ConfigPage from './pages/ConfigPage.vue'
import LogsPage from './pages/LogsPage.vue'
import RouteRulesPage from './pages/RouteRulesPage.vue'
import SettingsPage from './pages/SettingsPage.vue'
import StatsPage from './pages/StatsPage.vue'
import { useConfigStore } from './stores/config'
import { useLogStore } from './stores/logs'
import { useServerStore } from './stores/server'
import type { LogEntry, ServerStatus, StatsSnapshot } from './types'

type PageKey = 'dashboard' | 'connections' | 'logs' | 'stats' | 'config' | 'routes' | 'auth' | 'settings'

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
      { key: 'stats', label: '流量统计', icon: BarChart3 }
    ]
  },
  {
    title: '管理',
    items: [
      { key: 'config', label: '服务配置', icon: SlidersHorizontal },
      { key: 'routes', label: '路由规则', icon: RouteIcon },
      { key: 'auth', label: '认证管理', icon: Shield },
      { key: 'settings', label: '应用设置', icon: Settings }
    ]
  }
]

const pageLabels: Record<PageKey, string> = {
  dashboard: '仪表盘',
  connections: '活跃连接',
  logs: '实时日志',
  stats: '流量统计',
  config: '服务配置',
  routes: '路由规则',
  auth: '认证管理',
  settings: '应用设置'
}

const initialHash = window.location.hash.replace('#', '') as PageKey
const enabledKeys = navGroups.flatMap((group) => group.items).filter((item) => !item.disabled).map((item) => item.key)
const activePage = ref<PageKey>(enabledKeys.includes(initialHash) ? initialHash : 'dashboard')
const systemDark = ref(window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? true)
const serverActionLocked = ref(false)
const localIPs = ref<string[]>([])
const appVersion = 'V1.0.0'

const currentTheme = computed<'dark' | 'light'>(() => {
  const selected = config.draft?.ui.theme ?? 'dark'
  if (selected === 'auto') return systemDark.value ? 'dark' : 'light'
  return selected
})

const naiveTheme = computed(() => (currentTheme.value === 'dark' ? darkTheme : null))
const activeLabel = computed(() => pageLabels[activePage.value])
const dashboardListenState = computed(() => (server.status.running ? '监听状态 / RUNNING' : '监听状态 / STOPPED'))
const socksChip = computed(() => formatAddr('SOCKS5', server.status.socks5Addr, config.draft?.server.socks5.port))
const httpChip = computed(() => formatAddr('HTTP', server.status.httpAddr, config.draft?.server.http.port))
const localIPLabel = computed(() => (localIPs.value.length > 0 ? localIPs.value.join(' / ') : '未检测到网卡 IP'))

function selectPage(item: NavItem) {
  if (item.disabled) return
  activePage.value = item.key
  window.location.hash = item.key
}

function formatAddr(label: string, addr: string, fallbackPort?: number) {
  if (addr) return `${label} / ${addr}`
  if (fallbackPort) return `${label} / :${fallbackPort}`
  return `${label} / -`
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
  try {
    localIPs.value = await getLocalIPAddresses()
  } catch {
    localIPs.value = []
  }
  onEvent<LogEntry>('proxy:log', logs.append)
  onEvent<ServerStatus>('proxy:status', server.setStatus)
  onEvent<StatsSnapshot>('proxy:stats', server.setStats)

  const media = window.matchMedia?.('(prefers-color-scheme: dark)')
  media?.addEventListener('change', (event) => {
    systemDark.value = event.matches
  })
})
</script>

<template>
  <NConfigProvider :theme="naiveTheme">
    <NMessageProvider>
      <NDialogProvider>
        <div class="app-shell" :data-theme="currentTheme">
          <aside class="sidebar">
            <div class="nav-logo">
              <span class="live-dot" :class="{ stopped: !server.status.running }" />
              <div class="nav-logo-copy">
                <div class="status-line">
                  <span class="inline-status" :class="{ stopped: !server.status.running }">
                    {{ server.status.running ? '服务运行中' : '服务已停止' }}
                  </span>
                  <NPopover trigger="click" placement="right-start" :to="false">
                    <template #trigger>
                      <button class="status-info-btn" type="button" title="查看网卡 IP">
                        <NIcon :component="Info" />
                      </button>
                    </template>
                    <div class="ip-popover">
                      <div class="ip-popover-title">网卡 IP</div>
                      <div v-if="localIPs.length > 0" class="ip-popover-list">
                        <span v-for="ip in localIPs" :key="ip" class="ip-popover-chip">{{ ip }}</span>
                      </div>
                      <span v-else class="ip-empty">{{ localIPLabel }}</span>
                    </div>
                  </NPopover>
                </div>
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
              <div class="version-panel">
                <span class="version-label">版本</span>
                <span class="version-value">{{ appVersion }}</span>
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
                <button class="icon-btn" type="button" :title="currentTheme === 'dark' ? '切换到浅色' : '切换到深色'" @click="toggleTheme">
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
              <StatsPage v-else-if="activePage === 'stats'" />
              <RouteRulesPage v-else-if="activePage === 'routes'" />
              <AuthPage v-else-if="activePage === 'auth'" />
              <SettingsPage v-else-if="activePage === 'settings'" />
              <ConfigPage v-else />
            </div>
          </main>
        </div>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>
