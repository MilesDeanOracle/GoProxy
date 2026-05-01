package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

var executablePath = os.Executable

// AppBaseDir returns the directory that contains the current executable.
func AppBaseDir() (string, error) {
	executable, err := executablePath()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}

	base := filepath.Dir(executable)
	if base == "" || base == "." {
		return "", fmt.Errorf("resolve executable directory: empty path")
	}

	return base, nil
}

// ConfigPath returns the platform-specific YAML config file path.
func ConfigPath() (string, error) {
	base, err := AppBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "configs", "config.yaml"), nil
}

// LogPath returns the platform-specific runtime log file path.
func LogPath() (string, error) {
	base, err := AppBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "logs", "proxy-server.log"), nil
}
