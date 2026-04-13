package main

import (
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/duke-git/lancet/v2/fileutil"

	"mfa_reader/internal/storage"
	"mfa_reader/internal/theme"
	"mfa_reader/internal/ui"
)

func init() {
	os.Setenv("FYNE_THEME", "light")

	if tz := os.Getenv("TZ"); tz == "" {
		if loc, err := time.LoadLocation("Local"); err == nil {
			time.Local = loc
		}
	}

	fontPath := "C:\\Windows\\Fonts\\simhei.ttf"
	if fileutil.IsExist(fontPath) {
		os.Setenv("FYNE_FONT", fontPath)
	} else {
		altFont := "C:\\Windows\\Fonts\\msyh.ttf"
		if fileutil.IsExist(altFont) {
			os.Setenv("FYNE_FONT", altFont)
		}
	}
}

func main() {
	myApp := app.New()

	// 设置应用程序的图标
	myApp.SetIcon(theme.LoadIcon())

	myWindow := myApp.NewWindow("虚拟MFA")
	myWindow.Resize(fyne.NewSize(400, 700))

	// 加载账号数据
	accounts := storage.LoadMFAAccounts()

	// 初始化主界面（传递停止 channel）
	ui.SetupMainWindow(myWindow, accounts)

	myWindow.ShowAndRun()
}
