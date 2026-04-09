package main

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"sync"
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

// MFATheme 自定义主题，用于放大字体和动态改变进度条颜色
type MFATheme struct {
	fyne.Theme
	primaryColor color.Color
	lock         sync.RWMutex
}

func (m *MFATheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if name == theme.ColorNamePrimary && m.primaryColor != nil {
		return m.primaryColor
	}
	return m.Theme.Color(name, variant)
}

func (m *MFATheme) setPrimaryColor(c color.Color) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.primaryColor = c
}

func (m *MFATheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return 32
	}
	return m.Theme.Size(name)
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

	// 全局主题实例，用于动态控制所有卡片内的颜色和字体大小
	mfaTheme := &MFATheme{Theme: theme.DefaultTheme()}

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

			// 使用 widget.Label 代替 canvas.Text 以获得更好的线程安全性支持
			codeLabel := widget.NewLabelWithData(codeStrBinding)
			codeLabel.Alignment = fyne.TextAlignCenter
			codeLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

			// 将透明按钮覆盖在上面来实现点击复制事件
			copyBtn := widget.NewButton("", func() {
				val, _ := codeStrBinding.Get()
				if val != "--- ---" && val != "Error" {
					myWindow.Clipboard().SetContent(strings.ReplaceAll(val, " ", ""))

					infoDialog := dialog.NewInformation("已复制", "验证码 "+val+" 已复制到剪贴板！", myWindow)
					infoDialog.Show()

					time.AfterFunc(1*time.Second, func() {
						infoDialog.Hide()
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

			contentBox := container.NewVBox(
				largeLabelContainer,
				progressContainer,
			)

			// 使用 Fyne 自带的 Card 组件
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

			// 根据进度比例决定颜色
			// > 60%: 绿色, > 40%: 黄色, <= 40%: 红色
			var currentColor color.Color
			if progressVal > 0.6 {
				currentColor = color.RGBA{R: 76, G: 175, B: 80, A: 255} // Material Green
			} else if progressVal > 0.2 {
				currentColor = color.RGBA{R: 255, G: 193, B: 7, A: 255} // Material Amber/Yellow
			} else {
				currentColor = color.RGBA{R: 244, G: 67, B: 54, A: 255} // Material Red
			}

			// 更新全局主题颜色，使用互斥锁确保线程安全
			mfaTheme.setPrimaryColor(currentColor)

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
			fyne.Do(func() {
				// 更新所有项的数据和进度条
				for _, info := range infos {
					info.item.codeBinding.Set(info.val)
					info.item.progress.SetValue(progressVal)
				}
				// 刷新整个列表以应用新的主题颜色
				listVBox.Refresh()
			})
		}
	}()

	myWindow.ShowAndRun()
}
