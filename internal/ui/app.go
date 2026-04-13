package ui

import (
	"fmt"
	"image/color"
	"strings"
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

func SetupMainWindow(myWindow fyne.Window, accounts []model.MFAAccount) {
	// 停止 channel
	stopCh := make(chan struct{})

	// 窗口关闭标志
	var windowClosed atomic.Bool
	windowClosed.Store(false)

	// 监听窗口关闭事件
	myWindow.SetOnClosed(func() {
		windowClosed.Store(true)
		close(stopCh)
	})

	// 1. 顶部标题栏 - 现代化设计
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
	searchEntry.Validator = nil
	searchEntry.TextStyle = fyne.TextStyle{Monospace: false}

	type updateItem struct {
		codeBinding binding.String
		progress    *widget.ProgressBar
		secret      string
	}
	var updateItems []updateItem

	// 列表容器
	listVBox := container.NewVBox()

	// 全局主题实例，用于动态控制所有卡片内的颜色和字体大小
	mfaTheme := theme.NewMFATheme()

	// 重新渲染列表的方法，用于支持搜索过滤
	var renderList func(filterText string)
	renderList = func(filterText string) {
		listVBox.Objects = nil
		updateItems = nil

		for _, acc := range accounts {
			// 如果 filterText 不为空且名字不包含 filterText，则跳过
			if filterText != "" && !strings.Contains(acc.AccountName, filterText) {
				continue
			}

			codeStrBinding := binding.NewString()
			codeStrBinding.Set("--- ---")

			// 使用 widget.Label 代替 canvas.Text 以获得更好的线程安全性支持
			codeLabel := widget.NewLabelWithData(codeStrBinding)
			codeLabel.Alignment = fyne.TextAlignCenter
			codeLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

			// 将透明按钮覆盖在上面来实现点击复制事件
			copyBtn := widget.NewButton("", func() {
				val, _ := codeStrBinding.Get()
				if val != "--- ---" && val != "Error" {
					myWindow.Clipboard().SetContent(strings.ReplaceAll(val, " ", ""))

					// 现代化提示框
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

			// 应用局部主题来放大验证码文字并支持动态主色调
			largeLabelContainer := container.NewThemeOverride(clickableCode, mfaTheme)

			progress := widget.NewProgressBar()
			progress.TextFormatter = func() string { return "" }

			// 进度条也需要应用局部主题以支持动态颜色
			progressContainer := container.NewThemeOverride(progress, mfaTheme)

			// 删除按钮 - 现代化设计
			currentAcc := acc
			deleteBtn := widget.NewButtonWithIcon("", fyneTheme.DeleteIcon(), func() {
				dialog.ShowConfirm("⚠ 删除确认", "确定要删除账号 "+currentAcc.AccountName+" 吗？", func(b bool) {
					if b {
						// 找到当前账号的索引并删除
						for i, acc := range accounts {
							if acc.AccountName == currentAcc.AccountName && acc.Secret == currentAcc.Secret {
								accounts = append(accounts[:i], accounts[i+1:]...)
								break
							}
						}
						storage.SaveMFAAccounts(accounts)
						// 重新渲染列表
						renderList("")
					}
				}, myWindow)
			})
			deleteBtn.Importance = widget.LowImportance

			// 将删除按钮放在右侧
			header := container.NewBorder(nil, nil, nil, deleteBtn,
				widget.NewLabelWithStyle(currentAcc.AccountName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

			contentBox := container.NewVBox(
				header,
				largeLabelContainer,
				progressContainer,
			)

			// 现代化卡片设计 - 使用圆角矩形
			cardBg := canvas.NewRectangle(theme.CardBg)
			cardBg.CornerRadius = 12
			cardBg.SetMinSize(fyne.NewSize(380, 0))

			card := container.NewMax(cardBg, container.NewPadded(contentBox))
			listVBox.Add(container.NewPadded(card))

			secretStr := strings.ToUpper(strings.TrimSpace(currentAcc.Secret))
			secretStr = strings.ReplaceAll(secretStr, " ", "")
			secretStr = strings.ReplaceAll(secretStr, "-", "")
			secretStr = strings.TrimRight(secretStr, "=")
			updateItems = append(updateItems, updateItem{
				codeBinding: codeStrBinding,
				progress:    progress,
				secret:      secretStr,
			})
		}
		listVBox.Refresh()
	}

	addBtn := widget.NewButtonWithIcon("添加", fyneTheme.ContentAddIcon(), func() {
		showAddAccountDialog(myWindow, &accounts, renderList)
	})
	addBtn.Importance = widget.HighImportance
	addBtn.Resize(fyne.NewSize(80, 36))

	// 初始渲染
	renderList("")

	// 搜索框事件
	searchEntry.OnChanged = func(s string) {
		renderList(s)
	}

	scrollList := container.NewVScroll(listVBox)

	// 主布局 - 现代化设计
	searchContainer := container.NewPadded(searchEntry)
	searchContainer.Resize(fyne.NewSize(280, 40))

	topBar := container.NewBorder(
		nil, nil,
		nil,
		addBtn,
		searchContainer,
	)

	topBarContainer := container.NewPadded(topBar)

	mainContent := container.NewBorder(
		container.NewVBox(navBar, topBarContainer),
		nil, nil, nil,
		scrollList,
	)

	myWindow.SetContent(mainContent)

	// 定时更新器
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			// 使用非阻塞select监听stopCh，防止channel关闭后的panic
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				// 在窗口已关闭时跳过更新
				if windowClosed.Load() {
					return
				}

				now := time.Now()
				remaining := 30 - (now.Unix() % 30)
				progressVal := float64(remaining) / 30.0

				// 根据进度比例决定颜色 - 使用主题颜色方案
				currentColor := theme.GetProgressColor(progressVal)

				// 更新全局主题颜色，使用互斥锁确保线程安全
				mfaTheme.SetPrimaryColor(currentColor)

				// 在后台计算所有验证码
				type updateInfo struct {
					item *updateItem
					val  string
				}
				var infos []updateInfo

				for i := range updateItems {
					item := &updateItems[i]
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

				// 使用 fyne.Do() 统一在主 UI 线程中执行所有更新操作
				// 先检查窗口是否已关闭
				if windowClosed.Load() {
					return
				}
				fyne.Do(func() {
					// 再次检查窗口状态，防止在fyne.Do执行时窗口已关闭
					if windowClosed.Load() {
						return
					}
					// 更新所有项的数据和进度条
					for _, info := range infos {
						info.item.codeBinding.Set(info.val)
						info.item.progress.SetValue(progressVal)
					}
					// 刷新整个列表以应用新的主题颜色
					listVBox.Refresh()
				})
			}
		}
	}()
}
