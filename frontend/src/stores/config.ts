import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getConfig, saveConfig } from '../backend/api'
import type { AppConfig } from '../types'

function cloneConfig(config: AppConfig): AppConfig {
  return JSON.parse(JSON.stringify(config)) as AppConfig
}

function sameListenerConfig(a: AppConfig, b: AppConfig): boolean {
  return JSON.stringify(a.server) === JSON.stringify(b.server)
}

export const useConfigStore = defineStore('config', () => {
  const current = ref<AppConfig | null>(null)
  const draft = ref<AppConfig | null>(null)
  const loading = ref(false)
  const saving = ref(false)
  const error = ref('')
  const restartRequired = ref(false)

  const dirty = computed(() => {
    if (!current.value || !draft.value) return false
    return JSON.stringify(current.value) !== JSON.stringify(draft.value)
  })

  const listenerDirty = computed(() => {
    if (!current.value || !draft.value) return false
    return !sameListenerConfig(current.value, draft.value)
  })

  async function load() {
    loading.value = true
    error.value = ''
    try {
      const config = await getConfig()
      current.value = cloneConfig(config)
      draft.value = cloneConfig(config)
      restartRequired.value = false
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
    } finally {
      loading.value = false
    }
  }

  async function save(serverRunning: boolean) {
    if (!draft.value) return
    saving.value = true
    error.value = ''
    const needsRestart = serverRunning && listenerDirty.value
    try {
      await saveConfig(draft.value)
      current.value = cloneConfig(draft.value)
      restartRequired.value = needsRestart
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
      throw err
    } finally {
      saving.value = false
    }
  }

  function reset() {
    if (current.value) {
      draft.value = cloneConfig(current.value)
    }
    restartRequired.value = false
    error.value = ''
  }

  return {
    current,
    draft,
    loading,
    saving,
    error,
    dirty,
    listenerDirty,
    restartRequired,
    load,
    save,
    reset
  }
})
