package main

import (
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"mfa_reader/internal/storage"
	"mfa_reader/internal/theme"
	"mfa_reader/internal/ui"
)

func init() {
	os.Setenv("FYNE_THEME", "light")

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
	myApp := app.New()
	myApp.SetIcon(theme.LoadIcon())

	myWindow := myApp.NewWindow("虚拟MFA")
	myWindow.Resize(fyne.NewSize(400, 700))

	accounts := storage.LoadMFAAccounts()
	ui.SetupMainWindow(myWindow, accounts)

	myWindow.ShowAndRun()
}
