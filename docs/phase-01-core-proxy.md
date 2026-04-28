# Phase 1：代理核心与 CLI 验证

> 周期建议：第 1-2 周  
> 阶段目标：完成可独立运行和测试的代理服务核心，支持 SOCKS5、HTTP CONNECT、连接转发、基础配置和命令行启动。

## 1. 实现功能

- 初始化 Go 项目结构与基础工程配置。
- 实现 SOCKS5 TCP CONNECT 代理服务。
- 实现 HTTP/1.1 CONNECT 代理服务。
- 实现双向 Relay Engine，支持半关闭、超时、上下文取消和连接清理。
- 实现基础配置结构，可从 YAML 文件加载监听地址、端口、超时和最大连接数。
- 实现 ProxyServer 生命周期管理：启动、停止、状态查询。
- 提供 CLI 或临时入口用于本阶段本地验证。
- 补齐核心模块单元测试与基础集成测试。

## 2. 设计方案

### 2.1 模块边界

本阶段只关注后端代理能力，不引入 Wails 和前端 UI。核心模块应放在 `internal/` 下，避免被外部项目直接依赖。

建议目录：

```text
internal/
  config/
    config.go
    manager.go
    validator.go
  proxy/
    server.go
    socks5.go
    http_connect.go
    relay.go
    auth.go
  stats/
    collector.go
```

### 2.2 ProxyServer 生命周期

`ProxyServer` 负责统一管理 SOCKS5 与 HTTP CONNECT 监听器：

- `Start(ctx)`：读取配置、检查端口、启动监听 goroutine。
- `Stop()`：关闭 listener、取消上下文、等待连接退出。
- `Status()`：返回运行状态、监听端口、活跃连接数和启动时间。

所有 goroutine 必须受 `context.Context` 或 listener 关闭控制，避免停止服务后遗留后台任务。

### 2.3 SOCKS5 协议处理

第一阶段支持 RFC 1928 的 TCP CONNECT：

- 支持 `NO AUTHENTICATION REQUIRED`。
- 支持 IPv4、IPv6、Domain 三种目标地址类型。
- 不支持 BIND、UDP ASSOCIATE，收到后返回不支持命令。
- 协议解析必须设置读写 deadline，避免恶意慢连接占用资源。

### 2.4 HTTP CONNECT 处理

HTTP CONNECT 处理流程：

1. 读取并解析请求行和 Header。
2. 校验 Method 必须为 `CONNECT`。
3. 解析 `host:port` 目标地址。
4. 与目标建立 TCP 连接。
5. 返回 `HTTP/1.1 200 Connection Established`。
6. 进入 Relay Engine。

非 CONNECT 请求返回明确错误，不在当前阶段实现普通 HTTP 正向代理。

### 2.5 Relay Engine

Relay Engine 使用两个 goroutine 进行双向 `io.Copy`：

- 客户端到目标：统计上行字节。
- 目标到客户端：统计下行字节。
- 支持 TCP half-close：一侧 EOF 后调用 `CloseWrite` 通知对端。
- 任意方向发生不可恢复错误或上下文取消时关闭双方连接。

必须限制最大并发连接数。超出限制时拒绝新连接并返回清晰错误。

### 2.6 配置设计

本阶段配置项覆盖代理核心运行所需参数：

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

relay:
  dial_timeout_sec: 10
  read_timeout_sec: 30
  max_connections: 1000
  keepalive_sec: 15
```

配置加载必须提供默认值和校验：

- 端口范围：`1-65535`。
- 超时必须为正数。
- `max_connections` 必须大于 0。
- 至少启用一种入站协议。

## 3. 任务拆解

| 编号 | 任务 | 说明 |
|------|------|------|
| P1-01 | 初始化 Go module | 创建 `go.mod`，确定 Go 版本和基础依赖 |
| P1-02 | 定义配置结构 | 实现 Config struct、默认配置、YAML 加载和校验 |
| P1-03 | 实现 ProxyServer | 管理 listener、上下文、连接计数和服务状态 |
| P1-04 | 实现 SOCKS5 handler | 完成握手、CONNECT 解析、目标建连 |
| P1-05 | 实现 HTTP CONNECT handler | 完成 CONNECT 解析、响应和目标建连 |
| P1-06 | 实现 Relay Engine | 双向复制、半关闭、超时、错误返回 |
| P1-07 | 实现基础统计 | 活跃连接、总连接、上下行字节累加 |
| P1-08 | 增加 CLI 验证入口 | 支持指定配置文件启动代理 |
| P1-09 | 编写测试 | 协议解析、配置校验、relay、启动停止测试 |

## 4. 验收标准

- 可以通过命令行启动 SOCKS5 和 HTTP CONNECT 代理。
- SOCKS5 能代理 TCP CONNECT 请求，支持域名和 IP 目标地址。
- HTTP CONNECT 能代理 HTTPS 隧道请求。
- 达到最大连接数后，新连接会被拒绝且不会影响已有连接。
- 服务停止后 listener 和活跃连接都能释放。
- 单元测试覆盖配置校验、协议解析和 Relay Engine。
- `go test ./...` 通过。

## 5. 风险与处理

| 风险 | 处理方案 |
|------|----------|
| 协议解析遇到慢连接导致 goroutine 堆积 | 所有握手和建连过程设置 deadline |
| 半关闭处理不当导致连接无法正常结束 | Relay Engine 中对 TCP 连接使用 `CloseWrite`，并兜底关闭双方连接 |
| 并发计数不准确 | 使用 `sync/atomic` 或集中连接管理器维护活跃连接数 |
| 端口占用导致启动失败 | 启动前监听失败时返回明确错误，禁止吞错重试 |

