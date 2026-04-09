package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/duke-git/lancet/v2/datetime"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/pquerna/otp/totp"
)

// 加载图标
func loadIcon() fyne.Resource {
	iconPath := "icon.ico"
	if fileutil.IsExist(iconPath) {
		r, err := fyne.LoadResourceFromPath(iconPath)
		if err == nil {
			return r
		}
	}
	return theme.FyneLogo()
}

type MFAAccount struct {
	Name      string
	AddedTime string
	Secret    string
}

func init() {
	// 强制应用亮色主题，看起来更整洁
	os.Setenv("FYNE_THEME", "light")

	fontPath := "C:\\Windows\\Fonts\\simhei.ttf"
	if fileutil.IsExist(fontPath) {
		os.Setenv("FYNE_FONT", fontPath)
	} else {
		altFont := "C:\\Windows\\Fonts\\msyh.ttf"
		if fileutil.IsExist(altFont) {
			os.Setenv("FYNE_FONT", altFont)
		}
	}
}

func loadMFAAccounts(myWindow fyne.Window) []MFAAccount {
	var accounts []MFAAccount
	filePath := "mfa.txt"

	if !fileutil.IsExist(filePath) {
		return accounts
	}

	content, err := fileutil.ReadFileToString(filePath)
	if err != nil {
		return accounts
	}

	lines := strutil.SplitAndTrim(content, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) >= 3 {
			accounts = append(accounts, MFAAccount{
				Name:      parts[0],
				AddedTime: parts[1],
				Secret:    parts[2],
			})
		} else if len(parts) == 2 {
			nowStr := datetime.FormatTimeToStr(time.Now(), "yyyy-mm-dd hh:mm:ss")
			accounts = append(accounts, MFAAccount{
				Name:      parts[0],
				AddedTime: nowStr,
				Secret:    parts[1],
			})
		}
	}
	return accounts
}

func main() {
	myApp := app.New()

	// 设置应用程序的图标
	myApp.SetIcon(loadIcon())

	myWindow := myApp.NewWindow("虚拟MFA")
	myWindow.Resize(fyne.NewSize(400, 700))

	accounts := loadMFAAccounts(myWindow)

	// 1. 顶部标题栏
	title := canvas.NewText("虚拟MFA", theme.ForegroundColor())
	title.TextSize = 18
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	navBar := container.NewCenter(title)

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("搜索")
	searchEntry.ActionItem = widget.NewIcon(theme.SearchIcon())
	searchContainer := container.NewPadded(searchEntry)

	type updateItem struct {
		codeBinding binding.String
		progress    *widget.ProgressBar
		secret      string
	}
	var updateItems []updateItem

	// 列表容器
	listVBox := container.NewVBox()

	// 重新渲染列表的方法，用于支持搜索过滤
	renderList := func(filterText string) {
		listVBox.Objects = nil
		updateItems = nil

		for _, acc := range accounts {
			// 如果 filterText 不为空且名字不包含 filterText，则跳过
			if filterText != "" && !strings.Contains(strings.ToLower(acc.Name), strings.ToLower(filterText)) {
				continue
			}

			codeStrBinding := binding.NewString()
			codeStrBinding.Set("--- ---")

			// 使用 canvas.Text 来实现超大字体，并使用主题主色调突出显示
			codeText := canvas.NewText("--- ---", theme.PrimaryColor())
			codeText.TextSize = 42
			codeText.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

			// 通过 DataListener 在 UI 线程安全地更新 canvas.Text
			codeStrBinding.AddListener(binding.NewDataListener(func() {
				val, _ := codeStrBinding.Get()
				codeText.Text = val
				codeText.Refresh()
			}))

			// 将透明按钮覆盖在上面来实现点击复制事件
			copyBtn := widget.NewButton("", func() {
				val, _ := codeStrBinding.Get()
				if val != "--- ---" && val != "Error" {
					// 复制时去掉中间的空格
					myWindow.Clipboard().SetContent(strings.ReplaceAll(val, " ", ""))

					// 弹出一个提示框 (Toast / Information)
					// 使用 Fyne 自带的 dialog 提示用户，并设置 1 秒后自动关闭
					infoDialog := dialog.NewInformation("已复制", "验证码 "+val+" 已复制到剪贴板！", myWindow)
					infoDialog.Show()

					go func() {
						time.Sleep(1 * time.Second)
						infoDialog.Hide()
					}()
				}
			})
			copyBtn.Importance = widget.LowImportance

			// 将透明按钮放在底层，大字体文本放在上层
			// 这样鼠标悬停时按钮的灰色反馈会作为背景层，不会遮挡文字
			clickableCode := container.NewStack(
				copyBtn,
				container.NewPadded(codeText), // 增加一点内边距，更有呼吸感
			)

			progress := widget.NewProgressBar()
			progress.TextFormatter = func() string { return "" }

			contentBox := container.NewVBox(
				clickableCode,
				progress,
			)

			// 使用 Fyne 自带的 Card 组件，自带阴影与圆角，看起来非常美观
			card := widget.NewCard(acc.Name, "添加时间: "+acc.AddedTime, contentBox)

			listVBox.Add(card)

			updateItems = append(updateItems, updateItem{
				codeBinding: codeStrBinding,
				progress:    progress,
				secret:      acc.Secret,
			})
		}
		listVBox.Refresh()
	}

	// 初始渲染
	renderList("")

	// 搜索框事件
	searchEntry.OnChanged = func(s string) {
		renderList(s)
	}

	scrollList := container.NewVScroll(listVBox)

	// 主布局
	mainContent := container.NewBorder(
		container.NewVBox(navBar, searchContainer),
		nil, nil, nil,
		scrollList,
	)

	myWindow.SetContent(mainContent)

	// 定时更新器
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			now := time.Now()
			remaining := 30 - (now.Unix() % 30)
			progressVal := float64(remaining) / 30.0

			for _, item := range updateItems {
				code, err := totp.GenerateCode(item.secret, now)
				if err != nil {
					item.codeBinding.Set("Error")
				} else {
					// 插入空格让其更像截图中分散的数字，同时更容易阅读 (例如 123 456)
					if len(code) == 6 {
						formattedCode := fmt.Sprintf("%s %s", code[:3], code[3:])
						item.codeBinding.Set(formattedCode)
					} else {
						item.codeBinding.Set(code)
					}
				}

				item.progress.SetValue(progressVal)
			}
		}
	}()

	myWindow.ShowAndRun()
}
