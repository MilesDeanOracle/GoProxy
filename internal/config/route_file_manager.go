package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultRouteFileName = "default.rule"

// RouteFileManager reads and writes route policy files under the configs directory.
type RouteFileManager struct {
	dir string
}

// NewRouteFileManager creates a manager rooted at a configs directory.
func NewRouteFileManager(dir string) *RouteFileManager {
	return &RouteFileManager{dir: dir}
}

// Dir returns the route file directory.
func (m *RouteFileManager) Dir() string {
	return m.dir
}

// EnsureDefault creates default.rule when it is missing.
func (m *RouteFileManager) EnsureDefault() error {
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return fmt.Errorf("create route directory: %w", err)
	}
	path, err := m.filePath(DefaultRouteFileName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat default route file: %w", err)
	}
	return m.Save(DefaultRouteFileName, DefaultRouteRuleSet())
}

// EnsureActive returns an existing active file, falling back to default.rule.
func (m *RouteFileManager) EnsureActive(active string) (string, error) {
	if err := m.EnsureDefault(); err != nil {
		return "", err
	}
	if err := ValidateRouteFileName(active); err != nil {
		active = DefaultRouteFileName
	}
	path, err := m.filePath(active)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return active, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat route file: %w", err)
	}
	return DefaultRouteFileName, nil
}

// List returns all .rule files in stable name order.
func (m *RouteFileManager) List(active string) ([]RouteFileInfo, error) {
	if err := m.EnsureDefault(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, fmt.Errorf("read route directory: %w", err)
	}

	files := make([]RouteFileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".rule") {
			continue
		}
		if err := ValidateRouteFileName(entry.Name()); err != nil {
			continue
		}
		info := RouteFileInfo{Name: entry.Name(), IsActive: entry.Name() == active}
		if stat, err := entry.Info(); err == nil {
			info.UpdatedAt = stat.ModTime().Format(time.RFC3339)
		}
		files = append(files, info)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
	return files, nil
}

// Load reads and validates a route policy file.
func (m *RouteFileManager) Load(name string) (RouteRuleSet, error) {
	path, err := m.filePath(name)
	if err != nil {
		return RouteRuleSet{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return RouteRuleSet{}, fmt.Errorf("read route file: %w", err)
	}
	var set RouteRuleSet
	if err := yaml.Unmarshal(data, &set); err != nil {
		return RouteRuleSet{}, fmt.Errorf("parse route file yaml: %w", err)
	}
	if err := ValidateRouteRuleSet(set); err != nil {
		return RouteRuleSet{}, err
	}
	return set, nil
}

// Save validates and atomically writes a route policy file.
func (m *RouteFileManager) Save(name string, set RouteRuleSet) error {
	path, err := m.filePath(name)
	if err != nil {
		return err
	}
	if set.UpdatedAt == "" {
		set.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	if err := ValidateRouteRuleSet(set); err != nil {
		return err
	}
	data, err := yaml.Marshal(set)
	if err != nil {
		return fmt.Errorf("marshal route file yaml: %w", err)
	}
	return writeFileAtomic(path, data)
}

// Create writes a new default route file.
func (m *RouteFileManager) Create(name string) error {
	if _, err := m.filePath(name); err != nil {
		return err
	}
	path := filepath.Join(m.dir, name)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("route file %q already exists", name)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat route file: %w", err)
	}
	set := DefaultRouteRuleSet()
	set.Name = strings.TrimSuffix(name, ".rule")
	return m.Save(name, set)
}

// Delete removes a route policy file.
func (m *RouteFileManager) Delete(name string) error {
	path, err := m.filePath(name)
	if err != nil {
		return err
	}
	if name == DefaultRouteFileName {
		return errors.New("default route file cannot be deleted")
	}
	return os.Remove(path)
}

func (m *RouteFileManager) filePath(name string) (string, error) {
	if m.dir == "" {
		return "", errors.New("route directory is required")
	}
	if err := ValidateRouteFileName(name); err != nil {
		return "", err
	}
	cleanDir, err := filepath.Abs(m.dir)
	if err != nil {
		return "", fmt.Errorf("resolve route directory: %w", err)
	}
	path := filepath.Join(cleanDir, name)
	if filepath.Dir(path) != cleanDir {
		return "", errors.New("route file path escapes config directory")
	}
	return path, nil
}

// DefaultRouteRuleSet returns a catch-all route file.
func DefaultRouteRuleSet() RouteRuleSet {
	return RouteRuleSet{
		Name:        "Default Route Rules",
		Version:     1,
		Description: "Default outbound policy",
		Rules: []RouteRule{
			{
				ID:        "default",
				Name:      "Default Route",
				Enabled:   true,
				Priority:  10000,
				Protocols: []string{"socks5", "http"},
				MatchType: "any",
				Targets:   []string{"*"},
				Outbound: OutboundBinding{
					Mode: "default",
				},
			},
		},
	}
}

// ValidateRouteRuleSet checks file-level and rule-level route policy constraints.
func ValidateRouteRuleSet(set RouteRuleSet) error {
	if strings.TrimSpace(set.Name) == "" {
		return errors.New("route rule set name is required")
	}
	if set.Version < 1 {
		return errors.New("route rule set version must be at least 1")
	}
	if len(set.Rules) == 0 {
		return errors.New("route rule set must contain at least one rule")
	}

	seen := make(map[string]struct{}, len(set.Rules))
	hasCatchAll := false
	for index, rule := range set.Rules {
		if err := validateRouteRule(rule); err != nil {
			return fmt.Errorf("route rule %d: %w", index+1, err)
		}
		id := strings.TrimSpace(rule.ID)
		if _, ok := seen[id]; ok {
			return fmt.Errorf("route rule id %q is duplicated", id)
		}
		seen[id] = struct{}{}
		if rule.Enabled && rule.MatchType == "any" {
			hasCatchAll = true
		}
	}
	if !hasCatchAll {
		return errors.New("route rule set must contain an enabled catch-all rule")
	}
	return nil
}

func validateRouteRule(rule RouteRule) error {
	if strings.TrimSpace(rule.ID) == "" {
		return errors.New("id is required")
	}
	if strings.TrimSpace(rule.Name) == "" {
		return errors.New("name is required")
	}
	if rule.Priority <= 0 {
		return errors.New("priority must be greater than 0")
	}
	if len(rule.Protocols) == 0 {
		return errors.New("protocols are required")
	}
	for _, protocol := range rule.Protocols {
		switch strings.ToLower(strings.TrimSpace(protocol)) {
		case "socks5", "http":
		default:
			return fmt.Errorf("unsupported protocol %q", protocol)
		}
	}
	switch rule.MatchType {
	case "any", "ip", "cidr", "domain", "wildcard":
	default:
		return fmt.Errorf("unsupported match type %q", rule.MatchType)
	}
	if len(rule.Targets) == 0 {
		return errors.New("targets are required")
	}
	for _, target := range rule.Targets {
		if strings.TrimSpace(target) == "" {
			return errors.New("target must not be empty")
		}
		switch rule.MatchType {
		case "ip":
			if net.ParseIP(strings.TrimSpace(target)) == nil {
				return fmt.Errorf("invalid ip target %q", target)
			}
		case "cidr":
			if _, _, err := net.ParseCIDR(strings.TrimSpace(target)); err != nil {
				return fmt.Errorf("invalid cidr target %q", target)
			}
		}
	}

	switch rule.Outbound.Mode {
	case "", "default":
	case "intercept":
	case "local_ip":
		if net.ParseIP(strings.TrimSpace(rule.Outbound.LocalIP)) == nil {
			return errors.New("local_ip outbound requires a valid local IP")
		}
	case "interface":
		if strings.TrimSpace(rule.Outbound.Interface) == "" {
			return errors.New("interface outbound requires an interface name")
		}
	default:
		return fmt.Errorf("unsupported outbound mode %q", rule.Outbound.Mode)
	}
	return nil
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create route directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp route file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp route file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp route file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("set temp route file permissions: %w", err)
	}
	existing, err := os.ReadFile(path)
	hadExisting := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read current route file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		if hadExisting {
			if removeErr := os.Remove(path); removeErr == nil {
				err = os.Rename(tmpPath, path)
			}
		}
		if err != nil {
			if hadExisting {
				_ = os.WriteFile(path, existing, 0o600)
			}
			return fmt.Errorf("replace route file: %w", err)
		}
	}
	cleanup = false
	return nil
}
