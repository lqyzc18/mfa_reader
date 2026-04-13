package theme

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	fyneTheme "fyne.io/fyne/v2/theme"
	"github.com/duke-git/lancet/v2/fileutil"
)

// 加载图标
func LoadIcon() fyne.Resource {
	iconPath := "icon.png"
	if fileutil.IsExist(iconPath) {
		r, err := fyne.LoadResourceFromPath(iconPath)
		if err == nil {
			return r
		}
	}
	return fyneTheme.FyneLogo()
}

// 现代化配色方案
var (
	PrimaryBlue   = color.RGBA{R: 66, G: 133, B: 244, A: 255}  // 现代化蓝色
	PrimaryPurple = color.RGBA{R: 156, G: 39, B: 176, A: 255}  // 优雅紫色
	SuccessGreen  = color.RGBA{R: 76, G: 175, B: 80, A: 255}   // 成功绿
	WarningYellow = color.RGBA{R: 255, G: 193, B: 7, A: 255}   // 警告黄
	AlertRed      = color.RGBA{R: 244, G: 67, B: 54, A: 255}   // 警报红
	Background    = color.RGBA{R: 250, G: 250, B: 250, A: 255} // 浅灰背景
	CardBg        = color.RGBA{R: 255, G: 255, B: 255, A: 255} // 卡片白
	TextPrimary   = color.RGBA{R: 33, G: 33, B: 33, A: 255}    // 主文字
	TextSecondary = color.RGBA{R: 117, G: 117, B: 117, A: 255} // 次要文字
)

// MFATheme 自定义主题，用于放大字体和动态改变进度条颜色
type MFATheme struct {
	fyne.Theme
	primaryColor color.Color
	lock         sync.RWMutex
}

func NewMFATheme() *MFATheme {
	return &MFATheme{Theme: fyneTheme.DefaultTheme()}
}

func (m *MFATheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if name == fyneTheme.ColorNamePrimary && m.primaryColor != nil {
		return m.primaryColor
	}
	return m.Theme.Color(name, variant)
}

func (m *MFATheme) SetPrimaryColor(c color.Color) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.primaryColor = c
}

func (m *MFATheme) Size(name fyne.ThemeSizeName) float32 {
	if name == fyneTheme.SizeNameText {
		return 32
	}
	return m.Theme.Size(name)
}

// GetProgressColor 根据剩余时间比例返回渐变色
func GetProgressColor(progress float64) color.Color {
	if progress > 0.6 {
		return SuccessGreen
	} else if progress > 0.2 {
		return WarningYellow
	}
	return AlertRed
}
