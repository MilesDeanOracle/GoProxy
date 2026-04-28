package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigIsValid(t *testing.T) {
	if err := Validate(Default()); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestManagerSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	manager := NewManager(path)

	cfg := Default()
	cfg.Server.SOCKS5.Host = "127.0.0.1"
	cfg.Server.SOCKS5.Port = 1081
	cfg.Server.HTTP.Enabled = false
	cfg.Relay.MaxConnections = 32

	if err := manager.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.Server.SOCKS5.Port != 1081 {
		t.Fatalf("expected socks5 port 1081, got %d", loaded.Server.SOCKS5.Port)
	}
	if loaded.Server.HTTP.Enabled {
		t.Fatal("expected http listener to be disabled")
	}
	if loaded.Relay.MaxConnections != 32 {
		t.Fatalf("expected max connections 32, got %d", loaded.Relay.MaxConnections)
	}

	cfg.Relay.MaxConnections = 64
	if err := manager.Save(cfg); err != nil {
		t.Fatalf("save updated config: %v", err)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
}

func TestManagerLoadAppliesDefaultsForMissingFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte(`
server:
  socks5:
    host: 127.0.0.1
    port: 1081
  http:
    enabled: false
relay:
  max_connections: 10
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loaded, err := NewManager(path).Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !loaded.Server.SOCKS5.Enabled {
		t.Fatal("expected omitted socks5.enabled to keep default true")
	}
	if loaded.Relay.DialTimeoutSec != Default().Relay.DialTimeoutSec {
		t.Fatalf("expected default dial timeout, got %d", loaded.Relay.DialTimeoutSec)
	}
}

func TestManagerLoadMissingFile(t *testing.T) {
	_, err := NewManager(filepath.Join(t.TempDir(), "missing.yaml")).Load()
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestValidateRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Config)
		want string
	}{
		{
			name: "no listeners",
			edit: func(cfg *Config) {
				cfg.Server.SOCKS5.Enabled = false
				cfg.Server.HTTP.Enabled = false
			},
			want: "at least one inbound protocol",
		},
		{
			name: "invalid port",
			edit: func(cfg *Config) {
				cfg.Server.SOCKS5.Port = 0
			},
			want: "port must be between",
		},
		{
			name: "invalid host",
			edit: func(cfg *Config) {
				cfg.Server.SOCKS5.Host = "localhost"
			},
			want: "host must be an IP address",
		},
		{
			name: "invalid max connections",
			edit: func(cfg *Config) {
				cfg.Relay.MaxConnections = 0
			},
			want: "max_connections",
		},
		{
			name: "duplicate listeners",
			edit: func(cfg *Config) {
				cfg.Server.SOCKS5.Host = "127.0.0.1"
				cfg.Server.HTTP.Host = "127.0.0.1"
				cfg.Server.SOCKS5.Port = 9000
				cfg.Server.HTTP.Port = 9000
			},
			want: "same address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.edit(&cfg)

			err := Validate(cfg)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}
