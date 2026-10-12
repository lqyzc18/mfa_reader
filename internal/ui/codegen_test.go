package ui

import (
	"fmt"
	"testing"
	"time"

	"mfa_reader/internal/model"
)

func TestFormatTOTPCode(t *testing.T) {
	cases := []struct {
		name   string
		code   string
		digits int
		err    error
		want   string
	}{
		{name: "六位分组显示", code: "123456", digits: 6, want: "123 456"},
		{name: "八位分组显示", code: "12345678", digits: 8, want: "1234 5678"},
		{name: "非标准长度保持原样", code: "1234567", digits: 7, want: "1234567"},
		{name: "空结果", code: "", digits: 6, want: ""},
		{name: "错误标记", code: "123456", digits: 6, err: fmt.Errorf("boom"), want: "Error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatTOTPCode(c.code, c.digits, c.err); got != c.want {
				t.Fatalf("formatTOTPCode(%q, %d, %v) = %q, want %q", c.code, c.digits, c.err, got, c.want)
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
		at := base.Add(time.Duration(c.offset) * time.Second)
		if got := remainingSeconds(at, model.DefaultPeriod); got != c.want {
			t.Fatalf("remainingSeconds(offset=%d) = %d, want %d", c.offset, got, c.want)
		}
	}

	// 60 秒周期：剩余时间按各自的周期长度换算。
	if got := remainingSeconds(base, 60); got != 20 {
		t.Fatalf("remainingSeconds(period=60) = %d, want 20", got)
	}
	if got := remainingSeconds(time.Unix(1000020, 0), 60); got != 60 {
		t.Fatalf("remainingSeconds at period start = %d, want 60", got)
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
	period := model.DefaultPeriod
	// unix 1000020 % 30 == 0，是周期起点。
	start := time.Unix(1000020, 0)
	if p := periodOf(start, period); p != 1000020/int64(period) {
		t.Fatalf("periodOf(start) = %d, want %d", p, 1000020/int64(period))
	}
	if r := remainingSeconds(start, period); r != period {
		t.Fatalf("remainingSeconds(start) = %d, want %d", r, period)
	}
	if g := progressRatio(remainingSeconds(start, period), period); g != 1.0 {
		t.Fatalf("progressRatio at period start = %v, want 1.0", g)
	}

	// 周期最后一秒：剩余 1 秒，且仍属于同一周期。
	last := time.Unix(int64(1000020+period-1), 0)
	if periodOf(last, period) != periodOf(start, period) {
		t.Fatal("last second should stay in the same period")
	}
	if r := remainingSeconds(last, period); r != 1 {
		t.Fatalf("remainingSeconds(last) = %d, want 1", r)
	}

	// 跨过边界即进入下一周期。
	if periodOf(last.Add(time.Second), period) != periodOf(start, period)+1 {
		t.Fatal("crossing the boundary should advance the period")
	}
}

func TestCodeGenCache(t *testing.T) {
	const secret = "JBSWY3DPEHPK3PXP"
	now := time.Unix(1000000, 0)
	req := requestFor(model.MFAAccount{Secret: secret})

	g := newCodeGen()
	first := g.Text(req, now)
	second := g.Text(req, now)
	if first != second {
		t.Fatalf("same period should return cached value: %q vs %q", first, second)
	}
	if len(first) != 7 || first[3] != ' ' { // "XXX XXX"
		t.Fatalf("unexpected code format: %q", first)
	}
	// 同密钥不同周期应得到不同值（30 秒后）。
	next := g.Text(req, now.Add(30*time.Second))
	if next == first {
		t.Fatal("different period should produce a different code")
	}

	// 不同密钥缓存互相独立。
	other := requestFor(model.MFAAccount{Secret: "ABCDEFGHIJKLMNOP"})
	if g.Text(other, now) == first {
		t.Fatal("different secret should not alias the cache")
	}
}

func TestCodeGenInvalidSecret(t *testing.T) {
	g := newCodeGen()
	req := requestFor(model.MFAAccount{Secret: "!!!not-base32!!!"})
	if got := g.Text(req, time.Unix(1000000, 0)); got != "Error" {
		t.Fatalf("invalid secret should produce Error, got %q", got)
	}
}

// 非标准参数必须真正参与出码：同一密钥在 8 位 / SHA256 / 60 秒下结果各不相同。
func TestCodeGenHonorsParams(t *testing.T) {
	const secret = "JBSWY3DPEHPK3PXP"
	now := time.Unix(1000000, 0)

	g := newCodeGen()
	base := g.Text(requestFor(model.MFAAccount{Secret: secret}), now)
	if len(base) != 7 {
		t.Fatalf("default code = %q, want 6 digits", base)
	}

	eight := g.Text(requestFor(model.MFAAccount{Secret: secret, Digits: 8}), now)
	if len(eight) != 9 { // "XXXX XXXX"
		t.Fatalf("8-digit code = %q, want 9 chars", eight)
	}

	sha256 := g.Text(requestFor(model.MFAAccount{Secret: secret, Algorithm: "SHA256"}), now)
	if sha256 == base {
		t.Fatal("SHA256 should produce a different code than SHA1")
	}

	// 60 秒周期：30 秒后仍在同一周期内，验证码不变。
	period60 := requestFor(model.MFAAccount{Secret: secret, Period: 60})
	first := g.Text(period60, time.Unix(1000020, 0))
	stillSame := g.Text(period60, time.Unix(1000050, 0))
	if first != stillSame {
		t.Fatalf("60s period should keep the same code within the period: %q vs %q", first, stillSame)
	}
	if changed := g.Text(period60, time.Unix(1000080, 0)); changed == first {
		t.Fatal("60s period should roll over after 60 seconds")
	}
}
