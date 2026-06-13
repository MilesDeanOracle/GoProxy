# ============================================================
#  GoProxy Makefile — 一键管理开发、构建与部署
#  用法: make <目标>
# ============================================================

# ---------- 可配置变量 ----------
MODULE       := gitee.com/jiuhuidalan1/goproxy
GO           := go
NPM          := npm
DOCKER       := docker
GOFLAGS      :=
LDFLAGS      := -s -w
FRONTEND_DIR := frontend
BUILD_DIR    := build
CONFIG_FILE  := config.yaml
LISTEN_ADDR  ?= 0.0.0.0:9090
DOCKER_IMAGE ?= goproxy:latest
DOCKER_PLATFORM ?=
GO_BIN_DIR   ?= $(shell GOBIN=`$(GO) env GOBIN`; if [ -n "$$GOBIN" ]; then printf "%s" "$$GOBIN"; else printf "%s/bin" "`$(GO) env GOPATH`"; fi)
WAILS        ?= $(shell command -v wails 2>/dev/null || printf "%s/wails" "$(GO_BIN_DIR)")
WAILS_VERSION ?= $(shell awk '/github.com\/wailsapp\/wails\/v2/ {print $$2; exit}' go.mod)

# 颜色输出（终端支持时生效）
BLUE   := \033[0;34m
GREEN  := \033[0;32m
YELLOW := \033[0;33m
CYAN   := \033[0;36m
BOLD   := \033[1m
RESET  := \033[0m

# ---------- 默认目标 ----------
.DEFAULT_GOAL := help

# ============================================================
#  帮助信息
# ============================================================
.PHONY: help
help: ## 📋 显示此帮助信息
	@echo ""
	@echo "$(BOLD)$(BLUE)╔══════════════════════════════════════════════════╗$(RESET)"
	@echo "$(BOLD)$(BLUE)║          GoProxy Makefile 命令帮助手册           ║$(RESET)"
	@echo "$(BOLD)$(BLUE)╚══════════════════════════════════════════════════╝$(RESET)"
	@echo ""
	@echo "$(CYAN)$(BOLD)  开发环境$(RESET)"
	@echo "  $(GREEN)make dev$(RESET)             — 启动完整开发环境（前端 + Web服务）"
	@echo "  $(GREEN)make dev-frontend$(RESET)     — 仅启动前端开发服务器（Vite 热更新）"
	@echo "  $(GREEN)make dev-webserver$(RESET)     — 仅启动后端 Web 服务"
	@echo "  $(GREEN)make dev-wails$(RESET)         — 启动 Wails 桌面应用开发模式（需安装 Wails CLI）"
	@echo ""
	@echo "$(CYAN)$(BOLD)  依赖安装$(RESET)"
	@echo "  $(GREEN)make install$(RESET)           — 安装前端 + 后端 + Wails CLI 全部依赖"
	@echo "  $(GREEN)make install-all$(RESET)       — 安装前端 + 后端 + Wails CLI 全部依赖"
	@echo "  $(GREEN)make install-frontend$(RESET)   — 仅安装前端 npm 依赖"
	@echo "  $(GREEN)make install-backend$(RESET)    — 仅安装后端 Go 依赖"
	@echo "  $(GREEN)make install-wails-cli$(RESET)  — 仅安装 Wails CLI"
	@echo ""
	@echo "$(CYAN)$(BOLD)  构建编译$(RESET)"
	@echo "  $(GREEN)make build$(RESET)             — 构建前端 + 编译 Web 服务端二进制"
	@echo "  $(GREEN)make build-frontend$(RESET)     — 仅构建前端静态资源"
	@echo "  $(GREEN)make build-webserver$(RESET)    — 仅编译 Web 服务端"
	@echo "  $(GREEN)make build-cli$(RESET)          — 仅编译命令行代理服务"
	@echo "  $(GREEN)make build-wails$(RESET)        — 构建 Wails 桌面应用"
	@echo "  $(GREEN)make docker-build$(RESET)       — 构建 Docker 镜像（Web 管理端 + 代理端口）"
	@echo ""
	@echo "$(CYAN)$(BOLD)  Linux 交叉编译$(RESET)"
	@echo "  $(GREEN)make build-linux$(RESET)        — 交叉编译 Linux amd64 并打包部署目录"
	@echo ""
	@echo "$(CYAN)$(BOLD)  测试 & 检查$(RESET)"
	@echo "  $(GREEN)make test$(RESET)              — 运行全部 Go 测试"
	@echo "  $(GREEN)make lint$(RESET)              — 运行代码静态检查"
	@echo "  $(GREEN)make vet$(RESET)               — 运行 go vet 检查"
	@echo ""
	@echo "$(CYAN)$(BOLD)  配置 & 清理$(RESET)"
	@echo "  $(GREEN)make init-config$(RESET)        — 生成默认配置文件 $(CONFIG_FILE)"
	@echo "  $(GREEN)make clean$(RESET)             — 清理构建产物和缓存"
	@echo ""
	@echo "$(YELLOW)  提示: LISTEN_ADDR=0.0.0.0:9090 make dev-webserver$(RESET)"
	@echo "$(YELLOW)       可通过变量覆盖默认监听地址$(RESET)"
	@echo ""

# ============================================================
#  依赖安装
# ============================================================
.PHONY: install install-all install-frontend install-backend install-tools install-wails-cli

install: install-frontend install-backend install-tools ## 安装全部依赖

install-all: install ## 安装全部依赖

install-frontend: ## 安装前端 npm 依赖
	@echo "$(BOLD)$(BLUE)==> 安装前端依赖...$(RESET)"
	cd $(FRONTEND_DIR) && if [ -f package-lock.json ]; then $(NPM) ci --prefer-offline; else $(NPM) install --prefer-offline; fi
	@echo "$(BOLD)$(GREEN)==> 前端依赖安装完成$(RESET)"

install-backend: ## 整理后端 Go 依赖
	@echo "$(BOLD)$(BLUE)==> 整理后端依赖...$(RESET)"
	$(GO) mod download
	@echo "$(BOLD)$(GREEN)==> 后端依赖就绪$(RESET)"

install-tools: install-wails-cli ## 安装开发工具依赖

install-wails-cli: ## 安装 Wails CLI
	@echo "$(BOLD)$(BLUE)==> 检查 Wails CLI...$(RESET)"
	@if [ -x "$(WAILS)" ]; then \
		echo "$(BOLD)$(GREEN)==> Wails CLI 已就绪: $(WAILS)$(RESET)"; \
	else \
		echo "$(BOLD)$(BLUE)==> 安装 Wails CLI $(WAILS_VERSION)...$(RESET)"; \
		$(GO) install github.com/wailsapp/wails/v2/cmd/wails@$(WAILS_VERSION); \
		echo "$(BOLD)$(GREEN)==> Wails CLI 安装完成: $(GO_BIN_DIR)/wails$(RESET)"; \
	fi

# ============================================================
#  开发环境
# ============================================================
.PHONY: dev dev-frontend dev-webserver dev-wails

dev: ## 启动完整开发环境（前台运行 Web 服务）
	@echo "$(BOLD)$(CYAN)==> 启动 GoProxy 开发环境$(RESET)"
	@echo "$(BOLD)$(BLUE)==> 构建前端...$(RESET)"
	@cd $(FRONTEND_DIR) && npm run build
	@echo "$(BOLD)$(GREEN)==> 前端构建完成，启动 Web 服务...$(RESET)"
	@$(GO) run ./cmd/webserver -listen $(LISTEN_ADDR) -static $(CURDIR)/$(FRONTEND_DIR)/dist

dev-frontend: ## 仅启动前端开发服务器（Vite 热更新，端口 18606）
	@echo "$(BOLD)$(CYAN)==> 启动前端开发服务器...$(RESET)"
	cd $(FRONTEND_DIR) && npm run dev

dev-webserver: build-frontend ## 仅启动后端 Web 服务
	@echo "$(BOLD)$(CYAN)==> 启动 Web 服务，监听 $(LISTEN_ADDR)...$(RESET)"
	$(GO) run ./cmd/webserver -listen $(LISTEN_ADDR) -static $(CURDIR)/$(FRONTEND_DIR)/dist

dev-wails: install-wails-cli ## 启动 Wails 桌面应用开发模式
	@echo "$(BOLD)$(CYAN)==> 启动 Wails 开发模式...$(RESET)"
	$(WAILS) dev

# ============================================================
#  构建编译
# ============================================================
.PHONY: build build-frontend build-webserver build-cli build-wails build-linux docker-build

build: build-frontend build-webserver ## 构建前端 + Web 服务端

build-frontend: ## 构建前端静态资源
	@echo "$(BOLD)$(BLUE)==> 构建前端...$(RESET)"
	cd $(FRONTEND_DIR) && npm run build
	@echo "$(BOLD)$(GREEN)==> 前端构建完成: $(FRONTEND_DIR)/dist/$(RESET)"

build-webserver: build-frontend ## 编译 Web 服务端二进制
	@echo "$(BOLD)$(BLUE)==> 编译 webserver...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/goproxy-webserver ./cmd/webserver/
	@echo "$(BOLD)$(GREEN)==> 编译完成: $(BUILD_DIR)/goproxy-webserver$(RESET)"

build-cli: ## 编译命令行代理服务
	@echo "$(BOLD)$(BLUE)==> 编译 proxycli...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/goproxy-cli ./cmd/proxycli/
	@echo "$(BOLD)$(GREEN)==> 编译完成: $(BUILD_DIR)/goproxy-cli$(RESET)"

build-wails: install-wails-cli build-frontend ## 构建 Wails 桌面应用
	@echo "$(BOLD)$(BLUE)==> 构建 Wails 桌面应用...$(RESET)"
	$(WAILS) build
	@echo "$(BOLD)$(GREEN)==> Wails 构建完成$(RESET)"

build-linux: build-frontend ## 交叉编译 Linux amd64 并打包
	@echo "$(BOLD)$(BLUE)==> 交叉编译 Linux amd64 二进制...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/goproxy-webserver ./cmd/webserver/
	@echo "$(BOLD)$(BLUE)==> 打包部署目录...$(RESET)"
	@rm -rf $(BUILD_DIR)/goproxy-linux-amd64
	@mkdir -p $(BUILD_DIR)/goproxy-linux-amd64/frontend/dist
	@cp $(BUILD_DIR)/goproxy-webserver $(BUILD_DIR)/goproxy-linux-amd64/
	@cp -r $(FRONTEND_DIR)/dist/* $(BUILD_DIR)/goproxy-linux-amd64/frontend/dist/
	@echo "$(BOLD)$(GREEN)==> 打包完成: $(BUILD_DIR)/goproxy-linux-amd64/$(RESET)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)  部署步骤:$(RESET)"
	@echo "  1. 上传 $(BUILD_DIR)/goproxy-linux-amd64/ 到目标服务器"
	@echo "  2. 生成配置: ./goproxy-webserver -write-default"
	@echo "  3. 启动服务: ./goproxy-webserver"
	@echo "  4. 浏览器访问: http://<服务器IP>:9090"
	@echo "  5. 默认账号: admin / admin"

docker-build: ## 构建 Docker 镜像（Web 管理端 + SOCKS5 + HTTP 代理端口）
	@echo "$(BOLD)$(BLUE)==> 构建 Docker 镜像 $(DOCKER_IMAGE)...$(RESET)"
	$(DOCKER) build $(if $(DOCKER_PLATFORM),--platform $(DOCKER_PLATFORM),) -t $(DOCKER_IMAGE) .
	@echo "$(BOLD)$(GREEN)==> Docker 镜像构建完成: $(DOCKER_IMAGE)$(RESET)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)  运行示例:$(RESET)"
	@echo "  docker run -d --name goproxy \\"
	@echo "    -p 9090:9090 \\"
	@echo "    -p 1080:1080 \\"
	@echo "    -p 8080:8080 \\"
	@echo "    -v goproxy-data:/app/configs \\"
	@echo "    $(DOCKER_IMAGE)"

# ============================================================
#  测试 & 检查
# ============================================================
.PHONY: test lint vet

test: ## 运行全部 Go 测试
	@echo "$(BOLD)$(BLUE)==> 运行测试...$(RESET)"
	$(GO) test ./... -v -count=1
	@echo "$(BOLD)$(GREEN)==> 测试完成$(RESET)"

lint: ## 运行 golangci-lint 静态检查
	@echo "$(BOLD)$(BLUE)==> 运行静态检查...$(RESET)"
	golangci-lint run ./...
	@echo "$(BOLD)$(GREEN)==> 静态检查通过$(RESET)"

vet: ## 运行 go vet
	@echo "$(BOLD)$(BLUE)==> 运行 go vet...$(RESET)"
	$(GO) vet ./...
	@echo "$(BOLD)$(GREEN)==> go vet 通过$(RESET)"

# ============================================================
#  配置 & 清理
# ============================================================
.PHONY: init-config clean

init-config: ## 生成默认配置文件
	@echo "$(BOLD)$(BLUE)==> 生成默认配置: $(CONFIG_FILE)$(RESET)"
	$(GO) run ./cmd/webserver -write-default -config $(CONFIG_FILE)
	@echo "$(BOLD)$(GREEN)==> 配置文件已生成$(RESET)"

clean: ## 清理构建产物和缓存
	@echo "$(BOLD)$(YELLOW)==> 清理构建产物...$(RESET)"
	rm -rf $(BUILD_DIR)/goproxy-webserver
	rm -rf $(BUILD_DIR)/goproxy-cli
	rm -rf $(BUILD_DIR)/goproxy-linux-amd64
	rm -rf $(FRONTEND_DIR)/dist
	@echo "$(BOLD)$(BLUE)==> 清理 Go 缓存...$(RESET)"
	$(GO) clean -cache -testcache
	@echo "$(BOLD)$(GREEN)==> 清理完成$(RESET)"
