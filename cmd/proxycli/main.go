package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"gitee.com/jiuhuidalan1/goproxy/internal/proxy"
	"gitee.com/jiuhuidalan1/goproxy/internal/stats"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config file")
	writeDefault := flag.Bool("write-default", false, "write the default config and exit")
	flag.Parse()

	manager := config.NewManager(*configPath)

	if *writeDefault {
		if err := manager.Save(config.Default()); err != nil {
			log.Fatalf("write default config: %v", err)
		}
		log.Printf("default config written to %s", *configPath)
		return
	}

	cfg, err := manager.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("load config: %v", err)
		}
		cfg = config.Default()
		log.Printf("config file %s not found, using defaults", *configPath)
	}

	collector := stats.NewCollector()
	server := proxy.NewServer(cfg, collector)
	routeManager := config.NewRouteFileManager(filepath.Dir(*configPath))
	activeFile, err := routeManager.EnsureActive(cfg.Route.ActiveFile)
	if err != nil {
		log.Fatalf("initialize route files: %v", err)
	}
	if cfg.Route.Enabled {
		set, err := routeManager.Load(activeFile)
		if err != nil {
			log.Fatalf("load route file: %v", err)
		}
		server.SetRoutePolicy(true, set)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := server.Start(ctx); err != nil {
		log.Fatalf("start proxy server: %v", err)
	}

	status := server.Status()
	log.Printf("proxy server started, socks5=%s, http=%s", status.SOCKS5Addr, status.HTTPAddr)

	<-ctx.Done()
	log.Print("stopping proxy server")

	if err := server.Stop(); err != nil {
		log.Fatalf("stop proxy server: %v", err)
	}
}
