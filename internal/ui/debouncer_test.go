package ui

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestDebouncerCoalescesRapidCalls(t *testing.T) {
	var got atomic.Int32
	var last atomic.Value

	db := newDebouncer(20*time.Millisecond, func(s string) {
		last.Store(s)
		got.Add(1)
	})

	for i := 0; i < 10; i++ {
		db.schedule("g")
		db.schedule("go")
		db.schedule("goo")
		db.schedule("goog")
	}
	// 等待触发
	time.Sleep(60 * time.Millisecond)

	if n := got.Load(); n != 1 {
		t.Fatalf("expected 1 invocation, got %d", n)
	}
	if s := last.Load().(string); s != "goog" {
		t.Fatalf("expected last='goog', got %q", s)
	}
}

func TestDebouncerCancel(t *testing.T) {
	var fired atomic.Int32
	db := newDebouncer(20*time.Millisecond, func(string) { fired.Add(1) })
	db.schedule("x")
	db.cancel()
	time.Sleep(40 * time.Millisecond)
	if n := fired.Load(); n != 0 {
		t.Fatalf("expected 0 invocations after cancel, got %d", n)
	}
}

func TestDebouncerConcurrent(t *testing.T) {
	var fired atomic.Int32
	db := newDebouncer(5*time.Millisecond, func(string) { fired.Add(1) })
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				db.schedule("x")
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	time.Sleep(40 * time.Millisecond)
	// 多次 schedule 触发多次 timer reset；最终应仅触发 1 次（或极少）。
	if n := fired.Load(); n < 1 || n > 2 {
		t.Fatalf("expected 1-2 invocations, got %d", n)
	}
}
