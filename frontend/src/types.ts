export interface ProtocolConfig {
  enabled: boolean
  host: string
  port: number
}

export interface ServerConfig {
  socks5: ProtocolConfig
  http: ProtocolConfig
}

export interface RelayConfig {
  dialTimeoutSec: number
  readTimeoutSec: number
  maxConnections: number
  keepaliveSec: number
}

export interface LogConfig {
  level: 'debug' | 'info' | 'warn' | 'error'
  maxSizeMb: number
  maxBackups: number
  output: 'file' | 'console' | 'both'
}

export interface UIConfig {
  theme: 'light' | 'dark' | 'auto'
  language: string
  startMinimized: boolean
  showTrayIcon: boolean
}

export interface AppConfig {
  server: ServerConfig
  relay: RelayConfig
  log: LogConfig
  ui: UIConfig
}

export interface ServerStatus {
  running: boolean
  startedAt: string
  socks5Addr: string
  httpAddr: string
  activeConns: number
  totalConns: number
}

export interface StatsSnapshot {
  activeConns: number
  totalConns: number
  uploadBytes: number
  downloadBytes: number
}

export interface ActiveConnection {
  id: number
  protocol: string
  clientAddr: string
  targetAddr: string
  uploadBytes: number
  downloadBytes: number
  openedAt: string
}

export interface LogEntry {
  time: string
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'
  message: string
  source: string
}
