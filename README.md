# GoProxy

GoProxy 是一个基于 Go + Wails v2 的桌面代理服务端应用，支持 SOCKS5 和 HTTP CONNECT 代理、YAML 配置、实时日志与运行状态展示。前端使用 Vue 3、Naive UI 和 Pinia。

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
