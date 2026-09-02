package ui

import (
	"sync"
	"time"
)

// debouncer 将高频调用合并为最后一次触发后的单次执行。
// 仅用于搜索框输入：避免每次按键都重建整个列表视图。
//
// 设计要点：
//   - schedule(s) 重置内部定时器，延迟 delay 后调用 fn(latest)
//   - 多次调用只会保留最后一次的字符串
//   - cancel() 取消尚未触发的回调
//   - fn 由调用方负责派发到 UI 线程（通常用 fyne.Do 包裹）
type debouncer struct {
	delay time.Duration

	mu     sync.Mutex
	timer  *time.Timer
	latest string
	fn     func(string)
}

func newDebouncer(delay time.Duration, fn func(string)) *debouncer {
	return &debouncer{delay: delay, fn: fn}
}

func (d *debouncer) schedule(s string) {
	d.mu.Lock()
	d.latest = s
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.delay, d.fire)
	d.mu.Unlock()
}

func (d *debouncer) fire() {
	d.mu.Lock()
	s := d.latest
	d.mu.Unlock()
	d.fn(s)
}

func (d *debouncer) cancel() {
	d.mu.Lock()
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = nil
	d.mu.Unlock()
}
