//go:build windows

package platform

import (
	"fmt"
	"strings"

	"github.com/getlantern/systray"
)

var currentNativeMenu nativeTrayMenu

func supportsTrayMenu() bool {
	return true
}

func trayHideDescription() string {
	return "Windows 使用通知区域托盘图标保持后台运行。"
}

func (t *TrayManager) setNativeMenu(menu nativeTrayMenu) {
	t.mu.Lock()
	currentNativeMenu = menu
	t.mu.Unlock()
}

func (t *TrayManager) updateNativeTray() {
	t.mu.Lock()
	running := t.serverRunning
	showStatusIP := t.showStatusIP
	menu := currentNativeMenu
	localIPs := append([]string(nil), t.localIPs...)
	socksAddr := t.socksAddr
	httpAddr := t.httpAddr
	t.mu.Unlock()

	if menu.start == nil || menu.stop == nil {
		return
	}

	if showStatusIP {
		if menu.status != nil {
			menu.status.Show()
			if running {
				menu.status.SetTitle("服务状态：运行中")
			} else {
				menu.status.SetTitle("服务状态：已停止")
			}
		}
		if menu.ips != nil {
			menu.ips.Show()
			menu.ips.SetTitle("网卡 IP：" + trayIPSummary(localIPs))
			if len(localIPs) > 0 {
				menu.ips.SetTooltip(strings.Join(localIPs, "\n"))
			} else {
				menu.ips.SetTooltip("未检测到网卡 IP")
			}
		}
		if menu.socks != nil {
			menu.socks.Show()
			menu.socks.SetTitle("SOCKS5：" + emptyAsDash(socksAddr))
		}
		if menu.http != nil {
			menu.http.Show()
			menu.http.SetTitle("HTTP：" + emptyAsDash(httpAddr))
		}
	} else {
		if menu.status != nil {
			menu.status.Hide()
		}
		if menu.ips != nil {
			menu.ips.Hide()
		}
		if menu.socks != nil {
			menu.socks.Hide()
		}
		if menu.http != nil {
			menu.http.Hide()
		}
	}

	if running {
		menu.start.Disable()
		menu.stop.Enable()
		systray.SetTooltip("GoProxy - 服务运行中")
		return
	}
	menu.start.Enable()
	menu.stop.Disable()
	systray.SetTooltip("GoProxy - 服务已停止")
}

func emptyAsDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func trayIPSummary(localIPs []string) string {
	if len(localIPs) == 0 {
		return "未检测到"
	}
	first := strings.TrimSpace(localIPs[0])
	if first == "" {
		return "未检测到"
	}
	if len(localIPs) == 1 {
		return first
	}
	return fmt.Sprintf("%s 等 %d 个", first, len(localIPs))
}
