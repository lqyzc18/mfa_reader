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
