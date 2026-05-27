package theme

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2/theme"
)

func TestGetProgressColor(t *testing.T) {
	tests := []struct {
		name     string
		progress float64
		expected color.Color
	}{
		{
			name:     "高进度-绿色",
			progress: 0.8,
			expected: SuccessGreen,
		},
		{
			name:     "中等进度-黄色",
			progress: 0.4,
			expected: WarningYellow,
		},
		{
			name:     "低进度-红色",
			progress: 0.1,
			expected: AlertRed,
		},
		{
			name:     "边界值60%-绿色",
			progress: 0.61,
			expected: SuccessGreen,
		},
		{
			name:     "边界值60%-黄色",
			progress: 0.6,
			expected: WarningYellow,
		},
		{
			name:     "边界值20%-黄色",
			progress: 0.21,
			expected: WarningYellow,
		},
		{
			name:     "边界值20%-红色",
			progress: 0.2,
			expected: AlertRed,
		},
		{
			name:     "零进度-红色",
			progress: 0.0,
			expected: AlertRed,
		},
		{
			name:     "满进度-绿色",
			progress: 1.0,
			expected: SuccessGreen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProgressColor(tt.progress)
			if result != tt.expected {
				t.Errorf("GetProgressColor(%v) = %v, want %v", tt.progress, result, tt.expected)
			}
		})
	}
}

func TestMFATheme_SetPrimaryColor(t *testing.T) {
	mfaTheme := NewMFATheme()

	testColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	mfaTheme.SetPrimaryColor(testColor)

	mfaTheme.lock.RLock()
	if mfaTheme.primaryColor != testColor {
		t.Errorf("SetPrimaryColor() did not set color correctly")
	}
	mfaTheme.lock.RUnlock()
}

func TestMFATheme_Color_PrimaryColor(t *testing.T) {
	mfaTheme := NewMFATheme()

	testColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	mfaTheme.SetPrimaryColor(testColor)

	result := mfaTheme.Color(theme.ColorNamePrimary, theme.VariantLight)
	if result != testColor {
		t.Errorf("Color() for primary = %v, want %v", result, testColor)
	}
}

func TestDefaultTextSize(t *testing.T) {
	if defaultTextSize <= 0 || defaultTextSize > 100 {
		t.Errorf("defaultTextSize = %v, want between 0 and 100", defaultTextSize)
	}
}
