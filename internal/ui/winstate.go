package ui

import (
	"fyne.io/fyne/v2"

	"mfa_reader/internal/winpos"
)

const (
	windowTitle = "虚拟MFA"
	prefWidth   = "window.width"
	prefHeight  = "window.height"
	prefX       = "window.x"
	prefY       = "window.y"
	prefHasPos  = "window.hasPos"
)

func restoreWindowSize(w fyne.Window, p fyne.Preferences) {
	width := float32(p.FloatWithFallback(prefWidth, 400))
	height := float32(p.FloatWithFallback(prefHeight, 700))
	if width < 320 {
		width = 320
	}
	if height < 400 {
		height = 400
	}
	w.Resize(fyne.NewSize(width, height))
}

func restoreWindowPos(p fyne.Preferences) {
	if !p.Bool(prefHasPos) {
		return
	}
	x := int(p.Float(prefX))
	y := int(p.Float(prefY))
	if x < -100 || y < -100 || x > 20000 || y > 20000 {
		return
	}
	winpos.Set(windowTitle, x, y)
}

func saveWindowState(w fyne.Window, p fyne.Preferences) {
	if w == nil || p == nil {
		return
	}
	sz := w.Canvas().Size()
	if sz.Width >= 320 && sz.Height >= 400 {
		p.SetFloat(prefWidth, float64(sz.Width))
		p.SetFloat(prefHeight, float64(sz.Height))
	}
	if x, y, ok := winpos.Get(windowTitle); ok {
		p.SetFloat(prefX, float64(x))
		p.SetFloat(prefY, float64(y))
		p.SetBool(prefHasPos, true)
	}
}
