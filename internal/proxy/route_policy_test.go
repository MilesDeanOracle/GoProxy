package proxy

import (
	"testing"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
)

func TestRoutePolicyEnginePriorityAndFirstMatch(t *testing.T) {
	engine := NewRoutePolicyEngine(config.RouteRuleSet{
		Name:    "test",
		Version: 1,
		Rules: []config.RouteRule{
			{
				ID:        "late",
				Name:      "late",
				Enabled:   true,
				Priority:  200,
				Protocols: []string{"socks5"},
				MatchType: "wildcard",
				Targets:   []string{"*.example.com"},
				Outbound:  config.OutboundBinding{Mode: "local_ip", LocalIP: "127.0.0.2"},
			},
			{
				ID:        "early",
				Name:      "early",
				Enabled:   true,
				Priority:  100,
				Protocols: []string{"socks5"},
				MatchType: "domain",
				Targets:   []string{"api.example.com"},
				Outbound:  config.OutboundBinding{Mode: "local_ip", LocalIP: "127.0.0.1"},
			},
		},
	})

	decision := engine.Match(RouteContext{
		Protocol:   "socks5",
		TargetHost: "api.example.com",
		TargetPort: "443",
	})
	if decision.RuleID != "early" || decision.LocalIP != "127.0.0.1" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestRoutePolicyEngineMatchesCIDRAndProtocol(t *testing.T) {
	engine := NewRoutePolicyEngine(config.RouteRuleSet{
		Name:    "test",
		Version: 1,
		Rules: []config.RouteRule{
			{
				ID:        "subnet",
				Name:      "subnet",
				Enabled:   true,
				Priority:  100,
				Protocols: []string{"http"},
				MatchType: "cidr",
				Targets:   []string{"10.20.0.0/16"},
				Outbound:  config.OutboundBinding{Mode: "interface", Interface: "eth0"},
			},
			{
				ID:        "default",
				Name:      "default",
				Enabled:   true,
				Priority:  10000,
				Protocols: []string{"socks5", "http"},
				MatchType: "any",
				Targets:   []string{"*"},
				Outbound:  config.OutboundBinding{Mode: "default"},
			},
		},
	})

	httpDecision := engine.Match(RouteContext{Protocol: "http", TargetHost: "10.20.1.10", TargetPort: "80"})
	if httpDecision.RuleID != "subnet" || httpDecision.InterfaceName != "eth0" {
		t.Fatalf("expected subnet match, got %+v", httpDecision)
	}
	socksDecision := engine.Match(RouteContext{Protocol: "socks5", TargetHost: "10.20.1.10", TargetPort: "80"})
	if socksDecision.RuleID != "default" {
		t.Fatalf("expected protocol fallback, got %+v", socksDecision)
	}
}
