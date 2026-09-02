package ui

import (
	"fmt"
	"image/color"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"

	"mfa_reader/internal/theme"
)

// liveRefresher 每秒驱动列表卡片刷新验证码、进度条与剩余秒数。
// 设计为可在 SetupMainWindow 之外独立启动，逻辑不与具体窗口布局耦合。
type liveRefresher struct {
	cardsMu  *sync.RWMutex
	cards    *[]*accountCard
	gen      *codeGen
	force    *atomic.Bool
	mfaTheme *theme.MFATheme

	lastPeriod int64
	lastColor  color.Color
}

func (r *liveRefresher) snapshot() []*accountCard {
	r.cardsMu.RLock()
	defer r.cardsMu.RUnlock()
	out := make([]*accountCard, len(*r.cards))
	copy(out, *r.cards)
	return out
}

func (r *liveRefresher) run(stop <-chan struct{}, windowClosed func() bool) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	r.lastPeriod = -1
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			r.tick(windowClosed)
		}
	}
}

func (r *liveRefresher) tick(windowClosed func() bool) {
	now := time.Now()
	period := now.Unix() / 30
	remain := remainingSeconds(now)
	progressVal := float64(remain) / 30.0

	colorChanged := false
	if currentColor := theme.GetProgressColor(progressVal); currentColor != r.lastColor {
		r.lastColor = currentColor
		r.mfaTheme.SetPrimaryColor(currentColor)
		colorChanged = true
	}

	needCode := r.force.Swap(false) || period != r.lastPeriod
	r.lastPeriod = period
	remainText := fmt.Sprintf("%ds", remain)

	type itemUpdate struct {
		card    *accountCard
		code    string
		newCode bool
	}

	cards := r.snapshot()
	updates := make([]itemUpdate, len(cards))
	for i, card := range cards {
		u := itemUpdate{card: card}
		if needCode {
			// HMAC 计算放在后台 goroutine，避免阻塞 UI。
			u.code = r.gen.Text(card.secret, now)
			u.newCode = true
		}
		updates[i] = u
	}

	fyne.Do(func() {
		if windowClosed() {
			return
		}
		for _, u := range updates {
			if u.newCode {
				u.card.SetCode(u.code)
			}
			u.card.setCountdown(progressVal, remainText, colorChanged)
		}
	})
}
