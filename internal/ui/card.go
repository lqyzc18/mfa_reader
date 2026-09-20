package ui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"mfa_reader/internal/model"
	"mfa_reader/internal/theme"
)

const cardWidth float32 = 380

// accountCard 保存单条账号的显示状态，由 liveRefresher 每秒驱动刷新。
// 除 setCountdown/SetCode 外，其余字段在 UI 线程构建完成后只读。
type accountCard struct {
	secret      string
	codeLabel   *widget.Label
	codeText    string
	progress    *widget.ProgressBar
	remainLabel *canvas.Text
}

type accountCardOptions struct {
	account    model.MFAAccount
	gen        *codeGen
	now        time.Time
	mfaTheme   *theme.MFATheme
	canvas     fyne.Canvas
	onCopy     func(code string)
	onDelete   func()
	onEdit     func()
	onPin      func()
	onMoveUp   func()
	onMoveDown func()
}

// newAccountCard 构建一张账号卡片并返回可放入列表的根容器。
func newAccountCard(o accountCardOptions) (*accountCard, fyne.CanvasObject) {
	now := o.now
	remain := remainingSeconds(now)
	progressVal := progressRatio(remain)
	remainText := fmt.Sprintf("%ds", remain)

	normalized := o.account.NormalizeSecret()
	card := &accountCard{
		secret:   normalized,
		codeText: o.gen.Text(normalized, now),
	}

	card.codeLabel = widget.NewLabel(card.codeText)
	card.codeLabel.Alignment = fyne.TextAlignCenter
	card.codeLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

	copyHint := canvas.NewText("点击验证码复制", theme.TextSecondary)
	copyHint.TextSize = 11
	copyHint.Alignment = fyne.TextAlignCenter

	copyBtn := widget.NewButton("", func() {
		if card.codeText != "Error" {
			o.onCopy(card.codeText)
		}
	})
	copyBtn.Importance = widget.LowImportance

	clickableCode := container.NewStack(
		copyBtn,
		container.NewPadded(container.NewVBox(card.codeLabel, copyHint)),
	)
	largeLabel := container.NewThemeOverride(clickableCode, o.mfaTheme)

	card.progress = widget.NewProgressBar()
	card.progress.Min = 0
	card.progress.Max = 1
	card.progress.SetValue(progressVal)
	card.progress.TextFormatter = func() string { return "" }

	card.remainLabel = canvas.NewText(remainText, theme.RemainTextColor(progressVal))
	card.remainLabel.TextSize = 12
	card.remainLabel.Alignment = fyne.TextAlignTrailing

	progressRow := container.NewBorder(nil, nil, nil, card.remainLabel,
		container.NewThemeOverride(card.progress, o.mfaTheme))

	titleText := o.account.AccountName
	if o.account.Pinned {
		titleText = "★ " + titleText
	}

	var more *widget.Button
	more = widget.NewButtonWithIcon("", fyneTheme.MoreVerticalIcon(), func() {
		pinLabel := "置顶"
		if o.account.Pinned {
			pinLabel = "取消置顶"
		}
		m := fyne.NewMenu("",
			fyne.NewMenuItem(pinLabel, o.onPin),
			fyne.NewMenuItem("上移", o.onMoveUp),
			fyne.NewMenuItem("下移", o.onMoveDown),
			fyne.NewMenuItem("编辑", o.onEdit),
			fyne.NewMenuItem("删除", o.onDelete),
		)
		c := o.canvas
		if c == nil {
			c = fyne.CurrentApp().Driver().CanvasForObject(more)
		}
		if c != nil {
			widget.ShowPopUpMenuAtRelativePosition(m, c, fyne.NewPos(0, more.Size().Height), more)
		}
	})
	more.Importance = widget.LowImportance

	header := container.NewBorder(nil, nil, nil, more,
		widget.NewLabelWithStyle(titleText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

	content := container.NewVBox(header, largeLabel, progressRow)

	bg := canvas.NewRectangle(theme.CardBg)
	bg.CornerRadius = 12
	bg.Shadow = canvas.Shadow{
		Variant:    canvas.DropShadow,
		BlurRadius: 8,
		Offset:     fyne.Position{X: 0, Y: 2},
		Color:      color.RGBA{A: 30},
	}
	bg.SetMinSize(fyne.NewSize(cardWidth, 0))

	root := container.NewPadded(container.NewStack(bg, container.NewPadded(content)))
	return card, root
}

// SetCode 更新验证码显示（UI 线程调用）。
func (c *accountCard) SetCode(text string) {
	if c.codeText == text {
		return
	}
	c.codeText = text
	c.codeLabel.SetText(text)
}

// setCountdown 更新进度条与剩余秒数显示（UI 线程调用）。
// colorChanged 表示进度条主题色已变化，需要强制重绘。
func (c *accountCard) setCountdown(value float64, remain string, colorChanged bool) {
	c.progress.SetValue(value)

	if c.remainLabel != nil {
		if c.remainLabel.Text != remain {
			c.remainLabel.Text = remain
		}
		if color := theme.RemainTextColor(value); color != c.remainLabel.Color {
			c.remainLabel.Color = color
		}
		c.remainLabel.Refresh()
	}
	if colorChanged {
		c.progress.Refresh()
	}
}
