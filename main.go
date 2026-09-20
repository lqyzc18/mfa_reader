package main

import (
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	"mfa_reader/internal/storage"
	"mfa_reader/internal/theme"
	"mfa_reader/internal/ui"
)

func init() {
	// 仅在用户未显式设置时给出 light 默认值，避免覆盖系统/用户主题偏好。
	// 注意：自定义卡片使用了固定浅色背景，深色模式下 UI 会与系统背景不一致。
	if os.Getenv("FYNE_THEME") == "" {
		os.Setenv("FYNE_THEME", "light")
	}

	for _, fontPath := range []string{
		`C:\Windows\Fonts\simhei.ttf`,
		`C:\Windows\Fonts\msyh.ttf`,
	} {
		if _, err := os.Stat(fontPath); err == nil {
			os.Setenv("FYNE_FONT", fontPath)
			break
		}
	}
}

func main() {
	myApp := app.NewWithID("com.lqyzc18.mfa_reader")
	myApp.SetIcon(theme.LoadIcon())

	myWindow := myApp.NewWindow("虚拟MFA")

	store, err := storage.Open()
	if err != nil {
		ui.SetupMainWindow(myApp, myWindow, storage.NewEmpty())
		dialog.NewInformation("数据文件错误", err.Error()+"\n\n将以空列表启动。", myWindow).Show()
		myWindow.ShowAndRun()
		return
	}

	ui.SetupMainWindow(myApp, myWindow, store)
	myWindow.ShowAndRun()
}
