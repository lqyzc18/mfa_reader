package ui

import (
	"fmt"
	"testing"
	"time"
)

func TestFormatTOTPCode(t *testing.T) {
	cases := []struct {
		name string
		code string
		err  error
		want string
	}{
		{name: "six digits spaced", code: "123456", want: "123 456"},
		{name: "non-standard length kept", code: "1234567", want: "1234567"},
		{name: "empty result", code: "", want: ""},
		{name: "error marker", code: "123456", err: fmt.Errorf("boom"), want: "Error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatTOTPCode(c.code, c.err); got != c.want {
				t.Fatalf("formatTOTPCode(%q, %v) = %q, want %q", c.code, c.err, got, c.want)
			}
		})
	}
}

func TestRemainingSeconds(t *testing.T) {
	base := time.Unix(1000000, 0) // 1000000 % 30 == 10
	cases := []struct {
		offset int64
		want   int
	}{
		{offset: 0, want: 20},
		{offset: 9, want: 11},
		{offset: 10, want: 10},
		{offset: 20, want: 30}, // 周期边界
		{offset: 29, want: 21},
	}
	for _, c := range cases {
		if got := remainingSeconds(base.Add(time.Duration(c.offset) * time.Second)); got != c.want {
			t.Fatalf("remainingSeconds(offset=%d) = %d, want %d", c.offset, got, c.want)
		}
	}
}

func TestUntilNextSecond(t *testing.T) {
	cases := []struct {
		name string
		nsec int
		want time.Duration
	}{
		{name: "整秒", nsec: 0, want: time.Second},
		{name: "刚过整秒", nsec: 1e6, want: 999 * time.Millisecond},
		{name: "半秒", nsec: 5e8, want: 500 * time.Millisecond},
		{name: "临近下一秒", nsec: 999999999, want: time.Nanosecond},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			now := time.Unix(1000000, int64(c.nsec))
			if got := untilNextSecond(now); got != c.want {
				t.Fatalf("untilNextSecond(nsec=%d) = %v, want %v", c.nsec, got, c.want)
			}
			// 对齐后的唤醒点必须恰好落在整秒上。
			if next := now.Add(untilNextSecond(now)); next.Nanosecond() != 0 {
				t.Fatalf("aligned wakeup has nsec=%d, want 0", next.Nanosecond())
			}
		})
	}
}

func TestPeriodAndProgress(t *testing.T) {
	// unix 1000020 % 30 == 0，是周期起点。
	start := time.Unix(1000020, 0)
	if p := periodOf(start); p != 1000020/totpPeriod {
		t.Fatalf("periodOf(start) = %d, want %d", p, 1000020/totpPeriod)
	}
	if r := remainingSeconds(start); r != totpPeriod {
		t.Fatalf("remainingSeconds(start) = %d, want %d", r, totpPeriod)
	}
	if g := progressRatio(remainingSeconds(start)); g != 1.0 {
		t.Fatalf("progressRatio at period start = %v, want 1.0", g)
	}

	// 周期最后一秒：剩余 1 秒，且仍属于同一周期。
	last := time.Unix(1000020+totpPeriod-1, 0)
	if periodOf(last) != periodOf(start) {
		t.Fatal("last second should stay in the same period")
	}
	if r := remainingSeconds(last); r != 1 {
		t.Fatalf("remainingSeconds(last) = %d, want 1", r)
	}

	// 跨过边界即进入下一周期。
	if periodOf(last.Add(time.Second)) != periodOf(start)+1 {
		t.Fatal("crossing the boundary should advance the period")
	}
}

func TestCodeGenCache(t *testing.T) {
	const secret = "JBSWY3DPEHPK3PXP"
	now := time.Unix(1000000, 0)

	g := newCodeGen()
	first := g.Text(secret, now)
	second := g.Text(secret, now)
	if first != second {
		t.Fatalf("same period should return cached value: %q vs %q", first, second)
	}
	if len(first) != 7 || first[3] != ' ' { // "XXX XXX"
		t.Fatalf("unexpected code format: %q", first)
	}
	// 同密钥不同周期应得到不同值（30 秒后）。
	next := g.Text(secret, now.Add(30*time.Second))
	if next == first {
		t.Fatal("different period should produce a different code")
	}

	// 不同密钥缓存互相独立。
	if g.Text("ABCDEFGHIJKLMNOP", now) == first {
		t.Fatal("different secret should not alias the cache")
	}
}

func TestCodeGenInvalidSecret(t *testing.T) {
	g := newCodeGen()
	if got := g.Text("!!!not-base32!!!", time.Unix(1000000, 0)); got != "Error" {
		t.Fatalf("invalid secret should produce Error, got %q", got)
	}
}
