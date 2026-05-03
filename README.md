# GoProxy

## 项目介绍

GoProxy 是一款专为局域网共享上网场景设计的桌面代理服务端应用。在诸多企业、学校、机房或家庭网络环境中，常常存在这样的困境：多台内网电脑需要访问互联网，但由于安全策略、网络架构或成本限制，无法为每一台机器直接分配公网出口。GoProxy 正是为解决这一痛点而生——只需在局域网内选择一台能够连通外网的机器部署本应用，其他内网设备即可通过该代理服务器共享上网，无需为每台终端单独配置网络出口，从而最大程度地减少硬件采购成本与网络带宽费用。

传统的代理方案往往配置繁琐、缺乏可视化界面，且难以进行精细化的访问控制。GoProxy 采用现代化的技术栈，将高性能的 Go 语言后端与基于 Wails v2 的跨平台桌面应用相结合，配合 Vue 3 前端界面，让用户能够在图形界面中轻松完成代理服务的启动、配置、监控与管理工作。无论是网络管理员还是普通用户，都可以在短时间内完成部署，无需深入理解复杂的网络协议或命令行操作。

在权限管理方面，GoProxy 内置了账号分配与认证机制。管理员可以在服务端创建多个代理账号，并为每个账号配置独立的用户名与密码。只有持有有效凭据的客户端才能通过 SOCKS5 或 HTTP CONNECT 协议接入代理服务。这种设计不仅防止了未授权访问，还便于在多人共享网络的场景中追踪和管理各个终端的上网行为。当员工离职、学生毕业或访客离开时，管理员只需在界面上禁用或删除对应账号，即可立即切断其网络访问权限，避免了传统共享网关难以回收权限的尴尬。

GoProxy 的另一大亮点在于其强大的路由功能。在许多复杂的网络环境中，一台服务器可能同时连接了多条出口链路，例如主宽带、备用宽带、专线或不同的运营商网络。GoProxy 支持基于目标地址的路由规则，管理员可以按目标 IP、CIDR 网段或域名（包括通配符匹配）来分流不同的网络请求，将其绑定到指定的本地网卡或源 IP 地址发出。例如，可以将访问办公系统的流量固定走内网专线，将视频或下载流量分流到备用宽带，从而实现带宽的合理分配与网络质量的优化。这一功能特别适用于拥有多运营商线路或需要区分业务流量的场景，让网络资源的利用更加高效和智能。

除了账号管理和智能路由，GoProxy 还提供了完善的可观测性支持。应用内置了实时连接监控面板，管理员可以查看当前有哪些客户端正在使用代理、访问了哪些目标地址、产生了多少上行与下行流量。环形缓冲区保存的实时日志则帮助用户快速排查连接异常、认证失败或路由规则未命中的问题。所有的运行状态、配置变更和流量统计都在桌面应用中直观呈现，无需额外安装监控软件或登录远程服务器执行命令。

从技术架构上看，GoProxy 后端基于 Go 语言的高并发网络编程能力，支持 SOCKS5 和 HTTP CONNECT 两种主流代理协议，能够兼容绝大多数浏览器、操作系统和应用程序的代理设置。前端采用 Vue 3 + Naive UI 构建，界面简洁美观、交互流畅，数据状态通过 Pinia 统一管理。Wails v2 框架将前后端紧密集成，既保证了桌面应用的原生体验，又允许开发者使用现代 Web 技术栈进行 UI 开发。整个应用打包后体积小巧、资源占用低，能够在普通的办公电脑或老旧服务器上长期稳定运行。

无论你是需要为办公室数十台电脑提供统一的上网出口，还是希望在学校机房中控制学生的网络访问权限，又或者是在家庭环境中让多设备共享一条宽带的同时实现流量分流，GoProxy 都能提供一个轻量、易用且功能完备的解决方案。

---

## Introduction

GoProxy is a desktop proxy server application designed specifically for LAN shared internet access scenarios. In many enterprise, school, lab, or home network environments, a common challenge arises: multiple intranet computers need internet access, but due to security policies, network architecture, or cost constraints, it is impractical to assign a public network egress to every single machine. GoProxy addresses this exact pain point — by deploying the application on just one machine within the local network that can reach the internet, all other internal devices can share that connection through the proxy server. This eliminates the need to configure separate network egress for each terminal, significantly reducing hardware procurement costs and network bandwidth expenses.

Traditional proxy solutions are often cumbersome to configure, lack visual interfaces, and make fine-grained access control difficult. GoProxy leverages a modern technology stack, combining a high-performance Go backend with a cross-platform desktop application built on Wails v2, paired with a Vue 3 frontend. This allows users to easily start, configure, monitor, and manage the proxy service through a graphical interface. Whether you are a network administrator or an ordinary user, deployment can be completed within minutes without needing deep knowledge of complex network protocols or command-line operations.

Regarding access control, GoProxy features a built-in account allocation and authentication system. Administrators can create multiple proxy accounts on the server, each with an independent username and password. Only clients with valid credentials can connect to the proxy service via SOCKS5 or HTTP CONNECT protocols. This design not only prevents unauthorized access but also facilitates tracking and managing internet usage across different terminals in multi-user environments. When an employee leaves, a student graduates, or a guest departs, the administrator can simply disable or delete the corresponding account in the interface to immediately revoke their network access — avoiding the awkwardness of reclaiming permissions in traditional shared gateway setups.

Another major highlight of GoProxy is its powerful routing capabilities. In many complex network environments, a single server may be connected to multiple egress links, such as primary broadband, backup broadband, dedicated lines, or networks from different carriers. GoProxy supports routing rules based on destination addresses, allowing administrators to分流 different network requests according to target IPs, CIDR blocks, or domains (including wildcard matching), binding them to specified local network interfaces or source IP addresses. For example, traffic to office systems can be fixed to route through an internal dedicated line, while video or download traffic can be diverted to a backup broadband, enabling rational bandwidth allocation and network quality optimization. This feature is especially valuable in scenarios with multi-carrier lines or the need to separate business traffic, making network resource utilization more efficient and intelligent.

Beyond account management and smart routing, GoProxy offers comprehensive observability support. The application includes a real-time connection monitoring dashboard where administrators can see which clients are currently using the proxy, what target addresses they are accessing, and how much upstream and downstream traffic they are generating. A ring buffer preserves real-time logs to help users quickly troubleshoot connection anomalies, authentication failures, or routing rules that failed to match. All runtime states, configuration changes, and traffic statistics are presented intuitively within the desktop application, with no need to install additional monitoring software or log into remote servers to execute commands.

From a technical architecture perspective, GoProxy is built on Go's high-concurrency network programming capabilities, supporting both SOCKS5 and HTTP CONNECT — the two mainstream proxy protocols — ensuring compatibility with the vast majority of browsers, operating systems, and applications. The frontend is built with Vue 3 and Naive UI, offering a clean, aesthetically pleasing, and smoothly interactive interface, with data state managed uniformly through Pinia. The Wails v2 framework tightly integrates the frontend and backend, delivering a native desktop experience while allowing developers to use modern web technologies for UI development. The entire application is compact after packaging and consumes minimal resources, enabling stable long-term operation on ordinary office computers or older servers.

Whether you need to provide a unified internet egress for dozens of office computers, control students' network access in a school lab, or enable multiple devices at home to share a single broadband connection while achieving traffic分流, GoProxy offers a lightweight, user-friendly, and fully-featured solution.

---

## 主要特点 / Key Features

1. **账号分配与认证 / Account Allocation & Authentication**  
   支持在服务端创建多个代理账号，客户端需通过用户名和密码认证后方可接入。管理员可随时启用、禁用或删除账号，实现精细化的访问权限控制。

2. **智能路由与网卡绑定 / Smart Routing & Interface Binding**  
   支持按目标 IP、CIDR 或域名（含通配符）设定路由规则，将不同流量分配至指定网卡或本地源地址发出，适用于多线路、多运营商的网络环境。

3. **实时连接监控 / Real-time Connection Monitoring**  
   可视化展示当前活跃连接、客户端地址、目标地址及实时流量，帮助管理员直观掌握代理服务的运行状况。

4. **双协议支持 / Dual Protocol Support**  
   同时支持 SOCKS5 和 HTTP CONNECT 代理协议，兼容主流浏览器、操作系统及各类应用程序的代理配置。

5. **跨平台桌面应用 / Cross-platform Desktop Application**  
   基于 Wails v2 构建，支持 Windows 与 macOS，提供原生桌面体验，无需依赖浏览器即可运行。

6. **实时日志与审计 / Real-time Logging & Auditing**  
   内置 zap 日志与环形缓冲区，实时记录连接、认证、路由决策等关键事件，便于故障排查与安全审计。

7. **可视化配置管理 / Visual Configuration Management**  
   通过 Vue 3 前端界面即可修改监听端口、认证信息、路由规则等配置，修改保存后服务自动重载，无需手动重启。

8. **轻量高效 / Lightweight & Efficient**  
   Go 语言后端具备高并发与低内存占用特性，整机体积小巧，可在普通 PC 或旧服务器上长期稳定运行。

---

## 界面预览 / Screenshots

### 仪表盘 / Dashboard

![仪表盘](images/home.png)

### 活跃连接 / Active Connections

实时查看每一条代理连接的协议、目标地址、命中规则、出口网卡及流量信息。

![活跃连接](images/1.png)

### 实时日志 / Real-time Logs

查看规则命中日志，快速排查连接异常、认证失败或路由未命中问题。

![实时日志](images/2.png)

### 服务配置 / Service Configuration

在图形界面中配置 SOCKS5 与 HTTP CONNECT 的监听地址和端口。

![服务配置](images/3.png)

### 路由规则 / Route Rules

按目标 IP、CIDR 或域名匹配流量，并将其分配至指定网卡或源地址。

![路由规则](images/4.png)

### 新增规则 / Add Rule

支持域名、IP、CIDR 等多种匹配类型，灵活配置出口策略。

![新增规则](images/5.png)

### 认证管理 / Authentication

启用代理认证后，通过用户名和密码控制客户端接入权限。

![认证管理](images/6.png)

### 应用设置 / Application Settings

自定义外观主题、启动行为等偏好设置。

![应用设置](images/7.png)

---

## 环境要求

- Go 1.22 或更高版本
- Node.js 18 或更高版本
- npm
- Wails v2 CLI
- Windows 需要 Edge WebView2 Runtime，Win10/Win11 通常已内置
- macOS 建议 10.13 或更高版本

安装 Wails CLI：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

确认工具可用：

```bash
go version
node -v
npm -v
wails version
```

## 安装依赖

在项目根目录安装 Go 依赖：

```bash
go mod download
```

安装前端依赖：

```bash
cd frontend
npm install
cd ..
```

## 开发启动

启动完整桌面应用：

```bash
wails dev
```

Wails 会根据 `wails.json` 自动启动前端开发服务，并连接 Go 后端绑定方法。

如果只需要预览前端页面样式，可以单独启动 Vite：

```bash
cd frontend
npm run dev
```

默认访问地址：

```text
http://127.0.0.1:18606
```

注意：单独启动前端时没有完整 Wails 桌面运行时，部分后端调用会不可用；功能联调请使用 `wails dev`。

## CLI 代理验证

项目提供了命令行入口，可以不启动桌面 UI，直接验证代理核心：

```bash
go run ./cmd/proxycli
```

指定配置文件：

```bash
go run ./cmd/proxycli -config config.yaml
```

写入默认配置文件：

```bash
go run ./cmd/proxycli -config config.yaml -write-default
```

默认监听：

- SOCKS5：`0.0.0.0:1080`
- HTTP CONNECT：`0.0.0.0:8080`

## 测试与校验

运行 Go 单元测试：

```bash
go test ./...
```

运行 Go 静态检查：

```bash
go vet ./...
```

运行前端类型检查和生产构建：

```bash
cd frontend
npm run build
cd ..
```

发布或提交前建议至少执行：

```bash
go test ./...
go vet ./...
cd frontend && npm run build
```

## 构建桌面应用

构建当前平台产物：

```bash
wails build
```

构建 Windows amd64：

```bash
wails build -platform windows/amd64
```

构建 macOS Intel：

```bash
wails build -platform darwin/amd64
```

构建 macOS Apple Silicon：

```bash
wails build -platform darwin/arm64
```

构建产物默认输出到 Wails 的 `build/bin` 目录，应用名和输出文件名由 `wails.json` 中的 `name` 与 `outputfilename` 控制。

## 项目结构

```text
.
├── app.go                  # Wails 绑定层
├── main.go                 # 桌面应用入口
├── cmd/proxycli            # CLI 验证入口
├── internal/config         # YAML 配置、默认值和校验
├── internal/proxy          # SOCKS5、HTTP CONNECT、Relay 和服务生命周期
├── internal/stats          # 连接数与流量统计
├── internal/logger         # zap 日志和环形缓冲
├── internal/platform       # 平台路径等差异能力
├── frontend                # Vue 3 前端
├── docs                    # 设计文档和阶段方案
└── build                   # 图标和平台构建资源
```

## 配置文件

桌面应用默认配置路径：

- Windows / macOS：`<应用目录>/configs/config.yaml`

CLI 默认读取当前目录下的 `config.yaml`，也可以通过 `-config` 指定路径。

## 常见问题

### wails 命令不存在

请先安装 Wails CLI，并确认 `$GOPATH/bin` 或 `$HOME/go/bin` 已加入 `PATH`：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 前端提示 vue-tsc 不存在

说明前端依赖未安装：

```bash
cd frontend
npm install
```

### 端口被占用

修改配置里的 SOCKS5 或 HTTP 监听端口，或者停止占用端口的进程。监听配置在服务运行中保存后，需要重启代理服务生效。
