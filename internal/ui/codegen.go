package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/pquerna/otp/totp"
)

// formatTOTPCode 将 6 位验证码格式化为 "XXX XXX"，非标准长度保持原样，错误返回 "Error"。
func formatTOTPCode(code string, err error) string {
	if err != nil {
		return "Error"
	}
	if len(code) == 6 {
		return fmt.Sprintf("%s %s", code[:3], code[3:])
	}
	return code
}

// remainingSeconds 返回当前 30 秒周期剩余秒数（1~30）。
func remainingSeconds(now time.Time) int {
	return int(30 - (now.Unix() % 30))
}

// codeGen 按 (密钥, 周期) 缓存生成的验证码：同一周期内重复渲染不再重复计算 HMAC。
// 供 UI 线程（初始/搜索渲染）与刷新 goroutine 共用，是验证码生成的唯一入口。
type codeGen struct {
	mu    sync.Mutex
	cache map[string]codeEntry
}

type codeEntry struct {
	period int64
	text   string
}

// 缓存上限，防止长期运行/频繁增删导致内存无限增长。
const codeCacheLimit = 512

func newCodeGen() *codeGen {
	return &codeGen{cache: make(map[string]codeEntry)}
}

// Text 返回 secret 在 now 所在周期内的格式化验证码。
func (g *codeGen) Text(secret string, now time.Time) string {
	period := now.Unix() / 30

	g.mu.Lock()
	defer g.mu.Unlock()

	if e, ok := g.cache[secret]; ok && e.period == period {
		return e.text
	}
	code, err := totp.GenerateCode(secret, now)
	text := formatTOTPCode(code, err)

	if len(g.cache) >= codeCacheLimit {
		g.cache = make(map[string]codeEntry)
	}
	g.cache[secret] = codeEntry{period: period, text: text}
	return text
}
