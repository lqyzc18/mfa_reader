package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"mfa_reader/internal/hotkey"
	"mfa_reader/internal/theme"
)

func attachDesktop(a fyne.App, w fyne.Window) func() {
	hidden := false
	show := func() {
		hidden = false
		w.Show()
		w.RequestFocus()
	}
	hide := func() {
		saveWindowState(w, a.Preferences())
		hidden = true
		w.Hide()
	}

	w.SetCloseIntercept(func() {
		hide()
	})

	if desk, ok := a.(desktop.App); ok {
		desk.SetSystemTrayIcon(theme.LoadIcon())
		desk.SetSystemTrayMenu(fyne.NewMenu("虚拟MFA",
			fyne.NewMenuItem("显示", show),
			fyne.NewMenuItem("退出", func() {
				saveWindowState(w, a.Preferences())
				w.SetCloseIntercept(nil)
				a.Quit()
			}),
		))
	}

	stopHotkey, err := hotkey.Start(func() {
		fyne.Do(func() {
			if hidden {
				show()
			} else {
				hide()
			}
		})
	})
	if err != nil {
		log.Printf("[hotkey] 注册 Ctrl+Alt+M 失败: %v", err)
		stopHotkey = func() {}
	}

	a.Lifecycle().SetOnStarted(func() {
		restoreWindowPos(a.Preferences())
	})
	a.Lifecycle().SetOnStopped(func() {
		saveWindowState(w, a.Preferences())
		stopHotkey()
	})

	return stopHotkey
}
