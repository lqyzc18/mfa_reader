package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"mfa_reader/internal/model"
)

func showAddAccountDialog(ctx *appContext) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("例如: Google")

	secretEntry := widget.NewPasswordEntry()
	secretEntry.SetPlaceHolder("例如: JBSWY3DPEHPK3PXP")

	hint := widget.NewLabel("密钥仅支持 A-Z / 2-7，长度至少 16 位；回车可快速提交")
	hint.Wrapping = fyne.TextWrapWord
	hint.Importance = widget.LowImportance

	nameItem := widget.NewFormItem("账号名称", nameEntry)
	nameItem.Required = true
	secretItem := widget.NewFormItem("密　　钥", secretEntry)
	secretItem.Required = true
	form := widget.NewForm(nameItem, secretItem)

	var d dialog.Dialog

	trySubmit := func() {
		accountName := strings.TrimSpace(nameEntry.Text)
		secret := strings.TrimSpace(secretEntry.Text)

		if accountName == "" || secret == "" {
			dialog.NewInformation("提示", "账号名称和密钥不能为空", ctx.window).Show()
			return
		}

		acc := model.MFAAccount{AccountName: accountName, Secret: secret}
		normalized := acc.NormalizeSecret()
		if err := model.ValidateSecret(normalized); err != nil {
			dialog.NewInformation("提示", err.Error(), ctx.window).Show()
			return
		}

		if dup, msg := ctx.hasDuplicate(accountName, normalized); dup {
			dialog.NewInformation("提示", msg, ctx.window).Show()
			return
		}

		acc.Secret = normalized
		if err := ctx.addAccount(acc); err != nil {
			dialog.NewInformation("错误", "保存失败: "+err.Error(), ctx.window).Show()
			return
		}

		d.Hide()
		if ctx.searchEntry != nil {
			ctx.searchEntry.SetText("")
		}
		ctx.onChanged("")
		if ctx.showToast != nil {
			ctx.showToast("已添加「" + accountName + "」")
		}
	}

	cancelBtn := widget.NewButton("取消", func() { d.Hide() })
	addBtn := widget.NewButton("添加", trySubmit)
	addBtn.Importance = widget.HighImportance

	buttons := container.NewGridWithColumns(2, cancelBtn, addBtn)
	body := container.NewPadded(container.NewVBox(form, hint, buttons))

	d = dialog.NewCustomWithoutButtons("添加 MFA 账号", body, ctx.window)

	nameEntry.OnSubmitted = func(string) {
		ctx.window.Canvas().Focus(secretEntry)
	}
	secretEntry.OnSubmitted = func(string) {
		trySubmit()
	}

	d.Resize(fyne.NewSize(360, 280))
	d.Show()
	ctx.window.Canvas().Focus(nameEntry)
}
