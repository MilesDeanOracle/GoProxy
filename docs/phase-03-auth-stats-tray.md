# Phase 3：认证管理、流量统计与系统托盘

> 周期建议：第 5-6 周  
> 阶段目标：补齐桌面应用的增强能力，包括用户认证、实时流量统计、可视化图表、系统托盘和主题设置。

## 1. 实现功能

- 实现 SOCKS5 用户名密码认证。
- 实现 HTTP CONNECT Proxy-Authorization 认证。
- 实现认证管理页面：开启认证、用户列表、新增用户、删除用户、重置密码。
- 密码使用 bcrypt hash 存储，禁止明文落盘。
- 实现流量统计：活跃连接、总连接、上下行累计流量、实时上传和下载速率。
- 每秒通过 Wails 事件推送统计快照。
- 前端使用 ECharts 展示近 60 秒流量速率曲线。
- 实现系统托盘菜单：显示窗口、启动服务、停止服务、退出。
- 实现主题设置：亮色、暗色、跟随系统。

## 2. 设计方案

### 2.1 认证模块

认证配置：

```yaml
auth:
  enabled: false
  users:
    - username: admin
      password: "$2a$10$..."
```

认证规则：

- 认证关闭时，SOCKS5 使用 no-auth，HTTP CONNECT 不要求 Proxy-Authorization。
- 认证开启时，至少需要一个用户。
- 用户名必须唯一。
- 密码只在新增或重置时以明文进入后端，保存时立即 hash。
- 日志中禁止输出明文密码和完整认证头。

SOCKS5 认证实现 RFC 1929 username/password 子协商。HTTP CONNECT 认证使用 `Proxy-Authorization: Basic ...`。

### 2.2 StatsCollector

统计模块负责接收代理核心上报的连接和字节事件，并聚合为快照：

```go
type Stats struct {
    ActiveConns   int64 `json:"activeConns"`
    TotalConns    int64 `json:"totalConns"`
    UploadBytes   int64 `json:"uploadBytes"`
    DownloadBytes int64 `json:"downloadBytes"`
    UploadRate    int64 `json:"uploadRate"`
    DownloadRate  int64 `json:"downloadRate"`
    AuthFailures  int64 `json:"authFailures"`
}
```

设计要求：

- 字节和连接计数使用 `sync/atomic`。
- 每秒计算一次速率差值。
- 统计数据只保存在内存，本次运行有效，应用重启后清零。
- Relay Engine 在 `io.Copy` 外包裹计数 reader 或 writer，避免侵入协议 handler。

### 2.3 前端统计展示

页面分工：

- 仪表盘：活跃连接、上传速率、下载速率、总流量、近 60 秒折线图。
- 流量统计页：当前会话总连接数、上下行总量、认证失败次数、协议分布占位。

前端维护固定长度的 60 个采样点，收到 `proxy:stats` 后追加新点并丢弃最旧点。

### 2.4 系统托盘

托盘行为：

- 最小化时隐藏主窗口并保留托盘图标。
- 双击托盘图标恢复窗口。
- 右键菜单包含显示窗口、启动服务、停止服务、退出。
- 服务运行状态变更时更新菜单状态。

macOS 和 Windows 的托盘能力存在差异，实现时应把平台差异隔离在独立模块中。

### 2.5 主题与应用设置

设置项：

```yaml
ui:
  theme: auto
  language: zh-CN
  start_minimized: false
  show_tray_icon: true
```

主题状态由前端 store 管理，保存到 YAML 后下次启动恢复。`auto` 模式跟随系统主题。

## 3. 任务拆解

| 编号 | 任务 | 说明 |
|------|------|------|
| P3-01 | 实现 Auth Module | 用户校验、bcrypt hash、配置集成 |
| P3-02 | 接入 SOCKS5 认证 | 支持 username/password 子协商 |
| P3-03 | 接入 HTTP CONNECT 认证 | 支持 Basic Proxy-Authorization |
| P3-04 | 实现认证管理绑定方法 | `AddUser`、`RemoveUser`、密码重置 |
| P3-05 | 实现认证管理页 | 开关、用户表格、新增、删除、重置密码 |
| P3-06 | 完善 StatsCollector | 原子计数、速率计算、快照导出 |
| P3-07 | 接入 Relay 字节统计 | 上下行分别计数，统计连接生命周期 |
| P3-08 | 实现统计事件推送 | 每秒发送 `proxy:stats` |
| P3-09 | 实现图表页面 | ECharts 折线图、统计卡片、空状态 |
| P3-10 | 实现系统托盘 | 菜单、状态同步、窗口显示隐藏 |
| P3-11 | 实现主题设置 | 亮色、暗色、跟随系统和配置保存 |

## 4. 验收标准

- 开启认证后，未提供凭据或凭据错误的连接会被拒绝。
- SOCKS5 和 HTTP CONNECT 都能使用正确账号密码通过认证。
- 配置文件中不出现明文密码。
- UI 可以新增、删除用户并保存到配置。
- 仪表盘每秒刷新连接数和流量速率。
- ECharts 展示最近 60 秒上行和下行速率曲线。
- 最小化到托盘、从托盘恢复、托盘启动停止服务可用。
- 主题切换后立即生效，并能在重启后恢复。

## 5. 风险与处理

| 风险 | 处理方案 |
|------|----------|
| 明文密码泄露到日志或配置 | 对认证相关日志做字段白名单，配置只保存 bcrypt hash |
| 高并发统计产生锁竞争 | 使用 atomic 累加，定时快照时再聚合 |
| 图表频繁重绘影响性能 | 前端固定窗口数据，按秒更新，不按字节事件更新 |
| 托盘跨平台行为不一致 | 平台差异封装在独立模块，UI 只依赖统一状态 |

