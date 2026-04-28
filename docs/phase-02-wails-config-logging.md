# Phase 2：Wails 集成、配置界面与日志系统

> 周期建议：第 3-4 周  
> 阶段目标：将 Phase 1 的代理核心接入 Wails 桌面应用，完成基础 UI、配置读写、服务启停和实时日志展示。

## 1. 实现功能

- 初始化 Wails v2 项目，接入 Go 后端与 Vue 3 前端。
- 实现前端仪表盘页面，展示服务状态、监听端口、启动/停止按钮。
- 实现服务配置页面，支持 SOCKS5、HTTP CONNECT、超时和最大连接数配置。
- 实现 YAML 配置读写、保存校验、写入前备份和失败回滚。
- 实现日志系统：zap 结构化日志、文件轮转、内存环形缓冲。
- 通过 Wails 事件实时推送日志到前端。
- 暴露基础 Wails 绑定方法供前端调用。

## 2. 设计方案

### 2.1 应用层职责

`app.go` 是 Wails 绑定层，只负责连接 UI 与业务服务，不直接承载复杂业务逻辑。

建议暴露方法：

| 方法 | 职责 |
|------|------|
| `GetConfig() Config` | 获取当前配置 |
| `SaveConfig(Config) error` | 校验并保存配置 |
| `StartServer() error` | 启动代理服务 |
| `StopServer() error` | 停止代理服务 |
| `GetServerStatus() Status` | 获取服务状态 |
| `GetRecentLogs(n int) []LogEntry` | 获取最近日志 |

### 2.2 配置管理

配置路径：

- Windows：`%APPDATA%\ProxyServer\config.yaml`
- macOS：`~/Library/Application Support/ProxyServer/config.yaml`

配置保存流程：

1. 前端提交完整配置对象。
2. 后端执行 schema 和业务校验。
3. 备份当前配置文件。
4. 写入临时文件。
5. 原子替换正式配置。
6. 写入失败或重载失败时回滚备份。

监听端口、协议开关等影响 listener 的配置，在服务运行中保存后不直接热更新，UI 应提示“重启服务后生效”。

### 2.3 日志系统

日志输出分三层：

- zap logger：统一结构化日志入口。
- lumberjack writer：按大小滚动磁盘日志。
- ring buffer：保留最近 1000 条供 UI 首次加载。

每条日志结构：

```go
type LogEntry struct {
    Time    string `json:"time"`
    Level   string `json:"level"`
    Message string `json:"message"`
    Source  string `json:"source"`
}
```

实时日志通过 Wails `EventsEmit(ctx, "proxy:log", entry)` 推送。前端日志清空只清除 UI 列表，不删除磁盘日志。

### 2.4 前端页面

本阶段前端优先实现可用性：

- 左侧导航 + 右侧内容区。
- 仪表盘展示运行状态、协议端口、活跃连接数占位和服务控制。
- 服务配置页面使用表单控件编辑配置。
- 实时日志页面支持级别筛选、关键字搜索、自动滚动。

状态管理建议使用 Pinia：

- `useServerStore`：运行状态、启动时间、端口信息。
- `useConfigStore`：当前配置、保存状态、脏数据状态。
- `useLogStore`：日志列表、过滤条件、自动滚动开关。

## 3. 任务拆解

| 编号 | 任务 | 说明 |
|------|------|------|
| P2-01 | 初始化 Wails 项目 | 配置 `wails.json`、Go 入口、前端工程 |
| P2-02 | 接入 Phase 1 代理核心 | 在 App 生命周期中创建 ConfigManager 和 ProxyServer |
| P2-03 | 实现 Wails 绑定方法 | 提供配置、状态、启停、日志查询方法 |
| P2-04 | 完善配置管理 | 配置路径、默认文件创建、备份、回滚、校验 |
| P2-05 | 实现日志模块 | zap 初始化、文件轮转、环形缓冲、日志事件推送 |
| P2-06 | 实现仪表盘 | 服务状态、端口、启动停止操作 |
| P2-07 | 实现服务配置页 | 表单、校验、保存、重启提示 |
| P2-08 | 实现实时日志页 | 日志加载、事件监听、筛选、搜索、自动滚动 |
| P2-09 | 增加前后端联调测试 | 验证 Wails 方法调用和事件推送 |

## 4. 验收标准

- `wails dev` 能启动桌面应用。
- 可以在 UI 中启动和停止代理服务。
- 可以在 UI 中编辑配置并写入 YAML 文件。
- 非法配置会在 UI 中显示清晰错误，不会写入正式配置。
- 服务运行中修改端口后，UI 能提示需要重启服务。
- 后端日志能实时显示在前端日志页。
- 重新打开应用时能从磁盘加载上次保存的配置。

## 5. 风险与处理

| 风险 | 处理方案 |
|------|----------|
| Wails 绑定方法承担过多业务逻辑 | `app.go` 只编排服务调用，业务放在 `internal/` |
| 配置写入中断导致文件损坏 | 采用临时文件 + 原子替换 + 写前备份 |
| 日志推送过快导致 UI 卡顿 | 前端批量合并渲染，后端保留固定大小环形缓冲 |
| 服务状态与 UI 状态不一致 | 状态变更统一由后端事件 `proxy:status` 推送 |

