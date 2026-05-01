package proxy

import (
	"net"
	"sort"
	"strings"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
)

// RouteContext describes one outbound connection for route matching.
type RouteContext struct {
	Protocol   string
	TargetHost string
	TargetPort string
	IsIP       bool
}

// RouteDecision is the selected outbound policy for one connection.
type RouteDecision struct {
	RuleID        string
	RuleName      string
	OutboundMode  string
	InterfaceName string
	LocalIP       string
}

// RoutePolicyEngine matches route contexts against ordered rules.
type RoutePolicyEngine interface {
	Match(ctx RouteContext) RouteDecision
}

type compiledRouteEngine struct {
	rules []compiledRouteRule
}

type compiledRouteRule struct {
	order     int
	rule      config.RouteRule
	protocols map[string]struct{}
	ips       []net.IP
	cidrs     []*net.IPNet
	targets   []string
}

// NewRoutePolicyEngine compiles a rule set into a matcher.
func NewRoutePolicyEngine(set config.RouteRuleSet) RoutePolicyEngine {
	rules := make([]compiledRouteRule, 0, len(set.Rules))
	for index, rule := range set.Rules {
		if !rule.Enabled {
			continue
		}
		compiled := compiledRouteRule{
			order:     index,
			rule:      rule,
			protocols: make(map[string]struct{}, len(rule.Protocols)),
		}
		for _, protocol := range rule.Protocols {
			compiled.protocols[strings.ToLower(strings.TrimSpace(protocol))] = struct{}{}
		}
		for _, target := range rule.Targets {
			text := strings.ToLower(strings.TrimSpace(target))
			if text == "" {
				continue
			}
			switch rule.MatchType {
			case "ip":
				if ip := net.ParseIP(text); ip != nil {
					compiled.ips = append(compiled.ips, ip)
				}
			case "cidr":
				if _, cidr, err := net.ParseCIDR(text); err == nil {
					compiled.cidrs = append(compiled.cidrs, cidr)
				}
			default:
				compiled.targets = append(compiled.targets, text)
			}
		}
		rules = append(rules, compiled)
	}
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].rule.Priority == rules[j].rule.Priority {
			return rules[i].order < rules[j].order
		}
		return rules[i].rule.Priority < rules[j].rule.Priority
	})
	return &compiledRouteEngine{rules: rules}
}

func (e *compiledRouteEngine) Match(ctx RouteContext) RouteDecision {
	for _, rule := range e.rules {
		if !rule.matchesProtocol(ctx.Protocol) {
			continue
		}
		if !rule.matchesTarget(ctx) {
			continue
		}
		return decisionFromRule(rule.rule)
	}
	return RouteDecision{OutboundMode: "default"}
}

func (r compiledRouteRule) matchesProtocol(protocol string) bool {
	_, ok := r.protocols[strings.ToLower(strings.TrimSpace(protocol))]
	return ok
}

func (r compiledRouteRule) matchesTarget(ctx RouteContext) bool {
	host := strings.ToLower(strings.TrimSpace(ctx.TargetHost))
	ip := net.ParseIP(host)

	switch r.rule.MatchType {
	case "any":
		return true
	case "ip":
		if ip == nil {
			return false
		}
		for _, target := range r.ips {
			if target.Equal(ip) {
				return true
			}
		}
	case "cidr":
		if ip == nil {
			return false
		}
		for _, target := range r.cidrs {
			if target.Contains(ip) {
				return true
			}
		}
	case "domain":
		if ip != nil {
			return false
		}
		for _, target := range r.targets {
			if host == target {
				return true
			}
		}
	case "wildcard":
		if ip != nil {
			return false
		}
		for _, target := range r.targets {
			if matchWildcardDomain(target, host) {
				return true
			}
		}
	}
	return false
}

func matchWildcardDomain(pattern, host string) bool {
	pattern = strings.TrimSpace(strings.ToLower(pattern))
	host = strings.TrimSpace(strings.ToLower(host))
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(host, suffix) && len(host) > len(suffix)
	}
	return host == pattern
}

func decisionFromRule(rule config.RouteRule) RouteDecision {
	mode := strings.TrimSpace(rule.Outbound.Mode)
	if mode == "" {
		mode = "default"
	}
	return RouteDecision{
		RuleID:        rule.ID,
		RuleName:      rule.Name,
		OutboundMode:  mode,
		InterfaceName: rule.Outbound.Interface,
		LocalIP:       rule.Outbound.LocalIP,
	}
}
