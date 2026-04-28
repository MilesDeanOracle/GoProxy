package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

const appDirName = "ProxyServer"

// AppDataDir returns the platform-specific application data directory.
func AppDataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(base, appDirName), nil
}

// ConfigPath returns the platform-specific YAML config file path.
func ConfigPath() (string, error) {
	dir, err := AppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// LogPath returns the platform-specific runtime log file path.
func LogPath() (string, error) {
	dir, err := AppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "proxy-server.log"), nil
}
