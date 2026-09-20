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

	"mfa_reader/internal/storage"
	"mfa_reader/internal/theme"
)

type appContext struct {
	app         fyne.App
	window      fyne.Window
	store       *storage.Store
	searchEntry *widget.Entry
	refreshNow  func()
	showToast   func(string)
}

func (ctx *appContext) currentFilter() string {
	if ctx.searchEntry == nil {
		return ""
	}
	return ctx.searchEntry.Text
}

func SetupMainWindow(a fyne.App, w fyne.Window, store *storage.Store) {
	restoreWindowSize(w, a.Preferences())
	attachDesktop(a, w)
	buildMainUI(a, w, store)
}

func buildMainUI(a fyne.App, myWindow fyne.Window, store *storage.Store) {
	stopCh := make(chan struct{})
	var windowClosed atomic.Bool

	myWindow.SetOnClosed(func() {
		saveWindowState(myWindow, a.Preferences())
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
		app:         a,
		window:      myWindow,
		store:       store,
		searchEntry: searchEntry,
		showToast:   showToast,
	}

	var cardsMu sync.RWMutex
	cards := []*accountCard{}

	var renderList func(filter string)
	renderList = func(filter string) {
		listVBox.Objects = nil

		accountsCopy := ctx.store.Snapshot()
		matched := 0
		now := time.Now()
		mfaTheme.SetPrimaryColor(theme.GetProgressColor(progressRatio(remainingSeconds(now))))

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
				canvas:   myWindow.Canvas(),
				onCopy: func(code string) {
					myWindow.Clipboard().SetContent(strings.ReplaceAll(code, " ", ""))
					showToast("已复制  " + code)
				},
				onDelete: func() {
					dialog.ShowConfirm("删除确认", "确定要删除账号「"+currentAcc.AccountName+"」吗？", func(b bool) {
						if !b {
							return
						}
						if err := ctx.store.Delete(currentAcc.AccountName, currentAcc.Secret); err != nil {
							dialog.NewInformation("错误", "保存失败: "+err.Error(), myWindow).Show()
							return
						}
						showToast("已删除「" + currentAcc.AccountName + "」")
						renderList(ctx.currentFilter())
					}, myWindow)
				},
				onEdit: func() {
					showEditAccountDialog(ctx, currentAcc)
				},
				onPin: func() {
					if err := ctx.store.TogglePin(currentAcc.AccountName, currentAcc.Secret); err != nil {
						dialog.NewInformation("错误", err.Error(), myWindow).Show()
						return
					}
					renderList(ctx.currentFilter())
				},
				onMoveUp: func() {
					if err := ctx.store.Move(currentAcc.AccountName, currentAcc.Secret, -1); err != nil {
						return
					}
					renderList(ctx.currentFilter())
				},
				onMoveDown: func() {
					if err := ctx.store.Move(currentAcc.AccountName, currentAcc.Secret, 1); err != nil {
						return
					}
					renderList(ctx.currentFilter())
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

		listVBox.Refresh()
		listStack.Refresh()
	}

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

	var menuBtn *widget.Button
	menuBtn = widget.NewButtonWithIcon("", fyneTheme.MenuIcon(), func() {
		m := fyne.NewMenu("",
			fyne.NewMenuItem("导入...", func() { showImportDialog(ctx) }),
			fyne.NewMenuItem("导出...", func() { showExportDialog(ctx) }),
			fyne.NewMenuItem("关于", func() {
				dialog.NewInformation("虚拟MFA", "全局快捷键 Ctrl+Alt+M 显示或隐藏窗口。\n关闭窗口会最小化到托盘，从托盘选「退出」才会真正退出。", myWindow).Show()
			}),
		)
		widget.ShowPopUpMenuAtRelativePosition(m, myWindow.Canvas(), fyne.NewPos(0, menuBtn.Size().Height), menuBtn)
	})

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
	rightBtns := container.NewHBox(addBtn)
	topBar := container.NewBorder(nil, nil, menuBtn, rightBtns, searchContainer)
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
		mfaTheme: mfaTheme,
	}
	go refresher.run(stopCh, func() bool { return windowClosed.Load() })
}
