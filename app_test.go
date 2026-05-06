package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppConfigLifecycle(t *testing.T) {
	app := newTestApp(t)

	cfg := app.GetConfig()
	cfg.Server.SOCKS5.Host = "127.0.0.1"
	cfg.Server.SOCKS5.Port = freePort(t)
	cfg.Server.HTTP.Enabled = false
	cfg.Relay.MaxConnections = 16

	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded := app.GetConfig()
	if loaded.Server.SOCKS5.Port != cfg.Server.SOCKS5.Port {
		t.Fatalf("expected saved socks5 port %d, got %d", cfg.Server.SOCKS5.Port, loaded.Server.SOCKS5.Port)
	}

	invalid := loaded
	invalid.Relay.MaxConnections = 0
	if err := app.SaveConfig(invalid); err == nil {
		t.Fatal("expected invalid config error")
	}
	if app.GetConfig().Relay.MaxConnections != 16 {
		t.Fatal("invalid config should not replace current config")
	}
}

func TestAppStartStopAndLogs(t *testing.T) {
	app := newTestApp(t)

	cfg := app.GetConfig()
	cfg.Server.SOCKS5.Enabled = false
	cfg.Server.HTTP.Enabled = true
	cfg.Server.HTTP.Host = "127.0.0.1"
	cfg.Server.HTTP.Port = freePort(t)
	cfg.Relay.MaxConnections = 4
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	if err := app.StartServer(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	if !app.GetServerStatus().Running {
		t.Fatal("expected server to be running")
	}
	if err := app.StopServer(); err != nil {
		t.Fatalf("stop server: %v", err)
	}
	if app.GetServerStatus().Running {
		t.Fatal("expected server to be stopped")
	}

	logs := app.GetRecentLogs(10)
	if len(logs) == 0 {
		t.Fatal("expected runtime logs")
	}
	var foundStart bool
	for _, entry := range logs {
		if strings.Contains(entry.Message, "代理服务已启动") {
			foundStart = true
			break
		}
	}
	if !foundStart {
		t.Fatalf("expected start log, got %#v", logs)
	}
}

func TestAppClearLogsDeletesFileAndRing(t *testing.T) {
	app := newTestApp(t)

	app.logger.Info(appSource, "待清空日志")
	if _, err := os.Stat(app.logPath); err != nil {
		t.Fatalf("expected log file before clear: %v", err)
	}
	if len(app.GetRecentLogs(10)) == 0 {
		t.Fatal("expected in-memory logs before clear")
	}

	if err := app.ClearLogs(); err != nil {
		t.Fatalf("clear logs: %v", err)
	}
	if len(app.GetRecentLogs(10)) != 0 {
		t.Fatalf("expected in-memory logs to be cleared, got %#v", app.GetRecentLogs(10))
	}
	if _, err := os.Stat(app.logPath); !os.IsNotExist(err) {
		t.Fatalf("expected log file to be deleted, stat error: %v", err)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()

	dir := t.TempDir()
	app, err := NewAppWithPaths(filepath.Join(dir, "config.yaml"), filepath.Join(dir, "proxy.log"))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() {
		app.shutdown(context.Background())
	})
	return app
}

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free port: %v", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}
