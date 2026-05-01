//go:build windows

package platform

import (
	"runtime"

	"github.com/getlantern/systray"
)

type nativeTrayMenu struct {
	status *systray.MenuItem
	ips    *systray.MenuItem
	socks  *systray.MenuItem
	http   *systray.MenuItem
	show   *systray.MenuItem
	start  *systray.MenuItem
	stop   *systray.MenuItem
	quit   *systray.MenuItem
}

func (t *TrayManager) startNativeTray(icon []byte) {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		systray.Run(func() {
			systray.SetIcon(icon)
			systray.SetTitle("GoProxy 已停止")
			systray.SetTooltip("GoProxy 代理服务")
			systray.SetOnDblClick(func() {
				t.mu.Lock()
				action := t.actions.ShowWindow
				t.mu.Unlock()
				if action != nil {
					action()
				}
			})

			menu := nativeTrayMenu{
				status: systray.AddMenuItem("服务状态: 已停止", "当前代理服务状态"),
				ips:    systray.AddMenuItem("本地IP: 未检测到", "当前局域网IP地址"),
				socks:  systray.AddMenuItem("SOCKS5: -", "SOCKS5 监听地址"),
				http:   systray.AddMenuItem("HTTP: -", "HTTP CONNECT 监听地址"),
			}
			menu.status.Disable()
			menu.ips.Disable()
			menu.socks.Disable()
			menu.http.Disable()
			systray.AddSeparator()
			menu.show = systray.AddMenuItem("显示窗口", "显示 GoProxy 主窗口")
			menu.start = systray.AddMenuItem("启动服务", "启动代理服务")
			menu.stop = systray.AddMenuItem("停止服务", "停止代理服务")
			menu.quit = systray.AddMenuItem("退出", "退出 GoProxy")

			t.setNativeMenu(menu)
			t.updateNativeTray()

			go t.watchNativeMenu(menu)
		}, func() {
			t.mu.Lock()
			t.nativeStarted = false
			t.mu.Unlock()
		})
	}()
}

func (t *TrayManager) watchNativeMenu(menu nativeTrayMenu) {
	for {
		select {
		case <-menu.show.ClickedCh:
			if t.actions.ShowWindow != nil {
				t.actions.ShowWindow()
			}
		case <-menu.start.ClickedCh:
			t.runTrayAction(t.actions.StartServer)
		case <-menu.stop.ClickedCh:
			t.runTrayAction(t.actions.StopServer)
		case <-menu.quit.ClickedCh:
			if t.actions.Quit != nil {
				t.actions.Quit()
			}
			return
		}
	}
}

func (t *TrayManager) stopNativeTray() {
	if t.State().NativeStarted {
		systray.Quit()
	}
}
