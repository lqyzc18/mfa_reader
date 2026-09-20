package storage

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/pquerna/otp"

	"mfa_reader/internal/model"
)

// ParseQRText 解析扫码结果：otpauth 链接，或纯 Base32 密钥。
func ParseQRText(text string) ([]model.MFAAccount, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(strings.ToLower(text), "otpauth://") {
		return ParseImportText(text)
	}
	acc := model.MFAAccount{AccountName: "扫码导入", Secret: text}
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

// ParseOTPAuth 解析单条 otpauth://totp/... 链接。
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
		name = "未命名账号"
	}
	return model.MFAAccount{AccountName: name, Secret: secret}, nil
}

// FormatOTPAuth 把账号写成标准 otpauth URI。
func FormatOTPAuth(acc model.MFAAccount) string {
	label := url.PathEscape(acc.AccountName)
	q := url.Values{}
	q.Set("secret", acc.Secret)
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

func normalizeImported(accs []model.MFAAccount) ([]model.MFAAccount, error) {
	out := make([]model.MFAAccount, 0, len(accs))
	for _, acc := range accs {
		acc.AccountName = strings.TrimSpace(acc.AccountName)
		if acc.AccountName == "" {
			acc.AccountName = "未命名账号"
		}
		acc.Secret = acc.NormalizeSecret()
		if err := model.ValidateSecret(acc.Secret); err != nil {
			return nil, fmt.Errorf("账号「%s」: %w", acc.AccountName, err)
		}
		out = append(out, acc)
	}
	return out, nil
}
