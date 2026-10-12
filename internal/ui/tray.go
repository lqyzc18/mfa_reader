package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"mfa_reader/internal/hotkey"
	"mfa_reader/internal/theme"
)

// attachDesktop 装配系统托盘与全局热键：关闭窗口即退出程序，
// 需要临时收起窗口时用快捷键 Ctrl+Alt+M 隐藏，再从快捷键或托盘菜单恢复。
func attachDesktop(a fyne.App, w fyne.Window) {
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

	if desk, ok := a.(desktop.App); ok {
		desk.SetSystemTrayIcon(theme.LoadIcon())
		desk.SetSystemTrayMenu(fyne.NewMenu(WindowTitle,
			fyne.NewMenuItem("显示", show),
			fyne.NewMenuItem("退出", func() {
				saveWindowState(w, a.Preferences())
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
}
