import type {
  ActiveConnection,
  AppConfig,
  LogEntry,
  NetworkInterface,
  RouteFileInfo,
  RouteRuleSet,
  ServerStatus,
  StatsSnapshot
} from '../types'
import {
  AddUser,
  ClearLogs,
  CreateRouteFile,
  DeleteRouteFile,
  GetActiveConnections,
  GetConfig,
  GetLocalIPAddresses,
  GetNetworkInterfaces,
  GetRecentLogs,
  GetServerStatus,
  GetStats,
  GetTrayState,
  HideToTray,
  ListRouteFiles,
  LoadRouteFile,
  QuitApp,
  RemoveUser,
  ResetUserPassword,
  SaveRouteFile,
  SaveConfig,
  SetAuthEnabled,
  SetActiveRouteFile,
  ShowWindow,
  StartServer,
  StopServer
} from '../../wailsjs/go/main/App'
import { ClipboardSetText, EventsOn } from '../../wailsjs/runtime/runtime'

type EventDisposer = () => void

export function getConfig() {
  return GetConfig() as unknown as Promise<AppConfig>
}

export function getLocalIPAddresses() {
  return GetLocalIPAddresses() as unknown as Promise<string[]>
}

export function getNetworkInterfaces() {
  return GetNetworkInterfaces() as unknown as Promise<NetworkInterface[]>
}

export function saveConfig(config: AppConfig) {
  return SaveConfig(config as unknown as Parameters<typeof SaveConfig>[0])
}

export function listRouteFiles() {
  return ListRouteFiles() as unknown as Promise<RouteFileInfo[]>
}

export function loadRouteFile(name: string) {
  return LoadRouteFile(name) as unknown as Promise<RouteRuleSet>
}

export function saveRouteFile(name: string, ruleSet: RouteRuleSet) {
  return SaveRouteFile(name, ruleSet as unknown as Parameters<typeof SaveRouteFile>[1])
}

export function createRouteFile(name: string) {
  return CreateRouteFile(name)
}

export function deleteRouteFile(name: string) {
  return DeleteRouteFile(name)
}

export function setActiveRouteFile(name: string) {
  return SetActiveRouteFile(name)
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

export function clearLogs() {
  return ClearLogs()
}

export function setAuthEnabled(enabled: boolean) {
  return SetAuthEnabled(enabled)
}

export function addUser(username: string, password: string) {
  return AddUser(username, password)
}

export function removeUser(username: string) {
  return RemoveUser(username)
}

export function resetUserPassword(username: string, password: string) {
  return ResetUserPassword(username, password)
}

export function getTrayState() {
  return GetTrayState()
}

export function showWindow() {
  return ShowWindow()
}

export function hideToTray() {
  return HideToTray()
}

export function quitApp() {
  return QuitApp()
}

export function copyText(text: string) {
  return ClipboardSetText(text)
}

export function onEvent<T>(eventName: string, callback: (payload: T) => void): EventDisposer {
  try {
    return EventsOn(eventName, (payload: T) => callback(payload))
  } catch {
    return () => undefined
  }
}
