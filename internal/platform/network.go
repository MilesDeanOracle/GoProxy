package platform

import "net"

// NetworkInterface describes a local adapter and its addresses.
type NetworkInterface struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Addresses   []string `json:"addresses"`
	Up          bool     `json:"up"`
	Loopback    bool     `json:"loopback"`
}

// LocalIPAddresses returns IPv4 addresses from active non-loopback interfaces.
func LocalIPAddresses() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var ips []string
	for _, item := range interfaces {
		if item.Flags&net.FlagUp == 0 || item.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := item.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			ip = ip.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}
			text := ip.String()
			if _, ok := seen[text]; ok {
				continue
			}
			seen[text] = struct{}{}
			ips = append(ips, text)
		}
	}
	return ips, nil
}

// NetworkInterfaces returns local network adapters for route binding selection.
func NetworkInterfaces() ([]NetworkInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	result := make([]NetworkInterface, 0, len(interfaces))
	for _, item := range interfaces {
		info := NetworkInterface{
			Name:        item.Name,
			DisplayName: item.Name,
			Up:          item.Flags&net.FlagUp != 0,
			Loopback:    item.Flags&net.FlagLoopback != 0,
		}
		addrs, err := item.Addrs()
		if err == nil {
			for _, addr := range addrs {
				info.Addresses = append(info.Addresses, addr.String())
			}
		}
		result = append(result, info)
	}
	return result, nil
}
