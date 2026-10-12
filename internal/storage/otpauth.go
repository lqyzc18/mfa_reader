package storage

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pquerna/otp"

	"mfa_reader/internal/model"
)

// 导入时的兜底名称：名称重复由 Store.Import 自动追加序号解决。
const (
	scannedAccountName = "扫码导入"
	unnamedAccountName = "未命名账号"
)

// ParseQRText 解析扫码结果：otpauth 链接，或纯 Base32 密钥。
func ParseQRText(text string) ([]model.MFAAccount, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(strings.ToLower(text), "otpauth://") {
		return ParseImportText(text)
	}
	acc := model.MFAAccount{AccountName: scannedAccountName, Secret: text}
	return normalizeImported([]model.MFAAccount{acc})
}

// ParseImportText 解析 otpauth URI、多行文本或明文 JSON 账号数组。
func ParseImportText(text string) ([]model.MFAAccount, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("导入内容为空")
	}

	if text[0] == '[' {
		var accs []model.MFAAccount
		if err := json.Unmarshal([]byte(text), &accs); err != nil {
			return nil, fmt.Errorf("JSON 解析失败: %w", err)
		}
		return normalizeImported(accs)
	}

	var accs []model.MFAAccount
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		acc, err := ParseOTPAuth(line)
		if err != nil {
			return nil, err
		}
		accs = append(accs, acc)
	}
	if len(accs) == 0 {
		return nil, fmt.Errorf("未找到可导入的账号")
	}
	return normalizeImported(accs)
}

// ParseOTPAuth 解析单条 otpauth://totp/... 链接，并保留其中的位数、周期与算法参数。
func ParseOTPAuth(uri string) (model.MFAAccount, error) {
	key, err := otp.NewKeyFromURL(strings.TrimSpace(uri))
	if err != nil {
		return model.MFAAccount{}, fmt.Errorf("无效的 otpauth 链接: %w", err)
	}
	if !strings.EqualFold(key.Type(), "totp") {
		return model.MFAAccount{}, fmt.Errorf("仅支持 TOTP，当前类型: %s", key.Type())
	}
	secret := strings.TrimSpace(key.Secret())
	if secret == "" {
		return model.MFAAccount{}, fmt.Errorf("otpauth 缺少 secret")
	}
	name := strings.TrimSpace(key.AccountName())
	if issuer := strings.TrimSpace(key.Issuer()); issuer != "" {
		if name == "" {
			name = issuer
		} else if !strings.Contains(name, issuer) {
			name = issuer + ":" + name
		}
	}
	if name == "" {
		name = unnamedAccountName
	}

	acc := model.MFAAccount{
		AccountName: name,
		Secret:      secret,
		Digits:      int(key.Digits()),
		Period:      int(key.Period()),
		Algorithm:   key.Algorithm().String(),
	}
	if err := acc.NormalizeParams(); err != nil {
		return model.MFAAccount{}, fmt.Errorf("无效的 otpauth 参数: %w", err)
	}
	return acc, nil
}

// FormatOTPAuth 把账号写成标准 otpauth URI，非标准参数（位数/周期/算法）一并写出。
func FormatOTPAuth(acc model.MFAAccount) string {
	label := url.PathEscape(acc.AccountName)
	q := url.Values{}
	q.Set("secret", acc.Secret)
	if !acc.IsDefaultParams() {
		if digits := acc.EffectiveDigits(); digits != model.DefaultDigits {
			q.Set("digits", strconv.Itoa(digits))
		}
		if period := acc.EffectivePeriod(); period != model.DefaultPeriod {
			q.Set("period", strconv.Itoa(period))
		}
		if algorithm := acc.EffectiveAlgorithm(); algorithm != model.DefaultAlgorithm {
			q.Set("algorithm", strings.ToLower(algorithm))
		}
	}
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// ExportOTPAuth 导出全部账号为多行 otpauth 文本。
func ExportOTPAuth(accounts []model.MFAAccount) string {
	lines := make([]string, 0, len(accounts))
	for _, a := range accounts {
		lines = append(lines, FormatOTPAuth(a))
	}
	return strings.Join(lines, "\n")
}

// normalizeImported 统一清洗导入内容：名称补默认、密钥标准化并校验、OTP 参数补全。
func normalizeImported(accs []model.MFAAccount) ([]model.MFAAccount, error) {
	out := make([]model.MFAAccount, 0, len(accs))
	for _, acc := range accs {
		acc.AccountName = strings.TrimSpace(acc.AccountName)
		if acc.AccountName == "" {
			acc.AccountName = unnamedAccountName
		}
		acc.Secret = acc.NormalizeSecret()
		if err := model.ValidateSecret(acc.Secret); err != nil {
			return nil, fmt.Errorf("账号「%s」: %w", acc.AccountName, err)
		}
		if err := acc.NormalizeParams(); err != nil {
			return nil, fmt.Errorf("账号「%s」: %w", acc.AccountName, err)
		}
		out = append(out, acc)
	}
	return out, nil
}
