import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getActiveConnections, getServerStatus, getStats, startServer, stopServer } from '../backend/api'
import type { ActiveConnection, ServerStatus, StatsSnapshot } from '../types'

const emptyStatus: ServerStatus = {
  running: false,
  startedAt: '',
  socks5Addr: '',
  httpAddr: '',
  activeConns: 0,
  totalConns: 0
}

const emptyStats: StatsSnapshot = {
  activeConns: 0,
  totalConns: 0,
  uploadBytes: 0,
  downloadBytes: 0
}

export const useServerStore = defineStore('server', () => {
  const status = ref<ServerStatus>({ ...emptyStatus })
  const stats = ref<StatsSnapshot>({ ...emptyStats })
  const activeConnections = ref<ActiveConnection[]>([])
  const loading = ref(false)
  const error = ref('')

  const totalBytes = computed(() => stats.value.uploadBytes + stats.value.downloadBytes)

  async function refresh() {
    error.value = ''
    try {
      status.value = await getServerStatus()
      stats.value = await getStats()
      activeConnections.value = await getActiveConnections()
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
    }
  }

  async function start() {
    loading.value = true
    error.value = ''
    try {
      await startServer()
      await refresh()
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function stop() {
    loading.value = true
    error.value = ''
    try {
      await stopServer()
      await refresh()
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
      throw err
    } finally {
      loading.value = false
    }
  }

  function setStatus(next: ServerStatus) {
    status.value = next
  }

  return {
    status,
    stats,
    activeConnections,
    loading,
    error,
    totalBytes,
    refresh,
    start,
    stop,
    setStatus
  }
})
