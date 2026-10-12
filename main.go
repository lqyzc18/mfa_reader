package main

import (
	"errors"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	"mfa_reader/internal/singleinstance"
	"mfa_reader/internal/storage"
	"mfa_reader/internal/theme"
	"mfa_reader/internal/ui"
	"mfa_reader/internal/winpos"
)

// singleInstanceName 是单实例互斥量名称，与 FyneApp.toml 中的应用 ID 保持一致。
const singleInstanceName = `Local\com.lqyzc18.mfa_reader`

func init() {
	// 仅在用户未显式设置时给出 light 默认值，避免覆盖系统/用户主题偏好。
	// 注意：自定义卡片使用了固定浅色背景，深色模式下 UI 会与系统背景不一致。
	if os.Getenv("FYNE_THEME") == "" {
		os.Setenv("FYNE_THEME", "light")
	}

	// 同理尊重用户显式指定的 FYNE_FONT，只在未设置时才探测系统中文字体。
	if os.Getenv("FYNE_FONT") == "" {
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
}

func main() {
	// 只允许一个实例：两个实例各自持有内存副本，后写会覆盖先写，账号可能静默丢失。
	release, err := singleinstance.Acquire(singleInstanceName)
	if errors.Is(err, singleinstance.ErrAlreadyRunning) {
		winpos.Activate(ui.WindowTitle)
		return
	}
	if err == nil {
		defer release()
	}

	myApp := app.NewWithID("com.lqyzc18.mfa_reader")
	myApp.SetIcon(theme.LoadIcon())

	myWindow := myApp.NewWindow(ui.WindowTitle)
	// 主窗口即程序生命周期：关闭它直接退出程序（不驻留托盘）。
	myWindow.SetMaster()

	store, err := storage.Open()
	if err != nil {
		startReadOnly(myApp, myWindow, err)
		return
	}

	ui.SetupMainWindow(myApp, myWindow, store)
	myWindow.ShowAndRun()
}

// startReadOnly 处理数据文件读取失败：先把原文件留档，再以只读模式启动，
// 保证用户能查看提示、处理原文件，而不会被空表覆盖。
func startReadOnly(myApp fyne.App, myWindow fyne.Window, cause error) {
	backupPath, backupErr := storage.QuarantineDataFile()

	store := storage.NewEmpty()
	store.SetReadOnly(true)

	ui.SetupMainWindow(myApp, myWindow, store)

	notice := cause.Error() + "\n\n本次以只读模式启动，不会写入数据文件。"
	switch {
	case backupErr != nil:
		notice += "\n原文件留档失败：" + backupErr.Error()
	case backupPath != "":
		notice += "\n原文件已留档为：" + backupPath
	}
	dialog.NewInformation("数据文件错误", notice, myWindow).Show()

	myWindow.ShowAndRun()
}
