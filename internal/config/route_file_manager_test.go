package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRouteFileManagerEnsureDefaultAndList(t *testing.T) {
	manager := NewRouteFileManager(t.TempDir())
	if err := manager.EnsureDefault(); err != nil {
		t.Fatalf("ensure default: %v", err)
	}
	if _, err := os.Stat(filepath.Join(manager.Dir(), DefaultRouteFileName)); err != nil {
		t.Fatalf("expected default rule file: %v", err)
	}
	files, err := manager.List(DefaultRouteFileName)
	if err != nil {
		t.Fatalf("list route files: %v", err)
	}
	if len(files) != 1 || files[0].Name != DefaultRouteFileName || !files[0].IsActive {
		t.Fatalf("unexpected files: %+v", files)
	}
}

func TestRouteFileManagerRejectsUnsafeNames(t *testing.T) {
	manager := NewRouteFileManager(t.TempDir())
	for _, name := range []string{"../bad.rule", "bad/name.rule", "bad name.rule", "rules.yaml"} {
		if err := manager.Create(name); err == nil {
			t.Fatalf("expected unsafe route file name %q to fail", name)
		}
	}
}

func TestValidateRouteRuleSet(t *testing.T) {
	set := DefaultRouteRuleSet()
	set.Rules = append([]RouteRule{
		{
			ID:        "office",
			Name:      "office",
			Enabled:   true,
			Priority:  100,
			Protocols: []string{"http"},
			MatchType: "cidr",
			Targets:   []string{"10.0.0.0/8"},
			Outbound:  OutboundBinding{Mode: "local_ip", LocalIP: "127.0.0.1"},
		},
	}, set.Rules...)
	if err := ValidateRouteRuleSet(set); err != nil {
		t.Fatalf("valid route set rejected: %v", err)
	}

	set.Rules[0].ID = "default"
	if err := ValidateRouteRuleSet(set); err == nil {
		t.Fatal("expected duplicate id error")
	}
}
