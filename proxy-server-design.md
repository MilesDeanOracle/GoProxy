# 代理服务端软件 — 系统设计文档

> **Proxy Server Desktop Application · Technical Design Document**
> 版本：v1.0 · 技术栈：Go + Wails v2 · 目标平台：Windows / macOS · 日期：2026-04-28

---

## 目录

1. [项目概述](#1-项目概述)
2. [技术选型](#2-技术选型)
3. [系统架构](#3-系统架构)
4. [功能模块设计](#4-功能模块设计)
5. [UI 界面设计](#5-ui-界面设计)
6. [工程目录结构](#6-工程目录结构)
7. [前后端通信设计](#7-前后端通信设计wails)
8. [构建与发布](#8-构建与发布)
9. [开发路线图](#9-开发路线图)
10. [关键风险与缓解措施](#10-关键风险与缓解措施)
11. [附录：关键依赖版本](#11-附录关键依赖版本)

---

## 1. 项目概述

本文档描述代理服务端桌面应用程序的系统设计方案。该软件运行于 Windows 和 macOS 平台，提供原生桌面 UI，支持代理服务监听、配置管理、实时日志查看与流量统计功能。配置数据以 YAML 文件持久化存储，所有参数均可通过图形界面进行读写。

### 1.1 项目目标

- 提供高性能 SOCKS5 / HTTP CONNECT 代理服务端
- 原生桌面应用：Windows（.exe）和 macOS（.app）双平台发布
- 图形化配置界面：无需手动编辑配置文件即可完成所有设置
- 实时监控：日志流式展示、连接数、流量统计数据可视化
- 配置持久化：使用 YAML 文件保存，支持导入导出

### 1.2 非目标（当前版本不涉及）

- 客户端功能（本软件仅做服务端）
- Linux 桌面 UI 支持（命令行运行可行）
- Web 远程管理面板
- 集群部署 / 负载均衡

---

## 2. 技术选型

### 2.1 技术栈总览

| 层次 | 选型 | 说明 |
|------|------|------|
| 编程语言 | Go 1.22+ | 高并发 goroutine 模型，天然契合代理转发场景；编译为单二进制无依赖 |
| 桌面框架 | Wails v2 | Go 后端 + Web 前端（Vue 3），打包为原生可执行文件，无需 Node.js 运行时 |
| 前端 UI | Vue 3 + Naive UI | 响应式组件库，风格统一，图表使用 ECharts |
| 代理核心 | net 标准库 | 基于 Go net 包自研，支持 SOCKS5 与 HTTP CONNECT 入站 |
| 配置管理 | gopkg.in/yaml.v3 | YAML 序列化/反序列化，配合 fsnotify 热重载 |
| 日志 | uber-go/zap | 结构化高性能日志，支持日志级别过滤与文件轮转 |
| 打包发布 | wails build | Windows 生成 .exe（可选 NSIS 安装包），macOS 生成 .app（可选 DMG） |

### 2.2 选型理由

#### 2.2.1 Go 语言优势

- goroutine 轻量级（初始 2KB 栈），单机支撑数万并发连接
- `io.Copy` 双向流转发零拷贝，CPU 开销极低
- 编译为单一二进制文件，跨平台部署无依赖
- 标准库覆盖网络、TLS、HTTP，代理相关生态成熟

#### 2.2.2 Wails v2 优势

- 与 Electron 相比，内存占用减少约 60–70%（无 Chromium 进程，使用系统 WebView）
- Go 后端与前端通过双向绑定直接通信，无需 REST 或 IPC 额外协议
- Windows 使用 Edge WebView2，macOS 使用 WKWebView，渲染性能优秀
- 构建产物：Windows 单 .exe（约 10–20MB），macOS 单 .app bundle

---

## 3. 系统架构

### 3.1 整体架构分层

| 层次 | 组件与职责 |
|------|-----------|
| 表现层 | Vue 3 前端（运行在 Wails WebView）：配置页面、日志面板、流量图表、系统托盘菜单 |
| 应用层 | Wails App 绑定层：前端调用 Go 函数的桥接，事件广播（日志推送、流量数据更新） |
| 业务层 | ProxyServer（服务生命周期）、ConfigManager（YAML 读写）、StatsCollector（统计聚合）、LogManager（日志过滤输出） |
| 核心层 | SOCKS5 Handler、HTTP CONNECT Handler、Relay Engine（双向流转发）、Auth Module（用户认证） |
| 基础设施 | Go net/tls（TCP 监听）、yaml.v3（持久化）、zap（日志）、系统托盘 API（systray） |

### 3.2 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    桌面应用（Wails）                          │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              表现层  Vue 3 + Naive UI                  │  │
│  │   仪表盘  │  服务配置  │  日志面板  │  流量统计         │  │
│  └───────────────────────┬──────────────────────────────┘  │
│                          │ Wails 绑定 / EventsEmit          │
│  ┌───────────────────────▼──────────────────────────────┐  │
│  │              应用层  app.go                            │  │
│  │   GetConfig / SaveConfig / StartServer / StopServer   │  │
│  └──────┬──────────────────┬───────────────┬────────────┘  │
│         │                  │               │                │
│  ┌──────▼──────┐  ┌────────▼──────┐  ┌────▼───────────┐   │
│  │ ProxyServer │  │ ConfigManager │  │ StatsCollector │   │
│  │  生命周期    │  │  YAML 读写    │  │  流量 / 连接数  │   │
│  └──────┬──────┘  └───────────────┘  └────────────────┘   │
│         │                                                   │
│  ┌──────▼──────────────────────────────────────────────┐   │
│  │              核心层                                   │   │
│  │  SOCKS5 Handler │ HTTP CONNECT Handler │ Relay Engine │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 3.3 核心数据流

```
入站：  客户端 → [SOCKS5 / HTTP CONNECT] → 本地监听端口 → 协议解析 → 目标建连
转发：  本地连接 ↔ Relay Engine (io.Copy 双向) ↔ 目标服务器连接
统计：  Relay Engine → 字节计数器 → StatsCollector → 聚合 → 前端图表
日志：  各模块 → zap Logger → 环形缓冲 → Wails Event → 前端日志面板
```

---

## 4. 功能模块设计

### 4.1 代理服务核心（ProxyCore）

#### 4.1.1 支持的协议

| 协议 | 默认端口 | 说明 |
|------|---------|------|
| SOCKS5 | 1080 | 支持 TCP CONNECT，可选 USERNAME/PASSWORD 认证（RFC 1928） |
| HTTP CONNECT | 8080 | HTTP/1.1 CONNECT 隧道，适用于 HTTPS 代理场景，防火墙穿透友好 |

#### 4.1.2 核心转发引擎

每个入站连接由独立 goroutine 处理，通过 `io.Copy` 实现双向流转发，支持半关闭（Half-Close）以正确传递 EOF 信号。连接超时通过 `SetDeadline` 控制，避免僵尸连接占用资源。

**关键参数（均可通过配置文件调整）：**

| 参数 | 默认值 | 说明 |
|------|-------|------|
| 读写超时 | 30 秒 | 连接空闲超时 |
| 连接超时 | 10 秒 | 目标建连超时 |
| 最大并发连接数 | 1000 | 超出后拒绝新连接 |
| TCP Keep-Alive 间隔 | 15 秒 | 检测死连接 |

**转发引擎核心逻辑：**

```go
// 双向流转发，支持半关闭
func relay(ctx context.Context, a, b net.Conn) error {
    errCh := make(chan error, 2)
    copy := func(dst, src net.Conn) {
        _, err := io.Copy(dst, src)
        dst.(*net.TCPConn).CloseWrite() // 半关闭，通知对端 EOF
        errCh <- err
    }
    go copy(a, b)
    go copy(b, a)
    select {
    case err := <-errCh:
        return err
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### 4.2 配置管理（ConfigManager）

#### 4.2.1 配置文件路径

- **Windows：** `%APPDATA%\ProxyServer\config.yaml`
- **macOS：** `~/Library/Application Support/ProxyServer/config.yaml`

#### 4.2.2 配置文件结构（YAML）

```yaml
server:
  socks5:
    enabled: true
    host: 0.0.0.0
    port: 1080
  http:
    enabled: true
    host: 0.0.0.0
    port: 8080

auth:
  enabled: false
  users:
    - username: admin
      password: ""       # bcrypt hash

relay:
  dial_timeout_sec: 10
  read_timeout_sec: 30
  max_connections: 1000
  keepalive_sec: 15

log:
  level: info            # debug | info | warn | error
  max_size_mb: 50
  max_backups: 3
  output: both           # file | console | both

ui:
  theme: auto            # light | dark | auto
  language: zh-CN
  start_minimized: false
  show_tray_icon: true
```

#### 4.2.3 配置管理特性

- 页面修改配置后实时写入 YAML 文件
- 使用 `fsnotify` 监听文件变动，支持外部编辑器修改后热重载
- 配置变更前进行 Schema 校验，非法值给出明确错误提示
- 支持配置文件导出（备份）与导入（还原）
- 监听端口变更需重启服务生效，UI 给出明确提示
- 每次写入前自动备份当前配置，校验失败时自动回滚

### 4.3 日志系统（LogManager）

#### 4.3.1 日志级别与分类

| 级别 | 内容 |
|------|------|
| DEBUG | 连接详情、协议握手过程 |
| INFO | 连接建立/断开、配置变更、服务启停 |
| WARN | 连接超时、认证失败、配置校验警告 |
| ERROR | 监听失败、致命错误 |

#### 4.3.2 日志存储与推送机制

- **文件存储：** 使用 `lumberjack` 进行日志轮转，按大小切割，保留最近 N 个备份
- **内存缓冲：** 环形缓冲区保留最近 1000 条日志，供 UI 初始加载使用
- **实时推送：** 新日志条目通过 Wails `EventsEmit` 推送到前端，前端监听追加显示
- **日志过滤：** 前端支持按级别、关键字过滤展示，不影响文件存储
- **日志清空：** 仅清除 UI 显示，不删除磁盘文件

### 4.4 流量统计（StatsCollector）

#### 4.4.1 统计指标

| 指标 | 类型 | 说明 |
|------|------|------|
| 当前活跃连接数 | 实时 | 每秒刷新 |
| 总连接数（本次运行） | 累计 | 服务启动后累计 |
| 上行流量（字节） | 累计 + 速率 | 客户端 → 目标方向 |
| 下行流量（字节） | 累计 + 速率 | 目标 → 客户端方向 |
| 实时上传速率（B/s） | 实时 | 每秒滑动窗口计算 |
| 实时下载速率（B/s） | 实时 | 每秒滑动窗口计算 |
| 认证失败次数 | 累计 | 开启认证时统计 |

#### 4.4.2 统计数据更新策略

- `StatsCollector` 使用原子操作（`sync/atomic`）累加字节数，避免锁竞争
- 每秒计算一次速率，通过 Wails 事件推送到前端
- 前端使用 ECharts 展示近 60 秒的速率折线图，数据保存在内存中
- 统计数据不持久化，重启服务后清零

---

## 5. UI 界面设计

### 5.1 主窗口布局

主窗口采用**左侧导航 + 右侧内容区**的经典桌面应用布局，最小窗口尺寸 900×600 像素。

| 页面 | 主要内容 |
|------|---------|
| 🏠 仪表盘 | 服务运行状态（启动/停止按钮）、活跃连接数、实时流量速率卡片、流量折线图（近 60 秒） |
| ⚙️ 服务配置 | SOCKS5 / HTTP 开关、监听地址、端口输入框、最大连接数、超时设置，保存按钮 |
| 🔐 认证管理 | 认证开关、用户列表（增删改）、密码重置，表格展示 |
| 📋 实时日志 | 日志级别筛选 Tabs、关键字搜索框、自动滚动开关、日志条目列表（颜色区分级别）、导出按钮 |
| 📊 流量统计 | 当前会话统计卡片、上行/下行总量、连接数历史折线图、按协议分类饼图 |
| 🎨 应用设置 | 主题切换（亮色/暗色/跟随系统）、开机自启、最小化到托盘、语言选择、配置导入导出 |

### 5.2 仪表盘布局示意

```
┌────────────────────────────────────────────────────────────┐
│  ● 运行中   SOCKS5 :1080   HTTP :8080    [停止服务]         │
├──────────────┬──────────────┬──────────────┬───────────────┤
│  活跃连接     │  上传速率     │  下载速率     │  总流量        │
│    42        │  1.2 MB/s    │  8.5 MB/s    │  12.3 GB      │
├──────────────┴──────────────┴──────────────┴───────────────┤
│                     流量速率折线图（近 60 秒）                │
│   ↑                                                        │
│   │  ╭──╮         ╭───╮                                   │
│   │──╯  ╰────────╯   ╰────────────────────────────        │
│   └──────────────────────────────────────────────→ time   │
└────────────────────────────────────────────────────────────┘
```

### 5.3 系统托盘

- 最小化时缩至系统托盘，双击图标恢复主窗口
- 右键菜单：显示窗口 / 启动服务 / 停止服务 / 退出
- 托盘图标通过颜色区分服务状态（绿色 = 运行中，灰色 = 已停止）

### 5.4 前端技术细节

| 技术 | 用途 |
|------|------|
| Vue 3 + Composition API | 前端框架 |
| Naive UI | UI 组件库，原生支持暗色模式 |
| Apache ECharts | 折线图、饼图 |
| UnoCSS | 原子化 CSS，减少打包体积 |
| Pinia | 状态管理 |
| Wails runtime | Go 函数绑定 + 事件监听 |

---

## 6. 工程目录结构

```
proxy-server/
├── main.go                   # Wails 入口，注册 App
├── wails.json                # Wails 项目配置
├── app.go                    # App 结构体，前端绑定方法集合
│
├── internal/
│   ├── config/
│   │   ├── config.go         # Config 结构体定义
│   │   ├── manager.go        # 读写 YAML，fsnotify 热重载
│   │   └── validator.go      # 配置校验逻辑
│   │
│   ├── proxy/
│   │   ├── server.go         # ProxyServer 生命周期管理
│   │   ├── socks5.go         # SOCKS5 协议处理
│   │   ├── http_connect.go   # HTTP CONNECT 协议处理
│   │   ├── relay.go          # 双向流转发引擎
│   │   └── auth.go           # 用户认证
│   │
│   ├── stats/
│   │   └── collector.go      # 原子统计 + 速率计算
│   │
│   └── logger/
│       ├── logger.go         # zap 初始化，日志级别控制
│       └── ring_buffer.go    # 内存环形缓冲，供 UI 初始加载
│
├── frontend/
│   ├── src/
│   │   ├── App.vue
│   │   ├── pages/            # Dashboard / Config / Auth / Logs / Stats / Settings
│   │   ├── components/       # StatusBadge / TrafficChart / LogItem
│   │   ├── stores/           # useServerStore / useConfigStore / useStatsStore
│   │   └── wailsjs/          # Wails 自动生成的 Go 函数绑定
│   ├── package.json
│   └── vite.config.ts
│
└── build/
    ├── windows/              # 图标、NSIS 安装脚本
    └── darwin/               # 图标、DMG 配置
```

---

## 7. 前后端通信设计（Wails）

### 7.1 前端调用 Go 函数

在 `app.go` 中定义并绑定到 Wails，前端通过自动生成的 `wailsjs/` 调用：

| Go 方法签名 | 说明 |
|------------|------|
| `GetConfig() Config` | 返回当前完整配置 |
| `SaveConfig(Config) error` | 保存配置到 YAML，校验后返回错误信息 |
| `StartServer() error` | 启动代理服务，返回启动结果 |
| `StopServer() error` | 停止代理服务 |
| `GetServerStatus() Status` | 返回服务状态、监听端口、启动时间 |
| `GetStats() Stats` | 返回当前统计快照（初始加载用） |
| `GetRecentLogs(n int) []LogEntry` | 返回最近 n 条日志（初始加载用） |
| `AddUser(username, password string) error` | 新增认证用户 |
| `RemoveUser(username string) error` | 删除认证用户 |
| `ExportConfig(path string) error` | 导出配置文件到指定路径 |
| `ImportConfig(path string) error` | 从指定路径导入配置文件 |

### 7.2 后端推送事件（EventsEmit）

| 事件名 | 触发频率 | Payload 格式 |
|--------|---------|-------------|
| `proxy:log` | 实时（每条日志） | `{time, level, message, source}` |
| `proxy:stats` | 每 1 秒 | `{activeConns, totalConns, uploadBytes, downloadBytes, uploadRate, downloadRate}` |
| `proxy:status` | 状态变更时 | `{running: bool, startedAt: string}` |
| `config:changed` | 配置文件外部修改时 | 无 payload，前端重新调用 GetConfig |

### 7.3 前端事件监听示例

```typescript
import { EventsOn } from '../wailsjs/runtime'

// 监听实时日志
EventsOn('proxy:log', (entry: LogEntry) => {
  logStore.append(entry)
})

// 监听流量统计
EventsOn('proxy:stats', (stats: Stats) => {
  statsStore.update(stats)
  chartStore.push(stats)   // 维护近 60 秒数据窗口
})

// 监听服务状态变更
EventsOn('proxy:status', (status: ServerStatus) => {
  serverStore.setStatus(status)
})
```

---

## 8. 构建与发布

### 8.1 构建命令

```bash
# 开发模式（热重载）
wails dev

# 生产构建 - Windows
wails build -platform windows/amd64

# 生产构建 - macOS Intel
wails build -platform darwin/amd64

# 生产构建 - macOS Apple Silicon
wails build -platform darwin/arm64
```

### 8.2 发布产物

| 平台 | 产物 | 说明 |
|------|------|------|
| Windows | `ProxyServer.exe` / `Setup.exe` | 单文件可执行 或 NSIS 安装包（含快捷方式、注册表、卸载支持） |
| macOS | `ProxyServer.app` / `ProxyServer.dmg` | App Bundle 或 DMG 镜像，需代码签名避免 Gatekeeper 拦截 |

### 8.3 代码签名

- **Windows：** 使用 EV 代码签名证书（DigiCert / Sectigo），避免 SmartScreen 拦截
- **macOS：** 需 Apple Developer ID 证书签名 + 公证（notarize），否则需用户手动放行
- **CI/CD：** 推荐 GitHub Actions 矩阵构建，分别在 `windows-latest` 和 `macos-latest` Runner 上构建

### 8.4 GitHub Actions 构建矩阵示例

```yaml
jobs:
  build:
    strategy:
      matrix:
        include:
          - os: windows-latest
            platform: windows/amd64
          - os: macos-latest
            platform: darwin/amd64
          - os: macos-latest
            platform: darwin/arm64
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go install github.com/wailsapp/wails/v2/cmd/wails@latest
      - run: wails build -platform ${{ matrix.platform }}
```

---

## 9. 开发路线图

| 阶段 | 参考周期 | 交付内容 |
|------|---------|---------|
| Phase 1 | 第 1–2 周 | 项目初始化、Go 代理核心（SOCKS5 + HTTP CONNECT + Relay）、单元测试、CLI 可用 |
| Phase 2 | 第 3–4 周 | Wails 集成、基础 UI（仪表盘 + 服务配置页）、YAML 配置读写、日志系统 |
| Phase 3 | 第 5–6 周 | 认证管理页、流量统计 + ECharts 图表、系统托盘、暗色模式 |
| Phase 4 | 第 7–8 周 | 打包脚本（NSIS / DMG）、代码签名、CI/CD、文档、测试与 Bug 修复 |

---

## 10. 关键风险与缓解措施

| 风险 | 缓解措施 |
|------|---------|
| macOS Gatekeeper 拦截 | 申请 Apple Developer ID，构建时完成签名与公证流程 |
| Windows SmartScreen 拦截 | 使用 EV 代码签名证书，安装包注册防火墙例外规则 |
| 端口占用冲突 | 启动前检测端口可用性，给出明确错误提示及建议端口 |
| 内存缓冲日志过多 | 环形缓冲限制 1000 条，超出自动丢弃最旧条目 |
| 配置文件损坏 | 每次写入前备份当前文件，校验失败时自动回滚 |
| Wails WebView 兼容性 | Windows 要求 Edge WebView2（Win10+ 默认内置），macOS 要求 10.13+，安装包检测并提示 |

---

## 11. 附录：关键依赖版本

| 依赖 | 版本 | 用途 |
|------|------|------|
| Go | 1.22+ | 主语言 |
| wails | v2.9+ | 桌面框架 |
| vue | 3.x | 前端框架 |
| naive-ui | 2.x | UI 组件库 |
| echarts | 5.x | 数据图表 |
| pinia | 2.x | 状态管理 |
| gopkg.in/yaml.v3 | latest | YAML 解析 |
| go.uber.org/zap | latest | 结构化日志 |
| gopkg.in/natefinish/lumberjack.v2 | v2 | 日志轮转 |
| github.com/fsnotify/fsnotify | latest | 文件变更监听 |
| github.com/getlantern/systray | latest | 系统托盘 |

---

*文档结束 · Proxy Server Desktop Application Design Document v1.0*
