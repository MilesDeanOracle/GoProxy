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
  autoStartProxy: boolean
  showTrayIcon: boolean
  closeToTray: boolean
  trayStatusAndIp: boolean
}

export interface AppConfig {
  server: ServerConfig
  auth: AuthConfig
  relay: RelayConfig
  log: LogConfig
  ui: UIConfig
  route: RouteConfig
}

export interface AuthConfig {
  enabled: boolean
  users: AuthUser[]
}

export interface AuthUser {
  username: string
  password: string
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
  uploadRate: number
  downloadRate: number
  authFailures: number
}

export interface TrafficSample extends StatsSnapshot {
  time: string
}

export interface TrayState {
  enabled: boolean
  visible: boolean
  platform: string
  supportsMenu: boolean
  hideDescription: string
}

export interface ActiveConnection {
  id: number
  protocol: string
  clientAddr: string
  targetAddr: string
  routeRuleName: string
  outboundIp: string
  outboundIface: string
  uploadBytes: number
  downloadBytes: number
  openedAt: string
}

export interface RouteConfig {
  enabled: boolean
  activeFile: string
}

export interface RouteRuleSet {
  name: string
  version: number
  updatedAt: string
  description: string
  rules: RouteRule[]
}

export interface RouteRule {
  id: string
  name: string
  enabled: boolean
  priority: number
  protocols: string[]
  matchType: 'ip' | 'cidr' | 'domain' | 'wildcard' | 'any'
  targets: string[]
  outbound: OutboundBinding
  remark: string
}

export interface OutboundBinding {
  mode: 'default' | 'interface' | 'intercept'
  localIp: string
  interface: string
}

export interface RouteFileInfo {
  name: string
  isActive: boolean
  updatedAt: string
}

export interface NetworkInterface {
  name: string
  displayName: string
  addresses: string[]
  up: boolean
  loopback: boolean
}

export interface LogEntry {
  time: string
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'
  message: string
  source: string
}
