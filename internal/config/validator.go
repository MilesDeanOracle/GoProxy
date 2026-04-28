package config

import (
	"errors"
	"fmt"
	"net"
)

// Validate checks the complete runtime configuration.
func Validate(cfg Config) error {
	if !cfg.Server.SOCKS5.Enabled && !cfg.Server.HTTP.Enabled {
		return errors.New("at least one inbound protocol must be enabled")
	}

	if cfg.Server.SOCKS5.Enabled {
		if err := validateProtocol("socks5", cfg.Server.SOCKS5); err != nil {
			return err
		}
	}
	if cfg.Server.HTTP.Enabled {
		if err := validateProtocol("http", cfg.Server.HTTP); err != nil {
			return err
		}
	}

	if cfg.Relay.DialTimeoutSec <= 0 {
		return errors.New("relay.dial_timeout_sec must be greater than 0")
	}
	if cfg.Relay.ReadTimeoutSec <= 0 {
		return errors.New("relay.read_timeout_sec must be greater than 0")
	}
	if cfg.Relay.MaxConnections <= 0 {
		return errors.New("relay.max_connections must be greater than 0")
	}
	if cfg.Relay.KeepAliveSec <= 0 {
		return errors.New("relay.keepalive_sec must be greater than 0")
	}
	if err := validateLog(cfg.Log); err != nil {
		return err
	}
	if err := validateUI(cfg.UI); err != nil {
		return err
	}

	if cfg.Server.SOCKS5.Enabled && cfg.Server.HTTP.Enabled {
		if cfg.Server.SOCKS5.Host == cfg.Server.HTTP.Host && cfg.Server.SOCKS5.Port == cfg.Server.HTTP.Port {
			return errors.New("socks5 and http listeners cannot use the same address")
		}
	}

	return nil
}

func validateProtocol(name string, protocol ProtocolConfig) error {
	if protocol.Host == "" {
		return fmt.Errorf("server.%s.host is required", name)
	}
	if ip := net.ParseIP(protocol.Host); ip == nil {
		return fmt.Errorf("server.%s.host must be an IP address", name)
	}
	if protocol.Port < 1 || protocol.Port > 65535 {
		return fmt.Errorf("server.%s.port must be between 1 and 65535", name)
	}
	return nil
}

func validateLog(log LogConfig) error {
	switch log.Level {
	case "debug", "info", "warn", "error":
	default:
		return errors.New("log.level must be one of debug, info, warn, error")
	}
	if log.MaxSizeMB <= 0 {
		return errors.New("log.max_size_mb must be greater than 0")
	}
	if log.MaxBackups < 0 {
		return errors.New("log.max_backups cannot be negative")
	}
	switch log.Output {
	case "file", "console", "both":
	default:
		return errors.New("log.output must be one of file, console, both")
	}
	return nil
}

func validateUI(ui UIConfig) error {
	switch ui.Theme {
	case "light", "dark", "auto":
	default:
		return errors.New("ui.theme must be one of light, dark, auto")
	}
	if ui.Language == "" {
		return errors.New("ui.language is required")
	}
	return nil
}
