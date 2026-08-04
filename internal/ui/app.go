package ui

import (
	"fmt"
	"image/color"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/pquerna/otp/totp"

	"mfa_reader/internal/model"
	"mfa_reader/internal/storage"
	"mfa_reader/internal/theme"
)

type updateItem struct {
	codeBinding binding.String
	progress    *widget.ProgressBar
	remainLabel *canvas.Text
	secret      string
}

type appContext struct {
	window       fyne.Window
	accounts     *[]model.MFAAccount
	accountsMu   *sync.RWMutex
	searchEntry  *widget.Entry
	onChanged    func(string)
	forceCodeGen *atomic.Bool
	showToast    func(string)
}

func (ctx *appContext) snapshotAccounts() []model.MFAAccount {
	ctx.accountsMu.RLock()
	defer ctx.accountsMu.RUnlock()
	out := make([]model.MFAAccount, len(*ctx.accounts))
	copy(out, *ctx.accounts)
	return out
}

func (ctx *appContext) addAccount(acc model.MFAAccount) error {
	ctx.accountsMu.Lock()
	*ctx.accounts = append(*ctx.accounts, acc)
	toSave := make([]model.MFAAccount, len(*ctx.accounts))
	copy(toSave, *ctx.accounts)
	ctx.accountsMu.Unlock()
	return storage.SaveMFAAccounts(toSave)
}

func (ctx *appContext) deleteAccount(name, secret string) error {
	ctx.accountsMu.Lock()
	accounts := *ctx.accounts
	for i, a := range accounts {
		if a.AccountName == name && a.Secret == secret {
			*ctx.accounts = append(accounts[:i], accounts[i+1:]...)
			break
		}
	}
	toSave := make([]model.MFAAccount, len(*ctx.accounts))
	copy(toSave, *ctx.accounts)
	ctx.accountsMu.Unlock()
	return storage.SaveMFAAccounts(toSave)
}

func (ctx *appContext) currentFilter() string {
	if ctx.searchEntry == nil {
		return ""
	}
	return ctx.searchEntry.Text
}

func (ctx *appContext) hasDuplicate(name, secret string) (bool, string) {
	for _, a := range ctx.snapshotAccounts() {
		if a.Secret == secret {
			return true, "该密钥已存在（账号: " + a.AccountName + "）"
		}
		if strings.EqualFold(a.AccountName, name) {
			return true, "账号名称已存在"
		}
	}
	return false, ""
}

func formatTOTPCode(code string, err error) string {
	if err != nil {
		return "Error"
	}
	if len(code) == 6 {
		return fmt.Sprintf("%s %s", code[:3], code[3:])
	}
	return code
}

func remainingSeconds(now time.Time) int {
	return int(30 - (now.Unix() % 30))
}

func SetupMainWindow(myWindow fyne.Window, initialAccounts []model.MFAAccount) {
	stopCh := make(chan struct{})
	var windowClosed atomic.Bool
	var forceCodeGen atomic.Bool
	forceCodeGen.Store(true)

	accounts := initialAccounts
	var accountsMu sync.RWMutex

	myWindow.SetOnClosed(func() {
		windowClosed.Store(true)
		close(stopCh)
	})

	headerBg := canvas.NewRectangle(theme.PrimaryBlue)
	headerBg.CornerRadius = 0

	title := canvas.NewText("虚拟MFA", color.White)
	title.TextSize = 26
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	subtitle := canvas.NewText("您的两步验证码", color.RGBA{R: 255, G: 255, B: 255, A: 200})
	subtitle.TextSize = 13
	subtitle.Alignment = fyne.TextAlignCenter

	titleContainer := container.NewVBox(title, subtitle)
	navBar := container.NewStack(headerBg, container.NewPadded(titleContainer))

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("搜索账号...")

	clearSearchBtn := widget.NewButtonWithIcon("", fyneTheme.ContentClearIcon(), func() {
		searchEntry.SetText("")
	})
	clearSearchBtn.Importance = widget.LowImportance
	clearSearchBtn.Hide()
	searchEntry.ActionItem = clearSearchBtn

	toastLabel := canvas.NewText("", color.White)
	toastLabel.TextSize = 13
	toastLabel.Alignment = fyne.TextAlignCenter
	toastLabel.Hide()

	toastBg := canvas.NewRectangle(color.RGBA{R: 33, G: 33, B: 33, A: 220})
	toastBg.CornerRadius = 8
	toastBg.Hide()

	toastBox := container.NewStack(toastBg, container.NewPadded(toastLabel))
	toastBox.Hide()

	var toastSeq atomic.Uint64
	showToast := func(msg string) {
		if windowClosed.Load() {
			return
		}
		seq := toastSeq.Add(1)
		toastLabel.Text = msg
		toastLabel.Show()
		toastBg.Show()
		toastBox.Show()
		toastBox.Refresh()

		time.AfterFunc(1200*time.Millisecond, func() {
			fyne.Do(func() {
				if windowClosed.Load() || toastSeq.Load() != seq {
					return
				}
				toastBox.Hide()
				toastLabel.Hide()
				toastBg.Hide()
			})
		})
	}

	var itemsMu sync.RWMutex
	var updateItems []updateItem

	listVBox := container.NewVBox()
	mfaTheme := theme.NewMFATheme()

	emptyHint := canvas.NewText("暂无账号，点击右上角「添加」开始使用", theme.TextSecondary)
	emptyHint.TextSize = 14
	emptyHint.Alignment = fyne.TextAlignCenter
	emptyHint.Hide()

	emptySearchHint := canvas.NewText("没有匹配的账号", theme.TextSecondary)
	emptySearchHint.TextSize = 14
	emptySearchHint.Alignment = fyne.TextAlignCenter
	emptySearchHint.Hide()

	listStack := container.NewStack(
		container.NewCenter(emptyHint),
		container.NewCenter(emptySearchHint),
		listVBox,
	)

	ctx := &appContext{
		window:       myWindow,
		accounts:     &accounts,
		accountsMu:   &accountsMu,
		searchEntry:  searchEntry,
		forceCodeGen: &forceCodeGen,
		showToast:    showToast,
	}

	var renderList func(filterText string)
	renderList = func(filterText string) {
		listVBox.Objects = nil

		accountsCopy := ctx.snapshotAccounts()
		matched := 0
		now := time.Now()
		remain := remainingSeconds(now)
		progressVal := float64(remain) / 30.0
		currentColor := theme.GetProgressColor(progressVal)
		mfaTheme.SetPrimaryColor(currentColor)

		itemsMu.Lock()
		updateItems = nil

		for _, acc := range accountsCopy {
			if !acc.MatchName(filterText) {
				continue
			}
			matched++

			normalized := acc.NormalizeSecret()
			code, err := totp.GenerateCode(normalized, now)
			codeStrBinding := binding.NewString()
			_ = codeStrBinding.Set(formatTOTPCode(code, err))

			codeLabel := widget.NewLabelWithData(codeStrBinding)
			codeLabel.Alignment = fyne.TextAlignCenter
			codeLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

			copyHint := canvas.NewText("点击验证码复制", theme.TextSecondary)
			copyHint.TextSize = 11
			copyHint.Alignment = fyne.TextAlignCenter

			copyBtn := widget.NewButton("", func() {
				val, err := codeStrBinding.Get()
				if err != nil || val == "--- ---" || val == "Error" {
					return
				}
				myWindow.Clipboard().SetContent(strings.ReplaceAll(val, " ", ""))
				showToast("已复制  " + val)
			})
			copyBtn.Importance = widget.LowImportance

			clickableCode := container.NewStack(
				copyBtn,
				container.NewPadded(container.NewVBox(codeLabel, copyHint)),
			)
			largeLabelContainer := container.NewThemeOverride(clickableCode, mfaTheme)

			progress := widget.NewProgressBar()
			progress.Min = 0
			progress.Max = 1
			progress.SetValue(progressVal)
			progress.TextFormatter = func() string { return "" }

			remainLabel := canvas.NewText(fmt.Sprintf("%ds", remain), theme.TextSecondary)
			remainLabel.TextSize = 12
			remainLabel.Alignment = fyne.TextAlignTrailing

			progressRow := container.NewBorder(nil, nil, nil, remainLabel,
				container.NewThemeOverride(progress, mfaTheme))

			currentAcc := acc
			deleteBtn := widget.NewButtonWithIcon("", fyneTheme.DeleteIcon(), func() {
				dialog.ShowConfirm("删除确认", "确定要删除账号「"+currentAcc.AccountName+"」吗？", func(b bool) {
					if !b {
						return
					}
					if err := ctx.deleteAccount(currentAcc.AccountName, currentAcc.Secret); err != nil {
						dialog.NewInformation("错误", "保存失败: "+err.Error(), myWindow).Show()
						return
					}
					showToast("已删除「" + currentAcc.AccountName + "」")
					renderList(ctx.currentFilter())
				}, myWindow)
			})
			deleteBtn.Importance = widget.LowImportance

			header := container.NewBorder(nil, nil, nil, deleteBtn,
				widget.NewLabelWithStyle(currentAcc.AccountName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

			contentBox := container.NewVBox(header, largeLabelContainer, progressRow)

			cardBg := canvas.NewRectangle(theme.CardBg)
			cardBg.CornerRadius = 12
			cardBg.Shadow = canvas.Shadow{
				Variant:    canvas.DropShadow,
				BlurRadius: 8,
				Offset:     fyne.Position{X: 0, Y: 2},
				Color:      color.RGBA{A: 30},
			}
			cardBg.SetMinSize(fyne.NewSize(380, 0))

			card := container.NewMax(cardBg, container.NewPadded(contentBox))
			listVBox.Add(container.NewPadded(card))

			updateItems = append(updateItems, updateItem{
				codeBinding: codeStrBinding,
				progress:    progress,
				remainLabel: remainLabel,
				secret:      normalized,
			})
		}
		itemsMu.Unlock()

		emptyHint.Hide()
		emptySearchHint.Hide()
		switch {
		case len(accountsCopy) == 0:
			emptyHint.Show()
		case matched == 0:
			emptySearchHint.Show()
		}

		forceCodeGen.Store(false)
		listVBox.Refresh()
		listStack.Refresh()
	}

	ctx.onChanged = renderList

	addBtn := widget.NewButtonWithIcon("添加", fyneTheme.ContentAddIcon(), func() {
		showAddAccountDialog(ctx)
	})
	addBtn.Importance = widget.HighImportance

	renderList("")

	searchEntry.OnChanged = func(s string) {
		if strings.TrimSpace(s) == "" {
			clearSearchBtn.Hide()
		} else {
			clearSearchBtn.Show()
		}
		renderList(s)
	}

	scrollList := container.NewVScroll(listStack)

	searchContainer := container.NewPadded(searchEntry)
	topBar := container.NewBorder(nil, nil, nil, addBtn, searchContainer)
	topBarContainer := container.NewPadded(topBar)

	toastBar := container.NewPadded(toastBox)

	mainContent := container.NewBorder(
		container.NewVBox(navBar, topBarContainer),
		toastBar,
		nil, nil,
		scrollList,
	)

	myWindow.SetContent(mainContent)

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		var lastPeriod int64 = -1
		var lastColor color.Color

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				now := time.Now()
				period := now.Unix() / 30
				remain := remainingSeconds(now)
				progressVal := float64(remain) / 30.0

				currentColor := theme.GetProgressColor(progressVal)
				colorChanged := currentColor != lastColor
				if colorChanged {
					mfaTheme.SetPrimaryColor(currentColor)
					lastColor = currentColor
				}

				needCodeUpdate := forceCodeGen.Swap(false) || period != lastPeriod
				lastPeriod = period
				remainText := fmt.Sprintf("%ds", remain)

				itemsMu.RLock()
				snapshot := make([]updateItem, len(updateItems))
				copy(snapshot, updateItems)
				itemsMu.RUnlock()

				type updateInfo struct {
					item       *updateItem
					val        string
					updateCode bool
					refreshBar bool
				}
				infos := make([]updateInfo, 0, len(snapshot))
				for i := range snapshot {
					item := &snapshot[i]
					info := updateInfo{item: item, refreshBar: colorChanged}
					if needCodeUpdate {
						code, err := totp.GenerateCode(item.secret, now)
						info.val = formatTOTPCode(code, err)
						info.updateCode = true
					}
					infos = append(infos, info)
				}

				fyne.Do(func() {
					if windowClosed.Load() {
						return
					}
					for _, info := range infos {
						if info.updateCode {
							_ = info.item.codeBinding.Set(info.val)
						}
						info.item.progress.SetValue(progressVal)
						if info.item.remainLabel != nil {
							info.item.remainLabel.Text = remainText
							// 低剩余时间时用警示色提示紧迫感
							if progressVal <= 0.2 {
								info.item.remainLabel.Color = theme.AlertRed
							} else {
								info.item.remainLabel.Color = theme.TextSecondary
							}
							info.item.remainLabel.Refresh()
						}
						if info.refreshBar {
							info.item.progress.Refresh()
						}
					}
				})
			}
		}
	}()
}
