# 路由规则与网卡出站绑定设计

> 文档目标：为 GoProxy 增加类似 Proxifier 的“规则匹配 + 指定网卡出站”能力，支持按目标 IP、目标域名、CIDR、进程来源扩展字段进行路由决策，并将匹配到的连接绑定到指定本机网卡或本机源 IP 发出。

---

## 1. 目标与范围

### 1.1 目标

- 支持为代理流量定义多条路由规则。
- 支持按目标地址类型进行匹配：
  - 单个 IP
  - IP 段 / CIDR
  - 域名精确匹配
  - 域名通配匹配
- 支持将命中的连接绑定到指定出口：
  - 指定网卡
  - 指定本地源 IP
  - 默认系统路由
- 支持规则优先级与首条命中原则。
- 支持在 UI 中增删改查规则，并可实时查看当前规则是否生效。

### 1.2 非目标

- 不实现内核级路由表修改。
- 不直接替代 Windows/macOS 系统路由表。
- 不在第一期实现按进程名识别真实客户端进程。
- 不在第一期实现 UDP 路由。
- 不在第一期实现 GEOIP / ASN / 国家地区规则。

---

## 2. 用户场景

### 2.1 典型场景

- 将访问 `*.bilibili.com` 的流量固定从网卡 A 出口发送。
- 将访问 `192.168.50.0/24` 的流量固定从网卡 B 出口发送。
- 将某些目标地址强制走指定内网 IP，避免默认路由误选。
- 在双网卡、多运营商、多出口环境中，精细控制代理服务端的出站路径。

### 2.2 用户预期

- 规则像防火墙规则一样，按顺序匹配。
- 能看懂每条规则命中了什么。
- 能明确知道“这个连接最终从哪个本地 IP 发出”。
- 当绑定失败时，日志中能清楚说明失败原因。

---

## 3. 设计原则

- 规则求稳：首期只做 TCP 出站绑定，不碰系统全局路由。
- 行为可解释：每条连接都能追踪命中的规则与实际选中的出口。
- 默认安全：未命中规则时继续走系统默认路由，不影响现有用户。
- 逐步增强：先做目标地址规则，再做更复杂条件。
- 跨平台兼容：核心能力基于 Go `net.Dialer.LocalAddr` 实现。

---

## 4. 核心方案

### 4.1 总体思路

当前 GoProxy 的出站连接主要在：

- `internal/proxy/http_connect.go` 中的 `dialProxyTarget`
- `internal/proxy/socks5.go` 中对目标的拨号

新增“路由规则引擎”后，拨号流程调整为：

1. 解析目标地址。
2. 生成规则匹配上下文。
3. 根据规则列表进行匹配。
4. 得到出口策略：
   - 默认路由
   - 指定本地源 IP
   - 指定网卡
5. 构造带 `LocalAddr` 的 `net.Dialer`。
6. 执行目标解析与拨号。
7. 记录命中规则、绑定源 IP、最终连接结果。

### 4.2 为什么采用“绑定本地源 IP”而不是直接改系统路由

原因：

- 改系统路由需要管理员权限，风险高。
- 跨平台实现差异大。
- 容易影响整机网络，不仅影响 GoProxy。
- GoProxy 作为用户态代理，更适合控制“自己的出站连接”。

因此首期建议：

- UI 上允许用户“选择网卡”
- 后端实际将网卡解析成该网卡下的一个本地 IPv4 地址
- 拨号时通过 `Dialer.LocalAddr` 绑定本地源地址

这样实现更稳，也更符合当前项目架构。

---

## 5. 规则模型

### 5.1 规则字段

建议新增 `RouteRule`：

```go
type RouteRule struct {
    ID          string   `yaml:"id" json:"id"`
    Name        string   `yaml:"name" json:"name"`
    Enabled     bool     `yaml:"enabled" json:"enabled"`
    Priority    int      `yaml:"priority" json:"priority"`
    Protocols   []string `yaml:"protocols" json:"protocols"` // socks5, http
    MatchType   string   `yaml:"match_type" json:"matchType"` // ip, cidr, domain, wildcard, any
    Targets     []string `yaml:"targets" json:"targets"`
    Outbound    OutboundBinding `yaml:"outbound" json:"outbound"`
    Remark      string   `yaml:"remark" json:"remark"`
}

type OutboundBinding struct {
    Mode        string `yaml:"mode" json:"mode"` // default, local_ip, interface
    LocalIP     string `yaml:"local_ip" json:"localIp"`
    Interface   string `yaml:"interface" json:"interface"`
}
```

### 5.2 匹配类型

支持如下 `MatchType`：

| 类型 | 示例 | 说明 |
|------|------|------|
| `any` | `*` | 匹配全部目标 |
| `ip` | `1.2.3.4` | 单个 IP |
| `cidr` | `192.168.10.0/24` | IP 段 |
| `domain` | `api.github.com` | 精确域名 |
| `wildcard` | `*.bilibili.com` | 域名后缀匹配 |

### 5.3 优先级规则

- 按 `Priority` 从小到大排序。
- 数值越小优先级越高。
- 同优先级按配置顺序处理。
- 首条命中即停止。
- 若无规则命中，则走默认路由。

### 5.4 推荐默认规则

建议系统默认带一条不可删除的兜底规则：

```yaml
- id: default
  name: 默认路由
  enabled: true
  priority: 10000
  protocols: [socks5, http]
  match_type: any
  targets: ["*"]
  outbound:
    mode: default
```

---

## 6. 配置结构设计

### 6.1 设计目标

路由规则建议从主配置 `config.yaml` 中拆出，使用独立文件保存，并允许前台切换不同规则文件。

这样做的原因：

- 路由规则通常会频繁调整，单独存放更清晰。
- 用户可能有多套规则场景，需要快速切换。
- 主配置关注服务参数，路由文件关注流量策略，职责更单一。
- 便于备份、导入导出和共享规则文件。

### 6.2 文件存放方式

约定所有路由规则文件都放在应用目录下的 `configs` 目录中：

```text
<应用目录>/
  configs/
    config.yaml
    default.rule
    office.rule
    bilibili.rule
```

文件扩展名统一为：

- `.rule`

默认路由文件建议命名为：

- `default.rule`

### 6.3 主配置中的最小挂载信息

主配置中不再直接保存完整规则列表，而只保存“路由功能状态 + 当前选中的规则文件”：

```yaml
route:
  enabled: false
  active_file: "default.rule"
```

建议新增：

```go
type RouteConfig struct {
    Enabled    bool   `yaml:"enabled" json:"enabled"`
    ActiveFile string `yaml:"active_file" json:"activeFile"`
}
```

并在顶层 `Config` 中挂载：

```go
Route RouteConfig `yaml:"route" json:"route"`
```

### 6.4 路由规则文件结构

每个 `.rule` 文件保存一整套路由规则：

```yaml
name: 默认规则集
version: 1
updated_at: "2026-05-01T14:00:00+08:00"
description: 默认出口策略
rules:
  - id: bilibili
    name: 哔哩哔哩走网卡A
    enabled: true
    priority: 100
    protocols: [socks5, http]
    match_type: wildcard
    targets:
      - "*.bilibili.com"
    outbound:
      mode: interface
      interface: "以太网"
    remark: "视频流量固定走主宽带"

  - id: office-subnet
    name: 办公网段走内网卡
    enabled: true
    priority: 200
    protocols: [socks5, http]
    match_type: cidr
    targets:
      - "10.20.0.0/16"
    outbound:
      mode: local_ip
      local_ip: "10.20.1.15"

  - id: default
    name: 默认路由
    enabled: true
    priority: 10000
    protocols: [socks5, http]
    match_type: any
    targets: ["*"]
    outbound:
      mode: default
```

建议新增规则文件模型：

```go
type RouteRuleSet struct {
    Name        string      `yaml:"name" json:"name"`
    Version     int         `yaml:"version" json:"version"`
    UpdatedAt   string      `yaml:"updated_at" json:"updatedAt"`
    Description string      `yaml:"description" json:"description"`
    Rules       []RouteRule `yaml:"rules" json:"rules"`
}
```

### 6.5 路由文件管理接口

建议新增独立的 `RouteFileManager`，负责：

- 枚举 `configs` 目录中的 `.rule` 文件
- 加载指定 `.rule` 文件
- 保存指定 `.rule` 文件
- 创建新规则文件
- 删除规则文件
- 校验文件名合法性

建议新增后端接口：

```go
ListRouteFiles() ([]RouteFileInfo, error)
LoadRouteFile(name string) (RouteRuleSet, error)
SaveRouteFile(name string, data RouteRuleSet) error
CreateRouteFile(name string) error
DeleteRouteFile(name string) error
SetActiveRouteFile(name string) error
```

返回结构示例：

```go
type RouteFileInfo struct {
    Name      string `json:"name"`
    IsActive  bool   `json:"isActive"`
    UpdatedAt string `json:"updatedAt"`
}
```

### 6.6 文件名约束

建议路由文件名约束如下：

- 必须以 `.rule` 结尾
- 只能包含字母、数字、`-`、`_`
- 不允许路径穿越字符，如 `..`、`/`、`\`
- 不允许覆盖 `config.yaml`

建议示例：

- `default.rule`
- `office.rule`
- `mobile_backup.rule`

不建议：

- `../../test.rule`
- `规则1.rule`
- `route config.rule`

### 6.7 当前生效规则文件

运行时只加载一个“当前生效规则文件”。

流程建议：

1. 应用启动时读取 `config.yaml`
2. 获取 `route.active_file`
3. 从 `configs/<active_file>` 加载规则集
4. 构建内存中的路由规则引擎
5. 用户在前台切换规则文件时：
   - 更新 `config.yaml` 中的 `active_file`
   - 重新加载规则文件
   - 原子替换内存中的规则引擎

如果配置的活动文件不存在：

- 记录警告日志
- 自动回退到 `default.rule`
- 如果 `default.rule` 也不存在，则创建默认规则文件

### 6.8 默认规则文件初始化

建议首次启动时自动生成：

- `configs/default.rule`

内容至少包含一条兜底规则：

```yaml
name: 默认规则集
version: 1
rules:
  - id: default
    name: 默认路由
    enabled: true
    priority: 10000
    protocols: [socks5, http]
    match_type: any
    targets: ["*"]
    outbound:
      mode: default
```

### 6.9 校验规则

除了原有规则内容校验外，还需要补充文件级校验：

- `name` 非空
- `version` 必须大于等于 1
- `rules` 至少包含一条规则
- 建议必须包含一条 `match_type=any` 的兜底规则
- 文件内 `id` 不得重复

### 6.10 热切换行为

切换活动规则文件时，建议行为如下：

- 新连接立即使用新规则文件
- 已建立连接继续沿用旧决策，不强制中断

原因：

- 避免切换规则时影响现有连接稳定性
- 与代理服务“平滑变更”习惯一致

---

## 7. 后端架构设计

### 7.1 新增模块
建议新增：

```text
internal/config/
  route_file_manager.go   # .rule 文件读写与枚举

internal/proxy/
  route_policy.go         # 规则结构、匹配器、排序
  outbound_bind.go        # 网卡解析、本地 IP 选择、Dialer 构建
```

### 7.2 校验规则

建议在 `validator.go` 中增加主配置校验，在 `route_file_manager.go` 中增加规则文件校验：

- `active_file` 必须以 `.rule` 结尾
- `priority` 必须为正整数
- `match_type` 必须是允许值
- `targets` 不得为空
- `mode=local_ip` 时必须填写合法 IP
- `mode=interface` 时必须填写非空网卡名
- `protocols` 只能包含 `socks5` / `http`
- `id` 必须唯一
### 7.3 匹配上下文

建议定义：

```go
type RouteContext struct {
    Protocol   string // socks5 / http
    TargetHost string
    TargetPort string
    IsIP       bool
}
```

### 7.4 路由决策结果

```go
type RouteDecision struct {
    RuleID        string
    RuleName      string
    OutboundMode  string
    InterfaceName string
    LocalIP       string
}
```

### 7.5 核心接口

```go
type RoutePolicyEngine interface {
    Match(ctx RouteContext) RouteDecision
}
```

### 7.6 拨号流程建议

现有 `dialProxyTarget` 可以演进为：

```go
func (s *Server) dialProxyTarget(ctx context.Context, protocol, targetAddr string) (net.Conn, error)
```

内部步骤：

1. 拆分 `host:port`
2. 构造 `RouteContext`
3. 执行规则匹配
4. 构造 `net.Dialer`
5. 如果命中 `local_ip`，则：
   - `Dialer.LocalAddr = &net.TCPAddr{IP: parsedIP}`
6. 如果命中 `interface`，则：
   - 读取该网卡地址列表
   - 选一个合适 IPv4
   - 转成 `LocalAddr`
7. 继续执行当前“优先 IPv4、再试 IPv6”的目标连接逻辑

### 7.7 网卡绑定实现建议

Go 标准库不能直接“按网卡名拨号”，但可以：

1. `net.Interfaces()` 找到目标网卡
2. 遍历 `iface.Addrs()`
3. 选出一个可用的本地 IPv4 地址
4. 用它作为 `Dialer.LocalAddr`

因此“按网卡路由”本质会映射为“按该网卡 IP 绑定本地源地址”。

### 7.8 连接记录增强

建议扩展活跃连接快照：

```go
type ConnectionSnapshot struct {
    ...
    RouteRuleName string `json:"routeRuleName"`
    OutboundIP    string `json:"outboundIp"`
    OutboundIface string `json:"outboundIface"`
}
```

这样前端可以看到每条连接命中了哪条规则，以及最终从哪个本地 IP 发出。

---

## 8. 前端 UI 设计

### 8.1 新页面

建议新增左侧导航页：

- `路由规则`

### 8.2 页面布局

建议页面包含三个区域：

1. 顶部规则文件选择区
   - 当前活动规则文件下拉框
   - 新建规则文件
   - 另存为
   - 删除规则文件
2. 全局开关
   - 启用路由规则
3. 规则列表
   - 名称
   - 优先级
   - 匹配条件
   - 出口绑定
   - 启用状态
   - 操作按钮
4. 编辑弹窗 / 抽屉
   - 创建/编辑规则

### 8.3 规则文件交互

前台应支持：

- 枚举 `configs` 目录中的所有 `.rule`
- 切换当前生效文件
- 编辑当前文件中的规则
- 将当前规则另存为新文件
- 删除非活动文件

推荐交互：

- 顶部显示“当前规则文件：`default.rule`”
- 切换文件时弹出确认框：
  - “切换后，新连接将使用新的路由规则”

### 8.4 表单字段

- 规则名称
- 是否启用
- 优先级
- 协议范围
- 匹配类型
- 目标列表
- 出口模式
- 网卡选择 / 本地 IP 选择
- 备注

### 8.5 网卡选择数据来源

建议新增后端方法：

```go
GetNetworkInterfaces() ([]NetworkInterface, error)
```

返回：

```go
type NetworkInterface struct {
    Name        string   `json:"name"`
    DisplayName string   `json:"displayName"`
    Addresses   []string `json:"addresses"`
    Up          bool     `json:"up"`
    Loopback    bool     `json:"loopback"`
}
```

### 8.6 连接页面增强

建议在活跃连接页新增列：

- 命中规则
- 出口 IP
- 出口网卡

这样用户可以快速验证规则是否生效。

---

## 9. 日志与可观测性

### 9.1 新增日志

建议在每次出站连接前记录：

```text
[route] matched file="default.rule" rule="哔哩哔哩走网卡A" protocol=socks5 target=api.bilibili.com:443 mode=interface iface=以太网 local_ip=192.168.1.7
```

拨号成功后记录：

```text
[route] outbound connected target=api.bilibili.com:443 local=192.168.1.7:52341 remote=61.147.x.x:443
```

绑定失败时记录：

```text
[route] resolve interface failed iface="以太网2": no usable ipv4 address
```

切换规则文件时建议记录：

```text
[route] active rule file changed file="office.rule"
```

### 9.2 调试价值

这些日志可以帮助用户定位：

- 规则没命中
- 网卡选错
- 网卡没有可用 IPv4
- 绑定源 IP 后目标不可达

---

## 10. 风险与边界

### 10.1 绑定本地 IP 不等于绝对控制系统路由

即使绑定了本地 IP，最终路径仍受系统路由表影响。通常在多网卡主机上，这已经足够接近用户想要的效果，但仍有边界：

- 某个本地 IP 没有可达默认路由
- 某个目标只能通过特定路由表到达
- 某些系统会拒绝从不匹配出口的源地址发包

### 10.2 域名规则的匹配时机

SOCKS5 / HTTP CONNECT 在大多数情况下能拿到目标域名，因此域名规则可直接匹配。  
如果客户端直接提交的是 IP，则只能按 IP/CIDR 规则匹配。

### 10.3 Windows / macOS 差异

- Windows 网卡显示名与系统内部名可能不同
- macOS 存在 `en0/en1` 这类系统名
- UI 最好同时展示“显示名 + 地址列表”

### 10.4 IPv6 支持策略

首期建议：

- 规则绑定优先支持 IPv4
- 后续再补 IPv6 本地地址绑定

原因：

- 当前项目对“优先 IPv4”已经有明显优化诉求
- 先把 IPv4 稳定性做好更实用

---

## 11. 分期实施建议

### Phase A：最小可用版本

- 主配置中新增 `route.active_file`
- 实现 `.rule` 文件管理
- 支持 `ip / cidr / domain / wildcard / any`
- 支持 `default / local_ip / interface`
- 规则按优先级首条命中
- SOCKS5 / HTTP 共用一套路由决策
- UI 支持规则列表与编辑
- 活跃连接显示命中规则和出口 IP

### Phase B：增强可用性

- 支持拖拽排序
- 支持规则测试器
- 支持批量导入导出
- 支持日志筛选 `route:*`

### Phase C：高级能力

- 支持 IPv6 出站绑定
- 支持域名缓存与统计
- 支持 GEOIP / ASN 规则
- 支持按认证用户分流
- 支持按监听入口分流

---

## 12. 验收标准

- 用户可以创建、编辑、删除、启用/禁用规则。
- 规则命中逻辑符合“优先级 + 首条命中”。
- 指定本地 IP 的流量能从对应源地址发出。
- 指定网卡的流量能正确映射到该网卡 IP 发出。
- 未命中规则的连接行为与当前版本保持一致。
- 活跃连接页可看到规则命中结果。
- 日志可看到路由决策与失败原因。

---

## 13. 推荐落地顺序

1. 先改配置模型与校验。
2. 抽象统一拨号入口。
3. 实现规则匹配器。
4. 实现本地 IP / 网卡绑定拨号。
5. 扩展连接快照与日志。
6. 最后补前端规则页与联调。

---

## 14. 建议文件改动清单

后端：

- `internal/config/config.go`
- `internal/config/validator.go`
- `internal/config/route_file_manager.go` 新增
- `internal/proxy/http_connect.go`
- `internal/proxy/socks5.go`
- `internal/proxy/server.go`
- `internal/platform/network.go`
- `internal/proxy/route_policy.go` 新增
- `internal/proxy/outbound_bind.go` 新增

前端：

- `frontend/src/types.ts`
- `frontend/src/backend/api.ts`
- `frontend/src/stores/config.ts`
- `frontend/src/pages/RouteRulesPage.vue` 新增
- `frontend/src/pages/ActiveConnectionsPage.vue`
- `frontend/src/App.vue`

文档：

- `docs/route-policy-design.md`

---

## 15. 总结

这套方案的核心不是修改系统路由，而是让 GoProxy 在建立每一条目标连接时，先做一次规则决策，再将连接绑定到指定本地源 IP 或指定网卡对应的本地 IP。

这样做有几个优点：

- 不需要管理员权限
- 风险低
- 易于解释
- 与当前 GoProxy 架构高度兼容
- 可以逐步演进成更强的流量分流系统

如果后续你决定真正落地这个功能，建议优先实现 Phase A。它已经能覆盖大多数“像 Proxifier 一样按目标走不同出口”的核心需求。
