import type { ActiveConnection, AppConfig, LogEntry, ServerStatus, StatsSnapshot } from '../types'
import {
  GetActiveConnections,
  GetConfig,
  GetRecentLogs,
  GetServerStatus,
  GetStats,
  SaveConfig,
  StartServer,
  StopServer
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

type EventDisposer = () => void

export function getConfig() {
  return GetConfig() as unknown as Promise<AppConfig>
}

export function saveConfig(config: AppConfig) {
  return SaveConfig(config as unknown as Parameters<typeof SaveConfig>[0])
}

export function startServer() {
  return StartServer()
}

export function stopServer() {
  return StopServer()
}

export function getServerStatus() {
  return GetServerStatus() as unknown as Promise<ServerStatus>
}

export function getStats() {
  return GetStats() as unknown as Promise<StatsSnapshot>
}

export function getActiveConnections() {
  return GetActiveConnections() as unknown as Promise<ActiveConnection[]>
}

export function getRecentLogs(n: number) {
  return GetRecentLogs(n) as unknown as Promise<LogEntry[]>
}

export function onEvent<T>(eventName: string, callback: (payload: T) => void): EventDisposer {
  try {
    return EventsOn(eventName, (payload: T) => callback(payload))
  } catch {
    return () => undefined
  }
}
