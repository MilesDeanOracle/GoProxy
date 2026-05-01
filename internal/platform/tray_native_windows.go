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
			systray.SetTitle("GoProxy Stopped")
			systray.SetTooltip("GoProxy")
			systray.SetOnDblClick(func() {
				t.mu.Lock()
				action := t.actions.ShowWindow
				t.mu.Unlock()
				if action != nil {
					action()
				}
			})

			menu := nativeTrayMenu{
				status: systray.AddMenuItem("Service Status: Stopped", "Current proxy service status"),
				ips:    systray.AddMenuItem("Local IP: Not detected", "Current local network IP addresses"),
				socks:  systray.AddMenuItem("SOCKS5: -", "SOCKS5 listen address"),
				http:   systray.AddMenuItem("HTTP: -", "HTTP CONNECT listen address"),
			}
			menu.status.Disable()
			menu.ips.Disable()
			menu.socks.Disable()
			menu.http.Disable()
			systray.AddSeparator()
			menu.show = systray.AddMenuItem("Show Window", "Show the GoProxy main window")
			menu.start = systray.AddMenuItem("Start Service", "Start the proxy service")
			menu.stop = systray.AddMenuItem("Stop Service", "Stop the proxy service")
			menu.quit = systray.AddMenuItem("Quit", "Quit GoProxy")

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
