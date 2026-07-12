package ui

import (
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"mfa_reader/internal/model"
	"mfa_reader/internal/storage"
)

var base32Regex = regexp.MustCompile(`^[A-Z2-7]+=*$`)

func showAddAccountDialog(ctx *appContext) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("例如: Google")

	secretEntry := widget.NewEntry()
	secretEntry.SetPlaceHolder("例如: JBSWY3DPEHPK3PXP")

	nameItem := widget.NewFormItem("账号名称", nameEntry)
	nameItem.Required = true
	secretItem := widget.NewFormItem("密　　钥", secretEntry)
	secretItem.Required = true

	form := widget.NewForm(nameItem, secretItem)

	content := container.NewPadded(form)

	d := dialog.NewCustomConfirm("➕ 添加 MFA 账号", "添加", "取消", content, func(ok bool) {
		if !ok {
			return
		}

		accountName := strings.TrimSpace(nameEntry.Text)
		secret := strings.TrimSpace(secretEntry.Text)

		if accountName == "" || secret == "" {
			dialog.NewInformation("❌ 错误", "账号名称和密钥不能为空", ctx.window).Show()
			return
		}

		acc := model.MFAAccount{
			AccountName: accountName,
			Secret:      secret,
		}
		normalized := acc.NormalizeSecret()

		if !base32Regex.MatchString(normalized) {
			dialog.NewInformation("❌ 错误", "密钥包含无效字符，仅支持 A-Z 和 2-7", ctx.window).Show()
			return
		}

		if len(normalized) < 16 {
			dialog.NewInformation("❌ 错误", "密钥长度不足，请检查是否输入正确", ctx.window).Show()
			return
		}

		acc.Secret = normalized
		*ctx.accounts = append(*ctx.accounts, acc)

		if err := storage.SaveMFAAccounts(*ctx.accounts); err != nil {
			dialog.NewInformation("❌ 错误", "保存失败: "+err.Error(), ctx.window).Show()
			return
		}

		ctx.onChanged("")
	}, ctx.window)

	d.Resize(fyne.NewSize(340, 220))
	d.Show()
}
