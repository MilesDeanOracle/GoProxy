package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manager loads and saves YAML configuration files.
type Manager struct {
	path string
}

// NewManager creates a configuration manager for a YAML file path.
func NewManager(path string) *Manager {
	return &Manager{path: path}
}

// Path returns the managed configuration file path.
func (m *Manager) Path() string {
	return m.path
}

// Load reads the YAML config file, applies defaults for missing fields, and validates it.
func (m *Manager) Load() (Config, error) {
	if m.path == "" {
		return Config{}, errors.New("config path is required")
	}

	data, err := os.ReadFile(m.path)
	if err != nil {
		return Config{}, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config yaml: %w", err)
	}

	if err := Validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Save validates and writes the YAML config file.
func (m *Manager) Save(cfg Config) error {
	if m.path == "" {
		return errors.New("config path is required")
	}
	if err := Validate(cfg); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config yaml: %w", err)
	}

	dir := filepath.Dir(m.path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create config directory: %w", err)
		}
	}

	return m.writeWithBackup(data)
}

func (m *Manager) writeWithBackup(data []byte) error {
	dir := filepath.Dir(m.path)
	if dir == "" {
		dir = "."
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(m.path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp config file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp config file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("set temp config permissions: %w", err)
	}

	existing, err := os.ReadFile(m.path)
	hadExisting := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read current config for backup: %w", err)
	}
	if hadExisting {
		if err := os.WriteFile(m.backupPath(), existing, 0o600); err != nil {
			return fmt.Errorf("write config backup: %w", err)
		}
	}

	if err := os.Rename(tmpPath, m.path); err != nil {
		if hadExisting {
			if removeErr := os.Remove(m.path); removeErr == nil {
				err = os.Rename(tmpPath, m.path)
			}
		}
		if err != nil {
			if hadExisting {
				_ = os.WriteFile(m.path, existing, 0o600)
			}
			return fmt.Errorf("replace config file: %w", err)
		}
	}

	cleanupTmp = false
	return nil
}

func (m *Manager) backupPath() string {
	return m.path + ".bak"
}
