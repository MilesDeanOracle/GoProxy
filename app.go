package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"gitee.com/jiuhuidalan1/goproxy/internal/logger"
	"gitee.com/jiuhuidalan1/goproxy/internal/platform"
	"gitee.com/jiuhuidalan1/goproxy/internal/proxy"
	"gitee.com/jiuhuidalan1/goproxy/internal/stats"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
)

const appSource = "app"

// App is the Wails binding layer between the desktop UI and backend services.
type App struct {
	mu sync.Mutex

	ctx context.Context

	configPath    string
	logPath       string
	configManager *config.Manager
	logger        *logger.Manager

	cfg        config.Config
	runtimeCfg config.Config
	collector  *stats.Collector
	server     *proxy.Server
}

// NewApp creates the desktop application using platform-specific paths.
func NewApp() (*App, error) {
	configPath, err := platform.ConfigPath()
	if err != nil {
		return nil, err
	}
	logPath, err := platform.LogPath()
	if err != nil {
		return nil, err
	}
	return NewAppWithPaths(configPath, logPath)
}

// NewAppWithPaths creates the application with explicit paths, primarily for tests.
func NewAppWithPaths(configPath, logPath string) (*App, error) {
	manager := config.NewManager(configPath)
	cfg, err := manager.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("load config: %w", err)
		}
		cfg = config.Default()
		if err := manager.Save(cfg); err != nil {
			return nil, fmt.Errorf("write default config: %w", err)
		}
	}

	logManager, err := logger.NewManager(cfg.Log, logPath)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	collector := stats.NewCollector()
	return &App{
		configPath:    configPath,
		logPath:       logPath,
		configManager: manager,
		logger:        logManager,
		cfg:           cfg,
		runtimeCfg:    cfg,
		collector:     collector,
		server:        proxy.NewServer(cfg, collector),
	}, nil
}

func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	if a.logger != nil {
		a.subscribeLoggerLocked(a.logger)
		a.logger.Info(appSource, "应用已启动", zap.String("configPath", a.configPath))
	}
	a.emitStatusLocked()
	a.mu.Unlock()
}

func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	server := a.server
	logManager := a.logger
	a.mu.Unlock()

	if server != nil {
		_ = server.Stop()
	}
	if logManager != nil {
		logManager.Info(appSource, "应用正在退出")
		_ = logManager.Close()
	}
}

// GetConfig returns the current complete YAML-backed configuration.
func (a *App) GetConfig() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

// SaveConfig validates and persists configuration changes.
func (a *App) SaveConfig(cfg config.Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	oldCfg := a.cfg
	running := a.server != nil && a.server.Status().Running

	var newLogger *logger.Manager
	if oldCfg.Log != cfg.Log {
		var err error
		newLogger, err = logger.NewManager(cfg.Log, a.logPath)
		if err != nil {
			if a.logger != nil {
				a.logger.Warn(appSource, "日志配置无效", zap.Error(err))
			}
			return err
		}
	}

	if err := a.configManager.Save(cfg); err != nil {
		if a.logger != nil {
			a.logger.Warn(appSource, "配置保存失败", zap.Error(err))
		}
		return err
	}

	a.cfg = cfg
	if newLogger != nil {
		oldLogger := a.logger
		a.logger = newLogger
		a.subscribeLoggerLocked(newLogger)
		if oldLogger != nil {
			_ = oldLogger.Close()
		}
	}
	if !running {
		a.collector = stats.NewCollector()
		a.server = proxy.NewServer(cfg, a.collector)
		a.runtimeCfg = cfg
	}

	if a.logger != nil {
		a.logger.Info(appSource, "配置已保存")
		if running && listenerConfigChanged(oldCfg, cfg) {
			a.logger.Warn(appSource, "监听配置已保存，重启服务后生效")
		}
	}

	a.emitStatusLocked()
	return nil
}

// StartServer starts the proxy server using the current configuration.
func (a *App) StartServer() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil || !sameRuntimeConfig(a.cfg, a.runtimeCfg) {
		a.collector = stats.NewCollector()
		a.server = proxy.NewServer(a.cfg, a.collector)
		a.runtimeCfg = a.cfg
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	if err := a.server.Start(ctx); err != nil {
		if a.logger != nil {
			a.logger.Error(appSource, "代理服务启动失败", zap.Error(err))
		}
		return err
	}

	if a.logger != nil {
		status := a.server.Status()
		a.logger.Info(appSource, "代理服务已启动",
			zap.String("socks5Addr", status.SOCKS5Addr),
			zap.String("httpAddr", status.HTTPAddr),
		)
	}

	a.emitStatusLocked()
	return nil
}

// StopServer stops all listeners and active proxy connections.
func (a *App) StopServer() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil {
		return nil
	}
	if err := a.server.Stop(); err != nil {
		if a.logger != nil {
			a.logger.Error(appSource, "代理服务停止失败", zap.Error(err))
		}
		return err
	}

	if a.logger != nil {
		a.logger.Info(appSource, "代理服务已停止")
	}
	a.emitStatusLocked()
	return nil
}

// GetServerStatus returns the current server status.
func (a *App) GetServerStatus() proxy.Status {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil {
		return proxy.Status{}
	}
	return a.server.Status()
}

// GetStats returns a snapshot of current proxy counters.
func (a *App) GetStats() stats.Stats {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil {
		return stats.Stats{}
	}
	return a.server.Stats()
}

// GetActiveConnections returns current active proxy connection details.
func (a *App) GetActiveConnections() []proxy.ConnectionSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil {
		return nil
	}
	return a.server.ActiveConnections()
}

// GetRecentLogs returns the newest n log entries from the in-memory ring buffer.
func (a *App) GetRecentLogs(n int) []logger.Entry {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.logger == nil {
		return nil
	}
	return a.logger.Recent(n)
}

func (a *App) emitStatusLocked() {
	if a.ctx == nil || a.server == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "proxy:status", a.server.Status())
}

func (a *App) subscribeLoggerLocked(logManager *logger.Manager) {
	if a.ctx == nil || logManager == nil {
		return
	}
	emitCtx := a.ctx
	logManager.Subscribe(func(entry logger.Entry) {
		runtime.EventsEmit(emitCtx, "proxy:log", entry)
	})
}

func listenerConfigChanged(a, b config.Config) bool {
	return a.Server.SOCKS5 != b.Server.SOCKS5 || a.Server.HTTP != b.Server.HTTP
}

func sameRuntimeConfig(a, b config.Config) bool {
	return a.Server == b.Server && a.Relay == b.Relay
}
