//go:build windows

package platform

import "testing"

func TestTrayIPSummary(t *testing.T) {
	tests := []struct {
		name string
		ips  []string
		want string
	}{
		{
			name: "empty",
			want: "未检测到",
		},
		{
			name: "single",
			ips:  []string{"192.168.1.10"},
			want: "192.168.1.10",
		},
		{
			name: "multiple",
			ips:  []string{"192.168.1.10", "10.0.0.8", "172.16.0.4"},
			want: "192.168.1.10 等 3 个",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trayIPSummary(tt.ips); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
