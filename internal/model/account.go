package model

import (
	"fmt"
	"regexp"
	"strings"
)

const MinSecretLength = 16

var base32Regex = regexp.MustCompile(`^[A-Z2-7]+$`)

type MFAAccount struct {
	AccountName string `json:"accountName"`
	Secret      string `json:"secret"`
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
		return fmt.Errorf("密钥长度不足，请检查是否输入正确")
	}
	return nil
}

// MatchName 判断账号名是否匹配搜索词（不区分大小写）。
func (a *MFAAccount) MatchName(filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(a.AccountName), strings.ToLower(filter))
}
