package ui

import (
	"io"
	"os"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	fynestorage "fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"mfa_reader/internal/model"
	"mfa_reader/internal/qrscan"
	"mfa_reader/internal/storage"
)

func showImportDialog(ctx *appContext) {
	area := widget.NewMultiLineEntry()
	area.SetPlaceHolder("粘贴 otpauth:// 链接，或明文 JSON 数组")
	area.Wrapping = fyne.TextWrapWord

	var d dialog.Dialog

	doImportText := func(text string) {
		accs, err := storage.ParseImportText(text)
		if err != nil {
			dialog.NewInformation("导入失败", err.Error(), ctx.window).Show()
			return
		}
		added, skipped, err := ctx.store.Import(accs)
		if err != nil {
			dialog.NewInformation("导入失败", err.Error(), ctx.window).Show()
			return
		}
		d.Hide()
		ctx.refreshNow()
		ctx.showToast(formatImportResult(added, skipped))
	}

	pasteBtn := widget.NewButton("导入文本", func() {
		doImportText(area.Text)
	})
	pasteBtn.Importance = widget.HighImportance

	fileBtn := widget.NewButton("从图片导入二维码", func() {
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil || rc == nil {
				return
			}
			defer rc.Close()
			text, decErr := qrscan.FromFile(rc.URI().Path())
			if decErr != nil {
				dialog.NewInformation("导入失败", decErr.Error(), ctx.window).Show()
				return
			}
			accs, err := storage.ParseQRText(text)
			if err != nil {
				dialog.NewInformation("导入失败", err.Error(), ctx.window).Show()
				return
			}
			added, skipped, err := ctx.store.Import(accs)
			if err != nil {
				dialog.NewInformation("导入失败", err.Error(), ctx.window).Show()
				return
			}
			d.Hide()
			ctx.refreshNow()
			ctx.showToast(formatImportResult(added, skipped))
		}, ctx.window)
		fd.SetFilter(fynestorage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg", ".bmp", ".gif"}))
		fd.Show()
	})

	shotBtn := widget.NewButton("从屏幕截图导入二维码", func() {
		d.Hide()
		go captureAndImport(ctx)
	})

	bakBtn := widget.NewButton("从 JSON 备份导入", func() {
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil || rc == nil {
				return
			}
			path := rc.URI().Path()
			rc.Close()
			accs, err := storage.LoadBackup(path)
			if err != nil {
				dialog.NewInformation("导入失败", err.Error(), ctx.window).Show()
				return
			}
			finishImport(ctx, d, accs)
		}, ctx.window)
		fd.SetFilter(fynestorage.NewExtensionFileFilter([]string{".json"}))
		fd.Show()
	})

	hint := widget.NewLabel("支持 otpauth://totp 链接、JSON 账号数组、authenticator 二维码（图片或截图），以及本应用的 JSON 备份。")
	hint.Wrapping = fyne.TextWrapWord
	hint.Importance = widget.LowImportance

	body := container.NewPadded(container.NewVBox(
		area,
		pasteBtn,
		fileBtn,
		shotBtn,
		bakBtn,
		hint,
	))
	d = dialog.NewCustom("导入账号", "关闭", body, ctx.window)
	d.Resize(fyne.NewSize(400, 420))
	d.Show()
}

func finishImport(ctx *appContext, parent dialog.Dialog, accs []model.MFAAccount) {
	added, skipped, err := ctx.store.Import(accs)
	if err != nil {
		dialog.NewInformation("导入失败", err.Error(), ctx.window).Show()
		return
	}
	if parent != nil {
		parent.Hide()
	}
	ctx.refreshNow()
	ctx.showToast(formatImportResult(added, skipped))
}

func captureAndImport(ctx *appContext) {
	fyne.Do(func() { ctx.window.Hide() })
	time.Sleep(280 * time.Millisecond)
	text, err := qrscan.FromDisplays()
	fyne.Do(func() {
		ctx.window.Show()
		ctx.window.RequestFocus()
		if err != nil {
			dialog.NewInformation("导入失败", err.Error(), ctx.window).Show()
			return
		}
		accs, err := storage.ParseQRText(text)
		if err != nil {
			dialog.NewInformation("导入失败", err.Error(), ctx.window).Show()
			return
		}
		added, skipped, err := ctx.store.Import(accs)
		if err != nil {
			dialog.NewInformation("导入失败", err.Error(), ctx.window).Show()
			return
		}
		ctx.refreshNow()
		ctx.showToast(formatImportResult(added, skipped))
	})
}

func showExportDialog(ctx *appContext) {
	var d dialog.Dialog

	otpBtn := widget.NewButton("导出 otpauth 文本", func() {
		saveTextFile(ctx, "mfa-otpauth.txt", storage.ExportOTPAuth(ctx.store.Snapshot()))
		d.Hide()
	})

	bakBtn := widget.NewButton("导出 JSON 备份（复制当前数据文件）", func() {
		src := ctx.store.FilePath()
		fd := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
			if err != nil || wc == nil {
				return
			}
			defer wc.Close()
			in, err := os.Open(src)
			if err != nil {
				dialog.NewInformation("导出失败", err.Error(), ctx.window).Show()
				return
			}
			defer in.Close()
			if _, err := io.Copy(wc, in); err != nil {
				dialog.NewInformation("导出失败", err.Error(), ctx.window).Show()
				return
			}
			ctx.showToast("已导出 JSON 备份")
		}, ctx.window)
		fd.SetFileName("mfa-backup.json")
		fd.Show()
		d.Hide()
	})

	hint := widget.NewLabel("otpauth 与 JSON 备份都含明文密钥，请妥善保管。")
	hint.Wrapping = fyne.TextWrapWord
	hint.Importance = widget.LowImportance

	body := container.NewPadded(container.NewVBox(otpBtn, bakBtn, hint))
	d = dialog.NewCustom("导出账号", "关闭", body, ctx.window)
	d.Resize(fyne.NewSize(400, 280))
	d.Show()
}

func saveTextFile(ctx *appContext, name, content string) {
	fd := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
		if err != nil || wc == nil {
			return
		}
		defer wc.Close()
		if _, err := io.WriteString(wc, content); err != nil {
			dialog.NewInformation("导出失败", err.Error(), ctx.window).Show()
			return
		}
		ctx.showToast("已导出")
	}, ctx.window)
	fd.SetFileName(name)
	fd.Show()
}

func formatImportResult(added, skipped int) string {
	switch {
	case added == 0 && skipped == 0:
		return "没有可导入的账号"
	case skipped == 0:
		return "已导入 " + strconv.Itoa(added) + " 个账号"
	default:
		return "已导入 " + strconv.Itoa(added) + " 个，跳过重复 " + strconv.Itoa(skipped) + " 个"
	}
}
