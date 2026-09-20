package ui

import (
	"fmt"
	"image/color"
	"sync"
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

// run 在每个整秒唤醒一次。使用自对齐的 Timer 而非固定 Ticker：
// Ticker 从启动时刻起算，与墙钟秒边界有固定偏移，会让 TOTP 周期切换被延迟至多 1 秒，
// 期间界面仍显示上一周期的验证码与错误的倒计时。
func (r *liveRefresher) run(stop <-chan struct{}, windowClosed func() bool) {
	timer := time.NewTimer(untilNextSecond(time.Now()))
	defer timer.Stop()

	r.lastPeriod = -1
	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			r.tick(windowClosed)
			timer.Reset(untilNextSecond(time.Now()))
		}
	}
}

// untilNextSecond 返回距离下一个整秒的时长，用于把唤醒点对齐到墙钟秒边界。
func untilNextSecond(now time.Time) time.Duration {
	return time.Second - time.Duration(now.Nanosecond())*time.Nanosecond
}

func (r *liveRefresher) tick(windowClosed func() bool) {
	now := time.Now()
	period := periodOf(now)
	remain := remainingSeconds(now)
	progressVal := progressRatio(remain)

	colorChanged := false
	if currentColor := theme.GetProgressColor(progressVal); currentColor != r.lastColor {
		r.lastColor = currentColor
		r.mfaTheme.SetPrimaryColor(currentColor)
		colorChanged = true
	}

	// lastPeriod 初始为 -1，首次 tick 必然重新生成一遍验证码。
	needCode := period != r.lastPeriod
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
