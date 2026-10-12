package ui

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
)

// liveRefresher 每秒驱动列表卡片刷新验证码、进度条与剩余秒数。
// 每张卡片的周期长度可能不同（30 秒 / 60 秒），因此逐卡判断是否需要重新出码。
type liveRefresher struct {
	cardsMu *sync.RWMutex
	cards   *[]*accountCard
	gen     *codeGen
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

// cardUpdate 描述一张卡片本秒需要应用的变更：由刷新 goroutine 计算，UI 线程落地。
type cardUpdate struct {
	card    *accountCard
	code    string
	newCode bool
	ratio   float64
	remain  int
}

func (r *liveRefresher) tick(windowClosed func() bool) {
	if windowClosed() {
		return
	}

	now := time.Now()
	cards := r.snapshot()
	updates := make([]cardUpdate, 0, len(cards))

	for _, card := range cards {
		remain := remainingSeconds(now, card.period)
		u := cardUpdate{
			card:   card,
			ratio:  progressRatio(remain, card.period),
			remain: remain,
		}
		// 卡片进入新周期时重新出码；HMAC 计算在后台 goroutine 完成，避免阻塞 UI。
		if index := periodOf(now, card.period); index != card.lastIndex {
			u.code = r.gen.Text(card.req, now)
			u.newCode = true
			card.lastIndex = index
		}
		updates = append(updates, u)
	}

	fyne.Do(func() {
		if windowClosed() {
			return
		}
		for _, u := range updates {
			u.card.apply(u.code, u.newCode, u.ratio, u.remain)
		}
	})
}
