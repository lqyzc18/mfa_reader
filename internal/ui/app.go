package ui

import (
	"fmt"
	"image/color"
	"log"
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
	secret      string
}

type appContext struct {
	window     fyne.Window
	accounts   *[]model.MFAAccount
	accountsMu *sync.RWMutex
	onChanged  func(string)
}

func SetupMainWindow(myWindow fyne.Window, initialAccounts []model.MFAAccount) {
	stopCh := make(chan struct{})
	var windowClosed atomic.Bool

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
	searchEntry.ActionItem = widget.NewIcon(fyneTheme.SearchIcon())
	searchEntry.TextStyle = fyne.TextStyle{Monospace: false}

	var itemsMu sync.RWMutex
	var updateItems []updateItem

	listVBox := container.NewVBox()
	mfaTheme := theme.NewMFATheme()

	ctx := &appContext{
		window:     myWindow,
		accounts:   &accounts,
		accountsMu: &accountsMu,
	}

	var renderList func(filterText string)
	renderList = func(filterText string) {
		listVBox.Objects = nil

		accountsMu.RLock()
		accountsCopy := make([]model.MFAAccount, len(accounts))
		copy(accountsCopy, accounts)
		accountsMu.RUnlock()

		itemsMu.Lock()
		updateItems = nil

		for _, acc := range accountsCopy {
			if filterText != "" && !strings.Contains(acc.AccountName, filterText) {
				continue
			}

			codeStrBinding := binding.NewString()
			codeStrBinding.Set("--- ---")

			codeLabel := widget.NewLabelWithData(codeStrBinding)
			codeLabel.Alignment = fyne.TextAlignCenter
			codeLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

			copyBtn := widget.NewButton("", func() {
				val, _ := codeStrBinding.Get()
				if val != "--- ---" && val != "Error" {
					myWindow.Clipboard().SetContent(strings.ReplaceAll(val, " ", ""))

					infoDialog := dialog.NewInformation("✓ 已复制", "验证码 "+val+" 已复制到剪贴板！", myWindow)
					infoDialog.Show()

					time.AfterFunc(800*time.Millisecond, func() {
						fyne.Do(func() {
							if !windowClosed.Load() {
								infoDialog.Hide()
							}
						})
					})
				}
			})
			copyBtn.Importance = widget.LowImportance

			clickableCode := container.NewStack(
				copyBtn,
				container.NewPadded(codeLabel),
			)
			largeLabelContainer := container.NewThemeOverride(clickableCode, mfaTheme)

			progress := widget.NewProgressBar()
			progress.TextFormatter = func() string { return "" }
			progressContainer := container.NewThemeOverride(progress, mfaTheme)

			currentAcc := acc
			deleteBtn := widget.NewButtonWithIcon("", fyneTheme.DeleteIcon(), func() {
				dialog.ShowConfirm("⚠ 删除确认", "确定要删除账号 "+currentAcc.AccountName+" 吗？", func(b bool) {
					if b {
						accountsMu.Lock()
						for i, a := range accounts {
							if a.AccountName == currentAcc.AccountName && a.Secret == currentAcc.Secret {
								accounts = append(accounts[:i], accounts[i+1:]...)
								break
							}
						}
						accountsMu.Unlock()

						accountsMu.RLock()
						accountsToSave := make([]model.MFAAccount, len(accounts))
						copy(accountsToSave, accounts)
						accountsMu.RUnlock()

						if err := storage.SaveMFAAccounts(accountsToSave); err != nil {
							log.Printf("[ui] 保存失败: %v", err)
						}
						renderList("")
					}
				}, myWindow)
			})
			deleteBtn.Importance = widget.LowImportance

			header := container.NewBorder(nil, nil, nil, deleteBtn,
				widget.NewLabelWithStyle(currentAcc.AccountName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

			contentBox := container.NewVBox(header, largeLabelContainer, progressContainer)

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
				secret:      currentAcc.NormalizeSecret(),
			})
		}
		itemsMu.Unlock()

		listVBox.Refresh()
	}

	ctx.onChanged = renderList

	addBtn := widget.NewButtonWithIcon("添加", fyneTheme.ContentAddIcon(), func() {
		showAddAccountDialog(ctx)
	})
	addBtn.Importance = widget.HighImportance
	addBtn.Resize(fyne.NewSize(80, 36))

	renderList("")

	searchEntry.OnChanged = func(s string) {
		renderList(s)
	}

	scrollList := container.NewVScroll(listVBox)

	searchContainer := container.NewPadded(searchEntry)
	searchContainer.Resize(fyne.NewSize(280, 40))

	topBar := container.NewBorder(nil, nil, nil, addBtn, searchContainer)
	topBarContainer := container.NewPadded(topBar)

	mainContent := container.NewBorder(
		container.NewVBox(navBar, topBarContainer),
		nil, nil, nil,
		scrollList,
	)

	myWindow.SetContent(mainContent)

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				now := time.Now()
				remaining := 30 - (now.Unix() % 30)
				progressVal := float64(remaining) / 30.0

				currentColor := theme.GetProgressColor(progressVal)
				mfaTheme.SetPrimaryColor(currentColor)

				itemsMu.RLock()
				snapshot := make([]updateItem, len(updateItems))
				copy(snapshot, updateItems)
				itemsMu.RUnlock()

				type updateInfo struct {
					item *updateItem
					val  string
				}
				infos := make([]updateInfo, 0, len(snapshot))
				for i := range snapshot {
					item := &snapshot[i]
					code, err := totp.GenerateCode(item.secret, now)
					val := "Error"
					if err == nil {
						if len(code) == 6 {
							val = fmt.Sprintf("%s %s", code[:3], code[3:])
						} else {
							val = code
						}
					}
					infos = append(infos, updateInfo{item: item, val: val})
				}

				fyne.Do(func() {
					if windowClosed.Load() {
						return
					}
					for _, info := range infos {
						info.item.codeBinding.Set(info.val)
						info.item.progress.SetValue(progressVal)
					}
					listVBox.Refresh()
				})
			}
		}
	}()
}
