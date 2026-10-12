package ui

import (
	"strings"
	"sync"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"mfa_reader/internal/model"
)

// codeRequest 描述生成一个验证码所需的全部参数。
// 字段均为可比较类型，因此可直接作为缓存键。
type codeRequest struct {
	secret    string
	digits    int
	period    int
	algorithm string
}

// requestFor 从账号提取生成参数，密钥取标准化后的值。
func requestFor(acc model.MFAAccount) codeRequest {
	return codeRequest{
		secret:    acc.NormalizeSecret(),
		digits:    acc.EffectiveDigits(),
		period:    acc.EffectivePeriod(),
		algorithm: acc.EffectiveAlgorithm(),
	}
}

// periodOf 返回 now 所属的周期序号；周期长度按账号各自的设置换算，切换即需重新出码。
func periodOf(now time.Time, period int) int64 {
	return now.Unix() / int64(normalizePeriod(period))
}

// remainingSeconds 返回当前周期剩余秒数（1~period）。
func remainingSeconds(now time.Time, period int) int {
	p := int64(normalizePeriod(period))
	return int(p - now.Unix()%p)
}

// progressRatio 将剩余秒数换算为进度条比例（0~1）。
func progressRatio(remain, period int) float64 {
	return float64(remain) / float64(normalizePeriod(period))
}

func normalizePeriod(period int) int {
	if period <= 0 {
		return model.DefaultPeriod
	}
	return period
}

// formatTOTPCode 按位数分组显示验证码：6 位为 "XXX XXX"，8 位为 "XXXX XXXX"，
// 其他长度保持原样；生成失败返回 "Error"。
func formatTOTPCode(code string, digits int, err error) string {
	if err != nil {
		return "Error"
	}
	switch {
	case digits == 6 && len(code) == 6:
		return code[:3] + " " + code[3:]
	case digits == 8 && len(code) == 8:
		return code[:4] + " " + code[4:]
	}
	return code
}

// codeGen 按「参数 + 周期序号」缓存生成的验证码：同一周期内重复渲染不再重复计算 HMAC。
// 供 UI 线程（首屏/搜索渲染）与刷新 goroutine 共用，是验证码生成的唯一入口。
type codeGen struct {
	mu    sync.Mutex
	cache map[codeRequest]codeEntry
}

type codeEntry struct {
	index int64
	text  string
}

// 缓存上限，防止长期运行/频繁增删导致内存无限增长。
const codeCacheLimit = 512

func newCodeGen() *codeGen {
	return &codeGen{cache: make(map[codeRequest]codeEntry)}
}

// Text 返回 req 在 now 所属周期内的格式化验证码。
func (g *codeGen) Text(req codeRequest, now time.Time) string {
	index := periodOf(now, req.period)

	g.mu.Lock()
	defer g.mu.Unlock()

	if e, ok := g.cache[req]; ok && e.index == index {
		return e.text
	}
	code, err := totp.GenerateCodeCustom(req.secret, now, totp.ValidateOpts{
		Period:    uint(normalizePeriod(req.period)),
		Digits:    otp.Digits(req.digits),
		Algorithm: algorithmOf(req.algorithm),
	})
	text := formatTOTPCode(code, req.digits, err)
	g.storeLocked(req, index, text)
	return text
}

// storeLocked 写入缓存；达到上限时优先淘汰已过期周期的条目，仍满则整体重置。
func (g *codeGen) storeLocked(req codeRequest, index int64, text string) {
	if len(g.cache) >= codeCacheLimit {
		for k, v := range g.cache {
			if v.index != index {
				delete(g.cache, k)
			}
		}
	}
	if len(g.cache) >= codeCacheLimit {
		g.cache = make(map[codeRequest]codeEntry)
	}
	g.cache[req] = codeEntry{index: index, text: text}
}

// algorithmOf 把数据文件中的算法名换算为库的枚举，未知取值回落到 SHA1。
func algorithmOf(name string) otp.Algorithm {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "SHA256":
		return otp.AlgorithmSHA256
	case "SHA512":
		return otp.AlgorithmSHA512
	default:
		return otp.AlgorithmSHA1
	}
}
