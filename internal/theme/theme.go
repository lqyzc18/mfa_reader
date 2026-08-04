package theme

import (
	"image/color"
	"os"
	"strconv"
	"sync"

	"fyne.io/fyne/v2"
	th "fyne.io/fyne/v2/theme"
)

var defaultTextSize float32 = 24

func init() {
	if envSize := os.Getenv("MFA_TEXT_SIZE"); envSize != "" {
		if size, err := strconv.ParseFloat(envSize, 32); err == nil && size > 0 && size <= 100 {
			defaultTextSize = float32(size)
		}
	}
}

func LoadIcon() fyne.Resource {
	iconPath := "icon.png"
	if _, err := os.Stat(iconPath); err == nil {
		r, err := fyne.LoadResourceFromPath(iconPath)
		if err == nil {
			return r
		}
	}
	return th.FyneLogo()
}

var (
	PrimaryBlue   = color.RGBA{R: 66, G: 133, B: 244, A: 255}
	SuccessGreen  = color.RGBA{R: 76, G: 175, B: 80, A: 255}
	WarningYellow = color.RGBA{R: 255, G: 193, B: 7, A: 255}
	AlertRed      = color.RGBA{R: 244, G: 67, B: 54, A: 255}
	Background    = color.RGBA{R: 250, G: 250, B: 250, A: 255}
	CardBg        = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	TextPrimary   = color.RGBA{R: 33, G: 33, B: 33, A: 255}
	TextSecondary = color.RGBA{R: 117, G: 117, B: 117, A: 255}
)

// MFATheme 自定义主题，用于放大字体和动态改变进度条颜色
type MFATheme struct {
	fyne.Theme
	primaryColor color.Color
	lock         sync.RWMutex
}

func NewMFATheme() *MFATheme {
	return &MFATheme{Theme: th.DefaultTheme()}
}

func (m *MFATheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if name == th.ColorNamePrimary && m.primaryColor != nil {
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
	switch name {
	case th.SizeNameText:
		return defaultTextSize
	case th.SizeNameCardRadius:
		return 12
	case th.SizeNameButtonRadius:
		return 8
	case th.SizeNameDialogRadius:
		return 16
	}
	return m.Theme.Size(name)
}

// GetProgressColor 根据剩余时间比例返回渐变色
func GetProgressColor(progress float64) color.Color {
	if progress > 0.6 {
		return SuccessGreen
	}
	if progress > 0.2 {
		return WarningYellow
	}
	return AlertRed
}
