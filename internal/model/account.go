package model

import (
	"encoding/base32"
	"fmt"
	"regexp"
	"strings"
)

const MinSecretLength = 16

// OTP 参数默认值：字段为零值时按标准默认处理（6 位 / 30 秒 / SHA1）。
const (
	DefaultDigits    = 6
	DefaultPeriod    = 30
	DefaultAlgorithm = "SHA1"
)

// 支持的取值：位数仅 6 位与 8 位（与 otp 库一致），周期取 1~600 秒的合理区间。
const (
	minPeriod = 1
	maxPeriod = 600
)

var (
	base32Regex = regexp.MustCompile(`^[A-Z2-7]+$`)

	supportedAlgorithms = map[string]bool{
		"SHA1":   true,
		"SHA256": true,
		"SHA512": true,
	}
)

type MFAAccount struct {
	AccountName string `json:"accountName"`
	Secret      string `json:"secret"`
	Pinned      bool   `json:"pinned,omitempty"`
	// Digits/Period/Algorithm 为 otpauth 链接的可选参数，零值表示使用标准默认。
	Digits    int    `json:"digits,omitempty"`
	Period    int    `json:"period,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
}

// secretReplacer 去除密钥中的分隔符（空格与横线），包级复用避免每次调用重复构建。
var secretReplacer = strings.NewReplacer(" ", "", "-", "")

func (a *MFAAccount) NormalizeSecret() string {
	s := secretReplacer.Replace(strings.ToUpper(strings.TrimSpace(a.Secret)))
	return strings.TrimRight(s, "=")
}

// ValidateSecret 校验标准化后的密钥是否为合法 Base32，并满足最小长度。
func ValidateSecret(normalized string) error {
	if normalized == "" {
		return fmt.Errorf("密钥不能为空")
	}
	if !base32Regex.MatchString(normalized) {
		return fmt.Errorf("密钥包含无效字符，仅支持 A-Z 和 2-7")
	}
	if len(normalized) < MinSecretLength {
		return fmt.Errorf("密钥长度不足 %d 位，请检查是否输入正确", MinSecretLength)
	}
	if err := decodeBase32(normalized); err != nil {
		return fmt.Errorf("密钥长度不是合法的 Base32 编码，请检查是否输入完整")
	}
	return nil
}

// decodeBase32 按 otp 库的方式解码密钥：先把长度补齐到 8 的整数倍，再用带填充的标准解码。
// 这一步能拦下字符集合法但无法解码的密钥（长度落在 8n+1、8n+3、8n+6 上），
// 否则它们会被保存进列表，却永远只能显示「Error」。
func decodeBase32(secret string) error {
	padded := secret
	if rest := len(padded) % 8; rest != 0 {
		padded += strings.Repeat("=", 8-rest)
	}
	_, err := base32.StdEncoding.DecodeString(padded)
	return err
}

// NormalizeParams 校验并补全 OTP 参数：零值按标准默认填充，取值非法时返回错误且不改动账号。
func (a *MFAAccount) NormalizeParams() error {
	digits := a.Digits
	if digits == 0 {
		digits = DefaultDigits
	}
	period := a.Period
	if period == 0 {
		period = DefaultPeriod
	}
	algorithm := strings.ToUpper(strings.TrimSpace(a.Algorithm))
	if algorithm == "" {
		algorithm = DefaultAlgorithm
	}

	switch digits {
	case 6, 8:
	default:
		return fmt.Errorf("不支持的验证码位数 %d，仅支持 6 位或 8 位", digits)
	}
	if period < minPeriod || period > maxPeriod {
		return fmt.Errorf("不支持的刷新周期 %d 秒，支持 %d~%d 秒", period, minPeriod, maxPeriod)
	}
	if !supportedAlgorithms[algorithm] {
		return fmt.Errorf("不支持的哈希算法 %s，仅支持 SHA1、SHA256、SHA512", algorithm)
	}

	a.Digits, a.Period, a.Algorithm = digits, period, algorithm
	return nil
}

// IsDefaultParams 判断账号是否使用标准参数（6 位 / 30 秒 / SHA1）。
func (a *MFAAccount) IsDefaultParams() bool {
	return (a.Digits == 0 || a.Digits == DefaultDigits) &&
		(a.Period == 0 || a.Period == DefaultPeriod) &&
		(a.Algorithm == "" || strings.EqualFold(a.Algorithm, DefaultAlgorithm))
}

// EffectiveDigits 返回验证码位数，未设置时为标准默认。
func (a *MFAAccount) EffectiveDigits() int {
	if a.Digits == 0 {
		return DefaultDigits
	}
	return a.Digits
}

// EffectivePeriod 返回刷新周期（秒），未设置时为标准默认。
func (a *MFAAccount) EffectivePeriod() int {
	if a.Period == 0 {
		return DefaultPeriod
	}
	return a.Period
}

// EffectiveAlgorithm 返回大写哈希算法名，未设置时为标准默认。
func (a *MFAAccount) EffectiveAlgorithm() string {
	if strings.TrimSpace(a.Algorithm) == "" {
		return DefaultAlgorithm
	}
	return strings.ToUpper(strings.TrimSpace(a.Algorithm))
}

// ParamsSummary 返回非标准参数的简要描述（如 "SHA256 / 60s / 8位"），标准参数返回空串。
func (a *MFAAccount) ParamsSummary() string {
	if a.IsDefaultParams() {
		return ""
	}
	parts := make([]string, 0, 3)
	if alg := a.EffectiveAlgorithm(); alg != DefaultAlgorithm {
		parts = append(parts, alg)
	}
	if period := a.EffectivePeriod(); period != DefaultPeriod {
		parts = append(parts, fmt.Sprintf("%ds", period))
	}
	if digits := a.EffectiveDigits(); digits != DefaultDigits {
		parts = append(parts, fmt.Sprintf("%d位", digits))
	}
	return strings.Join(parts, " / ")
}

// MatchName 判断账号名是否匹配搜索词（不区分大小写）。
func (a *MFAAccount) MatchName(filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(a.AccountName), strings.ToLower(filter))
}
