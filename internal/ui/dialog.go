package ui

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"mfa_reader/internal/model"
	"mfa_reader/internal/storage"
)

func showAddAccountDialog(myWindow fyne.Window, accounts *[]model.MFAAccount, renderList func(string)) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("例如: Google")

	secretEntry := widget.NewEntry()
	secretEntry.SetPlaceHolder("例如: JBSWY3DPEHPK3PXP")

	form := widget.NewForm(
		widget.NewFormItem("账号名称", nameEntry),
		widget.NewFormItem("密　　钥", secretEntry),
	)

	content := container.NewPadded(form)

	d := dialog.NewCustomConfirm("➕ 添加 MFA 账号", "添加", "取消", content, func(ok bool) {
		if !ok {
			return
		}

		accountName := strings.TrimSpace(nameEntry.Text)
		secret := strings.TrimSpace(secretEntry.Text)

		if accountName == "" || secret == "" {
			dialog.NewInformation("❌ 错误", "账号名称和密钥不能为空", myWindow).Show()
			return
		}

		secret = strings.ToUpper(strings.ReplaceAll(secret, " ", ""))
		secret = strings.ReplaceAll(secret, "-", "")
		secret = strings.TrimRight(secret, "=")

		if len(secret) < 16 {
			dialog.NewInformation("❌ 错误", "密钥长度不足，请检查是否输入正确", myWindow).Show()
			return
		}

		*accounts = append(*accounts, model.MFAAccount{
			AccountName: accountName,
			Time:        time.Now().UnixMilli(),
			Secret:      secret,
		})

		if err := storage.SaveMFAAccounts(*accounts); err != nil {
			dialog.NewInformation("❌ 错误", "保存失败: "+err.Error(), myWindow).Show()
			return
		}

		renderList("")
	}, myWindow)

	d.Resize(fyne.NewSize(340, 220))
	d.Show()
}