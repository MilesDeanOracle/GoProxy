# 开发规范

> 本项目所有开发、评审、测试和发布工作都必须遵循本文档。若本文档与临时代码实现冲突，以本文档为准；确需调整规范时，必须先更新本文档并说明原因。

## 1. 基本原则

- 先保持系统可运行，再逐步扩展功能。
- 后端业务逻辑必须放在 `internal/` 模块中，`app.go` 只作为 Wails 绑定和编排层。
- UI、配置、代理核心、日志、统计、认证模块之间通过明确接口通信，禁止跨层直接读写内部状态。
- 所有外部输入都必须校验，包括 UI 表单、配置文件、协议请求、导入文件路径。
- 任何涉及密码、认证头、证书、密钥的信息禁止写入日志。
- 新功能必须同时考虑 Windows 和 macOS 行为差异。

## 2. 目录规范

推荐目录结构：

```text
.
  main.go
  app.go
  wails.json
  internal/
    config/
    proxy/
    stats/
    logger/
    platform/
  frontend/
    src/
      pages/
      components/
      stores/
  docs/
  build/
    windows/
    darwin/
```

目录职责：

| 目录 | 职责 |
|------|------|
| `internal/config` | 配置结构、默认值、YAML 读写、校验、热重载 |
| `internal/proxy` | SOCKS5、HTTP CONNECT、Relay、认证、服务生命周期 |
| `internal/stats` | 连接数、字节数、速率统计和快照 |
| `internal/logger` | zap 初始化、日志轮转、环形缓冲、日志事件 |
| `internal/platform` | 系统托盘、平台路径、开机自启等平台差异能力 |
| `frontend/src/pages` | 页面级组件 |
| `frontend/src/components` | 可复用 UI 组件 |
| `frontend/src/stores` | Pinia 状态管理 |
| `docs` | 阶段计划、设计、开发规范、用户文档 |

## 3. Go 编码规范

- Go 版本使用 `1.22+`。
- 所有代码必须通过 `gofmt`，推荐同时通过 `go vet ./...`。
- 包名使用小写单词，避免下划线和复数滥用。
- 公开类型、函数和方法必须有简洁注释。
- 错误必须向上返回并保留上下文，禁止静默吞错。
- 禁止在库代码中直接 `panic` 或 `os.Exit`，入口层除外。
- 长生命周期 goroutine 必须能通过 `context.Context` 或显式关闭方法退出。
- 网络连接、文件、ticker、watcher 必须明确关闭。
- 并发共享状态优先使用 channel、atomic 或受控 mutex，禁止裸 map 并发读写。

错误处理示例：

```go
if err := manager.Save(cfg); err != nil {
    return fmt.Errorf("save config: %w", err)
}
```

## 4. 后端分层规范

### 4.1 Wails 绑定层

`app.go` 只能做：

- 参数接收与返回。
- 调用 service 或 manager。
- Wails 事件发送。
- 应用生命周期初始化和释放。

禁止在 `app.go` 中直接实现协议解析、文件写入细节、统计计算和复杂 UI 状态逻辑。

### 4.2 业务模块

每个业务模块必须具备清晰职责：

- `ConfigManager`：只负责配置读写、校验、备份、导入导出。
- `ProxyServer`：只负责代理服务生命周期和连接调度。
- `AuthManager`：只负责用户认证和密码 hash 校验。
- `StatsCollector`：只负责统计数据更新和快照。
- `LogManager`：只负责日志写入、缓冲和推送。

模块之间通过接口或构造参数依赖，避免全局变量。

## 5. 配置规范

- 配置文件格式固定为 YAML。
- 所有配置字段必须有默认值、注释和校验规则。
- 配置写入必须采用“校验 -> 备份 -> 临时文件 -> 原子替换 -> 失败回滚”流程。
- 配置导入必须先校验完整文件，不允许部分导入污染当前配置。
- 监听地址、端口、协议开关等影响 listener 的配置，运行中保存后需提示重启服务生效。
- 密码字段只允许保存 hash。

## 6. 日志规范

- 日志库使用 zap。
- 日志级别固定为 `debug`、`info`、`warn`、`error`。
- 运行日志必须结构化，至少包含时间、级别、消息、来源模块。
- 错误日志应包含错误原因和上下文，但不得包含敏感信息。
- UI 日志环形缓冲默认保留最近 1000 条。
- “清空日志”只清空 UI 展示，不删除磁盘日志文件。

禁止记录：

- 明文密码。
- 完整认证头。
- 证书私钥。
- 用户本地敏感路径中的隐私部分。

## 7. 代理核心规范

- SOCKS5 必须按 RFC 1928 实现 CONNECT，认证按 RFC 1929 实现。
- HTTP CONNECT 只处理隧道代理，不在当前版本实现普通 HTTP 正向代理。
- 所有握手、读取和目标建连必须设置超时。
- Relay 必须支持双向转发、半关闭和上下文取消。
- 连接数必须受 `max_connections` 限制。
- 服务停止必须关闭 listener，并尽力释放所有活跃连接。
- 协议解析错误应返回明确响应并写入 warn 或 debug 日志。

## 8. 前端规范

- 前端使用 Vue 3 + Composition API。
- 状态管理使用 Pinia。
- UI 组件优先使用 Naive UI。
- 图表使用 ECharts。
- 页面组件负责展示和交互，业务状态放在 store 中。
- Wails 绑定调用必须集中封装，页面不直接散落大量后端调用细节。
- 所有表单必须有前端校验，同时以后端校验结果为准。
- 长列表日志必须限制渲染数量或采用虚拟列表。
- UI 文案使用中文，错误信息要面向用户可理解。

## 9. Wails 通信规范

后端方法命名使用动词开头，返回值必须稳定：

- `GetConfig`
- `SaveConfig`
- `StartServer`
- `StopServer`
- `GetServerStatus`
- `GetStats`
- `GetRecentLogs`
- `AddUser`
- `RemoveUser`
- `ExportConfig`
- `ImportConfig`

事件名使用 `domain:event` 格式：

| 事件 | 用途 |
|------|------|
| `proxy:log` | 实时日志 |
| `proxy:stats` | 每秒统计快照 |
| `proxy:status` | 服务状态变更 |
| `config:changed` | 配置文件外部变更 |

事件 payload 必须是可 JSON 序列化的结构体，字段名保持 camelCase。

## 10. 测试规范

- 新增 Go 业务逻辑必须配套单元测试。
- 协议处理、认证、配置校验、Relay Engine 必须有重点测试。
- 修复 bug 时必须新增能复现问题的测试，除非无法自动化。
- 发布前必须执行：

```bash
go test ./...
go vet ./...
```

前端发布前必须执行项目定义的 lint、typecheck 和 build 命令。

## 11. 安全规范

- 密码使用 bcrypt hash。
- 配置导入和导出路径必须校验，避免覆盖非预期文件。
- 日志和错误信息禁止泄露敏感数据。
- 默认监听地址可以是 `0.0.0.0`，但 UI 必须清晰提示公网暴露风险。
- 认证关闭时，UI 必须提示代理可能被局域网或公网滥用。
- 依赖升级必须关注安全公告和破坏性变更。

## 12. Git 与评审规范

- 提交应聚焦单一目的，避免混合无关改动。
- 提交信息建议格式：`type(scope): summary`。
- 常用 type：`feat`、`fix`、`docs`、`test`、`refactor`、`build`、`chore`。
- PR 或合并前必须说明功能变更、测试结果和风险。
- 禁止提交本地配置、证书、密钥、构建产物和依赖缓存。

## 13. 文档规范

- 设计变更必须同步更新 `docs/`。
- 配置字段新增或变更必须同步更新配置说明。
- Wails 绑定方法和事件 payload 变更必须同步更新通信文档。
- 发布流程、签名流程和用户安装步骤必须保持可执行。

## 14. 发布规范

- 发布版本必须通过全部自动化测试。
- 发布产物必须包含版本号、平台和架构。
- Windows 和 macOS 发布包必须分别完成冒烟测试。
- 签名证书、公证凭据和 CI secret 禁止进入仓库。
- 发布说明必须列出新增功能、修复问题、兼容性变化和已知问题。

