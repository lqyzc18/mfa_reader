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