import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { clearLogs, getRecentLogs } from '../backend/api'
import type { LogEntry } from '../types'
import { friendlyError } from '../utils/errors'

const maxEntries = 1000

export const useLogStore = defineStore('logs', () => {
  const entries = ref<LogEntry[]>([])
  const level = ref<'ALL' | LogEntry['level']>('ALL')
  const keyword = ref('')
  const autoScroll = ref(true)
  const loading = ref(false)
  const clearing = ref(false)
  const error = ref('')

  const filteredEntries = computed(() => {
    const query = keyword.value.trim().toLowerCase()
    return entries.value.filter((entry) => {
      const levelMatches = level.value === 'ALL' || entry.level === level.value
      const queryMatches =
        query.length === 0 ||
        entry.message.toLowerCase().includes(query) ||
        entry.source.toLowerCase().includes(query)
      return levelMatches && queryMatches
    })
  })

  async function load() {
    loading.value = true
    error.value = ''
    try {
      entries.value = await getRecentLogs(maxEntries)
    } catch (err) {
      error.value = friendlyError(err)
    } finally {
      loading.value = false
    }
  }

  function append(entry: LogEntry) {
    entries.value.push(entry)
    if (entries.value.length > maxEntries) {
      entries.value.splice(0, entries.value.length - maxEntries)
    }
  }

  async function clearDisplay() {
    clearing.value = true
    error.value = ''
    try {
      await clearLogs()
      entries.value = []
    } catch (err) {
      error.value = friendlyError(err)
    } finally {
      clearing.value = false
    }
  }

  return {
    entries,
    filteredEntries,
    level,
    keyword,
    autoScroll,
    loading,
    clearing,
    error,
    load,
    append,
    clearDisplay
  }
})
