//go:build windows

package platform

import (
	"strings"

	"github.com/getlantern/systray"
)

var currentNativeMenu nativeTrayMenu

func supportsTrayMenu() bool {
	return true
}

func trayHideDescription() string {
	return "Windows uses the notification area tray icon to keep the app running in the background."
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
				menu.status.SetTitle("服务状态: 运行中")
			} else {
				menu.status.SetTitle("服务状态: 已停止")
			}
		}
		if menu.ips != nil {
			menu.ips.Show()
			text := "未检测到"
			if len(localIPs) > 0 {
				text = strings.Join(localIPs, " / ")
			}
			menu.ips.SetTitle("本地IP: " + text)
		}
		if menu.socks != nil {
			menu.socks.Show()
			menu.socks.SetTitle("SOCKS5: " + emptyAsDash(socksAddr))
		}
		if menu.http != nil {
			menu.http.Show()
			menu.http.SetTitle("HTTP: " + emptyAsDash(httpAddr))
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
		systray.SetTitle("GoProxy 运行中")
		systray.SetTooltip("GoProxy - 服务运行中")
		return
	}

	menu.start.Enable()
	menu.stop.Disable()
	systray.SetTitle("GoProxy 已停止")
	systray.SetTooltip("GoProxy - 服务已停止")
}

func emptyAsDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
