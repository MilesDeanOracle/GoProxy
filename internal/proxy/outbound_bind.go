package proxy

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

var errRouteIntercepted = errors.New("route intercepted")

func applyOutboundBinding(dialer *net.Dialer, decision *RouteDecision) error {
	if dialer == nil || decision == nil {
		return nil
	}

	switch strings.TrimSpace(decision.OutboundMode) {
	case "", "default":
		decision.OutboundMode = "default"
		return nil
	case "intercept":
		decision.OutboundMode = "intercept"
		return fmt.Errorf("%w: connection blocked by rule %s", errRouteIntercepted, decision.RuleName)
	case "local_ip":
		ip := net.ParseIP(strings.TrimSpace(decision.LocalIP))
		if ip == nil {
			return fmt.Errorf("invalid local ip %q", decision.LocalIP)
		}
		dialer.LocalAddr = &net.TCPAddr{IP: ip}
		decision.LocalIP = ip.String()
		return nil
	case "interface":
		ip, err := interfaceIPv4(decision.InterfaceName)
		if err != nil {
			return err
		}
		dialer.LocalAddr = &net.TCPAddr{IP: ip}
		decision.LocalIP = ip.String()
		return nil
	default:
		return fmt.Errorf("unsupported outbound mode %q", decision.OutboundMode)
	}
}

func interfaceIPv4(name string) (net.IP, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("interface name is required")
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	for _, iface := range ifaces {
		if iface.Name != name {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			return nil, fmt.Errorf("interface %q is not up", name)
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("list interface %q addresses: %w", name, err)
		}
		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ipv4 := ip.To4(); ipv4 != nil && !ipv4.IsLoopback() {
				return ipv4, nil
			}
		}
		return nil, fmt.Errorf("interface %q has no usable ipv4 address", name)
	}
	return nil, fmt.Errorf("interface %q not found", name)
}
