package ui

import (
	"image/color"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"mfa_reader/internal/model"
	"mfa_reader/internal/storage"
	"mfa_reader/internal/theme"
)

type appContext struct {
	window       fyne.Window
	accounts     *[]model.MFAAccount
	accountsMu   *sync.RWMutex
	searchEntry  *widget.Entry
	refreshNow   func()
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

	listVBox := container.NewVBox()
	mfaTheme := theme.NewMFATheme()
	gen := newCodeGen()

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

	var cardsMu sync.RWMutex
	cards := []*accountCard{}

	// renderList 是唯一重建列表的地方；验证码经 codeGen 缓存，同周期重建不重复计算。
	var renderList func(filter string)
	renderList = func(filter string) {
		listVBox.Objects = nil

		accountsCopy := ctx.snapshotAccounts()
		matched := 0
		now := time.Now()
		mfaTheme.SetPrimaryColor(theme.GetProgressColor(float64(remainingSeconds(now)) / 30.0))

		cardsMu.Lock()
		cards = cards[:0]
		for _, acc := range accountsCopy {
			if !acc.MatchName(filter) {
				continue
			}
			matched++

			currentAcc := acc
			card, root := newAccountCard(accountCardOptions{
				account:  currentAcc,
				gen:      gen,
				now:      now,
				mfaTheme: mfaTheme,
				onCopy: func(code string) {
					myWindow.Clipboard().SetContent(strings.ReplaceAll(code, " ", ""))
					showToast("已复制  " + code)
				},
				onDelete: func() {
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
				},
			})
			listVBox.Add(root)
			cards = append(cards, card)
		}
		cardsMu.Unlock()

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

	// 搜索输入防抖：按键抖动期间只触发最后一次渲染。
	// 后台 goroutine 通过 fyne.Do 切回 UI 线程再调用 renderList。
	searchDebouncer := newDebouncer(150*time.Millisecond, func(s string) {
		fyne.Do(func() {
			if windowClosed.Load() {
				return
			}
			renderList(s)
		})
	})
	ctx.refreshNow = func() {
		searchDebouncer.cancel()
		renderList(ctx.currentFilter())
	}

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
		searchDebouncer.schedule(s)
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

	refresher := &liveRefresher{
		cardsMu:  &cardsMu,
		cards:    &cards,
		gen:      gen,
		force:    &forceCodeGen,
		mfaTheme: mfaTheme,
	}
	go refresher.run(stopCh, func() bool { return windowClosed.Load() })
}
